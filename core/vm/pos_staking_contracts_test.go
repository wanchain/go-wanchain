package vm

import (
	"testing"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/common"
	"math/big"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/pos/cloudflare"
)

type StakerStateDB struct {
}

func (StakerStateDB) CreateAccount(common.Address) {}

func (StakerStateDB) SubBalance(common.Address, *big.Int) {}
func (StakerStateDB) AddBalance(addr common.Address, pval *big.Int) {

}
func (StakerStateDB) GetBalance(addr common.Address) *big.Int {
	defaulVal, _ := new(big.Int).SetString("10000000000000000000", 10)
	return defaulVal
}
func (StakerStateDB) GetNonce(common.Address) uint64                                         { return 0 }
func (StakerStateDB) SetNonce(common.Address, uint64)                                        {}
func (StakerStateDB) GetCodeHash(common.Address) common.Hash                                 { return common.Hash{} }
func (StakerStateDB) GetCode(common.Address) []byte                                          { return nil }
func (StakerStateDB) SetCode(common.Address, []byte)                                         {}
func (StakerStateDB) GetCodeSize(common.Address) int                                         { return 0 }
func (StakerStateDB) AddRefund(*big.Int)                                                     {}
func (StakerStateDB) GetRefund() *big.Int                                                    { return nil }
func (StakerStateDB) GetState(common.Address, common.Hash) common.Hash                       { return common.Hash{} }
func (StakerStateDB) SetState(common.Address, common.Hash, common.Hash)                      {}
func (StakerStateDB) Suicide(common.Address) bool                                            { return false }
func (StakerStateDB) HasSuicided(common.Address) bool                                        { return false }
func (StakerStateDB) Exist(common.Address) bool                                              { return false }
func (StakerStateDB) Empty(common.Address) bool                                              { return false }
func (StakerStateDB) RevertToSnapshot(int)                                                   {}
func (StakerStateDB) Snapshot() int                                                          { return 0 }
func (StakerStateDB) AddLog(*types.Log)                                                      {}
func (StakerStateDB) AddPreimage(common.Hash, []byte)                                        {}
func (StakerStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool)     {}
func (StakerStateDB) ForEachStorageByteArray(common.Address, func(common.Hash, []byte) bool) {}

var (
	stakerepochId = uint64(0)
	stakerdb = make(map[common.Hash][]byte)
	stakergroupdb = make(map[uint64][]bn256.G1)
//	stakerdb = make(map[uint64]*big.Int)
)

func (StakerStateDB) GetStateByteArray(addr common.Address, hs common.Hash) []byte {
	return rbdb[hs]
}

func (StakerStateDB) SetStateByteArray(addr common.Address, hs common.Hash, data []byte) {
	rbdb[hs] = data
}

type dummyStakerRef struct {
	calledForEach bool
}

func (dummyStakerRef) ReturnGas(*big.Int)          {}
func (dummyStakerRef) Address() common.Address     { return common.Address{} }
func (dummyStakerRef) Value() *big.Int             { return new(big.Int) }
func (dummyStakerRef) SetCode(common.Hash, []byte) {}
func (d *dummyStakerRef) ForEachStorage(callback func(key, value common.Hash) bool) {
	d.calledForEach = true
}
func (d *dummyStakerRef) SubBalance(amount *big.Int) {}
func (d *dummyStakerRef) AddBalance(amount *big.Int) {}
func (d *dummyStakerRef) SetBalance(*big.Int)        {}
func (d *dummyStakerRef) SetNonce(uint64)            {}
func (d *dummyStakerRef) Balance() *big.Int          { return new(big.Int) }

type dummyStajerDB struct {
	CTStateDB
	ref *dummyCtRef
}

var (
	stakernr = 10
	stdb, _      = ethdb.NewMemDatabase()
	stakerstatedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	stakerref = &dummyCtRef{}
	stakerevm = NewEVM(Context{}, dummyCtDB{ref: ref}, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})

	stakercontract = &pos_staking{}

)


func TestStakeIn(t *testing.T) {

}

func TestStakeOut(t *testing.T) {

}




