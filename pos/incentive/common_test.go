package incentive

import (
	"fmt"
	"math/big"
	"testing"
)

func TestCalcPercent(t *testing.T) {
	total, ret := big.NewInt(0).SetString("1000000000000000000000000", 10)
	if !ret {
		t.FailNow()
	}
	value := calcPercent(total, 25.0/49.0*100.0)
	fmt.Println(value.String())

	if value.String() != "510204082000000000000000" {
		t.FailNow()
	}

	total.SetUint64(1000)
	value = calcPercent(total, 66)
	if value.String() != "660" {
		t.FailNow()
	}
}
