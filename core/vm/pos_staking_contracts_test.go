package vm

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"io/ioutil"
	"math/big"
	"os"
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
	dirname, _ = ioutil.TempDir(os.TempDir(), "pos_staking")
	posStakingDB *ethdb.LDBDatabase = nil
)

func clearDb() {
	if posStakingDB != nil {
		posStakingDB.Close()
		posStakingDB = nil
	}
	os.RemoveAll(dirname)
}

func initDb() bool {
	dbTmp, err := ethdb.NewLDBDatabase(dirname, 0, 0)
	if err != nil {
		println(err.Error())
		return false
	}
	posStakingDB = dbTmp
	return true
}

func reset() bool {
	clearDb()
	t := time.Now().Unix()
	posconfig.FirstEpochId, _ = util.CalEpochSlotID(uint64(t))
	evmtime = 0
	return initDb()
}

func (StakerStateDB) GetStateByteArray(addr common.Address, hs common.Hash) []byte {
	ret, _ := posStakingDB.Get(hs[:])
	return ret
}

func (StakerStateDB) SetStateByteArray(addr common.Address, hs common.Hash, data []byte) {
	posStakingDB.Put(hs[:], data)
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
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	// value >= 10000
	err := doStakeIn(9999)
	if err == nil {
		t.Fatal("should be error, stake < 10,000")
	}
	// fee too high
	err = doStakeInWithParam(10000, 10001)
	if err == nil {
		t.Fatal("fee should too high")
	}

	// normal, should success
	err = doStakeIn(10000)
	if err != nil {
		t.Fatal(err.Error())
	}

	// can't join twice
	err = doStakeIn(10000)
	if err == nil {
		t.Fatal("should not stakeIn twice")
	}

	// if posconfig.FirstEpochId == 0
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	posconfig.FirstEpochId = 0
	err = doStakeIn(10000)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !reset() {
		t.Fatal("pos staking db init error")
	}
	// value <= 10,500,000
	err = doStakeIn(10590001)
	if err == nil {
		t.Fatal("should be error, stake > 10,500,000")
	}
	err = doStakeIn(10500000)
	if err != nil {
		t.Fatal(err.Error())
	}
	clearDb()
}

func TestDelegateIn(t *testing.T) {
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	// stake holder not exist
	err := doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1)
	if err == nil {
		t.Fatal("should not find stake holder")
	}
	// FeeRate == 10000
	err = doStakeInWithParam(200000, 10000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 10000)
	if err == nil {
		t.Fatal("should failed, fee == 10000")
	}
	// < MinValidatorStake
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	err = doStakeIn(PSMinValidatorStake - 1)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 10000)
	if err == nil {
		t.Fatal("should failed, fee == 10000")
	}
	// contract.value.Cmp(minDelegatorStake) < 0
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	err = doStakeIn(PSMinValidatorStake)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 99)
	if err == nil {
		t.Fatal("first delegate stake should >= 100")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 100)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1)
	if err != nil {
		t.Fatal(err.Error())
	}

	////////////////////////////
	if !reset() {
		t.Fatal("pos staking db init error")
	}

	err = doStakeIn(200000)
	if err != nil {
		t.Fatal(err.Error())
	}
	// normal delegate, should success
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 200000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1800001)
	if err == nil {
		t.Fatal("should only delegate ten times")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1800000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 30000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 300001)
	if err == nil {
		t.Fatal("should only delegate ten times")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 300000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"),730000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 7240001)
	if err == nil {
		t.Fatal("should be error, stake > 10,500,000")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 7240000)
	if err != nil {
		t.Fatal(err.Error())
	}
	clearDb()
}

