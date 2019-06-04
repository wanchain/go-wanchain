package incentive

import (
	"fmt"
	"math/big"
	"testing"
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
			fmt.Println(finalIncentive[i][m].WalletAddr.Hex())
			fmt.Println("-------->")
			fmt.Println(finalIncentive[i][m].Incentive.String())
			if m == 0 {
				if finalIncentive[i][m].Incentive.String() != "122500000000000000" {
					t.FailNow()
				}
			} else {
				if finalIncentive[i][m].Incentive.String() != "22500000000000000" {
					t.FailNow()
				}
			}
			fmt.Println("<--------")
		}
	}
}

func TestCeilingCalc(t *testing.T) {
	value := big.NewInt(100)
	percent := 2.0
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
