package vm

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
