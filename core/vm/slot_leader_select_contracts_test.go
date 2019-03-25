package vm

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"
)

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

//func TestWanSlotLeaderCommitment(t *testing.T) {
//	contract := &slotLeaderSC{}
//
//	contract.RequiredGas(nil)
//
//	contract.Run(nil, nil, nil)
//
//	privKey, err := crypto.GenerateKey()
//	if err != nil {
//		t.Fail()
//	}
//
//	slot := slotleader.GetSlotLeaderSelection()
//	epochID := uint64(4096)
//
//	payload, err := slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
//	if err != nil {
//		t.Fail()
//	}
//
//	data, err := slot.PackStage1Data(payload)
//	if err != nil {
//		t.Fail()
//	}
//
//	contract.Run(data, nil, nil)
//}

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
