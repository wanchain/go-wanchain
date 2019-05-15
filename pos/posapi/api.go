package posapi

import (
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/core/types"
	"sort"
	"time"

	"github.com/wanchain/go-wanchain/pos/cfm"

	"github.com/wanchain/go-wanchain/params"

	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/rlp"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/incentive"
	"github.com/wanchain/go-wanchain/pos/util"

	"context"
	"errors"
	"math/big"

	"encoding/binary"

	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/internal/ethapi"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/rpc"
)

type PosChainReader interface {
	// Config retrieves the blockchain's chain configuration.
	Config() *params.ChainConfig

	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.Header

	// GetHeader retrieves a block header from the database by hash and number.
	GetHeader(hash common.Hash, number uint64) *types.Header

	// GetHeaderByNumber retrieves a block header from the database by number.
	GetHeaderByNumber(number uint64) *types.Header

	// GetHeaderByHash retrieves a block header from the database by its hash.
	GetHeaderByHash(hash common.Hash) *types.Header

	// GetBlock retrieves a block from the database by hash and number.
	GetBlock(hash common.Hash, number uint64) *types.Block

	//get chain quality,return quality * 1000
	ChainQuality(epochid uint64, slotid uint64) (uint64,error)
}


type PosApi struct {
	chain   PosChainReader
	backend ethapi.Backend
}

func APIs(chain PosChainReader, backend ethapi.Backend) []rpc.API {
	return []rpc.API{{
		Namespace: "pos",
		Version:   "1.0",
		Service:   &PosApi{chain, backend},
		Public:    true,
	}}
}

func (a PosApi) Version() string {
	return "1.0"
}

func (a PosApi) GetSlotLeadersByEpochID(epochID uint64) map[string]string {
	infoMap := make(map[string]string, 0)
	for i := uint64(0); i < posconfig.SlotCount; i++ {
		buf, err := posdb.GetDb().GetWithIndex(epochID, i, slotleader.SlotLeader)
		if err != nil {
			infoMap[fmt.Sprintf("%06d", i)] = fmt.Sprintf("epochID:%d, index:%d, error:%s \n", epochID, i, err.Error())
		} else {
			infoMap[fmt.Sprintf("%06d", i)] = hex.EncodeToString(buf)
		}
	}

	return infoMap
}

func (a PosApi) GetEpochLeadersByEpochID(epochID uint64) (map[string]string, error) {
	infoMap := make(map[string]string, 0)

	selector := epochLeader.GetEpocher()

	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}

	epochLeaders := selector.GetEpochLeaders(epochID)

	for i := 0; i < len(epochLeaders); i++ {
		infoMap[fmt.Sprintf("%06d", i)] = hex.EncodeToString(epochLeaders[i])
	}

	return infoMap, nil
}

func (a PosApi) GetLocalPK() (string, error) {
	pk, err := slotleader.GetSlotLeaderSelection().GetLocalPublicKey()
	if err != nil {
		return "nil", err
	}

	return hex.EncodeToString(crypto.FromECDSAPub(pk)), nil
}

func (a PosApi) GetBootNodePK() string {
	return posconfig.GenesisPK
}

func (a PosApi) GetSlotScCallTimesByEpochID(epochID uint64) uint64 {
	return vm.GetSlotScCallTimes(epochID)
}

func (a PosApi) GetSmaByEpochID(epochID uint64) (map[string]string, error) {
	pks, _, err := slotleader.GetSlotLeaderSelection().GetSma(epochID)
	if err != nil {
		return nil, err
	}

	info := make(map[string]string, len(pks))

	for i := 0; i < len(pks); i++ {
		info[fmt.Sprintf("%06d", i)] = hex.EncodeToString(crypto.FromECDSAPub(pks[i]))
	}

	return info, nil
}

func (a PosApi) GetRandomProposersByEpochID(epochID uint64) map[string]string {
	//leaders := posdb.GetRBProposerGroup(epochID)
	leaders := epochLeader.GetEpocher().GetRBProposer(epochID)
	info := make(map[string]string, 0)
	for i := 0; i < len(leaders); i++ {
		info[fmt.Sprintf("%06d", i)] = hex.EncodeToString(leaders[i])
	}
	return info
}

