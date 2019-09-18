package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
)

type MpcCtxFactory struct {
}

func (*MpcCtxFactory) CreateContext(ctxType int,
	mpcID uint64,
	peers []mpcprotocol.PeerInfo,
	preSetValue ...MpcValue) (MpcInterface, error) {

	switch ctxType {
	case mpcprotocol.MpcGPKLeader:
		return reqGPKMpc(mpcID, peers, preSetValue...)
	case mpcprotocol.MpcGPKPeer:
		return ackGPKMpc(mpcID, peers, preSetValue...)

	case mpcprotocol.MpcSignLeader:
		return reqSignMpc(mpcID, peers, preSetValue...)
	case mpcprotocol.MpcSignPeer:
		return ackSignMpc(mpcID, peers, preSetValue...)
	}

	return nil, mpcprotocol.ErrContextType
}
