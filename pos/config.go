package pos

import (
	"math/big"

	"github.com/wanchain/go-wanchain/node"
	bn256 "github.com/wanchain/pos/cloudflare"
)

var (
	// EpochBaseTime is the pos start time such as: 2018-12-12 00:00:00 == 1544544000
	EpochBaseTime = uint64(0)
	// SelfTestMode config whether it is in a simlate tese mode
	SelfTestMode = false
)

const (
	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 10
	// RandomProperCount is count of pk in random leader group which is select by stake
	RandomProperCount = 10
	// SlotCount is slot count in an epoch
	SlotCount = 30
	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 6
)

type Config struct {
	PolymDegree      uint
	K                uint
	MinRBProposerCnt uint
	EpochInterval    uint64
	PosStartTime     int64
	SelfPuK          *bn256.G1
	SelfPrK          *big.Int
	Dbpath           string
	NodeCfg          *node.Config
}

var DefaultConfig = Config{
	1,
	2,
	2,
	0,
	0,
	new(bn256.G1).ScalarBaseMult(big.NewInt(1)),
	big.NewInt(1),
	"",
	nil,
}

func Cfg() *Config {
	return &DefaultConfig
}
