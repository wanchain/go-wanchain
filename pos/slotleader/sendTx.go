package slotleader

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/postools/slottools"
)

//--------------Transacton create / send --------------------------------------------

func (s *SlotLeaderSelection) sendStage1Tx(data []byte) error {
	//test
	fmt.Println("Ready to send StageTx1 tx:", hex.EncodeToString(data))

	if s.rc == nil {
		return errors.New("rc is not ready")
	}

	//Set payload infomation--------------
	payload, err := slottools.PackStage1Data(data, vm.GetSlotLeaderScAbiString())
	if err != nil {
		log.Debug("PackStage1Data err:" + err.Error())
		return err
	}

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	//arg["gas"] = (*hexutil.Big)(big.NewInt(1500000)) //use default gas
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)
	log.Debug("Write data of payload", "length", len(payload))

	_, err = pos.SendTx(s.rc, arg)
	return err
}
func (s *SlotLeaderSelection) sendStage2Tx(data string) error {
	//test
	fmt.Println("Ready send tx:", data)

	if s.rc == nil {
		return errors.New("rc is not ready")
	}

	//Set payload infomation--------------
	payload, err := slottools.PackStage2Data(data, vm.GetSlotLeaderScAbiString())
	if err != nil {
		return err
	}

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(40000000))
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)
	log.Debug("Write data of payload", "length", len(payload))

	_, err = pos.SendTx(s.rc, arg)
	return err
}
