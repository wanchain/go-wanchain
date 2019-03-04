package incentive

import (
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/crypto"
)

var (
	addrsCount = 20
	epAddrs    = make([]common.Address, addrsCount)
	rpAddrs    = make([]common.Address, addrsCount)
	slAddrs    = make([]common.Address, addrsCount)
	epActs     = make([]int, addrsCount)
	rpActs     = make([]int, addrsCount)
	slBlks     = make([]int, addrsCount)
)

func testgetEpLeader(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int) {
	return epAddrs, epActs
}

func testgetRProposer(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int) {
	return rpAddrs, rpActs
}

func testgetSltLeader(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int, float64) {
	return slAddrs, slBlks, 1
}

func generateTestAddrs() {
	for i := 0; i < addrsCount; i++ {
		key, _ := crypto.GenerateKey()
		epAddrs[i] = crypto.PubkeyToAddress(key.PublicKey)
		epActs[i] = 1
	}

	for i := 0; i < addrsCount; i++ {
		key, _ := crypto.GenerateKey()
		rpAddrs[i] = crypto.PubkeyToAddress(key.PublicKey)
		rpActs[i] = 1
	}

	for i := 0; i < addrsCount; i++ {
		key, _ := crypto.GenerateKey()
		slAddrs[i] = crypto.PubkeyToAddress(key.PublicKey)
		slBlks[i] = 20
	}
}
func TestSetActivityInterface(t *testing.T) {
	generateTestAddrs()
	SetActivityInterface(testgetEpLeader, testgetRProposer, testgetSltLeader)
}

var (
	delegateStakerCount        = 40
	delegateStakerMap          = make(map[common.Address][]common.Address)
	delegateStakerProbilityMap = make(map[common.Address][]*big.Int)
)

func getInfo(addr common.Address, epochID uint64) ([]common.Address, []*big.Int, int, float64) {
	return delegateStakerMap[addr], delegateStakerProbilityMap[addr], 10, 0.025
}

func setInfo([]common.Address, []*big.Int, uint64) {}

func generateTestStaker() {
	for i := 0; i < addrsCount; i++ {
		stakers := make([]common.Address, delegateStakerCount)
		probility := make([]*big.Int, delegateStakerCount)
		for m := 0; m < delegateStakerCount; m++ {
			key, _ := crypto.GenerateKey()
			stakers[m] = crypto.PubkeyToAddress(key.PublicKey)
			probility[m] = key.D
		}
		stakers[0] = epAddrs[i]
		delegateStakerMap[epAddrs[i]] = stakers
		delegateStakerProbilityMap[epAddrs[i]] = probility
	}

	for i := 0; i < addrsCount; i++ {
		stakers := make([]common.Address, delegateStakerCount)
		probility := make([]*big.Int, delegateStakerCount)
		for m := 0; m < delegateStakerCount; m++ {
			key, _ := crypto.GenerateKey()
			stakers[m] = crypto.PubkeyToAddress(key.PublicKey)
			probility[m] = key.D
		}
		stakers[0] = rpAddrs[i]
		delegateStakerMap[rpAddrs[i]] = stakers
		delegateStakerProbilityMap[rpAddrs[i]] = probility
	}
}
func TestSetStakerInterface(t *testing.T) {
	generateTestAddrs()
	generateTestStaker()

	// fmt.Println(delegateStakerMap)
	// fmt.Println(delegateStakerProbilityMap)

	SetStakerInterface(getInfo, setInfo)
}
