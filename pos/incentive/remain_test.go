package incentive

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/pos/util"
)

func TestAddRemain(t *testing.T) {
	statedb.Reset(common.Hash{})

	remainConst := big.NewInt(0).SetUint64(99885844748858447)

	subsidy := getBaseSubsidyTotalForEpoch(statedb, subsidyReductionInterval)
	fmt.Println(subsidy.String(), util.FromWin(subsidy))

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

	subsidy2 := getBaseSubsidyTotalForEpoch(statedb, subsidyReductionInterval)
	fmt.Println(subsidy2.String(), util.FromWin(subsidy2))

	subsidy2 = subsidy2.Sub(subsidy2, subsidy)
	totalRemain := subsidy.Mul(subsidy2, big.NewInt(0).SetUint64(subsidyReductionInterval))
	fmt.Println(totalRemain.String())

	subValue := remainDef.Sub(remainDef, totalRemain).Int64()
	if subValue < 0 || subValue > 1e9 {
		fmt.Println("sub:", subValue)
		t.FailNow()
	}

	fmt.Println(subValue)
}
