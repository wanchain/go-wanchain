package slottools

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/crypto"
)

var (
	slotLeaderSCDef = `[
			{
				"constant": false,
				"type": "function",
				"inputs": [
					{
						"name": "data",
						"type": "string"
					}
				],
				"name": "slotLeaderStage1MiSave",
				"outputs": [
					{
						"name": "data",
						"type": "string"
					}
				]
			},
			{
				"constant": false,
				"type": "function",
				"inputs": [
					{
						"name": "data",
						"type": "string"
					}
				],
				"name": "slotLeaderStage2InfoSave",
				"outputs": [
					{
						"name": "data",
						"type": "string"
					}
				]
			}
		]`
)

func TestStage1DataPack(t *testing.T) {

	data := []byte("Each winter, npm, Inc. circulates a survey of software developers and npm users to solicit your ")

	payload, err := PackStage1Data(data, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}
	id1, _ := GetStage1FunctionID(slotLeaderSCDef)
	id2, _ := GetFuncIDFromPayload(payload)
	if id1 != id2 {
		t.Fail()
	}

	unpack, err := UnpackStage1Data(payload, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	if hex.EncodeToString(unpack) != hex.EncodeToString(data) {
		t.Fail()
	}
}

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

	epochIDUnpack, selfIndexUnpack, miUnpack, err := RlpUnpackStage1DataForTx(buf, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	if postools.Uint64ToString(epochID) != postools.Uint64ToString(epochIDUnpack) ||
		postools.Uint64ToString(selfIndex) != postools.Uint64ToString(selfIndexUnpack) ||
		hex.EncodeToString(crypto.FromECDSAPub(mi)) != hex.EncodeToString(crypto.FromECDSAPub(miUnpack)) {
		t.Fail()
	}

	epochIDBuf, selfIndexBuf, err := RlpGetStage1IDFromTx(buf, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	if postools.Uint64ToString(epochID) != postools.Uint64ToString(postools.BytesToUint64(epochIDBuf)) {
		t.Fail()
	}

	if postools.Uint64ToString(selfIndex) != postools.Uint64ToString(postools.BytesToUint64(selfIndexBuf)) {
		t.Fail()
	}
}

func TestStage2RlpCompress(t *testing.T) {
	epochID := uint64(88665544)
	selfIndex := uint64(789012)
	key, _ := crypto.GenerateKey()
	selfPK := &key.PublicKey

	alphaPki := make([]*ecdsa.PublicKey, 1000)
	proof := make([]*big.Int, 10)

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

	epochIDUnpack, selfIndexUnpack, selfPKUnpack, alphaPkiUnpack, proofUnpack, err := RlpUnpackStage2DataForTx(buf, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	if postools.Uint64ToString(epochID) != postools.Uint64ToString(epochIDUnpack) ||
		postools.Uint64ToString(selfIndex) != postools.Uint64ToString(selfIndexUnpack) {
		t.Fail()
	}

	if !postools.PkEqual(selfPK, selfPKUnpack) {
		t.Fail()
	}

	for i := 0; i < len(alphaPki); i++ {
		if !postools.PkEqual(alphaPki[i], alphaPkiUnpack[i]) {
			t.Fail()
		}
	}

	for i := 0; i < len(proof); i++ {
		if proof[i].String() != proofUnpack[i].String() {
			t.Fail()
		}
	}

	epochIDBuf, selfIndexBuf, err := RlpGetStage2IDFromTx(buf, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	if postools.Uint64ToString(epochID) != postools.Uint64ToString(postools.BytesToUint64(epochIDBuf)) {
		t.Fail()
	}

	if postools.Uint64ToString(selfIndex) != postools.Uint64ToString(postools.BytesToUint64(selfIndexBuf)) {
		t.Fail()
	}
}

func TestPkCompress(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pk := &key.PublicKey

	buf, err := PkCompress(pk)
	if err != nil {
		t.Fail()
	}

	fmt.Println("len(pk):", len(buf))

	pkUncompress, err := PkUncompress(buf)
	if err != nil {
		t.Fail()
	}

	if hex.EncodeToString(crypto.FromECDSAPub(pk)) != hex.EncodeToString(crypto.FromECDSAPub(pkUncompress)) {
		t.Fail()
	}
}
