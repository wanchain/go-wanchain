package storemanmpc

import (
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
)

type MpcCtxFactory struct {
}

func (*MpcCtxFactory) CreateContext(ctxType int,
	mpcID uint64,
	peers []mpcprotocol.PeerInfo,
	preSetValue ...MpcValue) (MpcInterface, error) {

	log.Info("============================CreateContext=====================")
	log.Info("CreateContext", "ctxType", ctxType)
	for i := 0; i < len(preSetValue); i++ {
		if preSetValue[i].Value != nil {
			log.Info("preSetValue", "key", preSetValue[i].Key, "value", preSetValue[i].Value)
		} else if preSetValue[i].ByteValue != nil {
			log.Info("preSetValue", "key", preSetValue[i].Key, "bytevalue", preSetValue[i].ByteValue)
		}
	}
	log.Info("============================CreateContext=====================")

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