func (a PosApi) GetSlotCreateStatusByEpochID(epochID uint64) bool {
	return slotleader.GetSlotLeaderSelection().GetSlotCreateStatusByEpochID(epochID)
}

func (a PosApi) GetRandom(epochId uint64, blockNr int64) (*big.Int, error) {
	state, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(blockNr))
	if err != nil {
		return nil, err
	}

	r := vm.GetStateR(state, epochId)
	if r == nil {
		return nil, errors.New("no random number exists")
	}

	return r, nil
}

func (a PosApi) GetChainQuality(epochid uint64, slotid uint64) (uint64,error) {
	return a.chain.ChainQuality(epochid,slotid)
}

func (a PosApi) GetReorgState(epochid uint64) ([]uint64, error) {
	reOrgDb := posdb.GetDbByName(posconfig.ReorgLocalDB)
	if reOrgDb == nil {
		return []uint64{0,0}, nil
	}

	var reOrgNum, reOrgLen uint64

	reOrgNum = 0

	reorBytes, err := reOrgDb.Get(epochid, "reorgNumber")
	if err == nil && reorBytes != nil {
		reOrgNum = binary.BigEndian.Uint64(reorBytes)
	}

	lenBytes, err := reOrgDb.Get(epochid, "reorgLength")
	if err == nil && reorBytes != nil {
		reOrgLen = binary.BigEndian.Uint64(lenBytes)
	}

	return []uint64{reOrgNum, reOrgLen}, nil
}

func (a PosApi) GetRbSignatureCount(epochId uint64, blockNr int64) (int, error) {
	state, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(blockNr))
	if err != nil {
		return 0, err
	}
	j := 0
	for i := 0; i < posconfig.RandomProperCount; i++ {
		sigData, err := vm.GetSig(state, epochId, uint32(i))
		if err != nil {
			return 0, err
		}
		if sigData != nil {
			j++
		}
	}
	return j, nil
}

func (a PosApi) GetEpochStakerInfo(epochID uint64, addr common.Address) (ApiStakerInfo, error) {
	skInfo := ApiStakerInfo{}
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return skInfo, errors.New("epocher instance does not exist")
	}
	infors, feeRate, total, err := epocherInst.GetEpochProbability(epochID, addr)
	if err != nil {
		return skInfo, err
	}
	skInfo.TotalProbability = (*math.HexOrDecimal256)(total)
	skInfo.FeeRate = feeRate
	skInfo.Infors = make([]ApiClientProbability, len(infors))
	for i := 0; i < len(infors); i++ {
		skInfo.Infors[i].Addr = infors[i].Addr
		skInfo.Infors[i].Probability = (*math.HexOrDecimal256)(infors[i].Probability)
	}
	skInfo.Addr = addr
	return skInfo, nil
}

// this is the static snap of stekers by the block Number.
func (a PosApi) GetStakerInfo(targetBlkNum uint64) ([]*StakerJson, error) {
	stakers := make([]*StakerJson, 0)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return stakers, errors.New("epocher instance do not exist")
	}

	block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root())
	if err != nil {
		return stakers, err
	}
	stateDb.ForEachStorageByteArray(vm.StakersInfoAddr, func(key common.Hash, value []byte) bool {

		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(value, &staker)
		if err != nil {
			log.SyslogErr(err.Error())
			return true
		}
		stakeJson := ToStakerJson(&staker)
		stakers = append(stakers, stakeJson)
		return true
	})
	return stakers, nil
}

func (a PosApi) GetEpochStakerInfoAll(epochID uint64) ([]ApiStakerInfo, error) {
	targetBlkNum := epochLeader.GetEpocher().GetTargetBlkNumber(epochID)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return nil, errors.New("epocher instance do not exist")
	}
	block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root())
	if err != nil {
		return nil, err
	}
	ess := make([]ApiStakerInfo, 0)
	stateDb.ForEachStorageByteArray(vm.StakersInfoAddr, func(key common.Hash, value []byte) bool {
		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(value, &staker)
		if err != nil {
			log.SyslogErr(err.Error())
			return true
		}

		infors, pb, err := epochLeader.CalEpochProbabilityStaker(&staker, epochID)
		if err != nil || pb == nil {
			// this validator has no enough
			return true
		}

		es := ApiStakerInfo{}
		es.Infors = make([]ApiClientProbability, len(infors))
		for i := 0; i < len(infors); i++ {
			es.Infors[i].Addr = infors[i].Addr
			es.Infors[i].Probability = (*math.HexOrDecimal256)(infors[i].Probability)
		}
		es.TotalProbability = (*math.HexOrDecimal256)(pb)
		es.FeeRate = staker.FeeRate
		es.Addr = staker.Address
		ess = append(ess, es)
		return true
	})
	return ess, nil
}

