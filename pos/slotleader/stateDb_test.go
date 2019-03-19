package slotleader

import (
	"fmt"
	"testing"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/consensus/ethash"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/rpc"
)

var s *SlotLeaderSelection

func testInitSlotleader() {
	s = GetSlotLeaderSelection()
	rc := &rpc.Client{}

	// Create the database in memory or in a temporary directory.
	db, _ := ethdb.NewMemDatabase()
	gspec := core.DefaultPPOWTestingGenesisBlock()
	gspec.MustCommit(db)

	ce := ethash.NewFaker(db)
	bc, _ := core.NewBlockChain(db, gspec.Config, ce, vm.Config{})

	s.Init(bc, rc, &keystore.Key{}, nil)
}

func TestGetCurrentStateDb(t *testing.T) {
	testInitSlotleader()
	stateDb, err := s.GetCurrentStateDb()
	if err != nil || stateDb == nil {
		t.FailNow()
	}

	epochID := s.getLastEpochIDFromChain()
	slotID := s.getLastSlotIDFromChain()
	number := s.getBlockChainHeight()
	if number != 0 || epochID != 0 || slotID != 0 {
		t.FailNow()
	}

	fmt.Println(epochID, slotID)
}
