package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
)

type getStakerInfoFn func(common.Address, uint64) ([]vm.ClientProbability, uint64, *big.Int, error)

type setStakerInfoFn func([][]vm.ClientIncentive, uint64) error

type getEpochLeaderInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int)

type getRandomProposerInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int)

type getSlotLeaderInfoFn func(chain consensus.ChainReader, epochID uint64, slotCount int) ([]common.Address, []int, float64)

type getRandomProposerAddressFn func(epochID uint64) []common.Address

var getStakerInfo getStakerInfoFn

var setStakerInfo setStakerInfoFn

var getEpochLeaderInfo getEpochLeaderInfoFn

var getRandomProposerInfo getRandomProposerInfoFn

var getSlotLeaderInfo getSlotLeaderInfoFn

var getRandomProposerAddress getRandomProposerAddressFn

// SetStakerInterface is used for Staker module to set its interface
func SetStakerInterface(get getStakerInfoFn, set setStakerInfoFn) {
	getStakerInfo = get
	setStakerInfo = set
}

// SetActivityInterface is used for get activty module to set its interface
func SetActivityInterface(getEpl getEpochLeaderInfoFn, getRnp getRandomProposerInfoFn, getSlr getSlotLeaderInfoFn) {
	getEpochLeaderInfo = getEpl
	getRandomProposerInfo = getRnp
	getSlotLeaderInfo = getSlr
}

// SetRBAddressInterface is used to get random proposer address of epoch
func SetRBAddressInterface(getRBAddress getRandomProposerAddressFn) {
	getRandomProposerAddress = getRBAddress
}
