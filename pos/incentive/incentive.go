package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"

	"github.com/wanchain/go-wanchain/core/state"
)

type getStakerInfoFn func(common.Address) ([]common.Address, []*big.Int, float64, float64)

type setStakerInfoFn func([]common.Address, []*big.Int)

var getStakerInfo getStakerInfoFn

var setStakerInfo setStakerInfoFn

// SetStakerInterface is use for Staker module to set its interface
func SetStakerInterface(get getStakerInfoFn, set setStakerInfoFn) {
	getStakerInfo = get
	setStakerInfo = set
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

	finished(stateDb, epochID)
	return true
}
