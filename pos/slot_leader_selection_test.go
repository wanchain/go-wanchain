package pos

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/rlp"

	"github.com/btcsuite/btcd/btcec"

	"github.com/wanchain/go-wanchain/crypto"
)

func TestSlotLeaderSelectionGetInstance(t *testing.T) {
	GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()
	if slot == nil {
		t.Fail()
	}
}

func TestGenerateCommitmentSuccess(t *testing.T) {
	GetDb().DbInit("test")
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

	epochID := new(big.Int).SetInt64(8192)
	payload, err := slot.GenerateCommitment(&privKey.PublicKey, epochID)
	if err != nil {
		t.Fail()
	}

	if payload == nil {
		t.Fail()
	}

	fmt.Println("payload len:", len(payload), " data: ", hex.EncodeToString(payload))

	alpha, err := slot.GetAlpha(epochID)
	if alpha == nil || err != nil {
		t.Fail()
	}

	var output [][]byte
	rlp.DecodeBytes(payload, &output)

	if hex.EncodeToString(pkCompress.SerializeCompressed()) != hex.EncodeToString(output[1]) {
		t.Fail()
	}

	fmt.Println("epochID: ", hex.EncodeToString(output[0]))
	fmt.Println("payload 0: ", hex.EncodeToString(output[1]))
	fmt.Println("payload 1: ", hex.EncodeToString(output[2]))
	fmt.Println("Alpha: ", alpha)
}

func TestGenerateCommitmentFailed(t *testing.T) {
	GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	epochID := new(big.Int).SetInt64(1)

	_, err = slot.GenerateCommitment(nil, epochID)
	if err == nil {
		t.Fail()
	}

	_, err = slot.GenerateCommitment(&privKey.PublicKey, nil)
	if err == nil {
		t.Fail()
	}

	privKey.PublicKey.X = nil
	privKey.PublicKey.Y = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey.PublicKey.Curve = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey2, _ := crypto.GenerateKey()

	privKey.X = privKey2.X
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID)
	if err == nil {
		t.Fail()
	}
}

func TestLoop(t *testing.T) {
	GetDb().DbInit("test")
	GetSlotLeaderSelection().Loop()
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
