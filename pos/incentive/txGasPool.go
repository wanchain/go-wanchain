package incentive

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pos/util/convert"
)

// AddEpochGas is used for every block's gas fee collection in each epoch
func AddEpochGas(stateDb vm.StateDB, gasValue *big.Int, epochID uint64) {
	if !openIncentive {
		return
	}

	if stateDb == nil || gasValue == nil {
		log.SyslogErr("AddEpochGas input param is nil")
		return
	}
	nowGas := getEpochGas(stateDb, epochID)
	nowGas.Add(nowGas, gasValue)
	stateDb.SetStateByteArray(getIncentivePrecompileAddress(), getGasHashKey(epochID), nowGas.Bytes())
}

func getEpochGas(stateDb vm.StateDB, epochID uint64) *big.Int {
	if stateDb == nil {
		log.SyslogErr("getEpochGas with an empty stateDb")
		return big.NewInt(0)
	}

	buf := stateDb.GetStateByteArray(getIncentivePrecompileAddress(), getGasHashKey(epochID))
	return big.NewInt(0).SetBytes(buf)
}

func getGasHashKey(epochID uint64) common.Hash {
	hash := crypto.Keccak256Hash(convert.Uint64ToBytes(epochID), []byte(dictGasCollection))
	return hash
}
