package slotleader

import (
	"errors"
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rpc"
)

var (
	errRCNotReady = errors.New("rc is not ready")
)

type SendTxFn func(rc *rpc.Client, tx map[string]interface{}) (common.Hash, error)

func (s *SlotLeaderSelection) sendSlotTx(data []byte, posSender SendTxFn) error {
	if s.rc == nil {
		return errRCNotReady
	}

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	//arg["gas"] = (*hexutil.Big)(big.NewInt(1500000)) //use default gas
	arg["txType"] = types.POS_TX
	arg["data"] = hexutil.Bytes(data)
	log.Debug("Write data of payload", "length", len(data))

	_, err := posSender(s.rc, arg)
	return err
}
