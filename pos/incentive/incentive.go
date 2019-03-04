package incentive

import (
	"fmt"
	"math/big"

	"github.com/wanchain/go-wanchain/pos"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"

	"github.com/wanchain/go-wanchain/core/state"
)

var (
	redutionYears            = 5
	subsidyReductionInterval = uint64((365 * 24 * 3600 * redutionYears) / (pos.SlotTime * pos.SlotCount)) // Epoch count in 5 years
	percentOfEpochLeader     = 20
	percentOfRandomProposer  = 20
	percentOfSlotLeader      = 60
	ceilingPercentS0         = 0.1
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
func protocalRunerAllocate(funds *big.Int, addrs []common.Address, acts []int, epochID uint64) ([]common.Address, []*big.Int, *big.Int) {
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

	finalAddrs, finalValues := delegate(fundAddrs, fundValues, epochID)
	return finalAddrs, finalValues, remains
}

// epochLeaderAllocate input funds, address and activity returns address and its amount allocate and remaining funds.
func epochLeaderAllocate(funds *big.Int, addrs []common.Address, acts []int, epochID uint64) ([]common.Address, []*big.Int, *big.Int) {
	return protocalRunerAllocate(funds, addrs, acts, epochID)
}

//randomProposerAllocate input funds, address and activity returns address and its amount allocate and remaining funds.
func randomProposerAllocate(funds *big.Int, addrs []common.Address, acts []int, epochID uint64) ([]common.Address, []*big.Int, *big.Int) {
	return protocalRunerAllocate(funds, addrs, acts, epochID)
}

//slotLeaderAllocate input funds, address, blocks and activity returns address and its amount allocate and remaining funds.
func slotLeaderAllocate(funds *big.Int, addrs []common.Address, blocks []int, act float64) ([]common.Address, []*big.Int, *big.Int) {
	remains := big.NewInt(0)
	return nil, nil, remains
}

func checkTotalValue(total *big.Int, readyToPay []*big.Int, remain *big.Int) bool {
	return true
}

func pay(addrs []common.Address, values []*big.Int) {

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

	epochLeaderSubsidy := calcPercent(total, percentOfEpochLeader)
	randomProposerSubsidy := calcPercent(total, percentOfRandomProposer)
	slotLeaderSubsidy := calcPercent(total, percentOfSlotLeader)

	addressAll := make([]common.Address, 0)
	valuesAll := make([]*big.Int, 0)
	remainsAll := big.NewInt(0)

	addrs, values, remains := epochLeaderAllocate(epochLeaderSubsidy, epAddrs, epAct, epochID)
	addressAll = append(addressAll, addrs...)
	valuesAll = append(valuesAll, values...)
	remainsAll.Add(remainsAll, remains)

	addrs, values, remains = randomProposerAllocate(randomProposerSubsidy, rpAddrs, rpAct, epochID)
	addressAll = append(addressAll, addrs...)
	valuesAll = append(valuesAll, values...)
	remainsAll.Add(remainsAll, remains)

	addrs, values, remains = slotLeaderAllocate(slotLeaderSubsidy, slAddrs, slBlk, slAct)
	addressAll = append(addressAll, addrs...)
	valuesAll = append(valuesAll, values...)
	remainsAll.Add(remainsAll, remains)

	if !checkTotalValue(total, valuesAll, remainsAll) {
		return false
	}

	addRemainIncentivePool(stateDb, epochID, remainsAll)

	pay(addressAll, valuesAll)

	setStakerInfo(addressAll, valuesAll, epochID)

	finished(stateDb, epochID)
	return true
}
