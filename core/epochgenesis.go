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
	epochGenesisCh     chan uint64
	lastEpochId        uint64

	epochGenDb *posdb.Db
	epgSetmu   sync.RWMutex
	epgGenmu   sync.RWMutex

	// white list map
	whiteMap map[common.Address] bool
	// epochId => slotId -> signer address
	slotLeaderCache map[uint64]map[uint64]common.Address
	//
	epochLeaderCache map[uint64]map[common.Address]bool

	epgGenesis *types.EpochGenesis

	// get: epoch genesis block, white header
	// summary: epoch header list, white header list
	// save : epoch genesis block,  list summary[100]
	//summaryList [][]*types.EpochGenesisSummary
	//summaryMap map[uint64]*types.EpochGenesisSummary
}

func NewEpochGenesisBlock(bc *BlockChain) *EpochGenesisBlock {
	f := &EpochGenesisBlock{}
	f.bc = bc
	f.epochGenesisCh = make(chan uint64, 1)
	f.lastEpochId = 0

	f.epochGenDb = posdb.NewDb("epochGendb")
	f.whiteMap = make(map[common.Address] bool)

	f.slotLeaderCache = make(map[uint64]map[uint64] common.Address)
	f.epochLeaderCache = make(map[uint64]map[common.Address] bool)

	// init white list
	for i:=0; i<len(posconfig.WhiteListOrig); i++ {
		pk := crypto.ToECDSAPub(hexutil.MustDecode(posconfig.WhiteListOrig[i]))
		addr := crypto.PubkeyToAddress(*pk)
		f.whiteMap[addr]= true
	}

	f.epgGenesis = nil

	//f.InitSummaryListAndMap()
	return f
}

//func (f *EpochGenesisBlock) GetSummary(epochId uint64) *types.EpochGenesisSummary {
//	return f.summaryMap[epochId]
//}

//func (f *EpochGenesisBlock) InitSummaryListAndMap() bool {
//	if f.summaryList == nil {
//		f.summaryList = make([][]*types.EpochGenesisSummary, 0)
//	}
//	if f.summaryMap == nil {
//		f.summaryMap = make(map[uint64]*types.EpochGenesisSummary)
//	}
//	for i:=uint64(0); ; i++ {
//		val, err := f.epochGenDb.Get(i,  "epochSummary")
//		if err != nil || val == nil {
//			return true
//		}
//		var summaryLine = []*types.EpochGenesisSummary{}
//		_ = rlp.DecodeBytes(val, &summaryLine)
//		l := len(summaryLine)
//		if l > 0 {
//			f.summaryList = append(f.summaryList, summaryLine)
//			for j:=0; j < l; j ++ {
//				f.summaryMap[summaryLine[j].EpochHeader.EpochId] = summaryLine[j]
//			}
//		}
//		if l < 100 {
//			return true
//		}
//	}
//}

