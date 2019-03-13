package vm

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/params"
	"math/big"
	"testing"
	"time"
)

type StakerStateDB struct {
}

func (StakerStateDB) CreateAccount(common.Address) {}

func (StakerStateDB) SubBalance(addr common.Address, pval *big.Int) {
	val,ok :=stakerdb[addr]
	if ok && (&val).Cmp(pval) >= 0{
		restVal := big.NewInt(0).Sub(&val,pval)
		stakerdb[addr] = *restVal
	}
}

func (StakerStateDB) AddBalance(addr common.Address, pval *big.Int) {
	val,ok :=stakerdb[addr]
	if !ok {
		stakerdb[addr] = *pval
	} else {
		total := big.NewInt(0).Add(&val,pval)
		stakerdb[addr] = *total
	}
}
func (StakerStateDB) GetBalance(addr common.Address) *big.Int {
	defaulVal, _ := new(big.Int).SetString("00000000000000000000", 10)
	val,ok :=stakerdb[addr]
	if ok {
		return &val
	} else {
		return defaulVal
	}
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
	stakerdb = make(map[common.Address]big.Int)
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

type dummyStakerDB struct {
	StakerStateDB
	ref *dummyStakerRef
}

var (
	pb = crypto.ToECDSAPub(common.FromHex("0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70"))

	stakerAddr =crypto.PubkeyToAddress(*pb)

	stakerValue = big.NewInt(0).Mul(big.NewInt(10),ether)

	//stdb, _      = ethdb.NewMemDatabase()
	//stakerstatedb, _ = state.New(common.Hash{}, state.NewDatabase(stdb))
	stakerref = &dummyStakerRef{}
	stakerevm = NewEVM(Context{}, dummyStakerDB{ref: stakerref}, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})

	contract = &Contract{value:big.NewInt(0).Mul(big.NewInt(10),ether),CallerAddress:stakerAddr}
	stakercontract = &PosStaking{}

)


func TestStakeInAndOutTimeOut(t *testing.T) {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	var input StakeInParam
	//input.SecPk = common.FromHex("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	input.SecPk = common.FromHex("0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70")
	input.Bn256Pk = common.FromHex("0x150b2b3230d6d6c8d1c133ec42d82f84add5e096c57665ff50ad071f6345cf45191fd8015cea72c4591ab3fd2ade12287c28a092ac0abf9ea19c13eb65fd4910")
	input.LockEpochs = big.NewInt(10)
	input.FeeRate = big.NewInt(100)

	bytes, err := cscAbi.Pack("stakeIn", input.SecPk, input.Bn256Pk, input.LockEpochs, input.FeeRate)
	if err != nil {
		t.Fatal("stakeIn pack failed:", err)
	}
	println(len(bytes))
	_, err = stakercontract.Run(bytes,contract,stakerevm)

	if err != nil {
		t.Fatal("stakeIn called failed")
	}
	//stakerevm.StateDB.AddBalance(WanCscPrecompileAddr,stakerValue)
	//
	//time.Sleep(20*time.Second)
	//
	//stakeOutInput := "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000003635c9adc5dea00000000000000000000000000000000000000000000000000000000000000000008230783164323535626362396138376336336463376636646532356336333961656164343339393161396566666231376139333336366565386663373538353463313732373330396235313534363932656137343564613839333431383431333563666263363737353933383736303137386239323235346235336237323864613265000000000000000000000000000000000000000000000000000000000000"
	//_,err = stakercontract.StakeOut(common.FromHex(stakeOutInput),contract,stakerevm)
	//
	//if err != nil {
	//	t.Fail()
	//}
	//
	//cscValue := stakerevm.StateDB.GetBalance(WanCscPrecompileAddr)
	//if cscValue.Cmp(big.NewInt(0)) != 0 {
	//	t.Fail()
	//}
	//
	//afterValue := stakerevm.StateDB.GetBalance(stakerAddr)
	//if stakerValue.Cmp(afterValue) != 0 {
	//	t.Fail()
	//}
}

func TestDelegateIn(t *testing.T) {
	var input DelegateInParam
	input.DelegateAddress = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	//input.LockEpochs = big.NewInt(10)

	// config
	stakerevm.Time = big.NewInt(time.Now().Unix())
	contract.CallerAddress = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")

	//bytes, err := cscAbi.Pack("delegateIn", input.Address, input.LockEpochs)
	bytes, err := cscAbi.Pack("delegateIn", input.DelegateAddress)
	if err != nil {
		t.Fatal("delegateIn pack failed")
	}
	println(len(bytes))
	_, err = stakercontract.Run(bytes,contract,stakerevm)

}


func TestStakeInAndOutNotTimeOut(t *testing.T) {
	stakeIninput := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000c328093e61ee4000000000000000000000000000000000000000000000000000000000000000000106307830346437646666653565303664326337303234643962623933663637356238323432653731393031656536366131626665336665353336393332346330613735626636663033336463346166363566356430666537303732653938373838666366613637303931396235626463303436663163613931663238646666353964623730307831353062326233323330643664366338643163313333656334326438326638346164643565303936633537363635666635306164303731663633343563663435313931666438303135636561373263343539316162336664326164653132323837633238613039326163306162663965613139633133656236356664343931300000000000000000000000000000000000000000000000000000"

	_,err := stakercontract.StakeIn(common.FromHex(stakeIninput),contract,stakerevm)

	if err != nil {
		t.Fail()
	}

	//stakeOutInput :=   "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000008ac7230489e80000000000000000000000000000000000000000000000000000000000000000008430783034643764666665356530366432633730323464396262393366363735623832343265373139303165653636613162666533666535333639333234633061373562663666303333646334616636356635643066653730373265393837383866636661363730393139623562646330343666316361393166323864666635396462373000000000000000000000000000000000000000000000000000000000"
	//_,err = stakercontract.StakeOut(common.FromHex(stakeOutInput),contract,stakerevm)
	//
	//if err.Error() != "lockTIme did not reach" {
	//	t.Fail()
	//}

}

//func TestRunFake(t *testing.T) {
//	runFake(stakerevm.StateDB)
//}



