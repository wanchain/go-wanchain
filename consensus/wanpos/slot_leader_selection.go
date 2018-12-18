package wanpos

import (
	"crypto/ecdsa"
	Rand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/uleaderselection"
)

//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	Alpha *big.Int //Local alpha
}

var slotLeaderSelection *SlotLeaderSelection

func init() {
	slotLeaderSelection = &SlotLeaderSelection{Alpha: nil}
}

//GetSlotLeaderSelection get the SlotLeaderSelection's object
func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}

//GenerateCommitment generate a commitment and send it by tx message
//Returns the commitment buffer []byte or err
//payload should be send with tx.
func (s *SlotLeaderSelection) GenerateCommitment(publicKey *ecdsa.PublicKey) ([]byte, error) {
	alpha, err := uleaderselection.RandFieldElement(Rand.Reader)
	if err != nil {
		return nil, err
	}

	fmt.Println("input pubkey: ", hex.EncodeToString(crypto.FromECDSAPub(publicKey)), "buflen: ", len(crypto.FromECDSAPub(publicKey)))
	//commitment = PublicKey || alpha * PublicKey
	commitment, err := uleaderselection.GenerateCommitment(publicKey, alpha)
	if err != nil {
		return nil, err
	}

	fmt.Println("generate m: ", hex.EncodeToString(crypto.FromECDSAPub(commitment[1])), "buflen: ", len(crypto.FromECDSAPub(publicKey)))

	buffer := make([]byte, 0)
	for i := 0; i < len(commitment); i++ {
		pk := commitment[i]
		buffer = append(buffer, crypto.FromECDSAPub(pk)...)
	}

	s.Alpha = alpha

	return buffer, nil
}
