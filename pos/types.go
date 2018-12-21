package pos

import (
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/pos/wanpos_crypto"
	"github.com/wanchain/go-wanchain/core/vm"
)

type RbDKGTxPayload struct {
	EpochId uint64
	ProposerId uint32
	Enshare []*bn256.G1
	Commit []*bn256.G2
	Proof []wanpos.DLEQproof
}

type RbSIGTxPayload struct {
	EpochId uint64
	ProposerId uint32
	Gsigshare *bn256.G1
}


type RbDKGDataCollector struct {
	data *RbDKGTxPayload
	pk *bn256.G1
}

type RbSIGDataCollector struct {
	data *RbSIGTxPayload
	pk *bn256.G1
}
