package slotleader

import (
	"errors"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
)

var (
	errNoStateDbInstance = errors.New("Do not have stateDb instance now")
)

const (
	// SafeBack2k is use to get a static safe block in 1k slots before
	SafeBack2k = uint64(posconfig.SlotCount * 1 / 10)
)

// GetCurrentStateDb use to get statedb instance of current state.
func (s *SlotLeaderSelection) GetCurrentStateDb() (stateDb *state.StateDB, err error) {
	return s.getCurrentStateDb()
}

func (s *SlotLeaderSelection) getCurrentStateDb() (stateDb *state.StateDB, err error) {
	s.updateToLastStateDb()
	if s.stateDb == nil {
		return nil, errNoStateDbInstance
	}
	return s.stateDb, nil
}

func (s *SlotLeaderSelection) updateToLastStateDb() {
	stateDb, err := s.blockChain.StateAt(s.blockChain.CurrentBlock().Root())
	if err != nil {
		log.Error("Update stateDb error in SlotLeaderSelection.updateToLastStateDb", "error", err.Error())
	}
	s.stateDb = stateDb
}

func (s *SlotLeaderSelection) getLastEpochIDFromChain() uint64 {
	lastEpochID := uint64((s.blockChain.CurrentBlock().Difficulty().Int64() >> 32))
	return lastEpochID
}

func (s *SlotLeaderSelection) getLastSlotIDFromChain() uint64 {
	curSlotID := uint64((s.blockChain.CurrentBlock().Difficulty().Int64() >> 8) & 0x00ffffff)
	return curSlotID
}

func (s *SlotLeaderSelection) getBlockChainHeight() uint64 {
	return s.blockChain.CurrentBlock().NumberU64()
}
