package incentive

import (
	"errors"
	"math"
	"math/big"

	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"

	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/pos/util/convert"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"

	"github.com/wanchain/go-wanchain/core/state"
)

var (
	redutionYears            = 1
	redutionRateBase         = 0.88                                                                                   //88% redution for every year
	subsidyReductionInterval = uint64((365 * 24 * 3600 * redutionYears) / (posconfig.SlotTime * posconfig.SlotCount)) // Epoch count in 1 years
	ceilingPercentS0         = 100.0                                                                                  //100% Turn off in current version.
	openIncentive            = true                                                                                   //If the incentive function is open
	firstPeriodReward        = big.NewInt(0).Mul(big.NewInt(2.5e6), big.NewInt(1e18))                                 // 2500000 wan coin for first year
)

const (
	dictGasCollection = "gas_collection"
	dictEpochRun      = "epoch_run"
	dictRemainPool    = "remain_pool"
	dictFinished      = "finished"
)

// Init is use to init the outsides interface of staker.
// Should be called at the node start
func Init(get GetStakerInfoFn, set SetStakerInfoFn, getRbAddr GetRandomProposerAddressFn) {
	activityInit()
	if get == nil || set == nil || getRbAddr == nil {
		log.SyslogErr("incentive Init input param error (get == nil || set == nil || getRbAddr == nil)")
	}

	setStakerInterface(get, set)
	setActivityInterface(getEpochLeaderActivity, getRandomProposerActivity, getSlotLeaderActivity)
	setRBAddressInterface(getRbAddr)

	initLocalDb(posconfig.IncentiveLocalDB)
	log.Info("--------Incentive Init Finish----------")
}

// Run is use to run the incentive should be called in Finalize of consensus
func Run(chain consensus.ChainReader, stateDb *state.StateDB, epochID uint64) bool {
	if chain == nil || stateDb == nil {
		log.SyslogErr("incentive Run input param error (chain == nil || stateDb == nil)")
		return false
	}

	if isFinished(stateDb, epochID) || !openIncentive {
		return true
	}
	finalIncentive := make([][]vm.ClientIncentive, 0)
	remainsAll := big.NewInt(0)

	total, foundation, gasPool := calculateIncentivePool(stateDb, epochID)
	saveIncentiveIncome(total, foundation, gasPool)

	epAddrs, epAct := getEpochLeaderInfo(stateDb, epochID)
	log.Info("epoch addr", "len", len(epAddrs))
	rpAddrs, rpAct := getRandomProposerInfo(stateDb, epochID)
	log.Info("rp Addrs", "len", len(rpAddrs))

	slAddrs, slBlk, slAct, ctrlCount := getSlotLeaderInfo(chain, epochID, posconfig.SlotCount)
	log.Info("sl Addr ", "len", len(slAddrs), "slAct", slAct, "ctrlCount", ctrlCount)
	log.Info("sl Blk ", "len", len(slBlk), "blks", slBlk)

	percentOfEpochLeader, percentOfRandomProposer, percentOfSlotLeader := calcIncentivePercent(stateDb, epochID)

	epochLeaderSubsidy := calcPercent(total, float64(percentOfEpochLeader*100.0))
	randomProposerSubsidy := calcPercent(total, float64(percentOfRandomProposer*100.0))
	slotLeaderSubsidy := calcPercent(total, float64(percentOfSlotLeader*100.0))
	saveIncentiveDivide(epochLeaderSubsidy, randomProposerSubsidy, slotLeaderSubsidy)

	sum := big.NewInt(0)
	sum.Add(sum, epochLeaderSubsidy)
	sum.Add(sum, randomProposerSubsidy)
	sum.Add(sum, slotLeaderSubsidy)
	sumRemain := big.NewInt(0).Sub(total, sum)
	remainsAll.Add(remainsAll, sumRemain)

	incentives, remains, err := epochLeaderAllocate(epochLeaderSubsidy, epAddrs, epAct, epochID)
	if err != nil {
		log.SyslogErr("Incentive epochLeaderAllocate error", "error", err.Error(), "epochLeaderSubsidy", epochLeaderSubsidy.String(), "epAddrs", epAddrs)
		return false
	}

	if incentives != nil {
		log.Info("epoch leader allocate", "total", sumToPay(incentives), "len", len(incentives))
		finalIncentive = append(finalIncentive, incentives...)
	} else {
		log.Warn("Nothing epoch Leader to incentive.")
	}

	remainsAll.Add(remainsAll, remains)

	incentives, remains, err = randomProposerAllocate(randomProposerSubsidy, rpAddrs, rpAct, epochID)
	if err != nil {
		log.SyslogErr("Incentive randomProposerAllocate error", "error", err.Error(), "randomProposerSubsidy", randomProposerSubsidy.String(), "rpAddrs", rpAddrs)
		return false
	}

	if incentives != nil {
		log.Info("random proposer allocate", "total", sumToPay(incentives), "len", len(incentives))
		finalIncentive = append(finalIncentive, incentives...)
	} else {
		log.Warn("Nothing random proposer to incentive.")
	}

	remainsAll.Add(remainsAll, remains)

	incentives, remains, err = slotLeaderAllocate(slotLeaderSubsidy, slAddrs, slBlk, slAct, posconfig.SlotCount-ctrlCount, epochID)
	if err != nil {
		log.SyslogErr("Incentive slotLeaderAllocate error", "slotLeaderSubsidy", slotLeaderSubsidy.String(), "slAddrs", slAddrs)
		return false
	}

	if incentives != nil {
		log.Info("slot leader allocate", "total", sumToPay(incentives), "len", len(incentives))
		finalIncentive = append(finalIncentive, incentives...)
	} else {
		log.Warn("Nothing slot leader to incentive.")
	}

	remainsAll.Add(remainsAll, remains)

	sumPay := sumToPay(finalIncentive)
	extraRemain := getExtraRemain(total, sumPay, remainsAll)
	remainsAll.Add(remainsAll, extraRemain)
	if !checkTotalValue(total, sumPay, remainsAll) {
		log.SyslogErr("Incentive checkTotalValue error", "sumPay", sumPay.String(), "remainsAll", remainsAll.String(), "total", total.String())
		return false
	}

	addRemainIncentivePool(stateDb, epochID, remainsAll)
	saveRemain(epochID, remainsAll)

	pay(finalIncentive, stateDb)

	setStakerInfo(epochID, finalIncentive)
	saveIncentiveHistory(epochID, finalIncentive)
	localDbSetValue(epochID, dictEpochBlock, chain.CurrentHeader().Number)

	finished(stateDb, epochID)
	return true
}

