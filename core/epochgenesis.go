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
	GetEpochLeadersPK(epochID uint64) []*ecdsa.PublicKey
}

type EpochGenesisBlock struct {
	//useEpochGenesis    		bool
	rbLeaderSelector   		RbLeadersSelInt
	slotLeaderSelector 		SlLeadersSelInt
	epochGenesisCh 			chan uint64
	lastEpochId				uint64

	epochGenDb 				*posdb.Db
	epsetmu  				sync.RWMutex // block processor lock
}

func NewEpochGenesisBlock() *EpochGenesisBlock {

	f := &EpochGenesisBlock{}
	//f.useEpochGenesis = false
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


func (f *EpochGenesisBlock) GenerateEpochGenesis(epochid uint64,lastblk *types.Block,rb []byte) (*types.EpochGenesis, error) {


	epGen := &types.EpochGenesis{}

	epGen.ProtocolMagic = []byte("wanchainpos")

	epGen.EpochId = epochid

	if lastblk == nil {
		epGen.PreEpochLastBlkHash = common.Hash{}
	} else {
		epGen.PreEpochLastBlkHash = lastblk.Hash()
	}

	epGen.EpochLeaders = f.rbLeaderSelector.GetEpochLeaders(epochid)

	epGen.RBLeaders = make([][]byte, 0)
	rbleaders := f.rbLeaderSelector.GetRBProposerGroup(epochid)
	if len(rbleaders) != 0 {
		for _, rbl := range rbleaders {
			epGen.RBLeaders = append(epGen.RBLeaders, rbl.PubSec256)
		}
	}

	epGen.SlotLeaders = make([][]byte, 0)
	pks := f.slotLeaderSelector.GetEpochLeadersPK(epochid)
	if len(pks) != 0 {
		for _, slpk := range pks {
			epGen.SlotLeaders = append(epGen.SlotLeaders, crypto.FromECDSAPub(slpk))
		}
	}

	epGen.GenesisBlkHash = common.Hash{}

	//fmt.Println(epGen)

	byteVal, err := json.Marshal(epGen)

	log.Info("generated hash data","",common.ToHex(byteVal))

	if err != nil {
		log.Debug("Failed to marshal epoch genesis data", "err", err)
		return nil, err
	}


	epGen.GenesisBlkHash = crypto.Keccak256Hash(byteVal)

	return epGen, nil
}


func (f *EpochGenesisBlock) preVerifyEpochGenesis(epGen *types.EpochGenesis) bool {



	res := bytes.Equal(epGen.ProtocolMagic,[]byte("wanchainpos"))
	if !res {
		return false
	}

	if len(epGen.RBLeaders)==0 || len(epGen.SlotLeaders)==0 || len(epGen.EpochLeaders)==0 {
		return false
	}

	epGenNew := &types.EpochGenesis{}

	epGenNew.ProtocolMagic = []byte("wanchainpos")
	epGenNew.PreEpochLastBlkHash = epGen.PreEpochLastBlkHash
	epGenNew.EpochId = epGen.EpochId
	epGenNew.RBLeaders = epGen.RBLeaders
	epGenNew.EpochLeaders = epGen.EpochLeaders
	epGenNew.SlotLeaders = epGen.SlotLeaders

	epGenNew.GenesisBlkHash = common.Hash{}


	//fmt.Println(epGen)

	byteVal, err := json.Marshal(epGenNew)
	log.Info("verify hash data","",common.ToHex(byteVal))

	if err != nil {
		log.Debug("Failed to marshal epoch genesis data", "err", err)
		return false
	}

	epGenNew.GenesisBlkHash = crypto.Keccak256Hash(byteVal)

	res = (epGenNew.GenesisBlkHash == epGen.GenesisBlkHash)

	return res
}

func (f *EpochGenesisBlock) IsFirstBlockInEpoch(firstBlk *types.Block) bool {
	_, slotid, err := f.GetBlockEpochIdAndSlotId(firstBlk.Header())
	if err != nil {
		log.Info("verify genesis failed because of wrong epochid or slotid")
		return false
	}

	if slotid == 0 {
		return true
	}

	return false
}

//updated specified epoch genesis
func (f *EpochGenesisBlock) UpdateEpochGenesis(epochID uint64) {
	if epochID != f.lastEpochId && epochID > 0{
		f.epochGenesisCh <- epochID
	}
}

func (f *EpochGenesisBlock) GetLastBlkInPreEpoch(bc *BlockChain, blk *types.Block) *types.Block {
	epochID := blk.Header().Difficulty.Uint64() >> 32
	blkNUm := posUtil.GetEpochBlock(epochID - 1)
	return bc.GetBlockByNumber(blkNUm)
}

func (f *EpochGenesisBlock) IsExistEpochGenesis(epochid uint64) bool {

	val, err := f.epochGenDb.Get(epochid, "epochgenesis")
	if err != nil || val == nil {
		return false
	}

	return true

}

func (f *EpochGenesisBlock) SetEpochGenesis(epochgen *types.EpochGenesis) error {
	f.epsetmu.Lock()
	defer f.epsetmu.Unlock()

	if epochgen == nil {
		return errors.New("inputing epoch genesis is nil")
	}

	res := f.preVerifyEpochGenesis(epochgen)
	if !res {
		return errors.New("epoch genesis preverify is failed")
	}

	val,err := rlp.EncodeToBytes(epochgen)
	if err != nil {
		return err
	}

	_,err = f.epochGenDb.Put(epochgen.EpochId,"epochgenesis",val)
	if err != nil {
		return err
	}

	log.Info("successfully input epochGenesis","",epochgen.EpochId)

	return nil
}

func (f *EpochGenesisBlock) GetEpochGenesis(epochid uint64) *types.EpochGenesis{

	val, err := f.epochGenDb.Get(epochid, "epochgenesis")
	if err != nil {
		return nil
	}

	epochGen := new(types.EpochGenesis)
	err = rlp.DecodeBytes(val,epochGen)
	if err != nil {
		return nil
	}

	return epochGen
}


