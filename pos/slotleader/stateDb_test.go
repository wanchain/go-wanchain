package slotleader

import (
	"fmt"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"testing"

	"github.com/wanchain/go-wanchain/accounts/keystore"

	"github.com/wanchain/go-wanchain/consensus/ethash"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/rpc"
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
