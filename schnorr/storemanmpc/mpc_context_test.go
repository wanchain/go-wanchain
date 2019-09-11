package storemanmpc

import (
	"github.com/wanchain/go-wanchain/common"
	mpcprotocol "github.com/wanchain/go-wanchain/schnorr/storemanmpc/protocol"
	"testing"
)

func TestContextQuit(t *testing.T) {
	testHash := []byte{130, 250, 171, 196, 164, 207, 30, 58, 73, 250, 188, 18, 53, 172, 48, 39, 187, 92, 60, 59, 208, 169, 165, 21, 184, 139, 28, 130, 176, 176, 66}
	t.Log(common.BytesToHash(testHash).String())
	nThread := 21
	nodeId := common.Hex2Bytes("8f8581b96c387d80c64b8924ec466b17d3994db98ea5601c4ccb4b0a5acead74f43b7ffde50cfcf96efd19005e3986186268d0c0865b4217d28eff61d693bc16")
	peers := make([]mpcprotocol.PeerInfo, nThread)
	for i := 0; i < nThread; i++ {
		copy(peers[i].PeerID[:], nodeId)
		peers[i].PeerID[0] = byte(i + 1)
		peers[i].PeerID[1] = byte(i + 1)
		peers[i].PeerID[2] = byte(i + 1)
		peers[i].Seed = uint64(i + 1)
	}

	mpc, err := requestTxSignMpc(1, peers)
	if err != nil {
		t.Error("mpc create error")
	}

	go func() {
		for _, mpcCt := range mpc.MpcSteps {
			err := mpcCt.InitMessageLoop(mpcCt)
			if err != nil {
				t.Error("mpc loop error error")
				break
			}
		}
	}()

	for i := 0; i < 10; i++ {
		mpc.quit(nil)
	}
}