func biToString(value *big.Int, err error) (string, error) {
	if err != nil {
		return "", nil
	}
	return value.String(), err
}

func (a PosApi) GetEpochIncentivePayDetail(epochID uint64) ([][]PayInfo, error) {
	c, err := incentive.GetEpochPayDetail(epochID)
	if err != nil {
		return nil, err
	}

	ret := make([][]PayInfo, len(c))
	for i := 0; i < len(c); i++ {
		ret[i] = make([]PayInfo, len(c[i]))
		for m := 0; m < len(c[i]); m++ {
			ret[i][m] = PayInfo{}
			ret[i][m].Addr = c[i][m].Addr
			ret[i][m].Incentive = (*math.HexOrDecimal256)(c[i][m].Incentive)
		}
	}

	return ret, nil
}

func (a PosApi) GetTotalIncentive() (string, error) {
	return biToString(incentive.GetTotalIncentive())
}

func (a PosApi) GetEpochIncentive(epochID uint64) (string, error) {
	return biToString(incentive.GetEpochIncentive(epochID))
}

func (a PosApi) GetEpochRemain(epochID uint64) (string, error) {
	return biToString(incentive.GetEpochRemain(epochID))
}

func (a PosApi) GetWhiteListConfig() ([]vm.UpgradeWhiteEpochLeaderParam, error) {
	epocherInst := epochLeader.GetEpocher()
	block := epocherInst.GetBlkChain().CurrentBlock()
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root())
	if err != nil {
		return nil, err
	}

	infos := make(vm.WhiteInfos, 0)
	infos = append(infos, vm.UpgradeWhiteEpochLeaderDefault)
	stateDb.ForEachStorageByteArray(vm.PosControlPrecompileAddr, func(key common.Hash, value []byte) bool {
		info := vm.UpgradeWhiteEpochLeaderParam{}
		err := rlp.DecodeBytes(value, &info)
		if err == nil {
			infos = append(infos, info)
		}
		return true
	})
	sort.Stable(infos)
	return infos, nil
}

func (a PosApi) GetWhiteListbyEpochID(epochID uint64) ([]string, error) {
	epocherInst := epochLeader.GetEpocher()
	return epocherInst.GetWhiteByEpochId(epochID)
}

func (a PosApi) GetTotalRemain() (string, error) {
	return biToString(incentive.GetTotalRemain())
}

func (a PosApi) GetIncentiveRunTimes() (string, error) {
	return biToString(incentive.GetRunTimes())
}

func (a PosApi) GetEpochGasPool(epochID uint64) (string, error) {
	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return "", err
	}
	return incentive.GetEpochGasPool(db, epochID).String(), nil
}

func (a PosApi) GetRBAddress(epochID uint64) []common.Address {
	return incentive.GetRBAddress(epochID)
}

func (a PosApi) GetIncentivePool(epochID uint64) ([]string, error) {
	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return nil, err
	}
	total, foundation, gasPool := incentive.GetIncentivePool(db, epochID)
	return []string{total.String(), foundation.String(), gasPool.String()}, nil
}

// GetActivity get epoch leader, random proposer, slot leader 's addresses and activity
func (a PosApi) GetActivity(epochID uint64) (*Activity, error) {
	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return nil, err
	}

	activity := Activity{}
	activity.EpLeader, activity.EpActivity = incentive.GetEpochLeaderActivity(db, epochID)
	activity.RpLeader, activity.RpActivity = incentive.GetEpochRBLeaderActivity(db, epochID)
	activity.SltLeader, activity.SlBlocks, activity.SlActivity, activity.SlCtrlCount = incentive.GetSlotLeaderActivity(s.GetChainReader(), epochID)
	return &activity, nil
}

func (a PosApi) GetEpochID() uint64 {
	ep, _ := util.CalEpochSlotID(uint64(time.Now().Unix()))
	return ep
}

