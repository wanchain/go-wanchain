package vm

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"
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

	nowTime := uint64(time.Now().Unix())

	baseTime := posconfig.EpochBaseTime
	posconfig.EpochBaseTime = nowTime

	testTime := nowTime + 1*posconfig.SlotCount*posconfig.SlotTime + 2*posconfig.K*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)

	kStart := uint64(2 * posconfig.K)
	kEnd := uint64(3 * posconfig.K)

	if isInValidStage(0, evm, kStart, kEnd) {
		t.Error("should out of range")
		t.Fail()
	}

	if !isInValidStage(1, evm, kStart, kEnd) {
		t.Error("should in range")
		t.Fail()
	}

	testTime = nowTime + 1*posconfig.SlotCount*posconfig.SlotTime + 2*posconfig.K*posconfig.SlotTime - 1
	evm.Time = big.NewInt(0).SetUint64(testTime)
	if isInValidStage(1, evm, kStart, kEnd) {
		t.Error("should in range")
		t.Fail()
	}

	testTime = nowTime + 1*posconfig.SlotCount*posconfig.SlotTime + 3*posconfig.K*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)
	if !isInValidStage(1, evm, kStart, kEnd) {
		t.Error("should in range")
		t.Fail()
	}

	testTime = nowTime + 1*posconfig.SlotCount*posconfig.SlotTime + (3*posconfig.K+1)*posconfig.SlotTime
	evm.Time = big.NewInt(0).SetUint64(testTime)
	if isInValidStage(1, evm, kStart, kEnd) {
		t.Error("should out of range")
		t.Fail()
	}

	posconfig.EpochBaseTime = baseTime
	evm = nil
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