func (f *EpochGenesisBlock) GetEpochGenesisZero() *types.EpochGenesis {
	if f.epgGenesis == nil {
		rb := big.NewInt(1)
		epgGenesis, _, err  := f.generateEpochGenesis(0, nil, rb.Bytes(), common.Hash{})
		if err != nil {
			log.Error("NewEpochGenesisBlock failed epgGenesis error")
		}
		//epgGenesis.SlotLeaders = posconfig.GenesisPK
		f.epgGenesis = epgGenesis
		f.updateCache(epgGenesis)
	}

	return f.epgGenesis
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

func (f *EpochGenesisBlock) GenerateEpochGenesis(epochid uint64) (*types.EpochGenesis, *types.Header, error) {

	log.Debug("generate epg", "", epochid)
	epg, header := f.GetEpochGenesis(epochid)
	if epg != nil {
		return epg, header, nil
	} else {
		return f.generateChainedEpochGenesis(epochid)
	}
}

func (f *EpochGenesisBlock) DoGenerateEpochGenesis(epochid uint64, isEnd bool) (*types.EpochGenesis,error) {
	epgPre, _ := f.GetEpochGenesis(epochid - 1)
	if epgPre == nil {
		if epochid == 1 {
			return f.GetEpochGenesisZero(), nil
		} else {
			return nil, errors.New("epoch genesis not exist")
		}
	}

	var rb 			*big.Int
	var blk 		*types.Block
	rb, blk = f.getEpochRandomAndPreEpLastBlk(epochid)
	epg, _, err := f.generateEpochGenesis(epochid, blk, rb.Bytes(), epgPre.GenesisBlkHash)
	if err != nil {
		return nil, err
	}
	return epg, nil
}

// TODO: test generate un exist epoch
func (f *EpochGenesisBlock) generateChainedEpochGenesis(epochid uint64) (*types.EpochGenesis, *types.Header, error) {
	//it is the first block of this epoch
	var rb *big.Int
	var blk *types.Block
	var epgPre *types.EpochGenesis
	var epg *types.EpochGenesis
	var whiteHeader *types.Header

	curEpid, _, err := f.GetBlockEpochIdAndSlotId(f.bc.currentBlock.Header())

	if curEpid <= epochid || err != nil || epochid == 0 {
		return nil, nil, errors.New("error epochid")
	}

	epgPre, _ = f.GetEpochGenesis(epochid - 1)
	if epgPre == nil {
		//start from ep1
		for i := uint64(1); i <= epochid; i++ {

			epg, _ = f.GetEpochGenesis(i)
			if epg != nil {
				continue
			}

			if i == 1 {
				epgPre = f.GetEpochGenesisZero()

			} else {
				epgPre, _ = f.GetEpochGenesis(i - 1)

			}

			rb, blk = f.getEpochRandomAndPreEpLastBlk(i)

			if epgPre == nil {
				return nil, nil, errors.New("pre epg is nil")
			}

			epg, whiteHeader, err = f.generateEpochGenesis(i, blk, rb.Bytes(), epgPre.GenesisBlkHash)
			if err != nil {
				return nil, nil, err
			}

			err = f.SetEpochGenesis(epg, whiteHeader)
			if err != nil {
				return nil, nil, err
			}

		}

	} else {

		rb, blk = f.getEpochRandomAndPreEpLastBlk(epochid)
		epg, whiteHeader, err = f.generateEpochGenesis(epochid, blk, rb.Bytes(), epgPre.GenesisBlkHash)
		if err != nil {
			return nil, nil, err
		}

		err = f.SetEpochGenesis(epg, whiteHeader)
		if err != nil {
			return nil, nil, err
		}
	}

	return epg, whiteHeader, nil
}

func (f *EpochGenesisBlock) getEpochRandomAndPreEpLastBlk(epochid uint64) (*big.Int, *types.Block) {
	//blkNum := f.rbLeaderSelector.GetEpochLastBlkNumber(epochid)

	blkNum := posUtil.GetEpochBlock(epochid)
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
	epkGnss, _ := f.GetEpochGenesis(epochid)
	if epkGnss == nil {
		log.Error("***dirty epoch genesis block data")
	}
	epkGnss, _, err := f.generateChainedEpochGenesis(epochid)
	if err != nil {
		return common.Hash{},errors.New("fail to generate epoch genesis " + err.Error())
	}
	return epkGnss.GenesisBlkHash, nil
}

func (f *EpochGenesisBlock) IsSignerValid(addr *common.Address, header *types.Header) bool {
	eid, _ := posUtil.CalEpochSlotID(header.Time.Uint64())

	if eid == 0 {
		return true
	}

	slotLeaderMap := f.getSlotLeaderMap(eid)
	if slotLeaderMap == nil {
		// todo: check it's the last epoch, and in full sync mode, it's check in valid body
		return true
	}

	add := (*slotLeaderMap)[header.Number.Uint64()]
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
		epkGnss, _, err = f.generateChainedEpochGenesis(epochid)
		if err != nil {
			return errors.New("VerifyEpochGenesisHash error, can't generate epoch genesis, id=" + strconv.Itoa(int(epochid)))
		}
	} else {
		epkGnss,_ = f.GetEpochGenesis(epochid)
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

// TODO: if epoch id not exist ? GetEpochLeaders? GetRBProposerGroup?
func (f *EpochGenesisBlock) generateEpochGenesis(epochid uint64,lastblk *types.Block,rb []byte,preHash common.Hash) (*types.EpochGenesis, *types.Header, error) {
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

	// TODO: white header may not exist !!
	var whiteHeader *types.Header = nil
	epGen.SlotLeaders, whiteHeader= f.getAllSlotLeaders(epochid)
	if whiteHeader == nil {
		whiteHeader = &types.Header{}
	}

	stakersBytes := make([][]byte, 0)
	//f.TryGetAndSaveAllStakerInfoBytes(epochid)
	//if err != nil {
	//	log.Debug("fail to get staker")
	//	return nil, err
	//}
	epGen.StakerInfos = stakersBytes

	epGen.PreEpochGenHash = preHash

	epGen.GenesisBlkHash = common.Hash{}

	byteVal, err := json.Marshal(epGen)

	//log.Info("generated epochGenesis data","",common.ToHex(byteVal))

	if err != nil {
		log.Debug("Failed to marshal epoch genesis data", "err", err)
		return nil, nil, err
	}

	epGen.GenesisBlkHash = crypto.Keccak256Hash(byteVal)
	return epGen, whiteHeader, nil
}

func (f *EpochGenesisBlock) updateCache(epkGnss *types.EpochGenesis) *map[uint64]common.Address {
	slotLeaderMap := make(map[uint64] common.Address)
	sz := len(epkGnss.SlotLeaders)
	for i:=0; i<sz; i++ {
		addr := epkGnss.SlotLeaders[i]
		slotLeaderMap[epkGnss.PreEpochLastBlkNumber - uint64(sz) + uint64(i + 1)] = addr
	}

	epochLeaderMap := make(map[common.Address] bool)
	sz = len(epkGnss.EpochLeaders)
	for i:=0; i<sz; i++ {
		pubkey := epkGnss.EpochLeaders[i]
		pk := crypto.ToECDSAPub(pubkey)
		epochLeaderMap[crypto.PubkeyToAddress(*pk)] = true
	}

	f.epgGenmu.Lock()
	defer f.epgGenmu.Unlock()

	f.slotLeaderCache[epkGnss.EpochId] = slotLeaderMap
	f.epochLeaderCache[epkGnss.EpochId] = epochLeaderMap

	return &slotLeaderMap
}

func (f *EpochGenesisBlock) dropCache(epochId uint64) {
	f.epgGenmu.Lock()
	defer f.epgGenmu.Unlock()

	delete(f.slotLeaderCache, epochId)
	delete(f.epochLeaderCache, epochId)
}

func (f *EpochGenesisBlock) tryCache(epochId uint64) bool {
	if epochId == 0 {
		eg := f.GetEpochGenesisZero()
		f.updateCache(eg)
	} else {
		eg, _ := f.GetEpochGenesis(epochId)
		if eg != nil {
			f.updateCache(eg)
		} else {
			return false
		}
	}
	return true
}

func (f *EpochGenesisBlock) getSlotLeaderMap(eid uint64) *map[uint64]common.Address {
	if eid < 1 {
		return nil
	}
	f.epgGenmu.RLock()
	slotLeaderMap, ok := f.slotLeaderCache[eid]
	f.epgGenmu.RUnlock()
	if !ok {
		if f.tryCache(eid) {
			slotLeaderMap, ok = f.slotLeaderCache[eid]
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}

	return &slotLeaderMap
}

func (f *EpochGenesisBlock) getEpochLeaderMap(eid uint64) *map[common.Address]bool {
	f.epgGenmu.RLock()
	epochLeaderMap, ok := f.epochLeaderCache[eid]
	f.epgGenmu.RUnlock()
	if !ok {
		if f.tryCache(eid) {
			epochLeaderMap, ok = f.epochLeaderCache[eid]
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}
	return &epochLeaderMap
}

func (f *EpochGenesisBlock) preVerifyEpochGenesis(epGen *types.EpochGenesis) bool {
	var epgPre *types.EpochGenesis
	var err error

	if epGen.EpochId <= 0 {
		return false
	}

	if epGen.EpochId == 1 {
		epgPre = f.GetEpochGenesisZero()
	} else {
		epgPre, _ = f.GetEpochGenesis(epGen.EpochId - 1)
		if epgPre == nil {
			return false
		}
	}

	res := epGen.PreEpochGenHash == epgPre.GenesisBlkHash
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
			f.GenerateEpochGenesis(epochID - 1)
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

func (f *EpochGenesisBlock) getAllSlotLeaders(epochID uint64) ([]common.Address,  *types.Header) {
	slotLeaders := make([]common.Address, 0)
	startBlkNum := uint64(0)
	var whiteHeader *types.Header
	if epochID > 0 {
		startBlkNum = posUtil.GetEpochBlock(epochID-1) + 1
		header := f.bc.GetHeaderByNumber(startBlkNum)
		eid, _ :=posUtil.CalEpochSlotID(header.Time.Uint64())
		if eid != epochID {
			return slotLeaders, whiteHeader
		}
	}

	endBlkNum := posUtil.GetEpochBlock(epochID)
	for i := startBlkNum; i <= endBlkNum; i++ {
		header := f.bc.GetHeaderByNumber(i)
		signer, err := f.recoverSigner(header)
		if err != nil {
			log.Error(err.Error())
			break
		}
		if whiteHeader == nil {
			_, ok := f.whiteMap[*signer]
			if ok {
				whiteHeader = header
			}
		}

		slotLeaders = append(slotLeaders, *signer)
	}

	return slotLeaders, whiteHeader

}

func (f *EpochGenesisBlock) SetEpochGenesis(epochgen *types.EpochGenesis, whiteHeader *types.Header) error {
	f.epgSetmu.Lock()
	defer f.epgSetmu.Unlock()

	if epochgen == nil || epochgen.EpochId <= 0 {
		return errors.New("inputing epoch genesis is nil")
	}

	res := f.preVerifyEpochGenesis(epochgen)
	if !res {
		return errors.New("epoch genesis preverify is failed")
	}

	// TODO: if whiteHeader is empty, may the chain had stopped?
	ok, err := f.checkSlotLeadersAndWhiteHeader(epochgen, whiteHeader)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("slot leader has no one on white list")
	}

	val, err := rlp.EncodeToBytes(epochgen)
	if err != nil {
		return err
	}
	var summary types.EpochGenesisSummary
	summary.EpochHeader = &types.EpochGenesisHeader{
		EpochId:epochgen.EpochId,
		PreEpochLastBlkNumber: epochgen.PreEpochLastBlkNumber,
		PreEpochLastBlkHash: epochgen.PreEpochLastBlkHash,
	}
	summary.WhiteHeader = whiteHeader
	valSummary, err := rlp.EncodeToBytes(summary)
	if err != nil {
		return err
	}

	// TODO: data batch
	_, _ = f.epochGenDb.Put(epochgen.EpochId, "epochgenesis", val)
	_, _ = f.epochGenDb.Put(epochgen.EpochId, "epochsummary", valSummary)
	_ = f.saveToPosDb(epochgen)

	posUtil.SetEpochBlock(epochgen.EpochId, epochgen.PreEpochLastBlkNumber, epochgen.PreEpochLastBlkHash)

	f.updateCache(epochgen)
	log.Info("successfully input epochGenesis", "", epochgen.EpochId)

	return nil
}

func (f *EpochGenesisBlock) GetEpochSummary(epochId uint64) *types.EpochGenesisSummary {
	val, err := f.epochGenDb.Get(epochId, "epochsummary")
	if err != nil || val == nil {
		return nil
	}
	summary := new (types.EpochGenesisSummary)
	err = rlp.DecodeBytes(val, summary)
	if err != nil {
		log.Debug(err.Error())
		return nil
	}
	return summary
}

func (f *EpochGenesisBlock) GetEpochGenesis(epochid uint64) (*types.EpochGenesis,*types.Header) {
	val, err := f.epochGenDb.Get(epochid, "epochgenesis")
	if err != nil || val == nil {
		return nil, nil
	}

	epochGen := new(types.EpochGenesis)
	err = rlp.DecodeBytes(val, epochGen)
	if err != nil {
		log.Debug(err.Error())
		return nil, nil
	}

	val, err = f.epochGenDb.Get(epochid, "epochsummary")
	if err != nil || val == nil {
		return nil, nil
	}

	summary := new(types.EpochGenesisSummary)
	err = rlp.DecodeBytes(val, summary)
	if err != nil {
		log.Debug(err.Error())
		return nil, nil
	}

	return epochGen, summary.WhiteHeader
}


// slot leaders should in pre epoch leader, and must have a member in white list
func (f * EpochGenesisBlock) checkSlotLeadersAndWhiteHeader(eg *types.EpochGenesis, whiteHeader *types.Header) (bool, error) {
	if eg.EpochId == 0 {
		// may be we should check, slot leader == posConfig.genesisPk
		return false, nil
	}

	preEpochLeaderMap := f.getEpochLeaderMap(eg.EpochId - 1)
	if preEpochLeaderMap == nil {
		return false, errors.New("get pre epoch leader map error")
	}

	//opks,_ := hex.DecodeString(posconfig.GenesisPK)
	//opk :=crypto.ToECDSAPub(opks)
	//oaddr := crypto.PubkeyToAddress(*opk)
	//println(common.Bytes2Hex(oaddr[:]))

	bHasWhite := false
	sz := len(eg.SlotLeaders)
	for i:=0; i<sz; i++ {
		addr := eg.SlotLeaders[i]

		if !bHasWhite {
			_, ok := f.whiteMap[addr]
			if ok {
				// check same with white header
				if whiteHeader.Difficulty != nil {
					whiteAddr, err := f.recoverSigner(whiteHeader)
					if err != nil {
						return false, errors.New("checkSlotLeaders whiteHeader un right")
					}
					if *whiteAddr != addr {
						return false, errors.New("checkSlotLeaders whiteHeader is not same with first white in the slot leaders")
					}
					bHasWhite = true
				}
			}
		}
		_, ok := (*preEpochLeaderMap)[addr]
		if !ok {
			return false, errors.New("slot leader should in pre epoch leader group ")
		}
	}

	if !bHasWhite {
		if whiteHeader.Difficulty != nil {
			return false, errors.New("not white leader in the slot leaders")
		} else {
			// todo: develop mode don't support white list
			//return false, errors.New("slot leader should has member in white group ")
		}
	}

	return true, nil
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

	var signerAddr = crypto.PubkeyToAddress(*pk)
	//if !bytes.Equal(signer, crypto.FromECDSAPub(pk)) {
	if *signer == signerAddr {
		log.Error("Pk signer verify failed in verifySeal", "number", block.NumberU64(),
			"epochID", epochID, "slotID", block.NumberU64(), "signer", common.ToHex(signer[:]), "PkAddress", crypto.PubkeyToAddress(*pk).Hex())
		return errors.New("failed to verify signer for fast synch verify")
	}

	//headerPkval := crypto.FromECDSAPub(pk)
	headerAddr := crypto.PubkeyToAddress(*pk)

	epg,_ := f.GetEpochGenesis(epochID)
	if epg == nil {
		return errors.New("failed to get epoch genesis")
	}
	if idx >= len(epg.SlotLeaders) {
		fmt.Println("idx=", idx, "len=", len(epg.SlotLeaders))
		return errors.New("blokc index is beyong slotleader")
	}

	//if !bytes.Equal(epg.SlotLeaders[idx], headerPkval) {
	if epg.SlotLeaders[idx] == headerAddr {
		log.Error("Pk signer verify with epoch genesis", "number", block.NumberU64(),
			"epochID", epochID, "slotID", block.NumberU64(), "signer", common.ToHex(signer[:]), "PkAddress", crypto.PubkeyToAddress(*pk).Hex())
		return errors.New("failed to verify signer with epoch genesis")
	}

	fmt.Println("end EpochGenesisBlock ValidateBody")
	return nil

}

func (f *EpochGenesisBlock) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}

func (f *EpochGenesisBlock) recoverSigner(header *types.Header) (*common.Address, error) {
	signature := header.Extra[len(header.Extra)-extraSeal:]

	log.Debug("signature", "hex", hex.EncodeToString(signature))

	log.Debug("sigHash(header)", "Bytes", hex.EncodeToString(sigHash(header).Bytes()))

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return nil, err
	}

	log.Debug("pubkey in ecrecover", "pk", hex.EncodeToString(pubkey))

	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	log.Debug("signer in ecrecover", "signer", signer.Hex())

	return &signer, nil

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
		v, err := rlp.EncodeToBytes(&tmp)
		if err != nil {
			return errors.New("decode epoch leader sec error")
		}
		_, err = elDb.PutWithIndex(epochgen.EpochId, uint64(i), "", v)
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
		v, err := rlp.EncodeToBytes(&tmp)
		if err != nil {
			return errors.New("decode random leader sec error")
		}
		if _, err = rbDb.PutWithIndex(epochgen.EpochId, uint64(i), "", v); err != nil {
			return errors.New("saveToPosDb to RbLocalDB failed" + err.Error())
		}
	}

	stDb := posdb.NewDb(posconfig.StakerLocalDB)
	if stDb == nil {
		return errors.New("create stakeholder local db error")
	}
	count = len(epochgen.StakerInfos)
	for _, pkBytes := range epochgen.StakerInfos {
		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(pkBytes, &staker)
		if err != nil {
			return errors.New("decode staker error")
		}
		if _, err = stDb.PutWithIndex(epochgen.EpochId, 0, common.ToHex(staker.Address[:]), pkBytes); err != nil {
			return errors.New("saveToPosDb to StakerLocalDB failed" + err.Error())
		}
	}

	return nil
}
