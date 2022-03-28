package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// StatusPacket is the network packet for the status message for eth/64 and later.
type StatusData struct {
	ProtocolVersion uint32
	NetworkID       uint64
	TD              *big.Int
	Head            common.Hash
	Genesis         common.Hash
}
