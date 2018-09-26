package protocol

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
)

type SendTxArgs struct {
	From      common.Address  `json:"from"`
	To        *common.Address `json:"to"`
	Gas       *hexutil.Big    `json:"gas"`
	GasPrice  *hexutil.Big    `json:"gasPrice"`
	Value     *hexutil.Big    `json:"value"`
	Data      hexutil.Bytes   `json:"data"`
	Nonce     *hexutil.Uint64 `json:"nonce"`
	ChainType string          `json:"chainType"`// 'WAN' or 'ETH'
	ChainID   *hexutil.Big    `json:"chainID"`
	SignType  string          `json:"signType"` //input 'hash' for hash sign (r,s,v), else for full sign(rawTransaction)
}

