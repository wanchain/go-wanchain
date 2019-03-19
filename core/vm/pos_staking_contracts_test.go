package vm

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/util/convert"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
	"reflect"
	"testing"
	"time"
)

type StakerStateDB struct {
}

func (StakerStateDB) CreateAccount(common.Address) {}

func (StakerStateDB) SubBalance(addr common.Address, pval *big.Int) {
	val, ok := stakerdb[addr]
	if ok && (&val).Cmp(pval) >= 0 {
		restVal := big.NewInt(0).Sub(&val, pval)
		stakerdb[addr] = *restVal
	}
}

func (StakerStateDB) AddBalance(addr common.Address, pval *big.Int) {
	val, ok := stakerdb[addr]
	if !ok {
		stakerdb[addr] = *pval
	} else {
		total := big.NewInt(0).Add(&val, pval)
		stakerdb[addr] = *total
	}
}
func (StakerStateDB) GetBalance(addr common.Address) *big.Int {
	defaulVal, _ := new(big.Int).SetString("00000000000000000000", 10)
	val, ok := stakerdb[addr]
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

	stakerAddr = crypto.PubkeyToAddress(*pb)

	stakerref = &dummyStakerRef{}
	stakerevm = NewEVM(Context{}, dummyStakerDB{ref: stakerref}, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})

	contract       = &Contract{value: big.NewInt(0).Mul(big.NewInt(10), ether), CallerAddress: stakerAddr}
	stakercontract = &PosStaking{}
)

func TestStakeIn(t *testing.T) {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	contract.CallerAddress = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	a := new(big.Int).Mul(big.NewInt(200000), ether)
	contract.Value().Set(a)
	eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())

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

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		t.Fatal("stakeIn called failed")
	}

	// check
	pub := crypto.ToECDSAPub(input.SecPk)
	secAddr := crypto.PubkeyToAddress(*pub)
	key := GetStakeInKeyHash(secAddr)
	bytes2 := evm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var info StakerInfo
	err = rlp.DecodeBytes(bytes2, &info)
	if err != nil {
		t.Fatal("stakeIn rlp decode failed")
	}
	if info.LockEpochs != input.LockEpochs.Uint64() ||
		info.FeeRate != input.FeeRate.Uint64() ||
		!reflect.DeepEqual(info.PubBn256, input.Bn256Pk) ||
		!reflect.DeepEqual(info.PubSec256, input.SecPk) {
		t.Fatal("stakeIn parse StakerInfo failed")
	}
	if info.Address != secAddr ||
		info.From != contract.CallerAddress ||
		info.Amount.Cmp(a) != 0 ||
		info.StakingEpoch != eidNow {
		t.Fatal("stakeIn from amount epoch address saved wrong")
	}
}

func TestDelegateIn(t *testing.T) {
	TestStakeIn(t)

	stakerevm.Time = big.NewInt(time.Now().Unix())
	contract.CallerAddress = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	a := new(big.Int).Mul(big.NewInt(20000), ether)
	contract.Value().Set(a)
	eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())

	var input DelegateInParam
	input.DelegateAddress = contract.CallerAddress

	bytes, err := cscAbi.Pack("delegateIn", input.DelegateAddress)
	if err != nil {
		t.Fatal("delegateIn pack failed")
	}

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		t.Fatal("delegateIn called failed")
	}
	// check
	key := GetStakeInKeyHash(input.DelegateAddress)
	bytes2 := evm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var infoS StakerInfo
	err = rlp.DecodeBytes(bytes2, &infoS)
	if err != nil {
		t.Fatal("delegateIn rlp decode failed")
	}

	lenth := len(infoS.Clients)
	if lenth <= 0 {
		t.Fatal("delegateIn save error")
	}
	info := infoS.Clients[lenth-1]
	if info.StakingEpoch != eidNow ||
		info.Amount.Cmp(a) != 0 ||
		info.Address != input.DelegateAddress {
		t.Fatal("delegateIn fields save error")
	}
}
