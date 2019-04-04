package incentive

import (
	"github.com/wanchain/go-wanchain/common"
)

type Activity struct {
	EpLeader   []common.Address
	EpActivity []int
	RpLeader   []common.Address
	RpActivity []int
	SltLeader  []common.Address
	SlBlocks   []int
	SlActivity float64
}
