package incentive

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/core/vm"
)

// delegatesCalc can calc the delegate division
func TestDelegate(t *testing.T) {
	TestSetActivityInterface(t)
	TestSetStakerInterface(t)

	epAddrs, _ := getEpochLeaderInfo(statedb, 0)

	values := make([]*big.Int, len(epAddrs))
	for i := 0; i < len(values); i++ {
		values[i] = big.NewInt(1e18)
	}

	finalIncentive, remain, err := delegate(epAddrs, values, 0)

	if err != nil {
		t.FailNow()
	}

	fmt.Println("remains:", remain)

	for i := 0; i < len(finalIncentive); i++ {
		fmt.Println("group:", i)
		for m := 0; m < len(finalIncentive[i]); m++ {
			fmt.Println(finalIncentive[i][m].Addr.Hex())
			fmt.Println("-------->")
			fmt.Println(finalIncentive[i][m].Incentive.String())
			fmt.Println("<--------")
		}
	}
}

func TestCeilingCalc(t *testing.T) {
	value := big.NewInt(100)
	percent := 0.02
	calcValue := ceilingCalc(value, percent)

	if calcValue.String() != value.String() {
		t.FailNow()
	}

	percent = ceilingPercentS0
	calcValue = ceilingCalc(value, percent)

	if calcValue.String() != value.String() {
		t.FailNow()
	}

	percent = ceilingPercentS0 * 2
	calcValue = ceilingCalc(value, percent)

	if calcValue.Int64() != 0 {
		t.FailNow()
	}

	percent = ceilingPercentS0 * 3
	calcValue = ceilingCalc(value, percent)

	if calcValue.Int64() != 0 {
		t.FailNow()
	}

	percent = ceilingPercentS0 * 1.5
	calcValue = ceilingCalc(value, percent)

	if calcValue.Int64() != 75 {
		fmt.Println(calcValue)
		t.FailNow()
	}
}

func TestCalcTotalPercent(t *testing.T) {
	testValues := make([]vm.ClientProbability, 3)
	testValues[0].Probability = big.NewInt(100)
	testValues[1].Probability = big.NewInt(200)
	testValues[2].Probability = big.NewInt(300)

	totalProb := big.NewInt(1200)

	percent := calcTotalPercent(testValues, totalProb)

	if math.Abs(percent-50) > 0.001 {
		fmt.Println(percent)
		t.FailNow()
	}
}
