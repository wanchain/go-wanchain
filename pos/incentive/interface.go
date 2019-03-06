package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
)

type GetStakerInfoFn func(uint64, common.Address) ([]vm.ClientProbability, uint64, *big.Int, error)

type SetStakerInfoFn func(uint64, [][]vm.ClientIncentive) error

type GetEpochLeaderInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int)

type GetRandomProposerInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int)

type GetSlotLeaderInfoFn func(chain consensus.ChainReader, epochID uint64, slotCount int) ([]common.Address, []int, float64)

type GetRandomProposerAddressFn func(epochID uint64) []common.Address

var getStakerInfo GetStakerInfoFn

var setStakerInfo SetStakerInfoFn

var getEpochLeaderInfo GetEpochLeaderInfoFn

var getRandomProposerInfo GetRandomProposerInfoFn

var getSlotLeaderInfo GetSlotLeaderInfoFn

var getRandomProposerAddress GetRandomProposerAddressFn

// SetStakerInterface is used for Staker module to set its interface
func SetStakerInterface(get GetStakerInfoFn, set SetStakerInfoFn) {
	getStakerInfo = get
	setStakerInfo = set
}

// SetActivityInterface is used for get activty module to set its interface
func SetActivityInterface(getEpl GetEpochLeaderInfoFn, getRnp GetRandomProposerInfoFn, getSlr GetSlotLeaderInfoFn) {
	getEpochLeaderInfo = getEpl
	getRandomProposerInfo = getRnp
	getSlotLeaderInfo = getSlr
}

// SetRBAddressInterface is used to get random proposer address of epoch
func SetRBAddressInterface(getRBAddress GetRandomProposerAddressFn) {
	getRandomProposerAddress = getRBAddress
}
