package incentive

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/dedis/kyber/util/random"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos"
)

// Prepare a simulate stateDB ---------------------------------------------
type CTStateDB struct {
}

func (CTStateDB) CreateAccount(common.Address) {}

func (CTStateDB) SubBalance(addr common.Address, pVal *big.Int) {
	account[addr] = account[addr].Add(account[addr], pVal.Mul(pVal, big.NewInt(-1)))
}
func (CTStateDB) AddBalance(addr common.Address, pval *big.Int) {
	account[addr] = account[addr].Add(account[addr], pval)
}

func (CTStateDB) GetBalance(addr common.Address) *big.Int {
	return account[addr]
}

func (CTStateDB) GetNonce(common.Address) uint64                                         { return 0 }
func (CTStateDB) SetNonce(common.Address, uint64)                                        {}
func (CTStateDB) GetCodeHash(common.Address) common.Hash                                 { return common.Hash{} }
func (CTStateDB) GetCode(common.Address) []byte                                          { return nil }
func (CTStateDB) SetCode(common.Address, []byte)                                         {}
func (CTStateDB) GetCodeSize(common.Address) int                                         { return 0 }
func (CTStateDB) AddRefund(*big.Int)                                                     {}
func (CTStateDB) GetRefund() *big.Int                                                    { return nil }
func (CTStateDB) GetState(common.Address, common.Hash) common.Hash                       { return common.Hash{} }
func (CTStateDB) SetState(common.Address, common.Hash, common.Hash)                      {}
func (CTStateDB) Suicide(common.Address) bool                                            { return false }
func (CTStateDB) HasSuicided(common.Address) bool                                        { return false }
func (CTStateDB) Exist(common.Address) bool                                              { return false }
func (CTStateDB) Empty(common.Address) bool                                              { return false }
func (CTStateDB) RevertToSnapshot(int)                                                   {}
func (CTStateDB) Snapshot() int                                                          { return 0 }
func (CTStateDB) AddLog(*types.Log)                                                      {}
func (CTStateDB) AddPreimage(common.Hash, []byte)                                        {}
func (CTStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool)     {}
func (CTStateDB) ForEachStorageByteArray(common.Address, func(common.Hash, []byte) bool) {}

var (
	rbdb    = make(map[common.Hash][]byte)
	account = make(map[common.Address]*big.Int)
)

func (CTStateDB) GetStateByteArray(addr common.Address, hs common.Hash) []byte {
	return rbdb[hs]
}

func (CTStateDB) SetStateByteArray(addr common.Address, hs common.Hash, data []byte) {
	rbdb[hs] = data
}

type dummyCtRef struct {
	calledForEach bool
}

func (dummyCtRef) ReturnGas(*big.Int)          {}
func (dummyCtRef) Address() common.Address     { return common.Address{} }
func (dummyCtRef) Value() *big.Int             { return new(big.Int) }
func (dummyCtRef) SetCode(common.Hash, []byte) {}
func (d *dummyCtRef) ForEachStorage(callback func(key, value common.Hash) bool) {
	d.calledForEach = true
}
func (d *dummyCtRef) SubBalance(amount *big.Int) {}
func (d *dummyCtRef) AddBalance(amount *big.Int) {}
func (d *dummyCtRef) SetBalance(*big.Int)        {}
func (d *dummyCtRef) SetNonce(uint64)            {}
func (d *dummyCtRef) Balance() *big.Int          { return new(big.Int) }

type dummyCtDB struct {
	CTStateDB
	ref *dummyCtRef
}

var (
	nr    = 21
	thres = pos.Cfg().PolymDegree + 1

	db, _      = ethdb.NewMemDatabase()
	statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	ref        = &dummyCtRef{}
	evm        = vm.NewEVM(vm.Context{}, dummyCtDB{ref: ref}, params.TestChainConfig, vm.Config{EnableJit: false, ForceJit: false})
)

//-------------------------------------------------------------------------------

func TestAddEpochGas(t *testing.T) {

	testTimes := 100
	testMaxWan := int64(1)
	randNums := make([]*big.Int, 0)

	for index := 0; index < testTimes; index++ {
		for i := 0; i < testTimes; i++ {
			gas := random.Int(big.NewInt(0).Mul(big.NewInt(testMaxWan), big.NewInt(1e18)), random.New()) //1000 WAN
			AddEpochGas(statedb, gas, uint64(index))
			randNums = append(randNums, gas)
		}
	}

	for index := 0; index < testTimes; index++ {
		totalInEpoch := big.NewInt(0)
		for i := 0; i < testTimes; i++ {
			totalInEpoch = totalInEpoch.Add(totalInEpoch, randNums[index*testTimes+i])
		}

		gas := getEpochGas(statedb, uint64(index))
		if gas.String() != totalInEpoch.String() {
			t.FailNow()
		}
	}
}

func getInfo(common.Address) ([]common.Address, []*big.Int, float64, float64) { return nil, nil, 0, 0 }

func setInfo([]common.Address, []*big.Int) {}
func TestSetStakerInterface(t *testing.T) {
	SetStakerInterface(getInfo, setInfo)
}

func testgetEpLeader(stateDb *state.StateDB, epochID uint64) ([]common.Address, int)

func testgetRProposer(stateDb *state.StateDB, epochID uint64) ([]common.Address, int)

func testgetSltLeader(stateDb *state.StateDB, epochID uint64) ([]common.Address, []*big.Int, float64)

func TestSetActivityInterface(t *testing.T) {
	SetActivityInterface(testgetEpLeader, testgetRProposer, testgetSltLeader)
}

func TestCalcBaseSubsidy(t *testing.T) {

	base := uint64(665905631659056310)

	subsidy := calcBaseSubsidy(0)
	fmt.Println(subsidy.String())

	if subsidy.String() != "665905631659056310" {
		t.FailNow()
	}

	for i := uint64(1); i < uint64(500); i++ {
		subsidyReductionInterval := uint64((365 * 24 * 3600 * 5) / (pos.SlotTime * pos.SlotCount)) // Epoch count in 5 years
		fmt.Println(subsidyReductionInterval)
		subsidy = calcBaseSubsidy(subsidyReductionInterval * i)
		fmt.Println(subsidy.String())
		if subsidy.Uint64() == 0 {
			fmt.Println("finish", i)
			return
		}
		if subsidy.Uint64() != (base >> i) {
			fmt.Println("error: ", subsidy.Uint64(), base/(i+1))
			t.FailNow()
		}
	}
}

func TestRun(t *testing.T) {
	testTimes := 100

	for i := 0; i < testTimes; i++ {
		Run(statedb, uint64(i))
		Run(statedb, uint64(i))
		Run(statedb, uint64(i))

		total, foundation, gasPool := calculateIncentivePool(statedb, uint64(i))
		if total.String() != (big.NewInt(0).Add(foundation, gasPool)).String() {
			t.FailNow()
		}
	}
}
