package slotleader

import (
	"errors"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
)

const (
	// SafeBack2k is use to get a static safe block in 2k slots before
	SafeBack2k = uint64(pos.SlotCount * 2 / 10)
)

func (s *SlotLeaderSelection) getStateDb() (stateDb *state.StateDB, err error) {
	s.updateStateDB()
	if s.stateDb == nil {
		return nil, errors.New("Do not have stateDb instance now")
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

func (s *SlotLeaderSelection) updateStateDB() {
	curNumber := s.blockChain.CurrentBlock().NumberU64()
	curSlotID := uint64((s.blockChain.CurrentBlock().Difficulty().Int64() >> 8) & 0x00ffffff)

	if uint64(curSlotID) < SlotStage1 {
		log.Warn("Current Slot ID is less than SlotStage1 (4k), do not use a SafeBack2k one, use last one")
		s.updateToLastStateDb()
		return
	}

	if curSlotID <= SafeBack2k {
		log.Warn("Current Slot ID is less than SafeBack2k, use last one")
		s.updateToLastStateDb()
		return
	}

	targetSlotID := curSlotID - SafeBack2k

	backIndex := uint64(1)
	for {
		block := s.blockChain.GetBlockByNumber(curNumber - backIndex)
		if block == nil {
			log.Error("Can not find a safe block in SlotLeaderSelection.updateStateDB use last one")
			backIndex = 0
			break
		}
		slotID := uint64((block.Difficulty().Int64() >> 8) & 0x00ffffff)
		if slotID <= targetSlotID {
			break
		}
	}

	stateDb, err := s.blockChain.StateAt(s.blockChain.GetBlockByNumber(curNumber - backIndex).Root())
	if err != nil {
		log.Error("Update stateDb error in SlotLeaderSelection.updateStateDB", "error", err.Error())
	}
	s.stateDb = stateDb
}
