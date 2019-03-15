package slotleader

import (
	"testing"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/rpc"
)

func testInit() {
	rc := &rpc.Client{}
	GetSlotLeaderSelection().Init(nil, rc, &keystore.Key{}, nil)
}

func testSender(rc *rpc.Client, tx map[string]interface{}) (common.Hash, error) {
	return common.Hash{}, nil
}

func TestSendStage1Tx(t *testing.T) {
	testInit()
	err := GetSlotLeaderSelection().sendStage1Tx(nil, testSender)
	if err != nil {
		t.FailNow()
	}
}

func TestSendStage2Tx(t *testing.T) {
	testInit()
	err := GetSlotLeaderSelection().sendStage2Tx(nil, testSender)
	if err != nil {
		t.FailNow()
	}
}
