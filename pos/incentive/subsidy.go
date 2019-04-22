package incentive

import (
	"math"
	"math/big"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/pos/posconfig"
)

// calcBaseSubsidy calc the base subsidy of slot base on year. input is wei.
func calcBaseSubsidy(baseValueOfYear *big.Int, slotTime int64) *big.Int {
	secondPerYear := big.NewInt(365 * 24 * 3600)
	slotPerYear := secondPerYear.Div(secondPerYear, big.NewInt(slotTime))
	subsidyPerSlot := big.NewInt(0).Div(baseValueOfYear, slotPerYear)
	return subsidyPerSlot
}

// getBaseSubsidyTotalForSlot returns the subsidy amount a slot at the provided epoch
// should have. This is mainly used for determining how much the incentive for
// newly generated blocks awards as well as validating the incentive for blocks
// has the expected value.
//
// The subsidy is halved every SubsidyReductionInterval
// this is: baseSubsidy / 2^(epoch/SubsidyReductionInterval)
//
// At the target block generation rate for the main network, this is
// approximately every 5 years.
func getBaseSubsidyTotalForSlot(stateDb *state.StateDB, epochID uint64) *big.Int {
	// 2500000 wan coin for first year
	year := big.NewInt(0).Mul(big.NewInt(2.5e6), big.NewInt(1e18))
	baseSubsidy := calcBaseSubsidy(year, posconfig.SlotTime)

	redutionRateNow := math.Pow(redutionRateBase, float64(epochID/subsidyReductionInterval))
	baseSubsidyReduction := calcPercent(baseSubsidy, redutionRateNow*100.0)

	// If 1 years later, need add the remain incentive pool value of last 1 years
	if (epochID / subsidyReductionInterval) >= 1 {
		remainLastPeriod := getRemainIncentivePool(stateDb, epochID)
		remainLastPerYears := remainLastPeriod.Div(remainLastPeriod, big.NewInt(int64(redutionYears)))
		baseRemain := calcBaseSubsidy(remainLastPerYears, posconfig.SlotTime)
		baseSubsidyReduction.Add(baseSubsidyReduction, baseRemain)
	}

	return baseSubsidyReduction
}

// calcWanFromFoundation returns subsidy Of Epoch from wan foundation by Wei
func calcWanFromFoundation(stateDb *state.StateDB, epochID uint64) *big.Int {
	subsidyOfSlot := getBaseSubsidyTotalForSlot(stateDb, epochID)
	subsidyOfEpoch := big.NewInt(0).Mul(subsidyOfSlot, big.NewInt(posconfig.SlotCount))
	return subsidyOfEpoch
}

// calculateIncentivePool returns subsidy of Epoch from all
func calculateIncentivePool(stateDb *state.StateDB, epochID uint64) (total *big.Int, foundation *big.Int, gasPool *big.Int) {
	foundation = calcWanFromFoundation(stateDb, epochID)
	gasPool = getEpochGas(stateDb, epochID)
	total = big.NewInt(0).Add(foundation, gasPool)
	return
}
