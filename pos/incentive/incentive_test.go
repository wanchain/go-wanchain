package incentive

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/pos/posconfig"
)

// Prepare a simulate stateDB ---------------------------------------------
var (
	db, _      = ethdb.NewMemDatabase()
	statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
)

func TestRun(t *testing.T) {
	posconfig.Init(nil, 4)
	Init(getInfo, setInfo, testGetRBAddress)
	TestSetActivityInterface(t)
	TestSetStakerInterface(t)

	testTimes := 1

	for i := 0; i < testTimes; i++ {
		for m := 0; m < posconfig.SlotCount; m++ {
			if !Run(&TestChainReader{}, statedb, uint64(i)) {
				t.FailNow()
			}
		}

		total, foundation, gasPool := calculateIncentivePool(statedb, uint64(i))
		if total.String() != (big.NewInt(0).Add(foundation, gasPool)).String() {
			t.FailNow()
		}
	}

	sumTotal := big.NewInt(0)
	for k, v := range delegateStakerMap {
		fmt.Println("Delegate of:", k.Hex())
		fmt.Println("---------->")
		for i := 0; i < len(v); i++ {
			sum := statedb.GetBalance(v[i])
			fmt.Println("addr:", v[i].Hex(), "balance:", sum.String())
			sumTotal.Add(sumTotal, sum)
		}
		fmt.Println("<----------")
	}

	fmt.Println("sum Total:", sumTotal.String())
}

func TestRunFail(t *testing.T) {
	if Run(nil, nil, 0) {
		t.FailNow()
	}
}

func TestCheckTotalValue(t *testing.T) {
	total := big.NewInt(1000)

	remain := big.NewInt(100)

	toPay := make([][]vm.ClientIncentive, 3)
	toPay[0] = make([]vm.ClientIncentive, 1)
	toPay[1] = make([]vm.ClientIncentive, 2)
	toPay[2] = make([]vm.ClientIncentive, 3)
	toPay[0][0].Incentive = big.NewInt(100)
	toPay[1][0].Incentive = big.NewInt(100)
	toPay[1][1].Incentive = big.NewInt(100)
	toPay[2][0].Incentive = big.NewInt(100)
	toPay[2][1].Incentive = big.NewInt(100)
	toPay[2][2].Incentive = big.NewInt(100)

	sumPay := sumToPay(toPay)

	if !checkTotalValue(total, sumPay, remain) {
		t.FailNow()
	}

	total = big.NewInt(700)
	if !checkTotalValue(total, sumPay, remain) {
		t.FailNow()
	}

	total = big.NewInt(699)
	if checkTotalValue(total, sumPay, remain) {
		t.FailNow()
	}
}

func TestInit(t *testing.T) {
	Init(getInfo, setInfo, testGetRBAddress)
}

func TestInitFail(t *testing.T) {
	Init(nil, nil, nil)
}
