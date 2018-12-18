package wanpos

import (
	"crypto/ecdsa"
	Rand "crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/wanchain/go-wanchain/rlp"

	"github.com/btcsuite/btcd/btcec"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/pos/uleaderselection"
)

//CompressedPubKeyLen means a compressed public key byte len.
const CompressedPubKeyLen = 33

//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	db *ethdb.LDBDatabase
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
	if publicKey == nil || epochID == nil || publicKey.X == nil || publicKey.Y == nil {
		return nil, errors.New("Invalid input parameters")
	}

	if !crypto.S256().IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, errors.New("Public key point is not on S256 curve")
	}

	alpha, err := uleaderselection.RandFieldElement(Rand.Reader)
	if err != nil {
		return nil, err
	}
	fmt.Println("alpha:", alpha)

	commitment, err := uleaderselection.GenerateCommitment(publicKey, alpha)
	if err != nil {
		return nil, err
	}

	pk := btcec.PublicKey(*commitment[0])
	mi := btcec.PublicKey(*commitment[1])

	pkCompress := pk.SerializeCompressed()
	miCompress := mi.SerializeCompressed()
	epochIDBuf := epochID.Bytes()

	buffer, err := rlp.EncodeToBytes([][]byte{epochIDBuf, pkCompress, miCompress})

	s.dbPut(epochID, "alpha", alpha.Bytes())

	return buffer, err
}

//GetAlpha get alpha of epochID
func (s *SlotLeaderSelection) GetAlpha(epochID *big.Int) (*big.Int, error) {
	buf, err := s.dbGet(epochID, "alpha")
	if err != nil {
		return nil, err
	}

	var alpha = new(big.Int).SetBytes(buf)
	return alpha, nil
}

func (s *SlotLeaderSelection) dbPut(epochID *big.Int, key string, value []byte) ([]byte, error) {

	newKey, err := rlp.EncodeToBytes([][]byte{
		epochID.Bytes(),
		[]byte(key),
	})

	s.db.Put(newKey, value)
	return newKey, err
}

func (s *SlotLeaderSelection) dbGet(epochID *big.Int, key string) ([]byte, error) {

	newKey, err := rlp.EncodeToBytes([][]byte{
		epochID.Bytes(),
		[]byte(key),
	})

	if err != nil {
		return nil, err
	}

	return s.db.Get(newKey)
}
