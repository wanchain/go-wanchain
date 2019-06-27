package incentive

import (
	"errors"
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
)

// delegate can calc the delegate division
func delegate(addrs []common.Address, values []*big.Int, epochID uint64) ([][]vm.ClientIncentive, *big.Int, error) {
	finalIncentive := make([][]vm.ClientIncentive, 0)
	remain := big.NewInt(0)
	for i := 0; i < len(addrs); i++ {
		stakers, division, totalProbility, err := getStakerInfoAndCheck(epochID, addrs[i])
		if err != nil {
			log.SyslogErr(err.Error())
			continue
		}

		incentive, subRemain := delegateDivision(addrs[i], values[i], stakers, division, totalProbility)
		finalIncentive = append(finalIncentive, incentive)
		remain.Add(remain, subRemain)
	}
	return finalIncentive, remain, nil
}

func getStakerInfoAndCheck(epochID uint64, addr common.Address) ([]vm.ClientProbability, uint64, *big.Int, error) {
	//stakers, division, totalProbility, err
	validator, err := getStakerInfo(epochID, addr)
	if err != nil {
		log.SyslogErr("getStakerInfo error", "error", err.Error())
		return nil, 0, nil, err
	}
	stakers := validator.Infos
	division := validator.FeeRate
	totalProbility := validator.TotalProbability
	if (stakers == nil) || (len(stakers) == 0) {
		log.SyslogErr("getStakerInfo get stakers error")
		return nil, 0, nil, errors.New("getStakerInfo get stakers error")
	}

	if division > 10000 {
		log.SyslogErr("getStakerInfo get division error")
		return nil, 0, nil, errors.New("getStakerInfo get division error")
	}

	// if totalProbility.Uint64() == 0 {
	// 	log.Error("getStakerInfo get totalProbility error")
	// 	return nil, 0, nil, errors.New("getStakerInfo get totalProbility error")
	// }

	return stakers, division, totalProbility, err
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

func sumStakerProbility(inputs []vm.ClientProbability) *big.Int {
	sumValue := big.NewInt(0)
	for i := 0; i < len(inputs); i++ {
		sumValue.Add(sumValue, inputs[i].Probability)
	}
	return sumValue
}

func delegateDivision(addr common.Address, value *big.Int, stakers []vm.ClientProbability,
	divisionPercent uint64, totalProbility *big.Int) ([]vm.ClientIncentive, *big.Int) {
	valueCeiling := value

	remain := big.NewInt(0).Sub(value, valueCeiling)

	//commission for delegator
	commission := calcPercent(valueCeiling, float64(divisionPercent)/100.0)
	lastValue := big.NewInt(0).Sub(valueCeiling, commission)
	tp := sumStakerProbility(stakers)
	result := make([]vm.ClientIncentive, len(stakers))

	for i := 0; i < len(stakers); i++ {
		result[i].ValidatorAddr = stakers[i].ValidatorAddr
		result[i].WalletAddr = stakers[i].WalletAddr

		result[i].Incentive = big.NewInt(0).Mul(lastValue, stakers[i].Probability)

		if result[i].Incentive.Cmp(big.NewInt(0)) != 0 {
			result[i].Incentive.Div(result[i].Incentive, tp)
		}

		// Position O is validator
		if i == 0 {
			result[0].Incentive.Add(result[0].Incentive, commission)
		}

		//Add check of incentive positive
		if result[i].Incentive.Cmp(big.NewInt(0)) == -1 {
			result[i].Incentive.SetUint64(0)
		}
	}
	return result, remain
}
