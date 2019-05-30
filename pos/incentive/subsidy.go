package incentive

import (
	"math"
	"math/big"

	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/log"
)

// calcBaseSubsidy calc the base subsidy of epoch base on subsidyReductionInterval. input is wei.
func calcBaseSubsidy(baseValue *big.Int) *big.Int {
	if baseValue == nil {
		log.SyslogErr("calcBaseSubsidy input is nil")
		return big.NewInt(0)
	}
	subsidyPerEpoch := big.NewInt(0).Div(baseValue, big.NewInt(0).SetUint64(subsidyReductionInterval))
	return subsidyPerEpoch
}

// getBaseSubsidyTotalForEpoch returns the subsidy amount a epoch at the provided epoch
// should have. This is mainly used for determining how much the incentive for
// newly generated blocks awards as well as validating the incentive for blocks
// has the expected value.
func getBaseSubsidyTotalForEpoch(stateDb *state.StateDB, epochID uint64) *big.Int {
	if stateDb == nil {
		log.SyslogErr("getBaseSubsidyTotalForEpoch with an empty stateDb")
		return big.NewInt(0)
	}

	if epochID < posconfig.FirstEpochId {
		return big.NewInt(0)
	}

	epochIDOffset := epochID - posconfig.FirstEpochId

	baseSubsidy := calcBaseSubsidy(firstPeriodReward)

	redutionRateNow := math.Pow(redutionRateBase, float64(epochIDOffset/subsidyReductionInterval))
	baseSubsidyReduction := calcPercent(baseSubsidy, redutionRateNow*100.0)

	log.Info("getBaseSubsidyTotalForEpoch",
		"FirstEpochId", posconfig.FirstEpochId,
		"epochID", epochID,
		"reduceTimes", epochIDOffset/subsidyReductionInterval,
		"reduceRate", redutionRateNow,
		"base", baseSubsidy.String(),
		"afterReduce", baseSubsidyReduction.String(),
	)

	// If 1 period later, need add the remain incentive pool value of last period
	if (epochIDOffset / subsidyReductionInterval) >= 1 {
		baseRemain := calcBaseSubsidy(getRemainIncentivePool(stateDb, epochIDOffset))
		baseSubsidyReduction.Add(baseSubsidyReduction, baseRemain)
	}

	return baseSubsidyReduction
}

// calcWanFromFoundation returns subsidy Of Epoch from wan foundation by Wei
func calcWanFromFoundation(stateDb *state.StateDB, epochID uint64) *big.Int {
	if stateDb == nil {
		log.SyslogErr("calcWanFromFoundation with an empty stateDb")
		return big.NewInt(0)
	}

	return getBaseSubsidyTotalForEpoch(stateDb, epochID)
}

// calculateIncentivePool returns subsidy of Epoch from all
func calculateIncentivePool(stateDb *state.StateDB, epochID uint64) (total *big.Int, foundation *big.Int, gasPool *big.Int) {
	if stateDb == nil {
		log.SyslogErr("calculateIncentivePool with an empty stateDb")
		return big.NewInt(0), big.NewInt(0), big.NewInt(0)
	}

	foundation = calcWanFromFoundation(stateDb, epochID)
	gasPool = getEpochGas(stateDb, epochID)
	total = big.NewInt(0).Add(foundation, gasPool)
	return
}
