package incentive

import (
	"fmt"
	"math/big"
	"testing"
)

// delegatesCalc can calc the delegate division
func TestDelegatesCalc(t *testing.T) {
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
