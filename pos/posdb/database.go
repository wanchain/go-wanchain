package posdb

import (
	"crypto/ecdsa"
	"encoding/hex"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"strings"

	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rlp"
)

//Db is the wanpos leveldb class
type Db struct {
	db *ethdb.LDBDatabase
}

func NewDb(dbPath string) *Db {
	dbInst := &Db{db: nil}
	dbInst.DbInit(dbPath)
	return dbInst
}

var dbInstance *Db

func init() {
	dbInstance = &Db{db: nil}
	dbInstance.DbInit("")
}

//GetDb can get a Db instance to use
func GetDb() *Db {
	return dbInstance
}

//DbInit use to init leveldb in this object, user should not use this. It is automate called in init().
func (s *Db) DbInit(dbPath string) {
	var dirname string
	var err error
	if dbPath == "" {
		dirname, err = ioutil.TempDir(os.TempDir(), "wanpos_tmpdb_")
		if err != nil {
			panic("failed to create wanpos_tmpdb file: " + err.Error())
		}
	} else {
		dirname = path.Join(dbPath, "wanposdb")
	}

	if s.db != nil {
		s.db.Close()
	}

	s.db, err = ethdb.NewLDBDatabase(dirname, 0, 0)
	if err != nil {
		panic("failed to create wanpos_tmpdb database: " + err.Error())
	}
}

//PutWithIndex use to set a key-value store with a given epochID and Index
func (s *Db) PutWithIndex(epochID uint64, index uint64, key string, value []byte) ([]byte, error) {

	newKey, err := rlp.EncodeToBytes([][]byte{
		Uint64ToBytes(epochID),
		Uint64ToBytes(index),
		[]byte(key),
	})

	s.db.Put(newKey, value)

	s.saveKey(newKey)

	return newKey, err
}

//GetWithIndex use to get a key-value store with a given epochID and Index
func (s *Db) GetWithIndex(epochID uint64, index uint64, key string) ([]byte, error) {

	newKey, err := rlp.EncodeToBytes([][]byte{
		Uint64ToBytes(epochID),
		Uint64ToBytes(index),
		[]byte(key),
	})

	if err != nil {
		return nil, err
	}

	return s.db.Get(newKey)
}

//Put use to set a key-value store with a given epochID
func (s *Db) Put(epochID uint64, key string, value []byte) ([]byte, error) {

	newKey, err := rlp.EncodeToBytes([][]byte{
		Uint64ToBytes(epochID),
		[]byte(key),
	})

	s.db.Put(newKey, value)

	s.saveKey(newKey)

	return newKey, err
}

//Get use to get a key-value store with a given epochID
func (s *Db) Get(epochID uint64, key string) ([]byte, error) {

	newKey, err := rlp.EncodeToBytes([][]byte{
		Uint64ToBytes(epochID),
		[]byte(key),
	})

	if err != nil {
		return nil, err
	}

	return s.db.Get(newKey)
}

func (s *Db) saveKey(key []byte) error {
	keyCount := s.getKeyCount()
	keyCount++

	keyName := "key_" + Uint64ToString(keyCount)

	_, err := s.putNoCount(0, keyName, key)
	if err != nil {
		return err
	}

	_, err = s.putNoCount(0, "keyCount", Uint64ToBytes(keyCount))
	if err != nil {
		return err
	}

	return nil
}

func (s *Db) getKeyCount() uint64 {
	ret, err := s.Get(0, "keyCount")
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	return BytesToUint64(ret)
}

func (s *Db) getAllKeys() [][]byte {
	keyCount := s.getKeyCount()

	keys := make([][]byte, 0)
	var i uint64
	for i = 0; i < keyCount; i++ {
		keyName := "key_" + Uint64ToString(i+1)
		ret, err := s.Get(0, keyName)
		if err != nil {
			log.Warn(err.Error())
			continue
		}

		keys = append(keys, ret)
	}

	return keys
}

func (s *Db) putNoCount(epochID uint64, key string, value []byte) ([]byte, error) {

	newKey, err := rlp.EncodeToBytes([][]byte{
		Uint64ToBytes(epochID),
		[]byte(key),
	})

	s.db.Put(newKey, value)
	return newKey, err
}

//DbClose use to close db file
func (s *Db) DbClose() {
	s.db.Close()
}

// GetStorageByteArray : cb is callback function. cb return true indicating like to continue, return false indicating stop
func (s *Db) GetStorageByteArray(epochID uint64) [][]byte {
	searchKey := hex.EncodeToString(Uint64ToBytes(epochID))

	keys := GetDb().getAllKeys()

	arrays := make([][]byte, 0)

	for i := 0; i < len(keys); i++ {
		if strings.Index(hex.EncodeToString(keys[i]), searchKey) == 0 {
			ret, err := s.db.Get(keys[i])
			if err != nil {
				log.Warn(err.Error())
				continue
			}
			arrays = append(arrays, ret)
		}
	}

	return arrays
}

//-------------common functions---------------------------------------

//PkEqual only can use in same curve. return whether the two points equal
func PkEqual(pk1, pk2 *ecdsa.PublicKey) bool {
	if pk1 == nil || pk2 == nil {
		return false
	}

	if hex.EncodeToString(pk1.X.Bytes()) == hex.EncodeToString(pk2.X.Bytes()) &&
		hex.EncodeToString(pk1.Y.Bytes()) == hex.EncodeToString(pk2.Y.Bytes()) {
		return true
	}
	return false
}

// Uint64ToBytes use a big.Int to transfer uint64 to bytes
// Must use big.Int to reverse
func Uint64ToBytes(input uint64) []byte {
	return big.NewInt(0).SetUint64(input).Bytes()
}

// BytesToUint64 use a big.Int to transfer uint64 to bytes
// Must input a big.Int bytes
func BytesToUint64(input []byte) uint64 {
	return big.NewInt(0).SetBytes(input).Uint64()
}

// Uint64ToString can change uint64 to string through a big.Int
func Uint64ToString(input uint64) string {
	return big.NewInt(0).SetUint64(input).String()
}

//-------------------------------------------------------------------
