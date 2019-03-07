package slottools

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

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
	epochIDUnpack, selfIndexUnpack, selfPKUnpack, alphaPkiUnpack, proofUnpack, err := RlpUnpackStage2DataForTx(buf, slotLeaderSCDef)
	if err != nil {
		t.Fail()
	}

	for index := 0; index < 49; index++ {
		epochIDUnpack, selfIndexUnpack, selfPKUnpack, alphaPkiUnpack, proofUnpack, err = RlpUnpackStage2DataForTx(buf, slotLeaderSCDef)
		if err != nil {
			t.Fail()
		}
	}

	fmt.Println("time:", time.Since(t1))

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

func Wadd(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	if x1.Cmp(x2) == 0 && y1.Cmp(y2) == 0 {
		return crypto.S256().Double(x1, y1)
	} else {
		return crypto.S256().Add(x1, y1, x2, y2)
	}
}

func VerifyDleqProof(PublicKeys []*ecdsa.PublicKey, AlphaPublicKeys []*ecdsa.PublicKey, Proof []*big.Int) bool {
	t1 := time.Now()

	if len(PublicKeys) == 0 || len(AlphaPublicKeys) == 0 || len(PublicKeys) != len(AlphaPublicKeys) || len(Proof) != 2 {
		return false
	}
	n := len(PublicKeys)
	var ebuffer bytes.Buffer
	for i := 0; i < n; i++ {
		ebuffer.Write(crypto.FromECDSAPub(PublicKeys[i]))
		ebuffer.Write(crypto.FromECDSAPub(AlphaPublicKeys[i]))
	}

	wLpublickey := new(ecdsa.PublicKey)
	wLpublickey.Curve = crypto.S256()
	wRpublickey := new(ecdsa.PublicKey)
	wRpublickey.Curve = crypto.S256()

	fmt.Println("VerifyDleqProof time 001:", time.Since(t1))

	for i := 0; i < n; i++ {

		t3 := time.Now()
		wLpublickey.X, wLpublickey.Y = crypto.S256().ScalarMult(PublicKeys[i].X, PublicKeys[i].Y, Proof[1].Bytes())
		fmt.Println("1:", time.Since(t3))

		wRpublickey.X, wRpublickey.Y = crypto.S256().ScalarMult(AlphaPublicKeys[i].X, AlphaPublicKeys[i].Y, Proof[0].Bytes())
		fmt.Println("2:", time.Since(t3))

		wLpublickey.X, wLpublickey.Y = Wadd(wLpublickey.X, wLpublickey.Y, wRpublickey.X, wRpublickey.Y)
		fmt.Println("3:", time.Since(t3))
		ebuffer.Write(crypto.FromECDSAPub(wLpublickey))
	}
	fmt.Println("VerifyDleqProof time 002:", time.Since(t1))

	ebyte := crypto.Keccak256(ebuffer.Bytes())
	e := new(big.Int).SetInt64(0)
	e.SetBytes(ebyte)
	fmt.Println("VerifyDleqProof time 003:", time.Since(t1))
	return e.Cmp(Proof[0]) == 0
}

func TestVerifyDleqProof(t *testing.T) {
	t0 := time.Now()

	pks := make([]*ecdsa.PublicKey, 50)
	alphaPks := make([]*ecdsa.PublicKey, 50)
	proof := make([]*big.Int, 2)

	for i := 0; i < len(pks); i++ {
		key, _ := crypto.GenerateKey()
		pks[i] = &key.PublicKey
		key, _ = crypto.GenerateKey()
		alphaPks[i] = &key.PublicKey

		if i < 2 {
			proof[i] = key.D
		}
	}

	t1 := time.Now()
	VerifyDleqProof(pks, alphaPks, proof)
	fmt.Println("VerifyDleqProof time:", time.Since(t1))

	fmt.Println("TestVerifyDleqProof total:", time.Since(t0))
}
