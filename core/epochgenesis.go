package core

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	 posUtil "github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"encoding/binary"
	"bytes"
	"github.com/wanchain/go-wanchain/common"
	"sync"
	"github.com/wanchain/go-wanchain/core/state"
	"math/big"
	"encoding/hex"
	"github.com/wanchain/go-wanchain/crypto/sha3"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"fmt"
)


func (bc *BlockChain) updateReOrg(epochId uint64, length uint64) {
	reOrgDb := posdb.GetDbByName("forkdb")
	if reOrgDb == nil {
		reOrgDb = posdb.NewDb("forkdb")
	}

	numberBytes, _ := reOrgDb.Get(epochId, "reorgNumber")

	num := uint64(0)
	if numberBytes != nil {
		num = binary.BigEndian.Uint64(numberBytes) + 1
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, num)

	reOrgDb.Put(epochId, "reorgNumber", b)

	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, length)
	reOrgDb.Put(epochId, "reorgLength", b)
}

func (bc *BlockChain) updateFork(epochId uint64) {
	reOrgDb := posdb.GetDbByName("forkdb")
	if reOrgDb == nil {
		reOrgDb = posdb.NewDb("forkdb")
	}

	numberBytes, _ := reOrgDb.Get(0, "forkNumber")

	num := uint64(0)
	if numberBytes != nil {
		num = binary.BigEndian.Uint64(numberBytes) + 1
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, num)

	reOrgDb.Put(epochId, "forkNumber", b)
}


type RbLeadersSelInt interface {
	GetEpochLastBlkNumber(epochId uint64) uint64
	GetRBProposerGroup(epochID uint64) []vm.Leader
	GetEpochLeaders(epochID uint64) [][]byte
}

type SlLeadersSelInt interface {

	ValidateBody(block *types.Block) error

	ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error

	GetInfoFromHeadExtra(epochID uint64, input []byte) ([]*big.Int, []*ecdsa.PublicKey, error)

}

type EpochGenesisBlock struct {
	bc 						*BlockChain
	rbLeaderSelector   		RbLeadersSelInt
	slotLeaderSelector 		SlLeadersSelInt
	epochGenesisCh 			chan uint64
	lastEpochId				uint64

	epochGenDb 				*posdb.Db
	epgSetmu  				sync.RWMutex
	epgGenmu  				sync.RWMutex
}

func NewEpochGenesisBlock(bc *BlockChain) *EpochGenesisBlock {

	f := &EpochGenesisBlock{}
	f.bc = bc
	f.epochGenesisCh = make(chan uint64,1)
	f.lastEpochId = 0

	f.epochGenDb = posdb.NewDb("epochGendb")


	return f
}

func (f *EpochGenesisBlock) GetBlockEpochIdAndSlotId(header *types.Header) (blkEpochId uint64, blkSlotId uint64, err error) {

	blkTd := header.Difficulty.Uint64()

	blkEpochId = (blkTd >> 32)
	blkSlotId = ((blkTd & 0xffffffff) >> 8)

	return
}

func (f *EpochGenesisBlock) tryAppendStakerInfoBytes(infos *[][]byte, pub []byte, epochid uint64)  error {
	addr := common.BytesToAddress(crypto.Keccak256(pub[1:])[12:])
	stakerBytes, err := posUtil.TryGetAndSaveStakerInfo(epochid, addr)
	if err != nil {
		return err
	}
	*infos = append(*infos, *stakerBytes)
	return nil
}

func (f *EpochGenesisBlock) SelfGenerateEpochGenesis(blk *types.Block){

	curEpid,_,err := f.GetBlockEpochIdAndSlotId(blk.Header())
	if err != nil {
		return
	}

	f.UpdateEpochGenesis(curEpid)
}



func (f *EpochGenesisBlock) GenerateEpochGenesis(epochid uint64) (*types.EpochGenesis,error) {

	fmt.Println("generate epg",epochid)
	epg := f.GetEpochGenesis(epochid)
	if epg != nil {
		return epg,nil
	} else {
		return f.generateChainedEpochGenesis(epochid)
	}
}


