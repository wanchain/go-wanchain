package slotleader

import (
	"errors"

	"github.com/ethereum/go-ethereum/pos/posconfig"

	"github.com/ethereum/go-ethereum/core/state"
)

var (
	errNoStateDbInstance = errors.New("Do not have stateDb instance now")
)

// GetCurrentStateDb use to get stateDB instance of current state.
func (s *SLS) GetCurrentStateDb() (stateDb *state.StateDB, err error) {
	return s.getCurrentStateDb()
}

func (s *SLS) getCurrentStateDb() (stateDb *state.StateDB, err error) {
	if posconfig.SelfTestMode {
		return s.stateDbTest, nil
	}
	return s.blockChain.StateAt(s.blockChain.CurrentBlock().Root())
}

func (s *SLS) getLastEpochIDFromChain() uint64 {
	lastEpochID := uint64((s.blockChain.CurrentBlock().Difficulty().Int64() >> 32))
	return lastEpochID
}

func (s *SLS) getLastSlotIDFromChain() uint64 {
	curSlotID := uint64((s.blockChain.CurrentBlock().Difficulty().Int64() >> 8) & 0x00ffffff)
	return curSlotID
}

func (s *SLS) getBlockChainHeight() uint64 {
	return s.blockChain.CurrentBlock().NumberU64()
}

func (s *SLS) getBlockTime(number uint64) uint64 {
	//return s.blockChain.GetBlockByNumber(number).Time().Uint64()
	return s.blockChain.GetBlockByNumber(number).Time()
}

func (s *SLS) getEpochIDFromBlockNumber(number uint64) uint64 {
	return uint64(s.blockChain.GetBlockByNumber(number).Difficulty().Int64() >> 32)
}
