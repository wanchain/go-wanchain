package incentive

import "math/big"

func sum(inputs []*big.Int) *big.Int {
	sumValue := big.NewInt(0)
	for i := 0; i < len(inputs); i++ {
		sumValue.Add(sumValue, inputs[i])
	}
	return sumValue
}

func calcPercent(total *big.Int, percent int) *big.Int {
	value := big.NewInt(0).Mul(total, big.NewInt(int64(percent)))
	value.Div(value, big.NewInt(100))
	return value
}
