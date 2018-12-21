package pos

import (
	"errors"
	"context"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/log"
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

func sendTx(tx map[string]interface{}) error {
	rc, err := rpc.Dial("http://localhost:8545")
	if err != nil {
		return err
	}

	if rc == nil {
		return errors.New("rc is not ready")
	}

	ctx := context.Background()
	var txHash common.Hash
	callErr := rc.CallContext(ctx, &txHash, "eth_sendTransaction", tx)
	if nil != callErr {
		log.Error("tx send failed")
		return errors.New("tx send failed")
	}

	log.Debug("tx send success")
	return nil
}

