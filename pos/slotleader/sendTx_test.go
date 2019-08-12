package slotleader

import (
	"github.com/wanchain/go-wanchain/rpc"
)


//func testInit() {
//	SlsInit()
//	GetSlotLeaderSelection().Init(nil, &rpc.Client{}, &keystore.Key{})
//}

func testSender(rc *rpc.Client, tx map[string]interface{}) {
	return
}

//func TestSendStage1Tx(t *testing.T) {
//	testInit()
//	err := GetSlotLeaderSelection().sendSlotTx(nil, testSender)
//	if err != nil {
//		t.FailNow()
//	}
//}
//
//func TestSendStage2Tx(t *testing.T) {
//	testInit()
//	err := GetSlotLeaderSelection().sendSlotTx(nil, testSender)
//	if err != nil {
//		t.FailNow()
//	}
//}
