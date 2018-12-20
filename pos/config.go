package pos

import "github.com/wanchain/pos/cloudflare"

type Config struct {
	PolymDegree uint
	K uint
	MinRBProposerCnt uint
	EpochInterval uint64
	PosStartTime int64
}

var DefaultConfig Config = Config{1,10,1,0,0}

func Cfg() *Config {
	return &DefaultConfig;
}




///>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>test>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func GetRBProposerGroup(epochId uint64) []bn256.G1 {
	return nil
}


///<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<test<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<