func TestDelegateOut(t *testing.T) {
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	// stake holder not exist
	err := doDelegateOut(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"))
	if err == nil {
		t.Fatal("should not find stake holder")
	}

	err = doStakeIn(200000)
	if err != nil {
		t.Fatal(err.Error())
	}
	// !found
	err = doDelegateOut(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"))
	if err == nil {
		t.Fatal("should not find delegate")
	}
	// good
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 500)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOut(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"))
	if err != nil {
		t.Fatal(err.Error())
	}
	// delegator has existed
	err = doDelegateOut(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"))
	if err == nil {
		t.Fatal("should failed if delegator has existed")
	}
}

func TestPartnerIn(t *testing.T) {
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	// stake holder not exist
	err := doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err == nil {
		t.Fatal("should be failed if there is no stake holder")
	}
	// realLockEpoch < 0
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	err = doStakeIn(20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	setEpochTime(posconfig.FirstEpochId + 11)
	err = doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err == nil {
		t.Fatal("should be failed if realLockEpoch < 0")
	}
	setEpochTime(posconfig.FirstEpochId + 10)
	err = doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err != nil {
		t.Fatal(err.Error())
	}
	// realLockEpoch > PSMaxEpochNum
	// TODO: li hua check
	setEpochTime(posconfig.FirstEpochId - 90 + 10 - 1)
	err = doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 0)
	if err == nil {
		t.Fatal("should be failed if realLockEpoch > PSMaxEpochNum")
	}
	setEpochTime(posconfig.FirstEpochId- 90 + 10)
	err = doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err != nil {
		t.Fatal(err.Error())
	}
	// if length >= maxPartners
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	err = doStakeIn(20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doPartnerOne(common.HexToAddress("0x11117c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doPartnerOne(common.HexToAddress("0x22227c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doPartnerOne(common.HexToAddress("0x33337c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doPartnerOne(common.HexToAddress("0x44447c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doPartnerOne(common.HexToAddress("0x55557c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doPartnerOne(common.HexToAddress("0x66667c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err == nil {
		t.Fatal("Too many partners, should fail")
	}
	///////////////////////
	// amount check
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	err = doStakeIn(20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 20000)
	if err == nil {
		t.Fatal("should be error, stake + partner < 50000")
	}
	err = doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 30000)
	if err != nil {
		t.Fatal("should be error, stake > 10,500,000")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 500000)
	if err == nil {
		t.Fatal("should be error, stake > 10*(stake + partner)")
	}
	err = doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err != nil {
		t.Fatal("should be success")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 9430001)
	if err == nil {
		t.Fatal("should be error, stake > 10,500,000")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 9430000)
	if err != nil {
		t.Fatal("should be success2")
	}
}

func TestStakeAppend(t *testing.T) {
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	// stake holder should exist
	err := doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err == nil {
		t.Fatal("should be failed if stake holder not exist")
	}
	err = doStakeIn(20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	// contract.CallerAddress != stakerInfo.From
	err = doStakeAppend(common.HexToAddress("0x44447c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err == nil {
		t.Fatal("should be failed if caller address != stake holder")
	}
	// realLockEpoch < 0
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	err = doStakeIn(20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	setEpochTime(posconfig.FirstEpochId + 11)
	err = doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err == nil {
		t.Fatal("should be failed if realLockEpoch < 0")
	}
	setEpochTime(posconfig.FirstEpochId + 10)
	err = doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 10)
	if err != nil {
		t.Fatal(err.Error())
	}
	// realLockEpoch > PSMaxEpochNum
	// TODO: li hua check
	setEpochTime(posconfig.FirstEpochId - 90 + 10 - 1)
	err = doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err == nil {
		t.Fatal("should be failed if realLockEpoch > PSMaxEpochNum")
	}
	setEpochTime(posconfig.FirstEpochId - 90 + 10)
	err = doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 20)
	if err != nil {
		t.Fatal(err.Error())
	}
	///////////////////////////////////////
	// amount check
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	err = doStakeIn(20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 30000)
	if err == nil {
		t.Fatal("should not delegate")
	}
	err = doStakeAppend(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 30000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 30000)
	if err != nil {
		t.Fatal("should delegate")
	}
	err = doPartnerOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 9420001)
	if err == nil {
		t.Fatal("should delegate failed, stake > 10,500,000")
	}
	err = doDelegateOne(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 9420000)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestStakeUpdate(t *testing.T)  {
	if !reset() {
		t.Fatal("pos staking db init error")
	}
	// stake holder == nil
	err := doStakeUpdate(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000, 0)
	if err == nil {
		t.Fatal("should be failed if stake holder not exist")
	}
	// contract.CallerAddress != stakeInfo.From
	err = doStakeIn(20000)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = doStakeUpdate(common.HexToAddress("0x11117c0813a51d3bd1d08246af2a8a7a57d8922e"), 1000000, 0)
	if err == nil {
		t.Fatal("should be failed if contract.CallerAddress != stakeInfo.From")
	}
	// cannot change at the last 3 epoch
	setEpochTime(posconfig.FirstEpochId + 2 + 10 - UpdateDelay + 1)
	err = doStakeUpdate(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 0, 0)
	if err == nil {
		t.Fatal("should be failed if contract.CallerAddress != stakeInfo.From")
	}
	// normal
	setEpochTime(posconfig.FirstEpochId + 2 + 10 - UpdateDelay)
	err = doStakeUpdate(common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"), 0, 10)
	if err != nil {
		t.Fatal(err.Error())
	}
}

// go test -test.bench=“.×”
func TestMultiDelegateIn(b *testing.T) {
	if !reset() {
		b.Fatal("pos staking db init error")
	}
	err := doStakeIn(200000)
	if err != nil {
		b.Fatal(err.Error())
	}
	count := 100 // 10000

	begin := time.Now()
	begin1 := time.Now()
	for i:=0; i<count + 5; i++ {
		if i== count {
			begin1 = time.Now()
		}
		key,_ := crypto.GenerateKey()
		address := crypto.PubkeyToAddress(key.PublicKey)
		err = doDelegateOne(address, 100)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
	tAll := time.Since(begin)
	tLast5 := time.Since(begin1)
	println("10005 delegate need time:", tAll)
	println("10000~10004 delegate need time:", tLast5)
	clearDb()
}

func TestStakeInParam(t *testing.T) {
	var input = getStakeInParam()
	// good
	err := doStakeInParam(input)
	if err != nil {
		t.Fatal(err.Error())
	}

	input = getStakeInParam()
	input.SecPk = common.FromHex("0x04d7dffe5e06d2c7024d9bb93f675b8242e71901")
	err = doStakeInParam(input)
	if err == nil {
		t.Fatal("secpk should be error")
	}

	input = getStakeInParam()
	input.Bn256Pk = common.FromHex("0x04d7dffe5e06d2c7024d9bb93f675b8242e71901")
	err = doStakeInParam(input)
	if err == nil {
		t.Fatal("Bn256Pk should be error")
	}

	one := big.NewInt(1)
	zero := big.NewInt(0)
	input = getStakeInParam()
	input.LockEpochs = new(big.Int).Sub(minEpochNum, one)
	err = doStakeInParam(input)
	if err == nil {
		t.Fatal("lock epoch should not < minEpochNum")
	}

	input.LockEpochs = new(big.Int).Add(maxEpochNum, one)
	err = doStakeInParam(input)
	if err == nil {
		t.Fatal("lock epoch should not > maxEpochNum")
	}

	input = getStakeInParam()
	input.FeeRate = new(big.Int).Sub(zero, one)
	err = doStakeInParam(input)
	if err == nil {
		t.Fatal("FeeRate should >= 0")
	}

	input.FeeRate = new(big.Int).Add(big.NewInt(10000), one)
	err = doStakeInParam(input)
	if err == nil {
		t.Fatal("FeeRate should <=  10000")
	}
}

func TestStakeUpdateParam(t *testing.T) {
	var input StakeUpdateParam
	input.Addr = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	input.LockEpochs = big.NewInt(0)
	// LockEpochs = 0
	err := doStakeUpdateParam(input)
	if err != nil {
		t.Fatal(err.Error())
	}
	// LockEpochs == min
	input.LockEpochs = minEpochNum
	err = doStakeUpdateParam(input)
	if err != nil {
		t.Fatal(err.Error())
	}
	// LockEpochs == max
	input.LockEpochs = maxEpochNum
	err = doStakeUpdateParam(input)
	if err != nil {
		t.Fatal(err.Error())
	}
	one := big.NewInt(1)
	// LockEpochs < min
	input.LockEpochs = new(big.Int).Sub(minEpochNum, one)
	err = doStakeUpdateParam(input)
	if err == nil {
		t.Fatal("LockEpochs < min should failed")
	}
	// LockEpochs > max
	input.LockEpochs = new(big.Int).Add(maxEpochNum, one)
	err = doStakeUpdateParam(input)
	if err == nil {
		t.Fatal("LockEpochs > max should failed")
	}
}

func doStakeInWithParam(amount int64, feeRate int) error {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	if evmtime != int64(0) {
		stakerevm.Time = big.NewInt(evmtime)
	}
	contract.CallerAddress = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	a := new(big.Int).Mul(big.NewInt(amount), ether)
	contract.Value().Set(a)
	contract.self = &dummyContractRef{}
	eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())
	stakerevm.BlockNumber = big.NewInt(10)

	var input StakeInParam
	//input.SecPk = common.FromHex("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	input.SecPk = common.FromHex("0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70")
	input.Bn256Pk = common.FromHex("0x150b2b3230d6d6c8d1c133ec42d82f84add5e096c57665ff50ad071f6345cf45191fd8015cea72c4591ab3fd2ade12287c28a092ac0abf9ea19c13eb65fd4910")
	input.LockEpochs = big.NewInt(10)
	input.FeeRate = big.NewInt(int64(feeRate))

	bytes, err := cscAbi.Pack("stakeIn", input.SecPk, input.Bn256Pk, input.LockEpochs, input.FeeRate)
	if err != nil {
		return errors.New("stakeIn pack failed")
	}

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		return errors.New("stakeIn called failed " + err.Error())
	}

	// check
	pub := crypto.ToECDSAPub(input.SecPk)
	secAddr := crypto.PubkeyToAddress(*pub)
	key := GetStakeInKeyHash(secAddr)
	bytes2 := stakerevm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var info StakerInfo
	err = rlp.DecodeBytes(bytes2, &info)
	if err != nil {
		return errors.New("stakeIn rlp decode failed")
	}
	if info.LockEpochs != input.LockEpochs.Uint64() ||
		info.FeeRate != input.FeeRate.Uint64() ||
		!reflect.DeepEqual(info.PubBn256, input.Bn256Pk) ||
		!reflect.DeepEqual(info.PubSec256, input.SecPk) {
		return errors.New("stakeIn parse StakerInfo failed")
	}
	if info.Address != secAddr ||
		info.From != contract.CallerAddress ||
		info.Amount.Cmp(a) != 0  {
		return errors.New("stakeIn from amount epoch address saved wrong")
	}
	if posconfig.FirstEpochId == 0 {
		if info.StakingEpoch != 0 {
			return errors.New("StakingEpoch saved wrong, should eq 0")
		}
	} else {
		if info.StakingEpoch != eidNow + 2 {
			return errors.New("StakingEpoch saved wrong")
		}
	}
	return nil
}

func doStakeIn(amount int64) error {
	return doStakeInWithParam(amount, 100)
}

func doDelegateOne(from common.Address, amount int64) error {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	if evmtime != int64(0) {
		stakerevm.Time = big.NewInt(evmtime)
	}
	contract.CallerAddress = from
	a := new(big.Int).Mul(big.NewInt(amount), ether)
	contract.Value().Set(a)
	//eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())

	var input common.Address
	input = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")

	bytes, err := cscAbi.Pack("delegateIn", input)
	if err != nil {
		return errors.New("delegateIn pack failed")
	}

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		return errors.New("delegateIn called failed " + err.Error())
	}
	// check
	key := GetStakeInKeyHash(input)
	bytes2 := stakerevm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var infoS StakerInfo
	err = rlp.DecodeBytes(bytes2, &infoS)
	if err != nil {
		return errors.New("delegateIn rlp decode failed")
	}

	lenth := len(infoS.Clients)
	if lenth <= 0 {
		return errors.New("delegateIn save error")
	}
	info := infoS.Clients[lenth-1]
	if info.QuitEpoch != 0 ||
		info.Amount.Cmp(a) < 0 ||
		info.Address != contract.CallerAddress {
		return errors.New("delegateIn fields save error")
	}
	return nil
}

func doDelegateOut(from common.Address) error {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	if evmtime != int64(0) {
		stakerevm.Time = big.NewInt(evmtime)
	}
	contract.CallerAddress = from
	a := new(big.Int).Mul(big.NewInt(0), ether)
	contract.Value().Set(a)
	eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())

	var input common.Address
	input = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")

	bytes, err := cscAbi.Pack("delegateOut", input)
	if err != nil {
		return errors.New("delegateOut pack failed")
	}

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		return errors.New("delegateOut called failed " + err.Error())
	}
	// check
	key := GetStakeInKeyHash(input)
	bytes2 := stakerevm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var infoS StakerInfo
	err = rlp.DecodeBytes(bytes2, &infoS)
	if err != nil {
		return errors.New("delegateOut rlp decode failed")
	}

	l := len(infoS.Clients)
	if l <= 0 {
		return errors.New("delegateOut save error")
	}
	for i := 0; i < l; i++ {
		info := infoS.Clients[i]
		if info.Address == contract.CallerAddress {
			if info.QuitEpoch == eidNow + QuitDelay {
				return nil
			}
		}
	}
	return errors.New("delegateOut fields save error")
}

func doPartnerOne(from common.Address, amount int64) error {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	if evmtime != int64(0) {
		stakerevm.Time = big.NewInt(evmtime)
	}
	contract.CallerAddress = from
	a := new(big.Int).Mul(big.NewInt(amount), ether)
	contract.Value().Set(a)
	//eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())

	var input PartnerInParam
	input.Addr = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	input.Renewal = false

	bytes, err := cscAbi.Pack("partnerIn", input.Addr,input.Renewal)
	if err != nil {
		return errors.New("partnerIn pack failed")
	}

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		return errors.New("partnerIn called failed " + err.Error())
	}
	// check
	key := GetStakeInKeyHash(input.Addr)
	bytes2 := stakerevm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var infoS StakerInfo
	err = rlp.DecodeBytes(bytes2, &infoS)
	if err != nil {
		return errors.New("partnerIn rlp decode failed")
	}

	lenth := len(infoS.Partners)
	if lenth <= 0 {
		return errors.New("partnerIn save error")
	}
	info := infoS.Partners[lenth-1]
	if info.Amount.Cmp(a) < 0 ||
		info.Address != contract.CallerAddress {
		return errors.New("partnerIn fields save error")
	}
	return nil
}

func doStakeAppend(from common.Address, amount int64) error {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	if evmtime != int64(0) {
		stakerevm.Time = big.NewInt(evmtime)
	}
	contract.CallerAddress = from
	a := new(big.Int).Mul(big.NewInt(amount), ether)
	contract.Value().Set(a)
	//eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())

	var input common.Address
	input = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")

	bytes, err := cscAbi.Pack("stakeAppend", input)
	if err != nil {
		return errors.New("stakeAppend pack failed " + err.Error())
	}

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		return errors.New("stakeAppend called failed " + err.Error())
	}
	// check
	key := GetStakeInKeyHash(input)
	bytes2 := stakerevm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var infoS StakerInfo
	err = rlp.DecodeBytes(bytes2, &infoS)
	if err != nil {
		return errors.New("stakeAppend rlp decode failed")
	}

	if infoS.Amount.Cmp(a) < 0 ||
		infoS.Address != contract.CallerAddress {
		return errors.New("stakeAppend fields save error")
	}
	return nil
}

func doStakeUpdate(from common.Address, amount int64, deltaEpoch int64) error {
	stakerevm.Time = big.NewInt(time.Now().Unix())
	if evmtime != int64(0) {
		stakerevm.Time = big.NewInt(evmtime)
	}
	contract.CallerAddress = from
	a := new(big.Int).Mul(big.NewInt(amount), ether)
	contract.Value().Set(a)
	//eidNow, _ := util.CalEpochSlotID(stakerevm.Time.Uint64())

	var input StakeUpdateParam
	input.Addr = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	input.LockEpochs = big.NewInt(deltaEpoch)

	bytes, err := cscAbi.Pack("stakeUpdate", input.Addr, input.LockEpochs)
	if err != nil {
		return errors.New("stakeUpdate pack failed " + err.Error())
	}

	_, err = stakercontract.Run(bytes, contract, stakerevm)

	if err != nil {
		return errors.New("stakeUpdate called failed " + err.Error())
	}
	// check
	key := GetStakeInKeyHash(input.Addr)
	bytes2 := stakerevm.StateDB.GetStateByteArray(StakersInfoAddr, key)
	var infoS StakerInfo
	err = rlp.DecodeBytes(bytes2, &infoS)
	if err != nil {
		return errors.New("stakeUpdate rlp decode failed")
	}

	if infoS.Amount.Cmp(a) < 0 ||
		infoS.NextLockEpochs < uint64(deltaEpoch) ||
		infoS.Address != contract.CallerAddress {
		return errors.New("stakeUpdate fields save error")
	}
	return nil
}

func doStakeInParam(input StakeInParam) error {
	bytes, err := cscAbi.Pack("stakeIn", input.SecPk, input.Bn256Pk, input.LockEpochs, input.FeeRate)
	if err != nil {
		return err
	}
	_, err = stakercontract.stakeInParseAndValid(bytes[4:])
	if err != nil {
		return err
	}
	return nil
}

func doStakeUpdateParam(input StakeUpdateParam) error {
	bytes, err := cscAbi.Pack("stakeUpdate", input.Addr, input.LockEpochs)
	if err != nil {
		return err
	}
	_, err = stakercontract.stakeUpdateParseAndValid(bytes[4:])
	if err != nil {
		return err
	}
	return nil
}

func getStakeInParam() StakeInParam {
	var input StakeInParam
	//input.SecPk = common.FromHex("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	input.SecPk = common.FromHex("0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70")
	input.Bn256Pk = common.FromHex("0x150b2b3230d6d6c8d1c133ec42d82f84add5e096c57665ff50ad071f6345cf45191fd8015cea72c4591ab3fd2ade12287c28a092ac0abf9ea19c13eb65fd4910")
	input.LockEpochs = big.NewInt(10)
	input.FeeRate = big.NewInt(int64(100))

	return input
}

var evmtime int64
func setEpochTime(epochId uint64) {
	epochTimespan := uint64(posconfig.SlotTime * posconfig.SlotCount)
	evmtime = int64(epochId * epochTimespan)
}