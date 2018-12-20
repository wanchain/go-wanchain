package pos

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/node"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/rpc"
)

func BackendTimerLoop() {
	ctx := context.Background()
	fmt.Println("xxxxxx")
	time.Sleep(10 * time.Second)
	url := node.DefaultIPCEndpoint("gwan")
	fmt.Println(url)
	rc, err := rpc.Dial(url)
	if err != nil {
		fmt.Println("err:", err)
		panic(err)
	}
	test := false
	for {
		select {
		case <-time.After(10 * time.Second):
			fmt.Println("time")

			if test {
				var to = common.HexToAddress("0x0102030405060708090a0102030405060708090a")
				amount := new(big.Int)
				amount.SetString("100", 10) // 1000 tokens

				//type SendTxArgs struct {
				//	From     common.Address  `json:"from"`
				//	To       *common.Address `json:"to"`
				//	Gas      *hexutil.Big    `json:"gas"`
				//	GasPrice *hexutil.Big    `json:"gasPrice"`
				//	Value    *hexutil.Big    `json:"value"`
				//	Data     hexutil.Bytes   `json:"data"`
				//	Nonce    *hexutil.Uint64 `json:"nonce"`
				//}
				arg := map[string]interface{}{}
				arg["from"] = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
				arg["to"] = &to
				arg["value"] = (*hexutil.Big)(amount)
				arg["txType"] = 1
				var txHash common.Hash
				callErr := rc.CallContext(ctx, &txHash, "eth_sendTransaction", arg)
				if nil != callErr {
					fmt.Println(callErr)
				}
				fmt.Println(txHash)
			}

			//Add for slot leader selection
			slotleader.GetSlotLeaderSelection().Loop(rc)
		}
	}
	return
}
