package slotleader

import (
	"errors"
	"math/big"

	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
)

var (
	ErrRCNotReady = errors.New("rc is not ready")
)

//--------------Transacton create / send --------------------------------------------

func (s *SlotLeaderSelection) sendStage1Tx(data []byte) error {
	if s.rc == nil {
		return ErrRCNotReady
	}

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	//arg["gas"] = (*hexutil.Big)(big.NewInt(1500000)) //use default gas
	arg["txType"] = 7
	arg["data"] = hexutil.Bytes(data)
	log.Debug("Write data of payload", "length", len(data))

	_, err := pos.SendTx(s.rc, arg)
	return err
}
func (s *SlotLeaderSelection) sendStage2Tx(data []byte) error {
	if s.rc == nil {
		return ErrRCNotReady
	}

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(1500000))
	arg["txType"] = 7
	arg["data"] = hexutil.Bytes(data)
	log.Debug("Write data of payload", "length", len(data))

	_, err := pos.SendTx(s.rc, arg)
	return err
}
