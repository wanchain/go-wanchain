package vm

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/uleaderselection"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestStage1RlpCompress(t *testing.T) {
	epochID := uint64(88665544)
	selfIndex := uint64(789012)
	key, _ := crypto.GenerateKey()
	mi := &key.PublicKey

	buf, err := RlpPackStage1DataForTx(epochID, selfIndex, mi, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	fmt.Println("len(buf):", len(buf))

	epochIDUnpack, selfIndexUnpack, miUnpack, err := RlpUnpackStage1DataForTx(buf)
	if err != nil {
		t.Fail()
	}

	if convert.Uint64ToString(epochID) != convert.Uint64ToString(epochIDUnpack) ||
		convert.Uint64ToString(selfIndex) != convert.Uint64ToString(selfIndexUnpack) ||
		hex.EncodeToString(crypto.FromECDSAPub(mi)) != hex.EncodeToString(crypto.FromECDSAPub(miUnpack)) {
		t.Fail()
	}

	epochIDBuf, selfIndexBuf, err := RlpGetStage1IDFromTx(buf)
	if err != nil {
		t.Fail()
	}

	if convert.Uint64ToString(epochID) != convert.Uint64ToString(convert.BytesToUint64(epochIDBuf)) {
		t.Fail()
	}

	if convert.Uint64ToString(selfIndex) != convert.Uint64ToString(convert.BytesToUint64(selfIndexBuf)) {
		t.Fail()
	}
}
func TestStage2RlpCompress(t *testing.T) {
	epochID := uint64(88665544)
	selfIndex := uint64(789012)
	key, _ := crypto.GenerateKey()
	selfPK := &key.PublicKey

	alphaPki := make([]*ecdsa.PublicKey, 50)
	proof := make([]*big.Int, 2)

	for i := 0; i < len(alphaPki); i++ {
		k, _ := crypto.GenerateKey()
		alphaPki[i] = &k.PublicKey
	}

	for i := 0; i < len(proof); i++ {
		k, _ := crypto.GenerateKey()
		proof[i] = k.D
	}

	buf, err := RlpPackStage2DataForTx(epochID, selfIndex, selfPK, alphaPki, proof, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	fmt.Println("len(buf):", len(buf))

	t1 := time.Now()
	epochIDUnpack, selfIndexUnpack, selfPKUnpack, alphaPkiUnpack, proofUnpack, err := RlpUnpackStage2DataForTx(buf)
	if err != nil {
		t.Fail()
	}

	for index := 0; index < 49; index++ {
		epochIDUnpack, selfIndexUnpack, selfPKUnpack, alphaPkiUnpack, proofUnpack, err = RlpUnpackStage2DataForTx(buf)
		if err != nil {
			t.Fail()
		}
	}

	fmt.Println("time:", time.Since(t1))

	if convert.Uint64ToString(epochID) != convert.Uint64ToString(epochIDUnpack) ||
		convert.Uint64ToString(selfIndex) != convert.Uint64ToString(selfIndexUnpack) {
		t.Fail()
	}

	if !util.PkEqual(selfPK, selfPKUnpack) {
		t.Fail()
	}

	for i := 0; i < len(alphaPki); i++ {
		if !util.PkEqual(alphaPki[i], alphaPkiUnpack[i]) {
			t.Fail()
		}
	}

	for i := 0; i < len(proof); i++ {
		if proof[i].String() != proofUnpack[i].String() {
			t.Fail()
		}
	}

	epochIDBuf, selfIndexBuf, err := RlpGetStage2IDFromTx(buf)
	if err != nil {
		t.Fail()
	}

	if convert.Uint64ToString(epochID) != convert.Uint64ToString(convert.BytesToUint64(epochIDBuf)) {
		t.Fail()
	}

	if convert.Uint64ToString(selfIndex) != convert.Uint64ToString(convert.BytesToUint64(selfIndexBuf)) {
		t.Fail()
	}
}

func TestGetSlotLeaderStage2KeyHash(t *testing.T) {
	for i := 0; i < 4; i++ {
		epochIDBuf := convert.Uint64ToBytes(uint64(i))
		for j := 0; j < 2; j++ {
			slfIndexBuf := convert.Uint64ToBytes(uint64(j))
			key := GetSlotLeaderStage2KeyHash(epochIDBuf, slfIndexBuf)
			fmt.Printf("TestGetSlotLeaderStage2KeyHash epochID:%v, index:%v, key: %v\n",
				i, j, hex.EncodeToString(key[:]))
		}
	}
}

func TestGetSlotLeaderStageKeyHash(t *testing.T) {
	epochID := convert.Uint64ToBytes(uint64(0))
	selfIndex := convert.Uint64ToBytes(uint64(1))
	var hash common.Hash
	hash = getSlotLeaderStageKeyHash(epochID, selfIndex, SlotLeaderStag1)
	t.Logf("hash:0x%v", hex.EncodeToString(hash[:]))

	hash = getSlotLeaderStageKeyHash(epochID, selfIndex, SlotLeaderStag2)
	t.Logf("hash:0x%v", hex.EncodeToString(hash[:]))

}

func TestGetSlotLeaderStageIndexesKeyHash(t *testing.T) {

	for i := 0; i < 4; i++ {
		epochIDBuf := convert.Uint64ToBytes(uint64(i))
		key := getSlotLeaderStageIndexesKeyHash(epochIDBuf, SlotLeaderStag1Indexes)
		fmt.Printf("slot leader stage indexes stage1,epoch %v\n", epochIDBuf)
		fmt.Println(hex.EncodeToString(key.Bytes()))

		fmt.Printf("slot leader stage indexes stage2,epoch %v\n", epochIDBuf)
		key = getSlotLeaderStageIndexesKeyHash(epochIDBuf, SlotLeaderStag2Indexes)
		fmt.Println(hex.EncodeToString(key.Bytes()))
	}
}

func TestIsInValidStage(t *testing.T) {
	evm := NewEVM(Context{}, nil, &params.ChainConfig{ChainId: big1}, Config{Debug: true})

	util.CalEpochSlotIDByNow()
	curEpId, _ := util.GetEpochSlotID()


	testTime := curEpId*posconfig.SlotCount*posconfig.SlotTime + 2*posconfig.K*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)

	kStart := uint64(2 * posconfig.K)
	kEnd := uint64(3 * posconfig.K)

	if isInValidStage(0, evm.Time.Uint64(), kStart, kEnd) {
		t.Error("should out of range")
		t.Fail()
	}

	if !isInValidStage(curEpId, evm.Time.Uint64(), kStart, kEnd) {
		t.Error("should in range")
		t.Fail()
	}

}

