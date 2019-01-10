package posapi

import (
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/rpc"
)
type PosApi struct {
	chain  consensus.ChainReader

}
func APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "pos",
		Version:   "1.0",
		Service:   &PosApi{chain: chain},
		Public:    false,
	}}
}

func (a PosApi)Version() string {
	return "1.0"
}