package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"math/big"
	"strconv"
	"sync"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/sha3"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/util"
	posUtil "github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
)

func (bc *BlockChain) updateReOrg(epochid uint64, slotid uint64, length uint64) {

	reOrgDb := posdb.GetDbByName(posconfig.ReorgLocalDB)
	if reOrgDb == nil {
		reOrgDb = posdb.NewDb(posconfig.ReorgLocalDB)
	}

	numberBytes, _ := reOrgDb.Get(epochid, "reorgNumber")

	num := uint64(0)
	if numberBytes != nil {
		num = binary.BigEndian.Uint64(numberBytes) + 1
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, num)

	reOrgDb.Put(epochid, "reorgNumber", b)

	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, length)
	reOrgDb.Put(epochid, "reorgLength", b)
}

type RbLeadersSelInt interface {
	GetEpochLastBlkNumber(epochId uint64) uint64
	GetEpochEndBlkNumber(epochId uint64) uint64
	GetRBProposerGroup(epochID uint64) []vm.Leader
	GetEpochLeaders(epochID uint64) [][]byte
}

type SlLeadersSelInt interface {
	ValidateBody(block *types.Block) error

	ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error

	GetInfoFromHeadExtra(epochID uint64, input []byte) ([]*big.Int, []*ecdsa.PublicKey, error)
}

type EpochGenesisBlock struct {
	bc                 *BlockChain
	rbLeaderSelector   RbLeadersSelInt
	slotLeaderSelector SlLeadersSelInt
	epochGenesisCh 			chan *types.EpochSync
	lastEpochId        uint64

	epochGenDb *posdb.Db
	epgSetmu   sync.RWMutex
	epgGenmu   sync.RWMutex
}

func NewEpochGenesisBlock(bc *BlockChain) *EpochGenesisBlock {

	f := &EpochGenesisBlock{}
	f.bc = bc
	f.epochGenesisCh = make(chan *types.EpochSync,1)
	f.lastEpochId = 0

	f.epochGenDb = posdb.NewDb("epochGendb")

	return f
}

var whiteMap map[common.Address] bool
// epoch => slotId -> signer address
var slotLeaderCache map[uint64]map[uint64]common.Address

func init() {
	whiteMap = make(map[common.Address] bool)
	slotLeaderCache = make(map[uint64]map[uint64] common.Address)

	// init white list
	for i:=0; i<len(posconfig.WhiteListOrig); i++ {
		pk := crypto.ToECDSAPub(hexutil.MustDecode(posconfig.WhiteListOrig[i]))
		addr := crypto.PubkeyToAddress(*pk)
		whiteMap[addr]= true
	}
}

func (f *EpochGenesisBlock) GetBlockEpochIdAndSlotId(header *types.Header) (blkEpochId uint64, blkSlotId uint64, err error) {

	blkTd := header.Difficulty.Uint64()

	blkEpochId = (blkTd >> 32)
	blkSlotId = ((blkTd & 0xffffffff) >> 8)

	return
}

func (f *EpochGenesisBlock) SelfGenerateEpochGenesis(blk *types.Block) {

	curEpid, _, err := f.GetBlockEpochIdAndSlotId(blk.Header())
	if err != nil {
		return
	}

	go f.UpdateEpochGenesis(curEpid)
}

func (f *EpochGenesisBlock) GenerateEpochGenesis(epochid uint64, isEnd bool) (*types.EpochGenesis,error) {

	log.Debug("generate epg", "", epochid)
	epg := f.GetEpochGenesis(epochid)
	if epg != nil {
		return epg, nil
	} else {
		return f.generateChainedEpochGenesis(epochid, isEnd)
	}
}

func (f *EpochGenesisBlock) DoGenerateEpochGenesis(epochid uint64, isEnd bool) (*types.EpochGenesis,error) {
	epgPre := f.GetEpochGenesis(epochid - 1)
	if epgPre == nil {
		if epochid == 1 {
			rb := big.NewInt(1)
			epgPre, _ = f.generateEpochGenesis(0, nil, rb.Bytes(), common.Hash{})
		} else {
			return nil, errors.New("epoch genesis not exist")
		}
	}

	var rb 			*big.Int
	var blk 		*types.Block
	rb, blk = f.getEpochRandomAndPreEpLastBlk(epochid)
	epg, err := f.generateEpochGenesis(epochid, blk, rb.Bytes(), epgPre.GenesisBlkHash)
	if err != nil {
		return nil, err
	}
	return epg, nil
}