func TestAddSlotScCallTimes(t *testing.T) {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	t.Log("Current dir path ", dir)
	os.RemoveAll(path.Join(dir, "sl_contract_test"))
	posdb.NewDb(path.Join(dir, "sl_contract_test"))

	epochID := uint64(0)
	loopCount := 10
	for i := 0; i < loopCount; i++ {
		addSlotScCallTimes(epochID)
	}

	if intByte, _ := posdb.GetDb().Get(epochID, scCallTimes); convert.BytesToUint64(intByte[:]) != uint64(loopCount) {
		t.Fail()
	}

	os.RemoveAll(path.Join(dir, "sl_contract_test"))
}

func TestUpdateSlotLeaderStageIndex(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		stateDb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	)

	var sendtransGet [posconfig.EpochLeaderCount]bool

	evm := NewEVM(Context{}, stateDb, &params.ChainConfig{ChainId: big1}, Config{Debug: true})
	epochIDBuf := convert.Uint64ToBytes(uint64(0))

	var index uint64
	// stage1 index
	index = 0
	updateSlotLeaderStageIndex(evm, epochIDBuf, SlotLeaderStag1Indexes, index)

	key := getSlotLeaderStageIndexesKeyHash(epochIDBuf, SlotLeaderStag1Indexes)
	bytes := evm.StateDB.GetStateByteArray(slotLeaderPrecompileAddr, key)
	rlp.DecodeBytes(bytes, &sendtransGet)

	t.Logf("Index %v, status is :%v", index, sendtransGet[index])
	if !sendtransGet[index] {
		t.Error("state of index should be true")
		t.Fail()
	}

	t.Logf("Index %v, status is :%v", index+1, sendtransGet[index+1])
	if sendtransGet[index+1] {
		t.Error("state of index should be false")
		t.Fail()
	}

	// clear
	for i := 0; i < len(sendtransGet); i++ {
		sendtransGet[i] = false
	}

	// stage2 index
	index = 0
	updateSlotLeaderStageIndex(evm, epochIDBuf, SlotLeaderStag2Indexes, index)

	key = getSlotLeaderStageIndexesKeyHash(epochIDBuf, SlotLeaderStag2Indexes)
	bytes = evm.StateDB.GetStateByteArray(slotLeaderPrecompileAddr, key)
	rlp.DecodeBytes(bytes, &sendtransGet)

	t.Logf("Index %v, status is :%v", index, sendtransGet[index])
	if !sendtransGet[index] {
		t.Error("state of index should be true")
		t.Fail()
	}

	t.Logf("Index %v, status is :%v", index+1, sendtransGet[index+1])
	if sendtransGet[index+1] {
		t.Error("state of index should be false")
		t.Fail()
	}

}