func (f *EpochGenesisBlock) generateChainedEpochGenesis(epochid uint64) (*types.EpochGenesis,error){
	//it is the first block of this epoch
	var rb 			*big.Int
	var blk 		*types.Block
	var epgPre		*types.EpochGenesis
	var epg			*types.EpochGenesis

	curEpid,_,err := f.GetBlockEpochIdAndSlotId(f.bc.currentBlock.Header())

	if curEpid < epochid || err !=nil || epochid == 0{
		return nil , errors.New("error epochid")
	}


	epgPre = f.GetEpochGenesis(epochid - 1)
	if epgPre == nil {
		//start from ep1
		for i := uint64(1);i <= epochid; i++ {

			epg := f.GetEpochGenesis(i)
			if epg != nil {
				continue
			}

			if i == 1 {
				rb = big.NewInt(1)
				epgPre, err = f.generateEpochGenesis(0, nil, rb.Bytes(),common.Hash{})
				if err != nil {
					return nil, err
				}

				err = f.SetEpochGenesis(epgPre)
				if err != nil {
					return nil, err
				}

			} else {
				epgPre = f.GetEpochGenesis(i - 1)

			}

			rb, blk = f.getEpochRandomAndPreEpLastBlk(i)

			if epgPre == nil {
				return nil, errors.New("pre epg is nil")
			}

			epg, err = f.generateEpochGenesis(i, blk, rb.Bytes(),epgPre.GenesisBlkHash)
			if err != nil {
				return nil, err
			}

			err = f.SetEpochGenesis(epg)
			if err != nil {
				return nil, err
			}

		}

	} else {

		rb, blk = f.getEpochRandomAndPreEpLastBlk(epochid)
		epg, err = f.generateEpochGenesis(epochid, blk, rb.Bytes(), epgPre.GenesisBlkHash)
		if err != nil {
			return nil, err
		}

		err = f.SetEpochGenesis(epg)
		if err != nil {
			return nil, err
		}
	}

	return epg,nil
}

func (f *EpochGenesisBlock) getEpochRandomAndPreEpLastBlk(epochid uint64)(*big.Int,*types.Block){
	blkNum := f.rbLeaderSelector.GetEpochLastBlkNumber(epochid)

	preEpLastblk := f.bc.GetBlockByNumber(blkNum - 1)

	stateDb, err := f.bc.StateAt(preEpLastblk.Root())
	if err != nil {
		return nil, nil
	}

	rb := vm.GetR(stateDb, epochid)

	return rb,preEpLastblk
}


func (f *EpochGenesisBlock) generateEpochGenesis(epochid uint64,lastblk *types.Block,rb []byte,preHash common.Hash) (*types.EpochGenesis, error) {


	epGen := &types.EpochGenesis{}

	epGen.ProtocolMagic = []byte("wanchainpos")

	epGen.EpochId = epochid

	if lastblk == nil {
		epGen.PreEpochLastBlkHash = common.Hash{}
	} else {
		epGen.PreEpochLastBlkHash = lastblk.Hash()
	}

	epGen.EpochLeaders = f.rbLeaderSelector.GetEpochLeaders(epochid)

	epGen.StakerInfos = make([][]byte, 0)
	for _, epLeader := range epGen.EpochLeaders {
		err := f.tryAppendStakerInfoBytes(&epGen.StakerInfos, epLeader, epochid)
		if err != nil {
			return nil, err
		}
	}
	epGen.RBLeadersSec256 = make([][]byte, 0)
	epGen.RBLeadersBn256 = make([][]byte, 0)
	rbleaders := f.rbLeaderSelector.GetRBProposerGroup(epochid)

	if len(rbleaders) != 0 {
		for _, rbl := range rbleaders {
			epGen.RBLeadersSec256 = append(epGen.RBLeadersSec256, rbl.PubSec256)
			epGen.RBLeadersBn256 = append(epGen.RBLeadersBn256,rbl.PubBn256)
			err := f.tryAppendStakerInfoBytes(&epGen.StakerInfos, rbl.PubSec256, epochid)
			if err != nil {
				return nil, err
			}
		}
	}

	epGen.SlotLeaders = f.getAllSlotLeaders(epochid)
	if len(epGen.SlotLeaders) != 0 {
		for _, pkBytes := range epGen.SlotLeaders  {
			err := f.tryAppendStakerInfoBytes(&epGen.StakerInfos, pkBytes, epochid)
			if err != nil {
				return nil, err
			}

		}
	}

	epGen.PreEpochGenHash = preHash

	epGen.GenesisBlkHash = common.Hash{}

	byteVal, err := json.Marshal(epGen)

	log.Info("generated epochGenesis data","",common.ToHex(byteVal))

	if err != nil {
		log.Debug("Failed to marshal epoch genesis data", "err", err)
		return nil, err
	}

	epGen.GenesisBlkHash = crypto.Keccak256Hash(byteVal)

	return epGen, nil
}


