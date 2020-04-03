package posapi

import (
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/wanchain/go-wanchain/core/types"

	"github.com/wanchain/go-wanchain/pos/cfm"
	"github.com/wanchain/go-wanchain/pos/util/convert"

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

var (
	maxUint64 = uint64(1<<64 - 1)
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
	ChainQuality(epochid uint64, slotid uint64) (uint64, error)
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

func (a PosApi) GetSlotLeaderByEpochIDAndSlotID(epochID uint64, slotID uint64) string {
	if !isPosStage() {
		return "Not POS stage."
	}
	slp, err := slotleader.GetSlotLeaderSelection().GetSlotLeader(epochID, slotID)
	if err != nil {
		return err.Error()
	}
	return hex.EncodeToString(crypto.FromECDSAPub(slp))
}

func (a PosApi) GetEpochLeadersByEpochID(epochID uint64) (map[string]string, error) {
	if !isPosStage() {
		return nil, nil
	}

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

func (a PosApi) GetEpochLeadersAddrByEpochID(epochID uint64) ([]common.Address, error) {
	if !isPosStage() {
		return nil, nil
	}

	selector := epochLeader.GetEpocher()
	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}

	leaders := selector.GetEpochLeaders(epochID)
	addres := make([]common.Address, len(leaders))
	for i := range leaders {
		pub := crypto.ToECDSAPub(leaders[i])
		if pub == nil {
			continue
		}

		addres[i] = crypto.PubkeyToAddress(*pub)

	}

	return addres, nil
}
func (a PosApi) GetLeaderGroupByEpochID(epochID uint64) ([]LeaderJson, error) {
	if !isPosStage() {
		return nil, nil
	}
	selector := epochLeader.GetEpocher()
	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}
	return ToLeaderJson(selector.GetLeaderGroup(epochID)), nil
}

func (a PosApi) GetLocalPK() (string, error) {
	if !isPosStage() {
		return "Not POS stage.", nil
	}
	SLS := slotleader.GetSlotLeaderSelection()
	if SLS == nil {
		return "nil", errors.New("This function can not use in POW stage.")
	}
	pk, err := SLS.GetLocalPublicKey()
	if err != nil {
		return "nil", err
	}

	return hex.EncodeToString(crypto.FromECDSAPub(pk)), nil
}

func (a PosApi) GetBootNodePK() string {
	if !isPosStage() {
		return "Not POS stage."
	}
	return posconfig.GenesisPK
}

func (a PosApi) GetSlotScCallTimesByEpochID(epochID uint64) uint64 {
	if !isPosStage() {
		return 0
	}
	return vm.GetSlotScCallTimes(epochID)
}

func (a PosApi) GetSmaByEpochID(epochID uint64) (map[string]string, error) {
	if !isPosStage() {
		return nil, nil
	}
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

func (a PosApi) GetRandomProposersByEpochID(epochID uint64) (map[string]string, error) {
	if !isPosStage() {
		return nil, nil
	}
	selector := epochLeader.GetEpocher()
	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}

	leaders := selector.GetRBProposer(epochID)
	info := make(map[string]string, 0)
	for i := 0; i < len(leaders); i++ {
		info[fmt.Sprintf("%06d", i)] = hex.EncodeToString(leaders[i])
	}

	return info, nil
}

func (a PosApi) GetRandomProposersAddrByEpochID(epochID uint64) ([]common.Address, error) {
	if !isPosStage() {
		return nil, nil
	}
	selector := epochLeader.GetEpocher()
	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}

	leaders := selector.GetRBProposerGroup(epochID)
	addres := make([]common.Address, len(leaders))
	for i := range leaders {
		addres[i] = leaders[i].SecAddr
	}

	return addres, nil
}

func (a PosApi) GetSlotCreateStatusByEpochID(epochID uint64) bool {
	if !isPosStage() {
		return false
	}
	return slotleader.GetSlotLeaderSelection().GetSlotCreateStatusByEpochID(epochID)
}

