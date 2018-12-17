package wanpos

import (
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
}
