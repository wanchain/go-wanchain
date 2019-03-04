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

	addrsFinal, valueFinal := delegate(epAddrs, values, 0)

	for i := 0; i < len(addrsFinal); i++ {
		fmt.Println(addrsFinal[i].Hex())
		fmt.Println("-------->")
		fmt.Println(valueFinal[i].String())
		fmt.Println("<--------")
	}

}