func getIncentivePrecompileAddress() common.Address {
	return vm.IncentivePrecompileAddr
}

func getRunFlagKey(epochID uint64) common.Hash {
	hash := crypto.Keccak256Hash(convert.Uint64ToBytes(epochID), []byte(dictEpochRun))
	return hash
}

func isFinished(stateDb *state.StateDB, epochID uint64) bool {
	buf := stateDb.GetStateByteArray(getIncentivePrecompileAddress(), getRunFlagKey(epochID))
	if buf == nil || len(buf) == 0 {
		return false
	}
	return true
}

func finished(stateDb *state.StateDB, epochID uint64) {
	stateDb.SetStateByteArray(getIncentivePrecompileAddress(), getRunFlagKey(epochID), []byte(dictFinished))
}

// protocalRunerAllocate use to calc the subsidy of protocal Participant (Epoch leader and Random proposer)
func protocalRunerAllocate(funds *big.Int, addrs []common.Address, acts []int,
	epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	remains := big.NewInt(0)

	if addrs == nil || len(addrs) == 0 {
		return nil, remains.Add(remains, funds), nil
	}

	count := len(addrs)
	if count == 0 {
		return nil, nil, errors.New("protocalRunerAllocate addrs length == 0")
	}

	if count != len(acts) {
		return nil, nil, errors.New("protocalRunerAllocate addrs length != acts length")
	}

	if funds == nil {
		return nil, nil, errors.New("protocalRunerAllocate funds == nil")
	}

	fundOne := funds.Div(funds, big.NewInt(int64(count)))
	fundAddrs := make([]common.Address, 0)
	fundValues := make([]*big.Int, 0)

	for i := 0; i < count; i++ {
		if acts[i] == 1 {
			fundAddrs = append(fundAddrs, addrs[i])
			fundValues = append(fundValues, fundOne)
		} else {
			remains.Add(remains, fundOne)
		}
	}

	finalIncentive, subRemain, err := delegate(fundAddrs, fundValues, epochID)
	if err != nil {
		return nil, nil, err
	}
	remains.Add(remains, subRemain)

	return finalIncentive, remains, nil
}

// epochLeaderAllocate input funds, address and activity returns address and its amount allocate and remaining funds.
func epochLeaderAllocate(funds *big.Int, addrs []common.Address, acts []int,
	epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	return protocalRunerAllocate(funds, addrs, acts, epochID)
}

