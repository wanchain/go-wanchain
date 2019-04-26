package incentive

import (
	"github.com/wanchain/go-wanchain/common"
)

// Activity is a struct use for outside api call return
type Activity struct {
	EpLeader   []common.Address
	EpActivity []int
	RpLeader   []common.Address
	RpActivity []int
	SltLeader  []common.Address
	SlBlocks   []int
	SlActivity float64
}