func (a PosApi) GetRandom(epochId uint64, blockNr int64) (*big.Int, error) {
	if !isPosStage() {
		return nil, nil
	}

	if blockNr > a.chain.CurrentHeader().Number.Int64() {
		blockNr = -1
	}

	epID, _ := util.CalEpSlbyTd(a.chain.CurrentHeader().Difficulty.Uint64())

	if epochId > epID {
		return nil, errors.New("wrong epochId (It hasn't arrived yet.):" + convert.Uint64ToString(epochId))
	}

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

func (a PosApi) GetChainQuality(epochid uint64, slotid uint64) (uint64, error) {
	if !isPosStage() {
		return 1000, nil
	}
	return a.chain.ChainQuality(epochid, slotid)
}

func (a PosApi) GetReorgState(epochid uint64) ([]uint64, error) {
	if !isPosStage() {
		return nil, nil
	}
	reOrgDb := posdb.GetDbByName(posconfig.ReorgLocalDB)
	if reOrgDb == nil {
		return []uint64{0, 0}, nil
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
	if !isPosStage() {
		return 0, nil
	}

	if blockNr > a.chain.CurrentHeader().Number.Int64() {
		blockNr = -1
	}

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
	validator, err := epocherInst.GetEpochProbability(epochID, addr)
	if err != nil {
		return skInfo, err
	}
	skInfo.TotalProbability = (*math.HexOrDecimal256)(validator.TotalProbability)
	skInfo.FeeRate = validator.FeeRate
	skInfo.Infors = make([]ApiClientProbability, len(validator.Infos))
	for i := 0; i < len(validator.Infos); i++ {
		if i == 0 {
			skInfo.Infors[i].Addr = validator.Infos[i].ValidatorAddr
		} else {
			skInfo.Infors[i].Addr = validator.Infos[i].WalletAddr
		}
		skInfo.Infors[i].Probability = (*math.HexOrDecimal256)(validator.Infos[i].Probability)
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

	//block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	block := epocherInst.GetBlkChain().GetHeaderByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root)
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
		// add NextFeeRate MaxFeeRate
		keyFee := vm.GetStakeInKeyHash(staker.Address)
		newFeeBytes, err := vm.GetInfo(stateDb, vm.StakersFeeAddr, keyFee)
		if err == nil && newFeeBytes != nil {
			var newFee vm.UpdateFeeRate
			err = rlp.DecodeBytes(newFeeBytes, &newFee)
			if err == nil {
				stakeJson.MaxFeeRate = newFee.MaxFeeRate
				stakeJson.FeeRateChangedEpoch = newFee.ChangedEpoch
			} else {
				stakeJson.MaxFeeRate = staker.FeeRate
				stakeJson.FeeRateChangedEpoch = 0
			}
		} else {
			stakeJson.MaxFeeRate = staker.FeeRate
			stakeJson.FeeRateChangedEpoch = 0
		}

		stakers = append(stakers, stakeJson)
		return true
	})
	return stakers, nil
}

func isPosStage() bool {
	return posconfig.FirstEpochId != 0
}

func (a PosApi) GetPosInfo() (info PosInfoJson) {
	info.FirstEpochId = posconfig.FirstEpochId
	info.FirstBlockNumber = posconfig.Pow2PosUpgradeBlockNumber
	return
}

func (a PosApi) GetEpochStakerInfoAll(epochID uint64) ([]ApiStakerInfo, error) {
	targetBlkNum := epochLeader.GetEpocher().GetTargetBlkNumber(epochID)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return nil, errors.New("epocher instance do not exist")
	}
	//block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	block := epocherInst.GetBlkChain().GetHeaderByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root)
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
			if i == 0 {
				es.Infors[i].Addr = infors[i].ValidatorAddr
			} else {
				es.Infors[i].Addr = infors[i].WalletAddr
			}
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

func (a PosApi) GetEpochIncentivePayDetail(epochID uint64) ([]ValidatorInfo, error) {
	if !isPosStage() {
		return nil, nil
	}
	c, err := incentive.GetEpochPayDetail(epochID)
	if err != nil {
		return []ValidatorInfo{}, nil
	}

	ret := make([]ValidatorInfo, len(c))
	for i := 0; i < len(c); i++ {
		if len(c[i]) == 0 {
			continue
		}

		delegators := make([]DelegatorInfo, len(c[i])-1)

		for m := 1; m < len(c[i]); m++ {
			delegators[m-1] = DelegatorInfo{}
			delegators[m-1].Address = c[i][m].WalletAddr
			delegators[m-1].Incentive = (*math.HexOrDecimal256)(c[i][m].Incentive)
			delegators[m-1].Type = "delegator"
		}

		ret[i] = ValidatorInfo{
			Address:       c[i][0].ValidatorAddr,
			WalletAddress: c[i][0].WalletAddr,
			Incentive:     (*math.HexOrDecimal256)(c[i][0].Incentive),
			Type:          "validator",
			Delegators:    delegators,
		}
	}

	return ret, nil
}

func (a PosApi) GetTotalIncentive() (string, error) {
	if !isPosStage() {
		return "Not POS stage.", nil
	}
	return biToString(incentive.GetTotalIncentive())
}
func (a PosApi) GetEpochIncentiveBlockNumber(epochID uint64) (uint64, error) {
	if !isPosStage() {
		return 0, nil
	}
	number, err := incentive.GetEpochIncentiveBlockNumber(epochID)
	if err == nil {
		return number.Uint64(), nil
	}
	return 0, err
}
func (a PosApi) GetEpochIncentive(epochID uint64) (string, error) {
	if !isPosStage() {
		return "Not POS stage.", nil
	}
	return biToString(incentive.GetEpochIncentive(epochID))
}

func (a PosApi) GetEpochRemain(epochID uint64) (string, error) {
	if !isPosStage() {
		return "Not POS stage.", nil
	}
	return biToString(incentive.GetEpochRemain(epochID))
}

func (a PosApi) GetWhiteListConfig() ([]vm.UpgradeWhiteEpochLeaderParam, error) {
	epocherInst := epochLeader.GetEpocher()
	infos := make(vm.WhiteInfos, 0)
	if epocherInst == nil {
		return infos, errors.New("epocher instance do not exist")
	}
	block := epocherInst.GetBlkChain().CurrentBlock()
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root())
	if err != nil {
		return nil, err
	}

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
	if epocherInst == nil {
		return make([]string, 0), errors.New("epocher instance do not exist")
	}
	return epocherInst.GetWhiteByEpochId(epochID)
}

