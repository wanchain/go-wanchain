package pos

import (
	"github.com/wanchain/pos/cloudflare"
	"math/big"
	"github.com/wanchain/go-wanchain/common"
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

var DefaultConfig Config = Config{1,10,1, 0, 0, new(bn256.G1).ScalarBaseMult(big.NewInt(1)), big.NewInt(1)}

func Cfg() *Config {
	return &DefaultConfig;
}




///>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>test>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func GetRBProposerGroup(epochId uint64) []bn256.G1 {
	return nil
}

func GetGenesisRandon() *big.Int {
	return big.NewInt(1)
}

func GetTxFrom() common.Address {
	return common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
}

func GetRBPrecompileAddr() common.Address {
	return common.BytesToAddress(big.NewInt(7700).Bytes())
}



///<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<test<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<


