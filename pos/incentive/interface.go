package incentive

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/vm"
)

// GetStakerInfoFn is a function use to get staker info
type GetStakerInfoFn func(uint64, common.Address) (*vm.ValidatorInfo,  error)

// SetStakerInfoFn is a function use to set payment info
type SetStakerInfoFn func(uint64, [][]vm.ClientIncentive) error

// GetEpochLeaderInfoFn is a function use to get epoch activity and address
type GetEpochLeaderInfoFn func(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int)

// GetRandomProposerInfoFn is use to get rb group and activity
type GetRandomProposerInfoFn func(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int)

// GetSlotLeaderInfoFn is use to get slotleader address and activity
type GetSlotLeaderInfoFn func(chain consensus.ChainReader, epochID uint64, slotCount int) ([]common.Address, []int, float64, int)

// GetRandomProposerAddressFn is use to get rb group address
type GetRandomProposerAddressFn func(epochID uint64) []vm.Leader

var getStakerInfo GetStakerInfoFn

var setStakerInfo SetStakerInfoFn

var getEpochLeaderInfo GetEpochLeaderInfoFn

var getRandomProposerInfo GetRandomProposerInfoFn

var getSlotLeaderInfo GetSlotLeaderInfoFn

var getRandomProposerAddress GetRandomProposerAddressFn

// setStakerInterface is used for Staker module to set its interface
func setStakerInterface(get GetStakerInfoFn, set SetStakerInfoFn) {
	getStakerInfo = get
	setStakerInfo = set
}

// setActivityInterface is used for get activty module to set its interface
func setActivityInterface(getEpl GetEpochLeaderInfoFn, getRnp GetRandomProposerInfoFn, getSlr GetSlotLeaderInfoFn) {
	getEpochLeaderInfo = getEpl
	getRandomProposerInfo = getRnp
	getSlotLeaderInfo = getSlr
}

// setRBAddressInterface is used to get random proposer address of epoch
func setRBAddressInterface(getRBAddress GetRandomProposerAddressFn) {
	getRandomProposerAddress = getRBAddress
}