func (a PosApi) GetTotalRemain() (string, error) {
	if !isPosStage() {
		return "Not POS stage.", nil
	}
	return biToString(incentive.GetTotalRemain())
}

func (a PosApi) GetIncentiveRunTimes() (string, error) {
	if !isPosStage() {
		return "Not POS stage.", nil
	}
	return biToString(incentive.GetRunTimes())
}

func (a PosApi) GetEpochGasPool(epochID uint64) (string, error) {
	if !isPosStage() {
		return "Not POS stage.", nil
	}
	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return "", err
	}
	return incentive.GetEpochGasPool(db, epochID).String(), nil
}

func (a PosApi) GetRBAddress(epochID uint64) []common.Address {
	if !isPosStage() {
		return nil
	}
	return incentive.GetRBAddress(epochID)
}

func (a PosApi) GetIncentivePool(epochID uint64) ([]string, error) {
	if !isPosStage() {
		return nil, nil
	}
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
	if !isPosStage() {
		return nil, nil
	}
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

// GetEpRnpActivity get epoch leader, random leader proposer activity
func (a PosApi) GetEpRnpActivity(epochID uint64) (*EpRnpActivity, error) {
	if !isPosStage() {
		return nil, nil
	}
	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return nil, err
	}

	activity := EpRnpActivity{}
	activity.EpLeader, activity.EpActivity = incentive.GetEpochLeaderActivity(db, epochID)
	activity.RpLeader, activity.RpActivity = incentive.GetEpochRBLeaderActivity(db, epochID)
	return &activity, nil
}

