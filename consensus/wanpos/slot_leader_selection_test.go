package wanpos

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
)

func TestSlotLeaderSelectionGetInstance(t *testing.T) {
	slot := GetSlotLeaderSelection()
	if slot == nil {
		t.Fail()
	}

	if slot.Alpha != nil {
		t.Fail()
	}
}

func TestGenerateCommitmentSuccess(t *testing.T) {
	slot := GetSlotLeaderSelection()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	fmt.Println("priv len:", len(crypto.FromECDSA(privKey)))
	fmt.Println("pk len:", len(crypto.FromECDSAPub(&privKey.PublicKey)))

	payload, err := slot.GenerateCommitment(&privKey.PublicKey)
	if err != nil {
		t.Fail()
	}

	if payload == nil {
		t.Fail()
	}

	if slot.Alpha == nil {
		t.Fail()
	}

	pk := payload[:65]
	m := payload[65:]

	fmt.Println("payload 0: ", hex.EncodeToString(pk))
	fmt.Println("payload 1: ", hex.EncodeToString(m))
	fmt.Println("Alpha: ", GetSlotLeaderSelection().Alpha)
}

func TestGenerateCommitmentFailed(t *testing.T) {
	slot := GetSlotLeaderSelection()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	_, err = slot.GenerateCommitment(nil)
	if err == nil {
		t.Fail()
	}

	privKey.PublicKey.X = nil
	privKey.PublicKey.Y = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey.PublicKey.Curve = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey)
	if err == nil {
		t.Fail()
	}
}