func (a PosApi) GetSlotID() uint64 {
	_, sl := util.CalEpochSlotID(uint64(time.Now().Unix()))
	return sl
}

func (a PosApi) GetSlotCount() int {
	return posconfig.SlotCount
}

func (a PosApi) GetSlotTime() int {
	return posconfig.SlotTime
}

func (a PosApi) GetMaxStableBlkNumber() uint64 {
	return cfm.GetCFM().GetMaxStableBlkNumber()
}

// CalProbability use to calc the probability of a staker with amount by stake wan coins.
// The probability is different in different time, so you should input each epoch ID you want to calc
// Such as CalProbability(390, 10000, 60, 360) means begin from epoch 360 lock 60 epochs stake 10000 to calc 390's probability.
func (a PosApi) CalProbability(amountCoin uint64, lockTime uint64) (string, error) {
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return "", errors.New("epocher instance do not exist")
	}

	amountWin := big.NewInt(0).SetUint64(amountCoin)
	amountWin.Mul(amountWin, big.NewInt(params.Wan))

	probablity := epocherInst.CalProbability(amountWin, lockTime)
	return biToString(probablity, nil)
}

//GetEpochIDByTime can get Epoch ID by input time second Unix.
func (a PosApi) GetEpochIDByTime(timeUnix uint64) uint64 {
	ep, _ := util.CalEpochSlotID(timeUnix)
	return ep
}

//GetSlotIDByTime can get Slot ID by input time second Unix.
func (a PosApi) GetSlotIDByTime(timeUnix uint64) uint64 {
	_, sl := util.CalEpochSlotID(timeUnix)
	return sl
}

//GetTimeByEpochID can get time second Unix by epoch ID.
func (a PosApi) GetTimeByEpochID(epochID uint64) uint64 {
	if posconfig.EpochBaseTime == 0 {
		return 0
	}

	time := posconfig.EpochBaseTime + epochID*posconfig.SlotCount*posconfig.SlotTime

	epochIDGet := a.GetEpochIDByTime(time)
	if epochIDGet < epochID {
		for {
			time += posconfig.SlotTime
			epochIDNew := a.GetEpochIDByTime(time)
			if epochIDNew == epochID {
				return time
			}

			if epochIDNew > epochID {
				log.Error("GetTimeByEpochID error: epochIDNew > epochID", "epochIDNew", epochIDNew, "epochID", epochID)
				return 0
			}
		}
	}

	return time
}

func (a PosApi) GetEpochBlkCnt(epochId uint64) uint64 {
	blkCnt := uint64(0)
	bgBlkNum := uint64(0)

	// todo : add pow switch to pos checking
	header := a.chain.CurrentHeader()
	if header != nil {
		for {
			epid := a.GetEpochIDByTime(header.Time.Uint64())
			if epid <= epochId {
				bgBlkNum = header.Number.Uint64()
				break
			}
		}
	}

	if bgBlkNum == 0 {
		return 0
	}

	for {
		header := a.chain.GetHeaderByNumber(bgBlkNum)
		if header == nil {
			break
		}

		epid := a.GetEpochIDByTime(header.Time.Uint64())
		if epid < epochId {
			break
		}

		blkCnt++
		bgBlkNum--
	}

	return blkCnt
}

func (a PosApi) GetValidSMACnt(epochId uint64) (uint64, uint64) {
	sma1, sma2 := uint64(0), uint64(0)

	stateDb, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(-1))
	if err != nil {
		return sma1, sma2
	}

	sma1 = vm.GetValidSMA1Cnt(stateDb, epochId)
	sma2 = vm.GetValidSMA2Cnt(stateDb, epochId)

	return sma1, sma2
}

func (a PosApi) GetSlStage(slotId uint64) uint64 {
	return vm.GetSlStage(slotId)
}

func (a PosApi) GetValidRBCnt(epochId uint64) (uint64, uint64, uint64) {
	dkg1, dkg2, sig := uint64(0), uint64(0), uint64(0)
	stateDb, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(-1))
	if err != nil {
		return dkg1, dkg2, sig
	}

	dkg1 = vm.GetValidDkg1Cnt(stateDb, epochId)
	dkg2 = vm.GetValidDkg2Cnt(stateDb, epochId)
	sig = vm.GetValidSigCnt(stateDb, epochId)

	return dkg1, dkg2, sig
}

