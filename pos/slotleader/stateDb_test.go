package slotleader

import (
	"fmt"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"
)

var s *SLS

func testInitSlotleader() {
	SlsInit()
	s = GetSlotLeaderSelection()

	// Create the database in memory or in a temporary directory.
	db, _ := ethdb.NewMemDatabase()
	gspec := core.DefaultPPOWTestingGenesisBlock()
	gspec.MustCommit(db)

	ce := ethash.NewFaker(db)
	bc, _ := core.NewBlockChain(db, gspec.Config, ce, vm.Config{},nil)

	s.Init(bc, &rpc.Client{}, &keystore.Key{})

	s.sendTransactionFn = testSender

}

func TestGetCurrentStateDb(t *testing.T) {

	posconfig.SelfTestMode = true
	testInitSlotleader()

	posconfig.SelfTestMode = false
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
	RmDB("epochGendb")
	posconfig.SelfTestMode = false
}
