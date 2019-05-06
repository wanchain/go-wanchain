package incentive

import (
	"math"
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos/util"
)

func calcPercent(total *big.Int, percent float64) *big.Int {
	scale := 10000000.0
	percentRound := round(percent * scale)
	value := big.NewInt(0).Mul(total, big.NewInt(percentRound))
	value.Div(value, big.NewInt(round(100.0*scale)))
	return value
}

func round(x float64) int64 {
	return int64(math.Floor(x + 0.5))
}

func getEpochIDFromDifficulty(difficulty *big.Int) uint64 {
	epochID, _ := util.GetEpochSlotIDFromDifficulty(difficulty)
	return epochID
}

func sumIntArray(array []int) int {
	sum := 0
	for i := 0; i < len(array); i++ {
		sum += array[i]
	}
	return sum
}

func addressInclude(addr common.Address, addrs []common.Address) bool {
	for i := 0; i < len(addrs); i++ {
		if addr.Hex() == addrs[i].Hex() {
			return true
		}
	}
	return false
}

func sumIncentive(payment [][]vm.ClientIncentive) *big.Int {
	sum := big.NewInt(0)
	if payment == nil {
		return sum
	}

	for i := 0; i < len(payment); i++ {
		for m := 0; m < len(payment[i]); m++ {
			sum.Add(sum, payment[i][m].Incentive)
		}
	}

	return sum
}
