package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/postools"
)

// AddEpochGas is used for every block's gas fee collection in each epoch
func AddEpochGas(stateDb *state.StateDB, gasValue *big.Int, epochID uint64) {
	nowGas := getEpochGas(stateDb, epochID)
	nowGas.Add(nowGas, gasValue)
	stateDb.SetStateByteArray(getIncentivePrecompileAddress(), getGasHashKey(epochID), nowGas.Bytes())
}

func getEpochGas(stateDb *state.StateDB, epochID uint64) *big.Int {
	buf := stateDb.GetStateByteArray(getIncentivePrecompileAddress(), getGasHashKey(epochID))
	return big.NewInt(0).SetBytes(buf)
}

func getGasHashKey(epochID uint64) common.Hash {
	hash := crypto.Keccak256Hash(postools.Uint64ToBytes(epochID), []byte(dictGasCollection))
	return hash
}