func TestGetSlotLeaderScAbiString(t *testing.T) {
	if len(GetSlotLeaderScAbiString()) == 0 {
		t.Fail()
	}
}

func TestGetStage1FunctionID(t *testing.T) {
	slotStage1ID, err := GetStage1FunctionID(GetSlotLeaderScAbiString())
	if err != nil {
		t.Fail()
	}

	fmt.Println(slotStage1ID)
	if len(slotStage1ID) != 4 {
		t.Error("length of contract function error")
	}
}

func TestGetStage2FunctionID(t *testing.T) {
	slotStage2ID, err := GetStage2FunctionID(GetSlotLeaderScAbiString())
	if err != nil {
		t.Fail()
	}

	fmt.Println(slotStage2ID)
	if len(slotStage2ID) != 4 {
		t.Error("length of contract function error")
	}
}

func TestGetSlotLeaderSCAddress(t *testing.T) {
	addr := GetSlotLeaderSCAddress()
	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())

	t.Logf("slot leader contract precompile address is :%v ", hex.EncodeToString(addr[:]))
	if len(addr) != len(slotLeaderPrecompileAddr) {
		t.Fail()
	}

	if addr != slotLeaderPrecompileAddr {
		t.Fail()
	}
}

func TestRlpPackStage1DataForTx(t *testing.T) {

	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	pubKey := prvKey.PublicKey

	// pack
	rlpPackBytes, err := RlpPackStage1DataForTx(0, 0, &pubKey, GetSlotLeaderScAbiString())
	// unpack
	epochID, selfIndex, mi, err := RlpUnpackStage1DataForTx(rlpPackBytes)
	// check
	if epochID != 0 || selfIndex != 0 || uleaderselection.PublicKeyEqual(&pubKey, mi) {
		t.Fail()
	}
}