// GetSlotActivity get slot activity of epoch
func (a PosApi) GetSlotActivity(epochID uint64) (*SlotActivity, error) {
	if !isPosStage() {
		return nil, nil
	}
	s := slotleader.GetSlotLeaderSelection()
	activity := SlotActivity{}
	activity.SltLeader, activity.SlBlocks, activity.SlActivity, activity.SlCtrlCount = incentive.GetSlotLeaderActivity(s.GetChainReader(), epochID)
	return &activity, nil
}

// GetValidatorActivity get epoch leader, random proposer addresses and activity
func (a PosApi) GetValidatorActivity(epochID uint64) (*ValidatorActivity, error) {
	if !isPosStage() {
		return nil, nil
	}
	epID := a.GetEpochID()
	if epochID >= epID {
		return nil, nil
	}

	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return nil, err
	}

	activity := ValidatorActivity{}
	activity.EpLeader, activity.EpActivity = incentive.GetEpochLeaderActivity(db, epochID)
	activity.RpLeader, activity.RpActivity = incentive.GetEpochRBLeaderActivity(db, epochID)
	if len(activity.EpLeader) == 0 &&
		len(activity.EpActivity) == 0 &&
		len(activity.RpLeader) == 0 &&
		len(activity.RpActivity) == 0 {
		return nil, nil
	}

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
	if !isPosStage() {
		return 0
	}
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
	//if posconfig.EpochBaseTime == 0 {
	//	return 0
	//}

	time := epochID * posconfig.SlotCount * posconfig.SlotTime

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

func (a PosApi) GetEpochBlkCnt(epochId uint64) (uint64, error) {
	fastBgBlkNum := maxUint64
	fastEdBlkNum := maxUint64
	step := uint64(posconfig.SlotCount)

	lastHeader := a.chain.CurrentHeader()
	if lastHeader == nil {
		return 0, nil
	}

	if !util.IsPosBlock(lastHeader.Number.Uint64()) {
		return 0, nil
	}

	epId := a.GetEpochIDByTime(lastHeader.Time.Uint64())
	if epId < epochId {
		return 0, nil
	}

	// fast find the begin and end block numbers
	curNum := lastHeader.Number.Uint64()
	for {
		header := a.chain.GetHeaderByNumber(curNum)
		if header == nil {
			log.Error("get header by number fail", "number", curNum)
			return 0, errors.New("get header by number fail")
		}

		epId := a.GetEpochIDByTime(header.Time.Uint64())
		if epId > epochId {
			fastEdBlkNum = curNum
		} else if epId == epochId {
			if curNum > step && util.IsPosBlock(curNum-step) {
				fastBgBlkNum = curNum - step
			} else {
				fastBgBlkNum = util.FirstPosBlockNumber()
			}

			if curNum+step > lastHeader.Number.Uint64() {
				fastEdBlkNum = lastHeader.Number.Uint64()
			} else {
				fastEdBlkNum = curNum + step
			}

			break
		} else {
			fastBgBlkNum = curNum
			break
		}

		// todo : add pow switch to pos checking
		if curNum == util.FirstPosBlockNumber() {
			return 0, nil
		}

		if curNum > step && util.IsPosBlock(curNum-step) {
			curNum -= step
		} else {
			curNum = util.FirstPosBlockNumber()
		}
	}

	if fastBgBlkNum == maxUint64 || fastEdBlkNum == maxUint64 {
		return 0, nil
	}

	// finely find the begin block number
	for {
		header := a.chain.GetHeaderByNumber(fastBgBlkNum)
		if header == nil {
			log.Error("get header by number fail", "number", fastBgBlkNum)
			return 0, errors.New("get header by number fail")
		}

		epId := a.GetEpochIDByTime(header.Time.Uint64())
		if epId == epochId {
			break
		} else if epId > epochId {
			return 0, nil
		}

		fastBgBlkNum++
	}

	// finely find the end block number
	for {
		header := a.chain.GetHeaderByNumber(fastEdBlkNum)
		if header == nil {
			log.Error("get header by number fail", "number", fastEdBlkNum)
			return 0, errors.New("get header by number fail")
		}

		epId := a.GetEpochIDByTime(header.Time.Uint64())
		if epId == epochId {
			break
		} else if epId < epochId {
			return 0, nil
		}

		// todo : add pow switch to pos checking
		if fastEdBlkNum == util.FirstPosBlockNumber() {
			return 0, nil
		}

		fastEdBlkNum--
	}

	if fastEdBlkNum < fastBgBlkNum {
		return 0, nil
	}

	return fastEdBlkNum - fastBgBlkNum + 1, nil
}