func (f *EpochGenesisBlock) preVerifyEpochGenesis(epGen *types.EpochGenesis) bool {
	var epPre	*types.EpochGenesis
	var err 	error

	if epGen.EpochId < 1{
		return false
	}

	epPre = f.GetEpochGenesis(epGen.EpochId - 1)
	if epPre == nil {
		return false
	}


	res := (epGen.PreEpochGenHash == epPre.GenesisBlkHash)
	if !res {
		return false
	}

	res = bytes.Equal(epGen.ProtocolMagic,[]byte("wanchainpos"))
	if !res {
		return false
	}

	if len(epGen.RBLeadersSec256)==0 || len(epGen.SlotLeaders)< posconfig.SlotCount || len(epGen.EpochLeaders)==0 {
		return false
	}

	epGenNew := &types.EpochGenesis{}

	epGenNew.ProtocolMagic = []byte("wanchainpos")
	epGenNew.PreEpochLastBlkHash = epGen.PreEpochLastBlkHash
	epGenNew.EpochId = epGen.EpochId
	epGenNew.RBLeadersSec256 = epGen.RBLeadersSec256
	epGenNew.RBLeadersBn256 = epGen.RBLeadersBn256
	epGenNew.EpochLeaders = epGen.EpochLeaders
	epGenNew.StakerInfos = epGen.StakerInfos	
	epGenNew.SlotLeaders = epGen.SlotLeaders
	epGenNew.PreEpochGenHash = epGen.PreEpochGenHash
	epGenNew.GenesisBlkHash = common.Hash{}


	byteVal, err := json.Marshal(epGenNew)
	log.Info("verify genesis data","",common.ToHex(byteVal))

	if err != nil {
		log.Debug("Failed to marshal epoch genesis data", "err", err)
		return false
	}

	epGenNew.GenesisBlkHash = crypto.Keccak256Hash(byteVal)

	res = (epGenNew.GenesisBlkHash == epGen.GenesisBlkHash)

	return res
}

//updated specified epoch genesis
func (f *EpochGenesisBlock) UpdateEpochGenesis(epochID uint64) {
	if epochID != f.lastEpochId {

		if epochID > 2 {
			f.GenerateEpochGenesis(epochID - 1)
		}

		f.lastEpochId = epochID
	}
}

func (f *EpochGenesisBlock) GetLastBlkInPreEpoch( blk *types.Block) *types.Block {
	epochID := blk.Header().Difficulty.Uint64() >> 32
	blkNUm := posUtil.GetEpochBlock(epochID - 1)
	return f.bc.GetBlockByNumber(blkNUm)
}

func (f *EpochGenesisBlock) IsExistEpochGenesis(epochid uint64) bool {

	val, err := f.epochGenDb.Get(epochid, "epochgenesis")
	if err != nil || val == nil {
		return false
	}
	return true

}


func (f *EpochGenesisBlock) getAllSlotLeaders(epochID uint64) ([][]byte){
	startBlkNum := uint64(0)
	if epochID > 0 {
		startBlkNum = posUtil.GetEpochBlock(epochID - 1) + 1
	}

	endBlkNum := posUtil.GetEpochBlock(epochID)
	slotLeaders :=  make([][]byte, 0)
	for i := startBlkNum;i <= endBlkNum;i++ {
		header := f.bc.GetHeaderByNumber(i)
		signer, err := f.recoverSigner(header)
		if err != nil {
			log.Error(err.Error())
			break
		}

		slotLeaders = append(slotLeaders,signer[:])
	}

	return slotLeaders

}

func (f *EpochGenesisBlock) SetEpochGenesis(epochgen *types.EpochGenesis) error {
	f.epgSetmu.Lock()
	defer f.epgSetmu.Unlock()

	if epochgen == nil {
		return errors.New("inputing epoch genesis is nil")
	}

	if epochgen.EpochId > 0 {
		res := f.preVerifyEpochGenesis(epochgen)
		if !res {
			return errors.New("epoch genesis preverify is failed")
		}
	}


	val,err := rlp.EncodeToBytes(epochgen)
	if err != nil {
		return err
	}

	_,err = f.epochGenDb.Put(epochgen.EpochId,"epochgenesis",val)
	if err != nil {
		return err
	}

	err = f.saveToPosDb(epochgen)
	if err != nil {
		return err
	}

	log.Info("successfully input epochGenesis","",epochgen.EpochId)

	return nil
}

func (f *EpochGenesisBlock) GetEpochGenesis(epochid uint64) *types.EpochGenesis{

	val, err := f.epochGenDb.Get(epochid, "epochgenesis")
	if err != nil || val == nil {
		return nil
	}

	epochGen := new(types.EpochGenesis)
	err = rlp.DecodeBytes(val,epochGen)
	if err != nil {
		log.Debug(err.Error())
		return nil
	}

	return epochGen
}

