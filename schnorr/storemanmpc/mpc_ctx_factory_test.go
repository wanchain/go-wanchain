package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/schnorr/storemanmpc/protocol"
	"testing"
)

func TestMpcCtxFactory(t *testing.T) {
	mpcFactory := MpcCtxFactory{}
	_, err := mpcFactory.CreateContext(mpcprotocol.MpcCreateLockAccountLeader, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(mpcprotocol.MpcCreateLockAccountPeer, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(mpcprotocol.MpcTXSignLeader, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(mpcprotocol.MpcTXSignPeer, 0, nil)
	if err != nil {
		t.Error("mpcFactory create err:", err)
	}

	_, err = mpcFactory.CreateContext(5, 0, nil)
	if err != nil {
		t.Log("mpcFactory create err:", err)
	}
}