func TestRlpPackStage2DataForTx(t *testing.T) {

	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	pubKey := prvKey.PublicKey

	// pack
	var alphaPkis [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	for i := 0; i < posconfig.EpochLeaderCount; i++ {

		key, _ := crypto.GenerateKey()
		alphaPkis[i] = &key.PublicKey
	}

	var proof [2]*big.Int
	proof[0] = big1
	proof[1] = big4

	stag2Bytes, err := RlpPackStage2DataForTx(0, 0, &pubKey, alphaPkis[:], proof[:], GetSlotLeaderScAbiString())
	if err != nil {
		t.Fail()
	}

	epochID, selfIndex, selfPK, alphaPkisDecoded, proofDecoded, err := RlpUnpackStage2DataForTx(stag2Bytes)
	if err != nil {
		t.Fail()
	}

	if epochID != 0 || selfIndex != 0 || uleaderselection.PublicKeyEqual(selfPK, &pubKey) {
		t.Fail()
	}

	for i, pk := range alphaPkisDecoded {
		if uleaderselection.PublicKeyEqual(alphaPkis[i], pk) {
			t.Fail()
		}
	}

	for j, value := range proofDecoded {
		if proof[j].Cmp(value) != 0 {
			t.Fail()
		}
	}
}

func TestHandleStgOne(t *testing.T) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	pubKey := prvKey.PublicKey

	// pack
	rlpPackBytes, err := RlpPackStage1DataForTx(0, 0, &pubKey, GetSlotLeaderScAbiString())

	var (
		db, _      = ethdb.NewMemDatabase()
		stateDb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	)

	// put data into state db
	evm := NewEVM(Context{}, stateDb, &params.ChainConfig{ChainId: big1}, Config{Debug: true})

	nowTime := uint64(time.Now().Unix())
	//baseTime := posconfig.EpochBaseTime
	//posconfig.EpochBaseTime = nowTime

	testTime := nowTime + (posconfig.Sma1Start+1)*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)

	handleStgOne(rlpPackBytes, nil, evm)

	keyHash := GetSlotLeaderStage1KeyHash(convert.Uint64ToBytes(uint64(0)),
		convert.Uint64ToBytes(uint64(0)))

	// get data from state db
	bytesGet := evm.StateDB.GetStateByteArray(GetSlotLeaderSCAddress(), keyHash)

	// unpack
	epochID, selfIndex, mi, err := RlpUnpackStage1DataForTx(bytesGet)
	// check
	if epochID != 0 || selfIndex != 0 || uleaderselection.PublicKeyEqual(&pubKey, mi) {
		t.Fail()
	}

	//posconfig.EpochBaseTime = baseTime
}

func TestGetStg1StateDbInfo(t *testing.T) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	pubKey := prvKey.PublicKey

	// pack
	rlpPackBytes, err := RlpPackStage1DataForTx(0, 0, &pubKey, GetSlotLeaderScAbiString())

	var (
		db, _      = ethdb.NewMemDatabase()
		stateDb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	)

	// put data into state db
	evm := NewEVM(Context{}, stateDb, &params.ChainConfig{ChainId: big1}, Config{Debug: true})

	nowTime := uint64(time.Now().Unix())
	//baseTime := posconfig.EpochBaseTime
	//posconfig.EpochBaseTime = nowTime

	testTime := nowTime + (posconfig.Sma1Start+1)*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)

	handleStgOne(rlpPackBytes, nil, evm)

	// get data from state db
	bytesMi, err := GetStg1StateDbInfo(stateDb, 0, 0)
	if err != nil {
		t.Fail()
	}
	// len of PK is 65
	if len(bytesMi) != 65 {
		t.Fail()
	}

	//posconfig.EpochBaseTime = baseTime
}

