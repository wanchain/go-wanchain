package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
)

type MpcTestCtxFactory struct {
}

func (*MpcTestCtxFactory) CreateContext(ctxType int, mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (MpcInterface, error) {
	switch ctxType {
	case mpcprotocol.MpcGPKLeader:
		return testCreatep2pMpc(mpcID, peers, preSetValue...)

	case mpcprotocol.MpcGPKPeer:
		return acknowledgeCreatep2pMpc(mpcID, peers, preSetValue...)

	case mpcprotocol.MpcSignLeader:
		return reqSignMpc(mpcID, peers, preSetValue...)

	case mpcprotocol.MpcSignPeer:
		return ackSignMpc(mpcID, peers, preSetValue...)
	}

	return nil, mpcprotocol.ErrContextType
}
