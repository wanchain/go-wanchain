package protocol

import (
	"fmt"
	"github.com/wanchain/go-wanchain/common/hexutil"
)

type SendData struct {
	PKBytes hexutil.Bytes `json:"pk"`
	Data    []byte        `json:"data"`
}

func (d *SendData) String() string {
	return fmt.Sprintf(
		"From:%s", hexutil.Encode(d.Data[:]))
}

type SignedResult struct {
	R hexutil.Bytes `json:"R"`
	S hexutil.Bytes `json:"S"`
}
