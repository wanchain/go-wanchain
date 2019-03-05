package incentive

import (
	"fmt"
	"math/big"

	"github.com/wanchain/go-wanchain/core/vm"

	"github.com/wanchain/go-wanchain/pos"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"

	"github.com/wanchain/go-wanchain/core/state"
)

var (
	redutionYears            = 5
	subsidyReductionInterval = uint64((365 * 24 * 3600 * redutionYears) / (pos.SlotTime * pos.SlotCount)) // Epoch count in 5 years
	percentOfEpochLeader     = 20                                                                         //20%
	percentOfRandomProposer  = 20                                                                         //20%
	percentOfSlotLeader      = 60                                                                         //60%
	ceilingPercentS0         = 10.0                                                                       //10%
)

const (
	dictGasCollection = "gas_collection"
	dictEpochRun      = "epoch_run"
	dictRemainPool    = "remain_pool"
	dictFinished      = "finished"
)

func getIncentivePrecompileAddress() common.Address {
	return common.BytesToAddress(big.NewInt(606).Bytes()) //0x25E
}

func getRunFlagKey(epochID uint64) common.Hash {
	hash := crypto.Keccak256Hash(postools.Uint64ToBytes(epochID), []byte(dictEpochRun))
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
func protocalRunerAllocate(funds *big.Int, addrs []common.Address, acts []int, epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	remains := big.NewInt(0)
	count := len(addrs)

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
func epochLeaderAllocate(funds *big.Int, addrs []common.Address, acts []int, epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	return protocalRunerAllocate(funds, addrs, acts, epochID)
}

//randomProposerAllocate input funds, address and activity returns address and its amount allocate and remaining funds.
func randomProposerAllocate(funds *big.Int, addrs []common.Address, acts []int, epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	return protocalRunerAllocate(funds, addrs, acts, epochID)
}

//slotLeaderAllocate input funds, address, blocks and activity returns address and its amount allocate and remaining funds.
func slotLeaderAllocate(funds *big.Int, addrs []common.Address, blocks []int, act float64, slotCount int, epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	remains := big.NewInt(0)

	scale := 100000.0

	incentiveOfSlot := big.NewInt(0).Div(funds, big.NewInt(int64(slotCount)))
	incentiveScale := big.NewInt(0).Mul(incentiveOfSlot, big.NewInt(int64(act*scale)))
	incentiveActive := incentiveScale.Div(incentiveScale, big.NewInt(int64(scale))) // get incentive after activity calc.
	singleRemain := big.NewInt(0).Sub(incentiveOfSlot, incentiveActive)

	remains.Add(remains, big.NewInt(0).Mul(singleRemain, big.NewInt(int64(slotCount))))

	fundAddrs := make([]common.Address, 0)
	fundValues := make([]*big.Int, 0)

	count := len(addrs)
	for i := 0; i < count; i++ {
		fundAddrs = append(fundAddrs, addrs[i])
		fundValues = append(fundValues, big.NewInt(0).Mul(incentiveActive, big.NewInt(int64(blocks[i]))))
	}

	finalIncentive, subRemain, err := delegate(fundAddrs, fundValues, epochID)
	if err != nil {
		return nil, nil, err
	}
	remains.Add(remains, subRemain)

	return finalIncentive, remains, nil
}

func checkTotalValue(total *big.Int, readyToPay [][]vm.ClientIncentive, remain *big.Int) bool {
	sumPay := big.NewInt(0)
	for i := 0; i < len(readyToPay); i++ {
		for m := 0; m < len(readyToPay[i]); m++ {
			sumPay.Add(sumPay, readyToPay[i][m].Incentive)
		}
	}

	sumPay.Add(sumPay, remain)
	if total.Cmp(sumPay) == -1 {
		return false
	}
	return true
}

func pay(incentives [][]vm.ClientIncentive, stateDb *state.StateDB) {
	for i := 0; i < len(incentives); i++ {
		for m := 0; m < len(incentives[i]); m++ {
			stateDb.AddBalance(incentives[i][m].Addr, incentives[i][m].Incentive)
		}
	}
}

// Run is use to run the incentive
func Run(stateDb *state.StateDB, epochID uint64) bool {
	if isFinished(stateDb, epochID) {
		return true
	}

	total, foundation, gasPool := calculateIncentivePool(stateDb, epochID)
	fmt.Println("total:", total.String(), "foundation:", foundation.String(), "gasPool:", gasPool.String())

	epAddrs, epAct := getEpochLeaderInfo(stateDb, epochID)
	rpAddrs, rpAct := getRandomProposerInfo(stateDb, epochID)
	slAddrs, slBlk, slAct := getSlotLeaderInfo(stateDb, epochID)

	epochLeaderSubsidy := calcPercent(total, float64(percentOfEpochLeader))
	randomProposerSubsidy := calcPercent(total, float64(percentOfRandomProposer))
	slotLeaderSubsidy := calcPercent(total, float64(percentOfSlotLeader))

	finalIncentive := make([][]vm.ClientIncentive, 0)
	remainsAll := big.NewInt(0)

	incentives, remains, err := epochLeaderAllocate(epochLeaderSubsidy, epAddrs, epAct, epochID)
	if err != nil {
		return false
	}
	finalIncentive = append(finalIncentive, incentives...)
	remainsAll.Add(remainsAll, remains)

	incentives, remains, err = randomProposerAllocate(randomProposerSubsidy, rpAddrs, rpAct, epochID)
	if err != nil {
		return false
	}
	finalIncentive = append(finalIncentive, incentives...)
	remainsAll.Add(remainsAll, remains)

	incentives, remains, err = slotLeaderAllocate(slotLeaderSubsidy, slAddrs, slBlk, slAct, pos.SlotCount, epochID)
	if err != nil {
		return false
	}
	finalIncentive = append(finalIncentive, incentives...)
	remainsAll.Add(remainsAll, remains)

	if !checkTotalValue(total, finalIncentive, remainsAll) {
		return false
	}

	addRemainIncentivePool(stateDb, epochID, remainsAll)

	pay(finalIncentive, stateDb)

	setStakerInfo(finalIncentive, epochID)

	finished(stateDb, epochID)
	return true
}
