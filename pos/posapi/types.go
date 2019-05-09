package posapi

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/math"
)

type Activity struct {
	EpLeader    []common.Address
	EpActivity  []int
	RpLeader    []common.Address
	RpActivity  []int
	SltLeader   []common.Address
	SlBlocks    []int
	SlActivity  float64
	SlCtrlCount int
}

type PayInfo struct {
	Addr      common.Address
	Incentive *math.HexOrDecimal256
}
