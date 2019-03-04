package incentive

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/pos"
)

func TestCalcBaseSubsidy(t *testing.T) {
	base := uint64(199771689497716894) //SlotTime:3, SlotCount:40

	year := big.NewInt(0).Mul(big.NewInt(2.1e6), big.NewInt(1e18))
	yearBase := year
	subsidy := calcBaseSubsidy(year, 3)
	fmt.Println(subsidy.String())

	if subsidy.Uint64() != base {
		fmt.Println("error subsidy:", subsidy.String(), base)
		t.FailNow()
	}

	bigBase := big.NewInt(0).SetUint64(base)
	bigBase = bigBase.Mul(bigBase, big.NewInt(365*24*3600/3))

	fmt.Println(bigBase)

	fmt.Println(yearBase)

	fmt.Println(yearBase.Sub(yearBase, bigBase))

	if yearBase.Int64() < 0 || yearBase.Int64() > 1e9 {
		t.FailNow()
	}
}

func TestGetBaseSubsidyTotalForSlot(t *testing.T) {
	statedb.Reset(common.Hash{})
	year := big.NewInt(0).Mul(big.NewInt(2.1e6), big.NewInt(1e18))
	base := calcBaseSubsidy(year, pos.SlotTime).Uint64()
	for i := uint64(1); i < uint64(500); i++ {
		fmt.Println(subsidyReductionInterval)
		subsidy := getBaseSubsidyTotalForSlot(statedb, subsidyReductionInterval*i)
		fmt.Println(subsidy.String(), float64(subsidy.Uint64())/float64(1e18))
		if subsidy.Uint64() == 0 {
			fmt.Println("finish", i)
			return
		}
		if subsidy.Uint64() != (base >> i) {
			fmt.Println("error: ", subsidy.Uint64(), base/(i+1))
			t.FailNow()
		}
	}
}
