package vm

import (
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/consensus/wanpos"
	"github.com/wanchain/go-wanchain/crypto"
)

func TestWanSlotLeaderCommitment(t *testing.T) {
	contract := &wanSlotLeaderCommitment{}

	contract.RequiredGas(nil)

	contract.Run(nil, nil, nil)

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	slot := wanpos.GetSlotLeaderSelection()
	epochID := new(big.Int).SetInt64(4096)

	payload, err := slot.GenerateCommitment(&privKey.PublicKey, epochID)

	contract.Run(payload, nil, nil)
}
