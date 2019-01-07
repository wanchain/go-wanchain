package slotleader

import (
	"errors"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/log"
)

func (s *SlotLeaderSelection) getStateDb() (stateDb *state.StateDB, err error) {
	s.updateStateDB()
	if s.stateDb == nil {
		return nil, errors.New("Do not have stateDb instance now")
	}
	return s.stateDb, nil
}

func (s *SlotLeaderSelection) updateStateDB() {
	stateDb, err := s.blockChain.StateAt(s.blockChain.CurrentBlock().Root())
	if err != nil {
		log.Error("Update stateDb error in SlotLeaderSelection.updateStateDB", "error", err.Error())
	}
	s.stateDb = stateDb
}