func (a PosApi) GetValidSMACnt(epochId uint64) ([]uint64, error) {
	smas := make([]uint64, 2)

	stateDb, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(-1))
	if err != nil {
		return smas, err
	}

	smas[0] = vm.GetValidSMA1Cnt(stateDb, epochId)
	smas[1] = vm.GetValidSMA2Cnt(stateDb, epochId)

	return smas, err
}

func (a PosApi) GetSlStage(slotId uint64) uint64 {
	return vm.GetSlStage(slotId)
}

func (a PosApi) GetValidRBCnt(epochId uint64) ([]uint64, error) {
	cnts := make([]uint64, 3)
	stateDb, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(-1))
	if err != nil {
		return cnts, err
	}

	cnts[0] = vm.GetValidDkg1Cnt(stateDb, epochId)
	cnts[1] = vm.GetValidDkg2Cnt(stateDb, epochId)
	cnts[2] = vm.GetValidSigCnt(stateDb, epochId)

	return cnts, err
}

func (a PosApi) GetRbStage(slotId uint64) uint64 {
	stage, _, _ := vm.GetRBStage(slotId)
	return uint64(stage)
}

func (a PosApi) GetEpochIdByBlockNumber(blockNumber uint64) uint64 {
	header := a.chain.GetHeaderByNumber(blockNumber)
	if header != nil {
		ep, _ := util.CalEpochSlotID(header.Time.Uint64())
		return ep
	}
	return uint64(0) ^ uint64(0)
}

func (a PosApi) GetEpochStakeOut(epochID uint64) ([]RefundInfo, error) {
	stakeOutByte, err := posdb.GetDb().Get(epochID, posconfig.StakeOutEpochKey)
	if err != nil {
		//return nil, err
		info := make([]RefundInfo, 0)
		return info, nil
	}
	stakeOut := make([]epochLeader.RefundInfo, 0)
	err = rlp.DecodeBytes(stakeOutByte, &stakeOut)
	if err != nil {
		return nil, err
	}
	refundInfo := convertReundInfo(stakeOut)
	return refundInfo, nil
}

// GetTps used to get tps value
func (a PosApi) GetTps(fromNumber uint64, toNumber uint64) (string, error) {
	sRet := fmt.Sprintf("Get tps from %d to %d, ", fromNumber, toNumber)
	s := slotleader.GetSlotLeaderSelection()
	reader := s.GetChainReader()

	totalTx := uint64(0)
	totalSecond := uint64(0)
	for i := fromNumber; i <= toNumber; i++ {
		header := reader.GetHeaderByNumber(i)
		block := reader.GetBlock(header.Hash(), i)
		if block != nil {
			totalTx += uint64(len(block.Transactions()))

			if i == fromNumber {
				totalSecond = block.Time().Uint64()
			}

			if i == toNumber {
				totalSecond = block.Time().Uint64() - totalSecond
			}
		}
	}

	sRet += fmt.Sprintf("Total tx: %d, ", totalTx)
	sRet += fmt.Sprintf("Total second: %d, ", totalSecond)
	sRet += fmt.Sprintf("TPS: %d", totalTx/totalSecond)

	return sRet, nil
}
