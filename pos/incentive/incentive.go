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

type getStakerInfoFn func(common.Address) ([]common.Address, []*big.Int, float64, float64)

type setStakerInfoFn func([]common.Address, []*big.Int)

type getEpochLeaderInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, int)

type getRandomProposerInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, int)

type getSlotLeaderInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []*big.Int, float64)

var getStakerInfo getStakerInfoFn

var setStakerInfo setStakerInfoFn

var getEpochLeaderInfo getEpochLeaderInfoFn

var getRandomProposerInfo getRandomProposerInfoFn

var getSlotLeaderInfo getSlotLeaderInfoFn

// SetStakerInterface is use for Staker module to set its interface
func SetStakerInterface(get getStakerInfoFn, set setStakerInfoFn) {
	getStakerInfo = get
	setStakerInfo = set
}

// SetActivityInterface is use for get activty module to set its interface
func SetActivityInterface(getEpl getEpochLeaderInfoFn, getRnp getRandomProposerInfoFn, getSlr getSlotLeaderInfoFn) {
	getEpochLeaderInfo = getEpl
	getRandomProposerInfo = getRnp
	getSlotLeaderInfo = getSlr
}

func getIncentivePrecompileAddress() common.Address {
	return common.BytesToAddress(big.NewInt(606).Bytes()) //0x25E
}

// AddEpochGas is used for every block's gas fee collection in each epoch
func AddEpochGas(stateDb *state.StateDB, gasValue *big.Int, epochID uint64) {
	nowGas := getEpochGas(stateDb, epochID)
	nowGas = nowGas.Add(nowGas, gasValue)
	stateDb.SetStateByteArray(getIncentivePrecompileAddress(), getGasHashKey(epochID), nowGas.Bytes())
}

func getEpochGas(stateDb *state.StateDB, epochID uint64) *big.Int {
	buf := stateDb.GetStateByteArray(getIncentivePrecompileAddress(), getGasHashKey(epochID))
	return big.NewInt(0).SetBytes(buf)
}

func getGasHashKey(epochID uint64) common.Hash {
	hash := crypto.Keccak256Hash(postools.Uint64ToBytes(epochID), []byte("gas_collection"))
	return hash
}

func getRunFlagKey(epochID uint64) common.Hash {
	hash := crypto.Keccak256Hash(postools.Uint64ToBytes(epochID), []byte("epoch_run"))
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
	stateDb.SetStateByteArray(getIncentivePrecompileAddress(), getRunFlagKey(epochID), []byte("finished"))
}

// Run is use to run the incentive
func Run(stateDb *state.StateDB, epochID uint64) bool {
	if isFinished(stateDb, epochID) {
		return true
	}

	total, foundation, gasPool := calculateIncentivePool(stateDb, epochID)

	fmt.Println("total:", total.String(), "foundation:", foundation.String(), "gasPool:", gasPool.String())

	finished(stateDb, epochID)
	return true
}

// calcBaseSubsidy returns the subsidy amount a slot at the provided epoch
// should have. This is mainly used for determining how much the incentive for
// newly generated blocks awards as well as validating the incentive for blocks
// has the expected value.
//
// The subsidy is halved every SubsidyReductionInterval
// this is: baseSubsidy / 2^(epoch/SubsidyReductionInterval)
//
// At the target block generation rate for the main network, this is
// approximately every 5 years.
// It will be Zero after 300 years.
func calcBaseSubsidy(epochID uint64) *big.Int {
	subsidyReductionInterval := uint64((365 * 24 * 3600 * 5) / (pos.SlotTime * pos.SlotCount)) // Epoch count in 5 years

	year := big.NewInt(2.1e6) // 2100000 for first year
	weiOfYear := big.NewInt(0).Mul(year, big.NewInt(1e18))
	secondPerYear := big.NewInt(365 * 24 * 3600)
	weiPerSecond := big.NewInt(0).Div(weiOfYear, secondPerYear)
	baseSubsidy := big.NewInt(0).Mul(weiPerSecond, big.NewInt(pos.SlotTime)) // base subsidy for one slot in first year

	return big.NewInt(0).SetUint64(baseSubsidy.Uint64() >> (epochID / subsidyReductionInterval))
}

// calcWanFromFoundation returns subsidy Of Epoch from wan foundation by Wei
func calcWanFromFoundation(epochID uint64) *big.Int {
	subsidyOfSlot := calcBaseSubsidy(epochID)
	subsidyOfEpoch := big.NewInt(0).Mul(subsidyOfSlot, big.NewInt(pos.SlotCount))
	return subsidyOfEpoch
}

func calculateIncentivePool(stateDb *state.StateDB, epochID uint64) (total *big.Int, foundation *big.Int, gasPool *big.Int) {
	foundation = calcWanFromFoundation(epochID)
	gasPool = getEpochGas(stateDb, epochID)
	total = big.NewInt(0).Add(foundation, gasPool)
	return
}
