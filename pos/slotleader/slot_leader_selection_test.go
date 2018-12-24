package slotleader

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"

	"github.com/btcsuite/btcd/btcec"

	"github.com/wanchain/go-wanchain/crypto"
)

// TestLoop use to test main loop
func TestLoop(t *testing.T) {
	posdb.GetDb().DbInit("test")
	GetSlotLeaderSelection().Loop(nil, nil)
	GetSlotLeaderSelection().Loop(nil, nil)

	GetSlotLeaderSelection().setCurrentWorkStage(slotLeaderSelectionStage1)

	GetSlotLeaderSelection().Loop(nil, nil)
	GetSlotLeaderSelection().Loop(nil, nil)

	GetSlotLeaderSelection().setWorkingEpochID(1)

	GetSlotLeaderSelection().Loop(nil, nil)

}

func TestGetEpochSlotID(t *testing.T) {
	epochID, slotID, err := GetEpochSlotID()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("epochID:", epochID, " slotID:", slotID)
	}
}

func TestSlotLeaderSelectionGetInstance(t *testing.T) {
	posdb.GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()
	if slot == nil {
		t.Fail()
	}
}

func TestGenerateCommitmentSuccess(t *testing.T) {
	posdb.GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	fmt.Println("priv len:", len(crypto.FromECDSA(privKey)))
	fmt.Println("pk len:", len(crypto.FromECDSAPub(&privKey.PublicKey)))
	fmt.Println("pk: ", hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey)))

	pkCompress := btcec.PublicKey(privKey.PublicKey)
	fmt.Println("compressed pk: :", hex.EncodeToString(pkCompress.SerializeCompressed()), "len: ", len(pkCompress.SerializeCompressed()))

	epochID := uint64(8192)
	payload, err := slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
	if err != nil {
		t.Fail()
	}

	if payload == nil {
		t.Fail()
	}

	fmt.Println("payload len:", len(payload), " data: ", hex.EncodeToString(payload))

	alpha, err := slot.GetAlpha(epochID, 0)
	if alpha == nil || err != nil {
		t.Fail()
	}

	var output [][]byte
	rlp.DecodeBytes(payload, &output)

	if hex.EncodeToString(pkCompress.SerializeCompressed()) != hex.EncodeToString(output[2]) {
		t.Fail()
	}

	fmt.Println("epochID: ", hex.EncodeToString(output[0]))
	fmt.Println("selfIndex: ", hex.EncodeToString(output[1]))

	fmt.Println("payload 0: ", hex.EncodeToString(output[2]))
	fmt.Println("payload 1: ", hex.EncodeToString(output[3]))
	fmt.Println("Alpha: ", alpha)
}

func TestGenerateCommitmentFailed(t *testing.T) {
	posdb.GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	epochID := uint64(1)

	_, err = slot.GenerateCommitment(nil, epochID, 0)
	if err == nil {
		t.Fail()
	}

	// _, err = slot.GenerateCommitment(&privKey.PublicKey, 0)
	// if err == nil {
	// 	t.Fail()
	// }

	privKey.PublicKey.X = nil
	privKey.PublicKey.Y = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey.PublicKey.Curve = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey2, _ := crypto.GenerateKey()

	privKey.X = privKey2.X
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}
}

func TestPublicKeyCompress(t *testing.T) {
	privKey, _ := crypto.GenerateKey()

	fmt.Println("Is on curve: ", crypto.S256().IsOnCurve(privKey.X, privKey.Y))

	fmt.Println("public key:", hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey)))

	pk := btcec.PublicKey(privKey.PublicKey)

	fmt.Println("public key uncompress:", hex.EncodeToString(pk.SerializeUncompressed()), "len: ", len(pk.SerializeUncompressed()))

	fmt.Println("public key compress:", hex.EncodeToString(pk.SerializeCompressed()), "len: ", len(pk.SerializeCompressed()))

	keyCompress := pk.SerializeCompressed()

	key, _ := btcec.ParsePubKey(keyCompress, btcec.S256())

	pKey := ecdsa.PublicKey(*key)

	fmt.Println("public key:", hex.EncodeToString(crypto.FromECDSAPub(&pKey)))
}

func TestRlpEncodeAndDecode(t *testing.T) {

	privKey, _ := crypto.GenerateKey()
	pk := btcec.PublicKey(privKey.PublicKey)
	keyCompress := pk.SerializeCompressed()

	var test = [][]byte{
		new(big.Int).SetInt64(1).Bytes(),
		keyCompress,
		keyCompress,
	}

	fmt.Println("before encode:", test)

	buf, err := rlp.EncodeToBytes(test)

	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("encode buf: ", hex.EncodeToString(buf))

	fmt.Println("encode len: ", len(buf))

	var output [][]byte
	rlp.DecodeBytes(buf, &output)

	fmt.Println("after decode:", output)
}

func TestAbiPack(t *testing.T) {

	s := GetSlotLeaderSelection()

	data := []byte("Each winter, npm, Inc. circulates a survey of software developers and npm users to solicit your ")

	payload, err := s.PackStage1Data(data)
	if err != nil {
		t.Fail()
	}
	id1, _ := s.GetStage1FunctionID()
	id2, _ := s.GetFuncIDFromPayload(payload)
	if id1 != id2 {
		t.Fail()
	}

	unpack, err := s.UnpackStage1Data(payload)
	if err != nil {
		t.Fail()
	}

	if hex.EncodeToString(unpack) != hex.EncodeToString(data) {
		t.Fail()
	}
}

// TestByteToString is test for bytes compare with string() convert
func TestByteToString(t *testing.T) {
	testBytes := make([]byte, 0)
	for i := 0; i < 255; i++ {
		testBytes = append(testBytes, byte(i))
	}
	fmt.Println("bytes: ", testBytes)
	fmt.Println("string: ", string(testBytes))
	fmt.Println("string len:", len(string(testBytes)))

	testBytes2 := make([]byte, 0)
	for i := 0; i < 255; i++ {
		testBytes2 = append(testBytes2, byte(i))
	}

	if string(testBytes) != string(testBytes2) {
		t.Fail()
	}
}
