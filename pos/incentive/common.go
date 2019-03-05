package incentive

import (
	"math"
	"math/big"
)

func calcPercent(total *big.Int, percent float64) *big.Int {
	value := big.NewInt(0).Mul(total, big.NewInt(round(percent)))
	value.Div(value, big.NewInt(100))
	return value
}

func round(x float64) int64 {
	return int64(math.Floor(x + 0.5))
}