func (f *EpochGenesisBlock) generateChainedEpochGenesis(epochid uint64, isEnd bool) (*types.EpochGenesis,error){
	//it is the first block of this epoch
	var rb *big.Int
	var blk *types.Block
	var epgPre *types.EpochGenesis
	var epg *types.EpochGenesis

	curEpid, _, err := f.GetBlockEpochIdAndSlotId(f.bc.currentBlock.Header())

	if curEpid == 0 {
		//curEpid,_,err = f.GetBlockEpochIdAndSlotId(f.bc.hc.CurrentHeader())
		log.Warn("generateChainedEpochGenesis  current block is 0")
	}

	if curEpid < epochid || err !=nil || epochid == 0{
		return nil , errors.New("error epochid")
	}

	epgPre = f.GetEpochGenesis(epochid - 1)
	if epgPre == nil {
		//start from ep1
		for i := uint64(1); i <= epochid; i++ {

			epg = f.GetEpochGenesis(i)
			if epg != nil {
				continue
			}

			if i == 1 {
				rb = big.NewInt(1)
				epgPre, err = f.generateEpochGenesis(0, nil, rb.Bytes(), common.Hash{})
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

			epg, err = f.generateEpochGenesis(i, blk, rb.Bytes(), epgPre.GenesisBlkHash)
			if err != nil {
				return nil, err
			}

			bEnd := i == epochid
			if bEnd {
				bEnd = isEnd
			}
			err = f.SetEpochGenesis(epg,  bEnd)
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

		err = f.SetEpochGenesis(epg, isEnd)
		if err != nil {
			return nil, err
		}
	}

	return epg, nil
}

func (f *EpochGenesisBlock) getEpochRandomAndPreEpLastBlk(epochid uint64) (*big.Int, *types.Block) {
	blkNum := f.rbLeaderSelector.GetEpochLastBlkNumber(epochid)

	preEpLastblk := f.bc.GetBlockByNumber(blkNum)

	stateDb, err := f.bc.State() //f.bc.StateAt(preEpLastblk.Root())
	if err != nil {
		return nil, nil
	}

	rb := vm.GetR(stateDb, epochid)

	return rb, preEpLastblk
}


func (bc *HeaderChain) IsEpochFirstBlkNumber(epochId uint64, blocknum uint64, parents []*types.Header) bool {
	if epochId > 1 && blocknum > 1 {
		var head *types.Header = nil
		size := len(parents)
		if size > 0 {
			head = parents[ size - 1]
		} else {
			head = bc.GetHeaderByNumber(blocknum - 1)
		}
		if head != nil {
			eid := head.Difficulty.Uint64() >> 32
			if eid + 1 == epochId {
				log.Info(" IsEpochFirstBlkNumber", "epoch", epochId, "first", blocknum)
				return true
			}
		}
	}

	return false
}
func (hc *HeaderChain) VerifyEpochGenesisHash(epochid uint64, hash common.Hash, bGenerate bool) error {
	return hc.epochgen.VerifyEpochGenesisHash(epochid, hash, bGenerate)
}

func (hc *HeaderChain) GenerateEGHash(epochid uint64) (common.Hash, error) {
	return hc.epochgen.GenerateEGHash(epochid)
}

func (hc *HeaderChain) IsSignerValid(addr *common.Address, header *types.Header) bool {
	return hc.epochgen.IsSignerValid(addr, header)
}

//func (hc *HeaderChain) GetOrGenerateEGHash(epochid uint64) (common.Hash, error) {
//	return hc.epochgen.GetOrGenerateEGHash(epochid)
//}



func (f *EpochGenesisBlock) GenerateEGHash(epochid uint64) (common.Hash, error) {
	epkGnss := f.GetEpochGenesis(epochid)
	if epkGnss == nil {
		log.Error("***dirty epoch genesis block data")
	}
	epkGnss, err := f.generateChainedEpochGenesis(epochid, false)
	if err != nil {
		return common.Hash{},errors.New("fail to generate epoch genesis " + err.Error())
	}
	return epkGnss.GenesisBlkHash, nil
}

func (f *EpochGenesisBlock) IsSignerValid(addr *common.Address, header *types.Header) bool {
	eid, _ := posUtil.CalEpochSlotID(header.Time.Uint64())

	f.epgGenmu.RLock()
	slotLeaderMap, ok := slotLeaderCache[eid]
	f.epgGenmu.RUnlock()
	if !ok {
		var epkGnss *types.EpochGenesis = nil
		if eid == 0 {
			// todo: genesisPk 's addr
			//GenesisPK == *addr
			return true
		} else {
			epkGnss = f.GetEpochGenesis(eid)
		}
		if epkGnss != nil {
			slotLeaderMap = make(map[uint64] common.Address)
			sz := len(epkGnss.SlotLeaders)
			for i:=0; i<sz; i++ {
				pubkey := epkGnss.SlotLeaders[i]

				//pk := crypto.ToECDSAPub(hexutil.MustDecode(posconfig.WhiteListOrig[i]))
				//addr := crypto.PubkeyToAddress(*pk)
				pk := crypto.ToECDSAPub(pubkey)
				slotLeaderMap[epkGnss.PreEpochLastBlkNumber - uint64(sz) + uint64(i + 1)] = crypto.PubkeyToAddress(*pk)
			}
			f.epgGenmu.Lock()
			slotLeaderCache[eid] = slotLeaderMap
			f.epgGenmu.Unlock()
		} else {
			log.Error("IsSignerValid GetEpochGenesis failed")
			return false
		}
	}

	add := slotLeaderMap[header.Number.Uint64()]
	if add == *addr {
		return true
	}

	return false
}

// seal, header --> verify
func (f *EpochGenesisBlock) VerifyEpochGenesisHash(epochid uint64, hash common.Hash, bGenerate bool) error {
	var epkGnss *types.EpochGenesis = nil
	var err error
	if bGenerate {
		epkGnss, err = f.generateChainedEpochGenesis(epochid, false)
		if err != nil {
			return errors.New("VerifyEpochGenesisHash error, can't generate epoch genesis, id=" + strconv.Itoa(int(epochid)))
		}
	} else {
		epkGnss = f.GetEpochGenesis(epochid)
	}
	if epkGnss == nil{
		return errors.New("VerifyEpochGenesisHash error, can't get epoch genesis, id=" + strconv.FormatUint(epochid, 10))
	}

	epkGnssHash := epkGnss.GenesisBlkHash
	if epkGnssHash != hash {
		return errors.New("VerifyEpochGenesisHash failed, epoch id="+ strconv.FormatUint(epochid, 10))
	}
	return nil
}

func (f *EpochGenesisBlock) generateEpochGenesis(epochid uint64,lastblk *types.Block,rb []byte,preHash common.Hash) (*types.EpochGenesis, error) {

	epGen := &types.EpochGenesis{}

	epGen.ProtocolMagic = []byte("wanchainpos")

	epGen.EpochId = epochid

	if lastblk == nil {
		epGen.PreEpochLastBlkHash = common.Hash{}
		epGen.PreEpochLastBlkNumber = 0
	} else {
		epGen.PreEpochLastBlkHash = lastblk.Hash()
		epGen.PreEpochLastBlkNumber = lastblk.NumberU64()
	}

	epGen.EpochLeaders = f.rbLeaderSelector.GetEpochLeaders(epochid)

	epGen.RBLeadersSec256 = make([][]byte, 0)
	epGen.RBLeadersBn256 = make([][]byte, 0)
	rbleaders := f.rbLeaderSelector.GetRBProposerGroup(epochid)

	if len(rbleaders) != 0 {
		for _, rbl := range rbleaders {
			epGen.RBLeadersSec256 = append(epGen.RBLeadersSec256, rbl.PubSec256)
			epGen.RBLeadersBn256 = append(epGen.RBLeadersBn256, rbl.PubBn256)
		}
	}

	epGen.SlotLeaders = f.getAllSlotLeaders(epochid)

	// TODO: save WhiteInfos --- func GetWlConfig(stateDb StateDB) WhiteInfos
	// TODO: SelectLeadersLoop ---
	stakersBytes := make([][]byte, 0) //f.TryGetAndSaveAllStakerInfoBytes(epochid)
	//if err != nil {
	//	log.Debug("fail to get staker")
	//	return nil, err
	//}
	epGen.StakerInfos = stakersBytes

	// TODO: save R? no need

	epGen.PreEpochGenHash = preHash

	epGen.GenesisBlkHash = common.Hash{}

	byteVal, err := json.Marshal(epGen)

	//log.Info("generated epochGenesis data","",common.ToHex(byteVal))

	if err != nil {
		log.Debug("Failed to marshal epoch genesis data", "err", err)
		return nil, err
	}

	epGen.GenesisBlkHash = crypto.Keccak256Hash(byteVal)
	return epGen, nil
}

func (f *EpochGenesisBlock) preVerifyEpochGenesis(epGen *types.EpochGenesis) bool {
	var epgPre *types.EpochGenesis
	var err error

	if epGen.EpochId <= 0 {
		return false
	}

	if epGen.EpochId == 1 {
		rb := big.NewInt(1)
		epgPre, err = f.generateEpochGenesis(0, nil, rb.Bytes(), common.Hash{})
		if err != nil {
			return false
		}
	} else {
		epgPre = f.GetEpochGenesis(epGen.EpochId - 1)
		if epgPre == nil {
			return false
		}
	}

	res := (epGen.PreEpochGenHash == epgPre.GenesisBlkHash)
	if !res {
		log.Debug("Failed to verify preEpoch hash", "", common.ToHex(epGen.PreEpochGenHash[:]), common.ToHex(epgPre.GenesisBlkHash[:]))
		return false
	}

	res = bytes.Equal(epGen.ProtocolMagic, []byte("wanchainpos"))
	if !res {
		return false
	}

	if len(epGen.RBLeadersSec256) == 0 || len(epGen.SlotLeaders) == 0 || len(epGen.EpochLeaders) == 0 {
		log.Debug("Failed to verify leaders")
		return false
	}

	epGenNew := &types.EpochGenesis{}

	epGenNew.ProtocolMagic = []byte("wanchainpos")
	epGenNew.PreEpochLastBlkNumber = epGen.PreEpochLastBlkNumber
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
	//log.Info("verify genesis data","",common.ToHex(byteVal))

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
			f.GenerateEpochGenesis(epochID - 1, false)
		}

		f.lastEpochId = epochID
	}
}

