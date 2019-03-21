package slotleader

import (
	"errors"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/log"
)

var (
	errNoStateDbInstance = errors.New("Do not have stateDb instance now")
)

// GetCurrentStateDb use to get stateDB instance of current state.
func (s *SLS) GetCurrentStateDb() (stateDb *state.StateDB, err error) {
	return s.getCurrentStateDb()
}

func (s *SLS) getCurrentStateDb() (stateDb *state.StateDB, err error) {
	s.updateToLastStateDb()
	if s.stateDb == nil {
		return nil, errNoStateDbInstance
	}
	return s.stateDb, nil
}

func (s *SLS) updateToLastStateDb() {
	stateDb, err := s.blockChain.StateAt(s.blockChain.CurrentBlock().Root())
	if err != nil {
		log.Error("Update stateDb error in SLS.updateToLastStateDb", "error", err.Error())
	}
	s.stateDb = stateDb
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