//randomProposerAllocate input funds, address and activity returns address and its amount allocate and remaining funds.
func randomProposerAllocate(funds *big.Int, addrs []common.Address, acts []int,
	epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	return protocalRunerAllocate(funds, addrs, acts, epochID)
}

//slotLeaderAllocate input funds, address, blocks and activity returns address and its amount allocate and remaining funds.
//slotCount is the slot count ctrled by others not foundation.
func slotLeaderAllocate(funds *big.Int, addrs []common.Address, blocks []int,
	act float64, slotCount int, epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	remains := big.NewInt(0)

	if addrs == nil || len(addrs) == 0 || slotCount == 0 || act == 0 {
		return nil, remains.Add(remains, funds), nil
	}

	scale := 100000.0

	incentiveOfSlot := big.NewInt(0).Div(funds, big.NewInt(int64(slotCount)))
	incentiveScale := big.NewInt(0).Mul(incentiveOfSlot, big.NewInt(int64(math.Floor(act*scale))))
	incentiveActive := incentiveScale.Div(incentiveScale, big.NewInt(int64(math.Floor(scale)))) // get incentive after activity calc.
	singleRemain := big.NewInt(0).Sub(incentiveOfSlot, incentiveActive)

	remains.Add(remains, big.NewInt(0).Mul(singleRemain, big.NewInt(int64(slotCount))))

	log.Info("-->slotLeaderAllocate", "funds", funds, "slotCount", slotCount,
		"incentiveOfSlot", incentiveOfSlot, "incentiveActive", incentiveActive,
		"singleRemain", singleRemain, "len", len(addrs))

	fundAddrs := make([]common.Address, 0)
	fundValues := make([]*big.Int, 0)

	count := len(addrs)
	for i := 0; i < count; i++ {
		fundAddrs = append(fundAddrs, addrs[i])
		fundValues = append(fundValues, big.NewInt(0).Mul(incentiveActive, big.NewInt(int64(blocks[i]))))
	}

	finalIncentive, subRemain, err := delegate(fundAddrs, fundValues, epochID-1)
	if err != nil {
		return nil, nil, err
	}
	remains.Add(remains, subRemain)

	return finalIncentive, remains, nil
}

func sumToPay(readyToPay [][]vm.ClientIncentive) *big.Int {
	sumPay := big.NewInt(0)
	for i := 0; i < len(readyToPay); i++ {
		for m := 0; m < len(readyToPay[i]); m++ {
			sumPay.Add(sumPay, readyToPay[i][m].Incentive)
		}
	}
	return sumPay
}

func checkTotalValue(total *big.Int, sumPay, remain *big.Int) bool {
	log.Info("checkTotalValue", "Total", total, "payout", sumPay, "remains", remain)

	sum := big.NewInt(0).Add(sumPay, remain)
	if total.Cmp(sum) == -1 {
		return false
	}

	return true
}

func pay(incentives [][]vm.ClientIncentive, stateDb *state.StateDB) {
	for i := 0; i < len(incentives); i++ {
		for m := 0; m < len(incentives[i]); m++ {
			stateDb.AddBalance(incentives[i][m].WalletAddr, incentives[i][m].Incentive)
		}
	}
}

func saveIncentiveIncome(total, foundation, gasPool *big.Int) {
	log.Info("Incentive total", "total", total, "foundation", foundation, "gasPool", gasPool)
}

func saveIncentiveDivide(ep, rp, sl *big.Int) {
	log.Info("Incentive Divide", "ep", ep, "rp", rp, "sl", sl)
}

func getExtraRemain(total, sumPay, remain *big.Int) *big.Int {
	sum := big.NewInt(0)
	sum.Add(sum, sumPay)
	sum.Add(sum, remain)

	extraRemain := big.NewInt(0)
	if total.Cmp(sum) == 1 {
		extraRemain.Sub(total, sum)
	}
	return extraRemain
}

func calcIncentivePercent(stateDb vm.StateDB, epochID uint64) (percentOfEpochLeader, percentOfRandomProposer, percentOfSlotLeader float64) {
	rnpCnt := float64(posconfig.RandomProperCount)

	wlInfo := vm.GetEpochWLInfo(stateDb, epochID)
	wlLen := wlInfo.WlCount.Uint64()
	elCnt := float64(posconfig.EpochLeaderCount - wlLen)

	totalMemberCnt := float64(rnpCnt + elCnt)

	percentOfEpochLeader = elCnt / 2.0 / totalMemberCnt //24.4898%
	percentOfRandomProposer = rnpCnt / totalMemberCnt   //51.0204%
	percentOfSlotLeader = elCnt / 2.0 / totalMemberCnt  //24.4898%

	return
}
