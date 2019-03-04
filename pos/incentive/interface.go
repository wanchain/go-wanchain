package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
)

type getStakerInfoFn func(common.Address) ([]common.Address, []*big.Int, float64, float64)

type setStakerInfoFn func([]common.Address, []*big.Int)

type getEpochLeaderInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int)

type getRandomProposerInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int)

type getSlotLeaderInfoFn func(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int, float64)

var getStakerInfo getStakerInfoFn

var setStakerInfo setStakerInfoFn

var getEpochLeaderInfo getEpochLeaderInfoFn

var getRandomProposerInfo getRandomProposerInfoFn

var getSlotLeaderInfo getSlotLeaderInfoFn

// SetStakerInterface is use for Staker module to set its interface
func SetStakerInterface(get getStakerInfoFn, set setStakerInfoFn) {
	getStakerInfo = get
	setStakerInfo = set
}

// SetActivityInterface is use for get activty module to set its interface
func SetActivityInterface(getEpl getEpochLeaderInfoFn, getRnp getRandomProposerInfoFn, getSlr getSlotLeaderInfoFn) {
	getEpochLeaderInfo = getEpl
	getRandomProposerInfo = getRnp
	getSlotLeaderInfo = getSlr
}
