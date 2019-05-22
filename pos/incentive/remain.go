package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/util/convert"
)

func addRemainIncentivePool(stateDb *state.StateDB, epochID uint64, remainValue *big.Int) {
	now := getRemainIncentivePool(stateDb, epochID+subsidyReductionInterval)
	now.Add(now, remainValue)
	// add input 1 years later pool
	hash := crypto.Keccak256Hash(convert.Uint64ToBytes((epochID/subsidyReductionInterval)+1), []byte(dictRemainPool))
	stateDb.SetStateByteArray(getIncentivePrecompileAddress(), hash, now.Bytes())
}

func getRemainIncentivePool(stateDb *state.StateDB, epochID uint64) *big.Int {
	// get return this 1 years pool
	hash := crypto.Keccak256Hash(convert.Uint64ToBytes(epochID/subsidyReductionInterval), []byte(dictRemainPool))
	buf := stateDb.GetStateByteArray(getIncentivePrecompileAddress(), hash)
	return big.NewInt(0).SetBytes(buf)
}
