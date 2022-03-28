package slotleader

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	errRCNotReady = errors.New("rc is not ready")
)

type SendTxFn func(rc *rpc.Client, tx map[string]interface{})

func (s *SLS) sendSlotTx(payload []byte, posSender SendTxFn) error {
	if s.rc == nil {
		return errRCNotReady
	}

	to := vm.GetSlotLeaderSCAddress()
	data := hexutil.Bytes(payload)
	//gas := core.IntrinsicGas(data, &to, true)
	gas, _ := core.IntrinsicGasWan(data, nil, false, false, false, &to)

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(0).SetUint64(gas))
	arg["txType"] = types.WanPosTxType
	arg["data"] = data
	log.Debug("Write data of payload", "length", len(data))

	go posSender(s.rc, arg)
	return nil
}
