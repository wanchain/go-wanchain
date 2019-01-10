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
	"time"
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
	StakerStateDB
	ref *dummyStakerRef
}

var (
	stakernr = 10
	stdb, _      = ethdb.NewMemDatabase()
	stakerstatedb, _ = state.New(common.Hash{}, state.NewDatabase(stdb))
	stakerref = &dummyStakerRef{}
	stakerevm = NewEVM(Context{}, dummyStajerDB{ref: stakerref}, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})

	contract = &Contract{value:big.NewInt(0).Mul(big.NewInt(10),ether)}
	stakercontract = &Pos_staking{}

)


func TestStakeInAndOutTimeOut(t *testing.T) {

	stakeIninput := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000008ac7230489e800000000000000000000000000000000000000000000000000000000000000000106307830346437646666653565303664326337303234643962623933663637356238323432653731393031656536366131626665336665353336393332346330613735626636663033336463346166363566356430666537303732653938373838666366613637303931396235626463303436663163613931663238646666353964623730307831353062326233323330643664366338643163313333656334326438326638346164643565303936633537363635666635306164303731663633343563663435313931666438303135636561373263343539316162336664326164653132323837633238613039326163306162663965613139633133656236356664343931300000000000000000000000000000000000000000000000000000"

	_,err := stakercontract.StakeIn(common.FromHex(stakeIninput),contract,stakerevm)

	if err != nil {
		t.Fail()
	}

	time.Sleep(20*time.Second)

	stakeOutInput :=   "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000008ac7230489e80000000000000000000000000000000000000000000000000000000000000000008430783034643764666665356530366432633730323464396262393366363735623832343265373139303165653636613162666533666535333639333234633061373562663666303333646334616636356635643066653730373265393837383866636661363730393139623562646330343666316361393166323864666635396462373000000000000000000000000000000000000000000000000000000000"
	_,err = stakercontract.StakeOut(common.FromHex(stakeOutInput),contract,stakerevm)

	if err != nil {
		t.Fail()
	}

}


func TestStakeInAndOutNotTimeOut(t *testing.T) {

	stakeIninput := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000c328093e61ee4000000000000000000000000000000000000000000000000000000000000000000106307830346437646666653565303664326337303234643962623933663637356238323432653731393031656536366131626665336665353336393332346330613735626636663033336463346166363566356430666537303732653938373838666366613637303931396235626463303436663163613931663238646666353964623730307831353062326233323330643664366338643163313333656334326438326638346164643565303936633537363635666635306164303731663633343563663435313931666438303135636561373263343539316162336664326164653132323837633238613039326163306162663965613139633133656236356664343931300000000000000000000000000000000000000000000000000000"

	_,err := stakercontract.StakeIn(common.FromHex(stakeIninput),contract,stakerevm)

	if err != nil {
		t.Fail()
	}

	stakeOutInput :=   "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000008ac7230489e80000000000000000000000000000000000000000000000000000000000000000008430783034643764666665356530366432633730323464396262393366363735623832343265373139303165653636613162666533666535333639333234633061373562663666303333646334616636356635643066653730373265393837383866636661363730393139623562646330343666316361393166323864666635396462373000000000000000000000000000000000000000000000000000000000"
	_,err = stakercontract.StakeOut(common.FromHex(stakeOutInput),contract,stakerevm)

	if err.Error() != "lockTIme did not reach" {
		t.Fail()
	}

}





