package pos

import (
	"math/big"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
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
	EpochLeaderCount = 50
	// RandomProperCount is count of pk in random leader group which is select by stake
	RandomProperCount = 9
	// SlotCount is slot count in an epoch
	SlotCount = 100
	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 10
	// GenesisPK is the epoch0 pk
	GenesisPK = "04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70"

	//Incentive should perform delay some epochs.
	IncentiveDelayEpochs = 1

	// Stage1K is divde a epoch into 10 pieces
	Stage1K  = uint64(SlotCount / 10)
	Stage2K  = Stage1K * 2
	Stage3K  = Stage1K * 3
	Stage4K  = Stage1K * 4
	Stage5K  = Stage1K * 5
	Stage6K  = Stage1K * 6
	Stage7K  = Stage1K * 7
	Stage8K  = Stage1K * 8
	Stage9K  = Stage1K * 9
	Stage10K = Stage1K * 10

	Sma1Start = 0
	Sma1End   = Stage2K
	Sma2Start = Stage5K
	Sma2End   = Stage7K
	Sma3Start = Stage9K
	Sma3End   = Stage10K
)

type Config struct {
	PolymDegree   uint
	K             uint
	RBThres       uint
	EpochInterval uint64
	PosStartTime  int64
	MinerKey      *keystore.Key
	Dbpath        string
	NodeCfg       *node.Config
	Dkg1End       uint64
	Dkg2Begin     uint64
	Dkg2End       uint64
	SignBegin     uint64
	SignEnd       uint64
}

var DefaultConfig = Config{
	1,
	SlotCount / 10,
	3,
	0,
	0,
	nil,
	"",
	nil,
	Stage1K - 1,
	Stage3K,
	Stage5K - 1,
	Stage7K,
	Stage8K - 1,
}

func Cfg() *Config {
	return &DefaultConfig
}

func (c *Config) GetMinerAddr() common.Address {
	if c.MinerKey == nil {
		return common.Address{}
	}

	return c.MinerKey.Address
}

func (c *Config) GetMinerBn256PK() *bn256.G1 {
	if c.MinerKey == nil {
		return nil
	}

	return new(bn256.G1).Set(c.MinerKey.PrivateKey3.PublicKeyBn256.G1)
}

func (c *Config) GetMinerBn256SK() *big.Int {
	if c.MinerKey == nil {
		return nil
	}

	return new(big.Int).Set(c.MinerKey.PrivateKey3.D)
}
