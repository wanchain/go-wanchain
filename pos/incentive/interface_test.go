package incentive

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
)

var (
	addrsCount = 20
	epAddrs    = make([]common.Address, addrsCount)
	epPks      = make([]*ecdsa.PublicKey, addrsCount)
	rpAddrs    = make([]common.Address, addrsCount)
	slAddrs    = make([]common.Address, addrsCount)
	epActs     = make([]int, addrsCount)
	rpActs     = make([]int, addrsCount)
	slBlks     = make([]int, addrsCount)
)

func testgetEpLeader(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int) {
	return epAddrs, epActs
}

func testgetRProposer(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int) {
	return rpAddrs, rpActs
}

func testgetSltLeader(chain consensus.ChainReader, epochID uint64, slotCount int) ([]common.Address, []int, float64, int) {
	return epAddrs, slBlks, 1, 0
}

func clearTestAddrs() {
	for i := 0; i < addrsCount; i++ {
		epAddrs[i] = common.Address{}
		epPks[i] = nil
	}

	for i := 0; i < addrsCount; i++ {
		rpAddrs[i] = common.Address{}
		rpActs[i] = 0
	}

	for i := 0; i < addrsCount; i++ {
		slAddrs[i] = epAddrs[i]
		slBlks[i] = 0
	}
}

func generateTestAddrs() {
	for i := 0; i < addrsCount; i++ {
		key, _ := crypto.GenerateKey()
		epAddrs[i] = crypto.PubkeyToAddress(key.PublicKey)
		epPks[i] = &key.PublicKey
		epActs[i] = 1
		if (i+1)%10 == 0 {
			//	epActs[i] = 0
		}
	}

	for i := 0; i < addrsCount; i++ {
		key, _ := crypto.GenerateKey()
		rpAddrs[i] = crypto.PubkeyToAddress(key.PublicKey)
		rpActs[i] = 1
	}

	for i := 0; i < addrsCount; i++ {
		slAddrs[i] = epAddrs[i]
		slBlks[i] = posconfig.SlotCount / addrsCount
	}
}

func TestSetActivityInterface(t *testing.T) {
	generateTestAddrs()
	setActivityInterface(testgetEpLeader, testgetRProposer, testgetSltLeader)
}

var (
	delegateStakerCount        = 40
	delegateStakerMap          = make(map[common.Address][]common.Address)
	delegateStakerProbilityMap = make(map[common.Address][]*big.Int)
)

func getInfo(epochID uint64, addr common.Address) (*vm.ValidatorInfo, error) {

	addrs := delegateStakerMap[addr]
	probs := delegateStakerProbilityMap[addr]
	count := len(addrs)
	if count == 0 {
		fmt.Println("Do not found address")
		return nil, errors.New("Do not found address")
	}

	client := make([]vm.ClientProbability, count)
	for i := 0; i < count; i++ {
		client[i].ValidatorAddr = addrs[i]
		client[i].WalletAddr = addrs[i]
		client[i].Probability = probs[i]
	}

	validator := &vm.ValidatorInfo{}
	validator.FeeRate = 1000
	validator.Infos = client

	return validator, nil
}

func setInfo(uint64, [][]vm.ClientIncentive) error { return nil }

func generateTestStaker() {
	for i := 0; i < addrsCount; i++ {
		stakers := make([]common.Address, delegateStakerCount)
		probility := make([]*big.Int, delegateStakerCount)
		for m := 0; m < delegateStakerCount; m++ {
			key, _ := crypto.GenerateKey()
			stakers[m] = crypto.PubkeyToAddress(key.PublicKey)
			probility[m] = big.NewInt(100)
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
			probility[m] = big.NewInt(100)
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

	setStakerInterface(getInfo, setInfo)
}
