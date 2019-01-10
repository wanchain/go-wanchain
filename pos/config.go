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
	EpochLeaderCount = 30
	// RandomProperCount is count of pk in random leader group which is select by stake
	RandomProperCount = 10
	// SlotCount is slot count in an epoch
	SlotCount = 30
	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 3
	// GenesisPK is the epoch0 pk
	GenesisPK = "04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70"
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
	SlotCount/10,
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
