package wanpos

import (
	"crypto/ecdsa"
	Rand "crypto/rand"
	"encoding/hex"
	"errors"
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
	slotLeaderSelection.DbInit()
}

//GetSlotLeaderSelection get the SlotLeaderSelection's object
func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}

//DbInit use to init leveldb in this object, user should not use this. It is automate called in init().
func (s *SlotLeaderSelection) DbInit() {
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
func (s *SlotLeaderSelection) GenerateCommitment(publicKey *ecdsa.PublicKey, epochID *big.Int) ([]byte, error) {
	if publicKey == nil || epochID == nil {
		return nil, errors.New("Invalid input parameters")
	}

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

	s.dbPut(epochID, "alpha", buffer)

	return buffer, nil
}

func (s *SlotLeaderSelection) dbPut(epochID *big.Int, key string, value []byte) error {

	keyBuf, err := hex.DecodeString(key)
	if err != nil {
		return err
	}
	epochBuf := epochID.Bytes()

	newKey := make([]byte, len(keyBuf)+len(epochBuf))

	copy(newKey, keyBuf)
	copy(newKey[len(keyBuf):], epochBuf)

	s.db.Put(newKey, value)
	return nil
}

func (s *SlotLeaderSelection) dbGet(epochID *big.Int, key string) ([]byte, error) {
	keyBuf, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	epochBuf := epochID.Bytes()

	newKey := make([]byte, len(keyBuf)+len(epochBuf))

	copy(newKey, keyBuf)
	copy(newKey[len(keyBuf):], epochBuf)

	return s.db.Get(newKey)
}
