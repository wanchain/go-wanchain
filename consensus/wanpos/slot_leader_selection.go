package wanpos

import (
	"crypto/ecdsa"
	Rand "crypto/rand"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/pos/uleaderselection"
)

//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	Alpha *big.Int //Local alpha
	db    *ethdb.LDBDatabase
}

var slotLeaderSelection *SlotLeaderSelection

func init() {
	slotLeaderSelection = &SlotLeaderSelection{db: nil}
	slotLeaderSelection.Init()
}

//GetSlotLeaderSelection get the SlotLeaderSelection's object
func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}

//Init use to init leveldb in this object, user should not use this. It is automate called in init().
func (s *SlotLeaderSelection) Init() {
	dirname, err := ioutil.TempDir(os.TempDir(), "wanpos_tmpdb_")
	if err != nil {
		panic("failed to create wanpos_tmpdb file: " + err.Error())
	}
	s.db, err = ethdb.NewLDBDatabase(dirname, 0, 0)
	if err != nil {
		panic("failed to create wanpos_tmpdb database: " + err.Error())
	}
}

//GenerateCommitment generate a commitment and send it by tx message
//Returns the commitment buffer []byte which is publicKey and alpha * publicKey
//payload should be send with tx.
func (s *SlotLeaderSelection) GenerateCommitment(publicKey *ecdsa.PublicKey) ([]byte, error) {
	alpha, err := uleaderselection.RandFieldElement(Rand.Reader)
	if err != nil {
		return nil, err
	}

	commitment, err := uleaderselection.GenerateCommitment(publicKey, alpha)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, 130)
	for i := 0; i < len(commitment); i++ {
		pk := commitment[i]
		buffer = append(buffer, crypto.FromECDSAPub(pk)...)
	}

	s.Alpha = alpha

	return buffer, nil
}
