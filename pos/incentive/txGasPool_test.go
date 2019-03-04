package incentive

import (
	"math/big"
	"testing"

	"github.com/dedis/kyber/util/random"
)

func TestAddEpochGas(t *testing.T) {

	testTimes := 100
	testMaxWan := int64(1)
	randNums := make([]*big.Int, 0)

	for index := 0; index < testTimes; index++ {
		for i := 0; i < testTimes; i++ {
			gas := random.Int(big.NewInt(0).Mul(big.NewInt(testMaxWan), big.NewInt(1e18)), random.New()) //1000 WAN
			AddEpochGas(statedb, gas, uint64(index))
			randNums = append(randNums, gas)
		}
	}

	for index := 0; index < testTimes; index++ {
		totalInEpoch := big.NewInt(0)
		for i := 0; i < testTimes; i++ {
			totalInEpoch = totalInEpoch.Add(totalInEpoch, randNums[index*testTimes+i])
		}

		gas := getEpochGas(statedb, uint64(index))
		if gas.String() != totalInEpoch.String() {
			t.FailNow()
		}
	}
}
