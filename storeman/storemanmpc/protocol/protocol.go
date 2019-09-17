package protocol

import (
	"bytes"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"math/big"
	"strconv"
	"time"
)

const (
	MpcSchnrThr        = 26 // MpcSchnrThr >= number(storeman)/2 +1
	MpcSchnrDegree     = MpcSchnrThr - 1
	MpcSchnrNodeNumber = 50 // At least MpcSchnrNodeNumber MPC nodes
	MPCDegree          = 8
	//MPCDegree          = 1
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

	MPCTimeOut         = time.Second * 100
	ProtocolName       = "storeman"
	ProtocolVersion    = uint64(1)
	ProtocolVersionStr = "1.1"
)
const (
	MpcPrivateShare  = "MpcPrivateShare"  // skShare
	RMpcPrivateShare = "RMpcPrivateShare" // rskShare
	MpcPublicShare   = "MpcPublicShare"   // pkShare
	RMpcPublicShare  = "RMpcPublicShare"  // rpkShare
	MpcContextResult = "MpcContextResult"

	PublicKeyResult  = "PublicKeyResult"
	RPublicKeyResult = "RPublicKeyResult" // R
	MpcM             = "MpcM"             // M
	Mpcm             = "Mpcm"             // m
	MpcS             = "MpcS"             // S

	MpcSignAPoint  = "MpcSignAPoint"
	MpcTxHash      = "MpcTxHash"
	MpcTransaction = "MpcTransaction"
	MpcChainType   = "MpcChainType"
	MpcSignType    = "MpcSignType"
	MpcChainID     = "MpcChainID"
	MpcAddress     = "MpcAddress"
	MPCActoin      = "MPCActoin"
	MPCSignedFrom  = "MPCSignedFrom"
	MpcStmAccType  = "MpcStmAccType"
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
	Msgcode   uint64 //message code
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

func CheckAccountType(accType string) bool {
	if accType == "WAN" || accType == "ETH" || accType == "BTC" {
		return true
	}

	return false
}

func GetPreSetKeyArr(keySeed string, num int) []string {
	keyArr := []string{}
	for i := 0; i < num; i++ {
		keyArr = append(keyArr, keySeed+"_"+strconv.Itoa(i))
	}

	return keyArr
}
