package incentive

import "math/big"

func calcPercent(total *big.Int, percent int) *big.Int {
	value := big.NewInt(0).Mul(total, big.NewInt(int64(percent)))
	value.Div(value, big.NewInt(100))
	return value
}
