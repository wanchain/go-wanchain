package slottools

import (
	"crypto/ecdsa"
	"math/big"
)

type SlotLeader interface {
	GetStg1StateDbInfo(epochID uint64, index uint64) (mi []byte, err error)
	GetStage2TxAlphaPki(epochID uint64, selfIndex uint64) (alphaPkis []*ecdsa.PublicKey, proofs []*big.Int, err error)
	GetEpochLeadersPK(epochID uint64) []*ecdsa.PublicKey
}

var slotLeaderBridge SlotLeader

func SetSlotLeaderInst(sor SlotLeader) {
	slotLeaderBridge = sor
}
func GetSlotLeaderInst() SlotLeader {
	if slotLeaderBridge == nil {
		panic("GetSlotLeaderInst")
	}
	return slotLeaderBridge
}
