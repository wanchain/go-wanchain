package pos

type Config struct {
	PolymDegree uint
	K uint
	MinRBProposerCnt uint
}

var DefaultConfig Config = Config{1,10,1}

func Cfg() *Config {
	return &DefaultConfig;
}




