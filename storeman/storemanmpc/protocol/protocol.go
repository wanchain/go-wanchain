package protocol

import (
	"bytes"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"math/big"
	"time"
)

const (
	//MpcSchnrThr        = 26 // MpcSchnrThr >= number(storeman)/2 +1
	MpcSchnrThr = 2 // MpcSchnrThr >= number(storeman)/2 +1
	MPCDegree   = MpcSchnrThr - 1
	//MpcSchnrNodeNumber = 50 // At least MpcSchnrNodeNumber MPC nodes
	MpcSchnrNodeNumber = 3 // At least MpcSchnrNodeNumber MPC nodes
)

const (
	MpcGPKLeader = iota + 0
	MpcGPKPeer
	MpcSignLeader
	MpcSignPeer
)
const (
	StatusCode = iota + 0 // used by storeman protocol
	KeepaliveCode
	KeepaliveOkCode
	MPCError
	RequestMPC // ask for a new mpc Context
	MPCMessage // get a message for a Context
	RequestMPCNonce
	KeepaliveCycle
	NumberOfMessageCodes

	//MPCTimeOut = time.Second * 100
	MPCTimeOut = time.Second * 10
	PName      = "storeman"
	PVer       = uint64(1)
	PVerStr    = "1.1"
)
const (
	MpcPrivateShare  = "MpcPrivateShare"  // skShare
	RMpcPrivateShare = "RMpcPrivateShare" // rskShare
	MpcPublicShare   = "MpcPublicShare"   // pkShare
	RMpcPublicShare  = "RMpcPublicShare"  // rpkShare
	MpcContextResult = "MpcContextResult"

	PublicKeyResult  = "PublicKeyResult"  // gpk
	RPublicKeyResult = "RPublicKeyResult" // R: rpk
	MpcM             = "MpcM"             // M
	MpcS             = "MpcS"             // S: s

	MpcTxHash  = "MpcTxHash"
	MpcAddress = "MpcAddress"
	MPCAction  = "MPCAction"
)

type PeerInfo struct {
	PeerID discover.NodeID
	Seed   uint64
}
type SliceStoremanGroup []discover.NodeID

func (s SliceStoremanGroup) Len() int {
	return len(s)
}
func (s SliceStoremanGroup) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SliceStoremanGroup) Less(i, j int) bool {
	return bytes.Compare(s[i][:], s[j][:]) < 0
}

type GetMessageInterface interface {
	HandleMessage(*StepMessage) bool
}

type StepMessage struct {
	MsgCode   uint64 //message code
	PeerID    *discover.NodeID
	Peers     *[]PeerInfo
	Data      []big.Int //message data
	BytesData [][]byte
}

type MpcMessage struct {
	ContextID uint64
	StepID    uint64
	Peers     []byte
	Data      []big.Int //message data
	BytesData [][]byte
}
