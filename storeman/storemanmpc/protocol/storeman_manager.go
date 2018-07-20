package protocol

import (
	"github.com/wanchain/go-wanchain/p2p/discover"
)

type StoremanManager interface {
	P2pMessage(*discover.NodeID, uint64, interface{}) error
	BoardcastMessage([]discover.NodeID, uint64, interface{}) error
	SetMessagePeers(*MpcMessage, *[]PeerInfo)
	SelfNodeId() *discover.NodeID
	CreateKeystore(MpcResultInterface, *[]PeerInfo) error
	SignTransaction(MpcResultInterface) error
}
