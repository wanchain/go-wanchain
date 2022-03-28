package incentive

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/pos/posconfig"
)

func TestCalcBaseSubsidy(t *testing.T) {
	base, _ := big.NewInt(0).SetString("6849315068493150684931", 10)

	subsidy := calcBaseSubsidy(firstPeriodReward)
	fmt.Println(subsidy.String())

	if subsidy.String() != base.String() {
		fmt.Println("error subsidy:", subsidy.String(), base)
		t.FailNow()
	}

	calcBaseSubsidy(nil)
}

func TestGetBaseSubsidyTotal(t *testing.T) {
	statedb.Reset(common.Hash{})
	year := big.NewInt(0).Mul(big.NewInt(2.5e6), big.NewInt(1e18))
	base := calcBaseSubsidy(year)
	fmt.Println(subsidyReductionInterval)

	for i := uint64(1); i < uint64(500); i++ {
		subsidy := getBaseSubsidyTotalForEpoch(statedb, subsidyReductionInterval*i)
		if subsidy.Uint64() == 0 {
			fmt.Println("finish", i)
			return
		}

		reduce := math.Pow(redutionRateBase, float64(i))
		fmt.Println(i, float64(subsidy.Uint64())/float64(1e18), reduce)
		base := calcPercent(base, reduce*100.0)
		if subsidy.Uint64() != base.Uint64() {
			fmt.Println("error: ", subsidy.Uint64(), base.String())
			t.FailNow()
		}
	}
}

func TestYearReward(t *testing.T) {
	posconfig.FirstEpochId = 18141

	for i := 0; i < 100; i++ {
		ret := YearReward(posconfig.FirstEpochId + uint64(365*i))
		fmt.Println(ret)
	}
}
