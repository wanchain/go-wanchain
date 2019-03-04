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

func getInfo(common.Address) ([]common.Address, []*big.Int, float64, float64) { return nil, nil, 0, 0 }

func setInfo([]common.Address, []*big.Int) {}
func TestSetStakerInterface(t *testing.T) {
	SetStakerInterface(getInfo, setInfo)
}