func (f *EpochGenesisBlock) GetLastBlkInPreEpoch(blk *types.Block) *types.Block {
	epochID, _ := util.GetEpochSlotIDFromDifficulty(blk.Header().Difficulty)
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

func (f *EpochGenesisBlock) getAllSlotLeaders(epochID uint64) [][]byte {
	startBlkNum := uint64(0)
	if epochID > 0 {
		startBlkNum = f.rbLeaderSelector.GetEpochLastBlkNumber(epochID - 1) + 1
	}

	endBlkNum := f.rbLeaderSelector.GetEpochLastBlkNumber(epochID)
	slotLeaders :=  make([][]byte, 0)
	for i := startBlkNum;i <= endBlkNum;i++ {
		header := f.bc.GetHeaderByNumber(i)
		signer, err := f.recoverSigner(header)
		if err != nil {
			log.Error(err.Error())
			break
		}

		slotLeaders = append(slotLeaders, signer[:])
	}

	return slotLeaders

}

func (f *EpochGenesisBlock) SetEpochGenesis(epochgen *types.EpochGenesis, isEnd bool) error {
	f.epgSetmu.Lock()
	defer f.epgSetmu.Unlock()

	if epochgen == nil || epochgen.EpochId <= 0 {
		return errors.New("inputing epoch genesis is nil")
	}

	res := f.preVerifyEpochGenesis(epochgen)
	if !res {
		return errors.New("epoch genesis preverify is failed")
	}

	val, err := rlp.EncodeToBytes(epochgen)
	if err != nil {
		return err
	}

	if !isEnd {
		_,err = f.epochGenDb.Put(epochgen.EpochId,"epochgenesis",val)
		if err != nil {
			return err
		}
		posUtil.SetEpochBlock(epochgen.EpochId, epochgen.PreEpochLastBlkNumber, epochgen.PreEpochLastBlkHash)
	}

	err = f.saveToPosDb(epochgen, isEnd)
	if err != nil {
		return err
	}

	log.Info("successfully input epochGenesis", "", epochgen.EpochId)

	return nil
}

func (f *EpochGenesisBlock) GetEpochGenesis(epochid uint64) *types.EpochGenesis {

	val, err := f.epochGenDb.Get(epochid, "epochgenesis")
	if err != nil || val == nil {
		return nil
	}

	epochGen := new(types.EpochGenesis)
	err = rlp.DecodeBytes(val, epochGen)
	if err != nil {
		log.Debug(err.Error())
		return nil
	}

	return epochGen
}

func (f *EpochGenesisBlock) ValidateBody(block *types.Block) error {
	log.Debug("begin EpochGenesisBlock ValidateBody")
	extraSeal := 65
	header := block.Header()
	blkTd := block.Difficulty().Uint64()
	epochID := (blkTd >> 32)

	if epochID == 0 {
		return nil
	}

	blKBegin := posUtil.GetEpochBlock(epochID-1) + 1

	idx := int(block.NumberU64() - blKBegin)

	extraType := header.Extra[0]
	start := 1
	if extraType == 'g' {
		start = 33
	}

	_, proofMeg, err := f.slotLeaderSelector.GetInfoFromHeadExtra(epochID, header.Extra[start:len(header.Extra)-extraSeal])

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

	if !bytes.Equal(signer, crypto.FromECDSAPub(pk)) {
		log.Error("Pk signer verify failed in verifySeal", "number", block.NumberU64(),
			"epochID", epochID, "slotID", block.NumberU64(), "signer", common.ToHex(signer), "PkAddress", crypto.PubkeyToAddress(*pk).Hex())
		return errors.New("failed to verify signer for fast synch verify")
	}

	headerPkval := crypto.FromECDSAPub(pk)

	epg := f.GetEpochGenesis(epochID)
	if epg == nil {
		return errors.New("failed to get epoch genesis")
	}
	if idx >= len(epg.SlotLeaders) {
		fmt.Println("idx=", idx, "len=", len(epg.SlotLeaders))
		return errors.New("blokc index is beyong slotleader")
	}

	if !bytes.Equal(epg.SlotLeaders[idx], headerPkval) {
		log.Error("Pk signer verify with epoch genesis", "number", block.NumberU64(),
			"epochID", epochID, "slotID", block.NumberU64(), "signer", common.ToHex(signer), "PkAddress", crypto.PubkeyToAddress(*pk).Hex())
		return errors.New("failed to verify signer with epoch genesis")
	}

	fmt.Println("end EpochGenesisBlock ValidateBody")
	return nil

}

func (f *EpochGenesisBlock) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}

