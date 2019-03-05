package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
)

// delegate can calc the delegate division
func delegate(addrs []common.Address, values []*big.Int, epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	finalIncentive := make([][]vm.ClientIncentive, len(addrs))
	remain := big.NewInt(0)
	for i := 0; i < len(addrs); i++ {
		stakers, division, totalProbility, err := getStakerInfo(addrs[i], epochID)
		if err != nil {
			return nil, nil, err
		}
		var subRemain *big.Int
		finalIncentive[i], subRemain = delegateDivision(addrs[i], values[i], stakers, division, totalProbility)
		remain.Add(remain, subRemain)
	}
	return finalIncentive, remain, nil
}

func ceilingCalc(value *big.Int, totalPercent float64) *big.Int {
	if totalPercent <= ceilingPercentS0 {
		return value
	}

	if totalPercent > 2*ceilingPercentS0 {
		return big.NewInt(0)
	}

	percent := 1 - ((totalPercent-ceilingPercentS0)*(totalPercent-ceilingPercentS0))/(ceilingPercentS0*ceilingPercentS0)
	return calcPercent(value, percent*100.0)
}

func calcTotalPercent(stakers []vm.ClientProbability, totalProbility *big.Int) float64 {
	totalCalc := big.NewInt(0)
	for i := 0; i < len(stakers); i++ {
		totalCalc.Add(totalCalc, stakers[i].Probability)
	}
	totalCalc.Mul(totalCalc, big.NewInt(100))
	percent := totalCalc.Div(totalCalc, totalProbility)
	return float64(percent.Uint64())
}

func sumStakerProbility(inputs []vm.ClientProbability) *big.Int {
	sumValue := big.NewInt(0)
	for i := 0; i < len(inputs); i++ {
		sumValue.Add(sumValue, inputs[i].Probability)
	}
	return sumValue
}

func delegateDivision(addr common.Address, value *big.Int, stakers []vm.ClientProbability, divisionPercent uint64, totalProbility *big.Int) ([]vm.ClientIncentive, *big.Int) {
	totalPercent := calcTotalPercent(stakers, totalProbility)
	valueCeiling := ceilingCalc(value, totalPercent)

	remain := big.NewInt(0).Sub(value, valueCeiling)

	//commission for delegator
	commission := calcPercent(valueCeiling, float64(divisionPercent))
	lastValue := big.NewInt(0).Sub(valueCeiling, commission)
	tp := sumStakerProbility(stakers)
	result := make([]vm.ClientIncentive, len(stakers))

	for i := 0; i < len(stakers); i++ {
		result[i].Addr = stakers[i].Addr
		result[i].Incentive = big.NewInt(0).Mul(lastValue, stakers[i].Probability)
		result[i].Incentive.Div(result[i].Incentive, tp)

		if stakers[i].Addr.String() == addr.String() {
			result[i].Incentive.Add(result[i].Incentive, commission)
		}
	}
	return result, remain
}
