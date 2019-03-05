package incentive

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/state"
)

func getEpochLeaderActivity(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int) {
	return nil, nil
}

func getRandomProposerActivity(stateDb *state.StateDB, epochID uint64) ([]common.Address, []int) {
	return nil, nil
}

func getSlotLeaderActivity(chain consensus.ChainReader, epochID uint64, slotCount int) ([]common.Address, []int, float64) {
	currentNumber := chain.CurrentHeader().Number.Uint64()
	miners := make(map[common.Address]int)
	for i := currentNumber - 1; i > 0; i-- {
		header := chain.GetHeaderByNumber(i)
		epID := getEpochIDFromDifficulty(header.Difficulty)
		if epID == epochID {
			cnt, ok := miners[header.Coinbase]
			if ok {
				cnt++
				miners[header.Coinbase] = cnt
			} else {
				miners[header.Coinbase] = 1
			}
		}

		if epID < epochID {
			break
		}
	}

	addrs := make([]common.Address, 0)
	blocks := make([]int, 0)

	for k, v := range miners {
		addrs = append(addrs, k)
		blocks = append(blocks, v)
	}

	epochBlockCnt := sumIntArray(blocks)
	if epochBlockCnt > slotCount {
		epochBlockCnt = slotCount
	}
	activePercent := float64(epochBlockCnt) / float64(slotCount)
	return addrs, blocks, activePercent
}