func (f *EpochGenesisBlock) ValidateBody(block *types.Block) error {
    fmt.Println("begin EpochGenesisBlock ValidateBody")
	extraSeal := 65
	header := block.Header()
	blkTd := block.Difficulty().Uint64()
	epochID := (blkTd >> 32)
	slotID := ((blkTd & 0xffffffff) >> 8)

	if epochID == 0 {
		return nil
	}


	_, proofMeg, err := f.slotLeaderSelector.GetInfoFromHeadExtra(epochID, header.Extra[:len(header.Extra)-extraSeal])

	if err != nil {
		log.Error("Can not GetInfoFromHeadExtra, verify failed", "error", err.Error())
		return errors.New("Can not GetInfoFromHeadExtra, verify failed")
	}

	log.Debug("verifySeal GetInfoFromHeadExtra", "pk", hex.EncodeToString(crypto.FromECDSAPub(proofMeg[0])))
	pk := proofMeg[0]

	signer, err := f.recoverSigner(header)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if bytes.Equal(signer,crypto.FromECDSAPub(pk)) {
		log.Error("Pk signer verify failed in verifySeal", "number", block.NumberU64(),
			"epochID", epochID, "slotID", slotID, "signer", common.ToHex(signer), "PkAddress", crypto.PubkeyToAddress(*pk).Hex())
		return errors.New("failed to verify signer for fast synch verify")
	}

	headerPkval := crypto.FromECDSAPub(pk)

	epg := f.GetEpochGenesis(epochID)
	if epg == nil {
		return errors.New("failed to get epoch genesis")
	}

	if !bytes.Equal(epg.SlotLeaders[slotID],headerPkval) {
		log.Error("Pk signer verify with epoch genesis", "number", block.NumberU64(),
			"epochID", epochID, "slotID", slotID, "signer", common.ToHex(signer), "PkAddress", crypto.PubkeyToAddress(*pk).Hex())
		return errors.New("failed to verify signer with epoch genesis")
	}

	fmt.Println("end EpochGenesisBlock ValidateBody")
	return nil

}

func (f *EpochGenesisBlock) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}

func (f *EpochGenesisBlock) recoverSigner(header *types.Header) ([]byte,error) {
	signature := header.Extra[len(header.Extra)-extraSeal:]

	log.Debug("signature", "hex", hex.EncodeToString(signature))

	log.Debug("sigHash(header)", "Bytes", hex.EncodeToString(sigHash(header).Bytes()))

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return nil, err
	}

	log.Debug("pubkey in ecrecover", "pk", hex.EncodeToString(pubkey))

	return pubkey,nil


}

func sigHash(header *types.Header) (hash common.Hash) {

	hasher := sha3.NewKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-extraSeal], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	})

	hasher.Sum(hash[:0])
	return hash
}

func (f *EpochGenesisBlock) saveToPosDb(epochgen *types.EpochGenesis) error {
	elDb := posdb.NewDb(posconfig.EpLocalDB)
	if elDb == nil {
		return errors.New("create epoch local db error")
	}
	count := len(epochgen.EpochLeaders)
	// TODO: 1. check valid public key, 2. check exist? 3. put failed? 4. how to rollback
	for i := 0; i < count; i++ {
		tmp := posdb.Proposer{
		//PubSec256:make([]byte,0),
		//PubBn256:nil,
		//Probabilities: big.NewInt(0),
		}
		tmp.PubSec256 = epochgen.EpochLeaders[i]
		v, _ := rlp.EncodeToBytes(&tmp)
		elDb.PutWithIndex(epochgen.EpochId, uint64(i), "", v)
	}
	rbDb := posdb.NewDb(posconfig.RbLocalDB)
	if rbDb == nil {
		return errors.New("create rb local db error")
	}
	count = len(epochgen.RBLeadersSec256)
	for i := 0; i < count; i++ {
		tmp := posdb.Proposer{}
		tmp.PubBn256 = epochgen.RBLeadersBn256[i]
		tmp.PubSec256 = epochgen.RBLeadersSec256[i]
		v, _ := rlp.EncodeToBytes(&tmp)
		rbDb.PutWithIndex(epochgen.EpochId, uint64(i), "", v)
	}

	stDb := posdb.NewDb(posconfig.StakerLocalDB)
	if stDb == nil {
		return errors.New("create stakeholder local db error")
	}
	count = len(epochgen.StakerInfos)
	for _, pkBytes := range epochgen.StakerInfos {
		staker := vm.StakerInfo{}
		_ = rlp.DecodeBytes(pkBytes, &staker)
		stDb.PutWithIndex(epochgen.EpochId, 0, common.ToHex(staker.Address[:]), pkBytes)
	}

	return nil
}


