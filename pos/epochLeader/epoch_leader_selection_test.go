package epochLeader

import (
	"testing"
	"fmt"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/ethdb"
	"math/big"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/consensus/ethash"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/state"
	"time"
	"bytes"
)

type bproc struct{}

func (bproc) ValidateBody(*types.Block) error { return nil }
func (bproc) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}
func (bproc) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, *big.Int, error) {
	return nil, nil, new(big.Int), nil
}

func newTestBlockChain(fake bool) (*core.BlockChain, *core.ChainEnv) {

	db, _ := ethdb.NewMemDatabase()
	gspec := core.DefaultPlutoGenesisBlock()
	gspec.Difficulty = big.NewInt(1)
	gspec.MustCommit(db)
	engine := ethash.NewFullFaker(db)
	if !fake {
		engine = ethash.NewTester(db)
	}
	blockchain, err := core.NewBlockChain(db, gspec.Config, engine, vm.Config{})
	if err != nil {
		panic(err)
	}
	chainEnv := core.NewChainEnv(params.TestChainConfig, gspec, engine, blockchain, db)
	blockchain.SetValidator(bproc{})

	return blockchain, chainEnv
}

func TestGetGetEpochLeaders(t *testing.T) {

	epochID, slotID := slotleader.GetEpochSlotID()
	fmt.Println("epochID:", epochID, " slotID:", slotID)

    blkChain,_ := newTestBlockChain(true)

	epocher1 := NewEpocherWithLBN(blkChain,"rb1","epdb1")
	epocher2 := NewEpocherWithLBN(blkChain,"rb2","epdb2")

	epocher1.SelectLeadersLoop(0)

	time.Sleep(30*time.Second)
	epocher2.SelectLeadersLoop(0)

	epl1 := epocher1.GetEpochLeaders(0)
	epl2 := epocher2.GetEpochLeaders(0)

	if len(epl1) != len(epl2) {
		t.Fail()
	}

	for idx,val := range epl1 {
		if !bytes.Equal(val,epl2[idx]) {
			t.Fail()
		}
	}

	rbl1 := epocher1.GetRBProposerGroup(0)
	rbl2 := epocher2.GetRBProposerGroup(0)

	if len(epl1) != len(epl2) {
		t.Fail()
	}

	for idx,val := range rbl1 {
		if !bytes.Equal(val.Marshal(),rbl2[idx].Marshal()) {
			t.Fail()
		}
	}


}
