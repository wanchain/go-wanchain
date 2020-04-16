package util

import (
	"context"
	"errors"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"math/rand"
	"time"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rpc"
)

//type SendTxArgs struct {
//  From     common.Address  `json:"from"`
//  To       *common.Address `json:"to"`
//  Gas      *hexutil.Big    `json:"gas"`
//  GasPrice *hexutil.Big    `json:"gasPrice"`
//  Value    *hexutil.Big    `json:"value"`
//  Data     hexutil.Bytes   `json:"data"`
//  Nonce    *hexutil.Uint64 `json:"nonce"`
//}
func SendTx(rc *rpc.Client, tx map[string]interface{}) (common.Hash, error) {
	log.Info("begin send pos tx")
	if rc == nil {
		log.SyslogErr("connect rpc fail, rc is nil")
		return common.Hash{}, errors.New("rc is not ready")
	}

	tx["gasPrice"] = "0x"+posconfig.Cfg().DefaultGasPrice.Text(16)

	ctx := context.Background()
	var txHash common.Hash
	err := rc.CallContext(ctx, &txHash, "eth_sendPosTransaction", tx)
	if nil != err {
		log.SyslogErr("send pos tx fail", "err", err)
		return common.Hash{}, err
	}

	log.SyslogInfo("send pos tx success", "txHash", txHash.String())
	return txHash, nil
}

func SendPosTx(rc *rpc.Client, tx map[string]interface{})  {
	if posconfig.TxDelay != 0 {
		delay := rand.Intn(posconfig.TxDelay)
		time.Sleep(time.Duration(delay)*time.Second)
		log.Debug("SendPosTx", "delay",delay )
	}



	SendTx(rc, tx)
}