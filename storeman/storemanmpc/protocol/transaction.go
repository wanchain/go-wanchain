package protocol

import (
	"fmt"
	"github.com/wanchain/go-wanchain/common/hexutil"
)

type SendData struct {
	PKBytes []byte `json:"pk"`
	Data    []byte `json:"data"`
}

func (d *SendData) String() string {
	return fmt.Sprintf(
		"From:%s", hexutil.Encode(d.Data[:]))
}

type SignedResult struct {
	R []byte `json:"R"`
	S []byte `json:"S"`
}
