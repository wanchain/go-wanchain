package pos

import (
	"github.com/wanchain/pos/cloudflare"
	"math/big"
)

type Config struct {
	PolymDegree uint
	K uint
	MinRBProposerCnt uint
	EpochInterval uint64
	PosStartTime int64
	SelfPuK *bn256.G1
	SelfPrK *big.Int
}

var DefaultConfig Config = Config{1,1,1, 0, 0, new(bn256.G1).ScalarBaseMult(big.NewInt(1)), big.NewInt(1)}

func Cfg() *Config {
	return &DefaultConfig;
}


