package vm

import (
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"testing"
)

func TestGetSlotLeaderStageIndexesKeyHash(t *testing.T) {

	for i := 0; i < 4; i++ {
		epochIDBuf := posdb.Uint64ToBytes(uint64(i))
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