func TestGetStg2TxAlphaPki(t *testing.T) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	pubKey := prvKey.PublicKey

	tempInt := new(big.Int).SetInt64(0)
	tempInt.SetBytes(crypto.Keccak256(crypto.FromECDSAPub(&pubKey)))
	alpha := tempInt

	mi0 := new(ecdsa.PublicKey)
	mi0.Curve = crypto.S256()
	mi0.X, mi0.Y = crypto.S256().ScalarMult(pubKey.X, pubKey.Y, alpha.Bytes())

	// pack
	rlpPackBytes, err := RlpPackStage1DataForTx(0, 0, mi0, GetSlotLeaderScAbiString())

	var (
		db, _      = ethdb.NewMemDatabase()
		stateDb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	)

	// put data into state db
	evm := NewEVM(Context{}, stateDb, &params.ChainConfig{ChainId: big1}, Config{Debug: true})

	nowTime := uint64(time.Now().Unix())
	//baseTime := posconfig.EpochBaseTime
	//posconfig.EpochBaseTime = nowTime

	testTime := nowTime + (posconfig.Sma1Start+1)*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)

	handleStgOne(rlpPackBytes, nil, evm)

	// get data from state db
	bytesMi, err := GetStg1StateDbInfo(stateDb, 0, 0)
	if err != nil {
		t.Fail()
	}
	// len of PK is 65
	if len(bytesMi) != 65 {
		t.Fail()
	}

	// build stage2 data
	var alphaPkis [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	alphaPkis[0] = mi0
	for i := 1; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKey := prvKey.PublicKey

		mi := new(ecdsa.PublicKey)
		mi.Curve = crypto.S256()
		mi.X, mi.Y = crypto.S256().ScalarMult(pubKey.X, pubKey.Y, alpha.Bytes())

		alphaPkis[i] = mi
	}
	var proof [2]*big.Int
	proof[0] = big1
	proof[1] = big4
	stg2Bytes, _ := RlpPackStage2DataForTx(0, 0, &pubKey, alphaPkis[:], proof[:], GetSlotLeaderScAbiString())

	handleStgTwo(stg2Bytes[:], nil, evm)

	//posconfig.EpochBaseTime = baseTime
}

func TestHandleStgTwo(t *testing.T) {
	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	pubKey := prvKey.PublicKey

	tempInt := new(big.Int).SetInt64(0)
	tempInt.SetBytes(crypto.Keccak256(crypto.FromECDSAPub(&pubKey)))
	alpha := tempInt

	mi0 := new(ecdsa.PublicKey)
	mi0.Curve = crypto.S256()
	mi0.X, mi0.Y = crypto.S256().ScalarMult(pubKey.X, pubKey.Y, alpha.Bytes())

	// pack
	rlpPackBytes, err := RlpPackStage1DataForTx(0, 0, mi0, GetSlotLeaderScAbiString())

	var (
		db, _      = ethdb.NewMemDatabase()
		stateDb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	)

	// put data into state db
	evm := NewEVM(Context{}, stateDb, &params.ChainConfig{ChainId: big1}, Config{Debug: true})

	nowTime := uint64(time.Now().Unix())
	//baseTime := posconfig.EpochBaseTime
	//posconfig.EpochBaseTime = nowTime

	testTime := nowTime + (posconfig.Sma1Start+1)*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)

	handleStgOne(rlpPackBytes, nil, evm)

	// get data from state db
	bytesMi, err := GetStg1StateDbInfo(stateDb, 0, 0)
	if err != nil {
		t.Fail()
	}
	// len of PK is 65
	if len(bytesMi) != 65 {
		t.Fail()
	}

	// build stage2 data
	var alphaPkis [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	alphaPkis[0] = mi0
	for i := 1; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKey := prvKey.PublicKey

		mi := new(ecdsa.PublicKey)
		mi.Curve = crypto.S256()
		mi.X, mi.Y = crypto.S256().ScalarMult(pubKey.X, pubKey.Y, alpha.Bytes())

		alphaPkis[i] = mi
	}
	var proof [2]*big.Int
	proof[0] = big1
	proof[1] = big4
	stg2Bytes, _ := RlpPackStage2DataForTx(0, 0, &pubKey, alphaPkis[:], proof[:], GetSlotLeaderScAbiString())

	testTime = nowTime + (posconfig.Sma2Start+1)*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)

	ret, err := handleStgTwo(stg2Bytes[:], nil, evm)

	if err != nil || ret != nil {
		t.Fail()
	}
	//posconfig.EpochBaseTime = baseTime
}
