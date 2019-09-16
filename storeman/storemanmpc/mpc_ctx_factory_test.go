package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"testing"
)

func TestMpcCtxFactory(t *testing.T) {
	mpcFactory := MpcCtxFactory{}
	_, err := mpcFactory.CreateContext(mpcprotocol.MpcGPKLeader, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(mpcprotocol.MpcGPKPeer, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(mpcprotocol.MpcSignLeader, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(mpcprotocol.MpcSignPeer, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(5, 0, nil)
	if err != nil {
		t.Log("mpcFactory create err:", err)
	}
}
