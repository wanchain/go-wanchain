package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/common"
)

// delegate can calc the delegate division
func delegate(addrs []common.Address, values []*big.Int, epochID uint64) ([]common.Address, []*big.Int) {
	finalAddrs := make([]common.Address, 0)
	finalValues := make([]*big.Int, 0)

	for i := 0; i < len(addrs); i++ {
		stakers, probilities, division, percent := getStakerInfo(addrs[i], epochID)
		if division == 1 {
			finalAddrs = append(finalAddrs, addrs[i])
			finalValues = append(finalValues, values[i])
		} else {
			oneAddrs, oneValues := delegateDivision(addrs[i], values[i], stakers, probilities, division, percent)
			finalAddrs = append(finalAddrs, oneAddrs...)
			finalValues = append(finalValues, oneValues...)
		}
	}

	return finalAddrs, finalValues
}

func ceilingCalc(value *big.Int, totalPercent float64) *big.Int {
	if totalPercent <= ceilingPercentS0 {
		return value
	}

	if totalPercent > 2*ceilingPercentS0 {
		return big.NewInt(0)
	}

	percent := 1 - ((totalPercent-ceilingPercentS0)*(totalPercent-ceilingPercentS0))/(ceilingPercentS0*ceilingPercentS0)
	return calcPercent(value, int(percent*100))
}

func delegateDivision(addr common.Address, value *big.Int, stakers []common.Address, probilities []*big.Int, divisionPercent int, totalPercent float64) ([]common.Address, []*big.Int) {

	valueCeiling := ceilingCalc(value, totalPercent)

	//commission for delegator
	commission := calcPercent(valueCeiling, divisionPercent)
	lastValue := big.NewInt(0).Sub(valueCeiling, commission)
	totalProbilities := sum(probilities)
	valueForStakers := make([]*big.Int, len(stakers))

	for i := 0; i < len(stakers); i++ {
		valueForStakers[i] = big.NewInt(0).Mul(lastValue, probilities[i])
		valueForStakers[i].Div(valueForStakers[i], totalProbilities)

		if stakers[i].String() == addr.String() {
			valueForStakers[i].Add(valueForStakers[i], commission)
		}
	}
	return stakers, valueForStakers
}
