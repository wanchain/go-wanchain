package pos

import "wanpos/cloudflare"

type Config struct {
	PolymDegree uint
	K uint
	MinRBProposerCnt uint
}

var DefaultConfig Config = Config{1,10,1}

func Cfg() *Config {
	return &DefaultConfig;
}




///>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>test>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func GetRBProposerGroup(epochId uint64) []bn256.G1 {
	return nil
}


///<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<test<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<