func (f *EpochGenesisBlock) recoverSigner(header *types.Header) ([]byte, error) {
	signature := header.Extra[len(header.Extra)-extraSeal:]

	log.Debug("signature", "hex", hex.EncodeToString(signature))

	log.Debug("sigHash(header)", "Bytes", hex.EncodeToString(sigHash(header).Bytes()))

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return nil, err
	}

	log.Debug("pubkey in ecrecover", "pk", hex.EncodeToString(pubkey))

	return pubkey, nil

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

func (f *EpochGenesisBlock) saveToPosDb(epochgen *types.EpochGenesis, isEnd bool) error {
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
		_, err := elDb.PutWithIndex(epochgen.EpochId, uint64(i), "", v)
		if err != nil {
			return errors.New("saveToPosDb to EpLocalDB failed" + err.Error())
		}
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
		if _, err := rbDb.PutWithIndex(epochgen.EpochId, uint64(i), "", v); err != nil {
			return errors.New("saveToPosDb to RbLocalDB failed" + err.Error())
		}
	}

	if !isEnd {
		stDb := posdb.NewDb(posconfig.StakerLocalDB)
		if stDb == nil {
			return errors.New("create stakeholder local db error")
		}
		count = len(epochgen.StakerInfos)
		for _, pkBytes := range epochgen.StakerInfos {
			staker := vm.StakerInfo{}
			_ = rlp.DecodeBytes(pkBytes, &staker)
			if _, err := stDb.PutWithIndex(epochgen.EpochId, 0, common.ToHex(staker.Address[:]), pkBytes); err != nil {
				return errors.New("saveToPosDb to StakerLocalDB failed" + err.Error())
			}
		}
	}

	return nil
}
