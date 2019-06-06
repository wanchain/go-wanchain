package storemanmpc

import "github.com/wanchain/go-wanchain/p2p/discover"

type testP2pMessager struct {
}

func (ts *testP2pMessager) SendToPeer(*discover.NodeID, uint64, interface{}) error {
	return nil
}

func (ts *testP2pMessager) IsActivePeer(*discover.NodeID) bool {
	return true
}
