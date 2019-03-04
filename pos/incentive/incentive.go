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

// Run is use to run the incentive
func Run(stateDb *state.StateDB, epochID uint64) bool {
	if isFinished(stateDb, epochID) {
		return true
	}

	total, foundation, gasPool := calculateIncentivePool(stateDb, epochID)
	fmt.Println("total:", total.String(), "foundation:", foundation.String(), "gasPool:", gasPool.String())

	epochLeaderSubsidy := big.NewInt(0).Mul(total, big.NewInt(int64(percentOfEpochLeader)))
	epochLeaderSubsidy.Div(epochLeaderSubsidy, big.NewInt(100))

	randomProposerSubsidy := big.NewInt(0).Mul(total, big.NewInt(int64(percentOfRandomProposer)))
	randomProposerSubsidy.Div(randomProposerSubsidy, big.NewInt(100))

	slotLeaderSubsidy := big.NewInt(0).Mul(total, big.NewInt(int64(percentOfSlotLeader)))
	slotLeaderSubsidy.Div(slotLeaderSubsidy, big.NewInt(100))

	epAddrs, epAct := getEpochLeaderInfo(stateDb, epochID)
	rpAddrs, rpAct := getRandomProposerInfo(stateDb, epochID)
	slAddrs, slBlk, slAct := getSlotLeaderInfo(stateDb, epochID)

	fmt.Println(epAddrs, epAct)
	fmt.Println(rpAddrs, rpAct)
	fmt.Println(slAddrs, slBlk, slAct)

	finished(stateDb, epochID)
	return true
}
