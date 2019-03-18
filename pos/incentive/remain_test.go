package incentive

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/pos/posconfig"
)

func TestAddRemain(t *testing.T) {
	statedb.Reset(common.Hash{})

	remainConst := big.NewInt(0).SetUint64(99885844748858447)

	subsidy := getBaseSubsidyTotalForSlot(statedb, subsidyReductionInterval)
	fmt.Println(subsidy.String(), float64(subsidy.Uint64())/float64(1e18))

	fmt.Println(subsidyReductionInterval)
	for i := uint64(0); i < subsidyReductionInterval; i++ {
		addRemainIncentivePool(statedb, i, remainConst)
	}

	remain := getRemainIncentivePool(statedb, subsidyReductionInterval)
	fmt.Println(remain)
	remainDef := big.NewInt(0).Mul(remainConst, big.NewInt(0).SetUint64(subsidyReductionInterval))

	if remain.String() != remainDef.String() {
		fmt.Println(remain, remainDef)
		t.FailNow()
	}

	subsidy2 := getBaseSubsidyTotalForSlot(statedb, subsidyReductionInterval)
	fmt.Println(subsidy2.String(), float64(subsidy2.Uint64())/float64(1e18))

	subsidy2 = subsidy2.Sub(subsidy2, subsidy)
	totalRemain := subsidy.Mul(subsidy2, big.NewInt(0).SetUint64(subsidyReductionInterval*posconfig.SlotCount))
	fmt.Println(totalRemain.String())

	subValue := remainDef.Sub(remainDef, totalRemain).Int64()
	if subValue < 0 || subValue > 1e9 {
		fmt.Println("sub:", subValue)
		t.FailNow()
	}

	fmt.Println(subValue)
}
