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
)

//Db is the wanpos leveldb class
type Db struct {
	db *ethdb.LDBDatabase
}
var (
	dbInstMap = make(map[string]*Db)
)

func NewDb(dbPath string) *Db {
	dbInst := &Db{db: nil}
	dbInst.DbInit(dbPath)
	dbInstMap[dbPath] = dbInst
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

func GetDbByName(name string) (*Db) {
	return dbInstMap[name]
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

func (s *Db) put(epochID uint64, index uint64, key string, value []byte, saveKey bool) ([]byte, error) {
	newKey := s.getUniqueKeyBytes(epochID, index, key)

	has, err := s.db.Has(newKey)
	if err != nil {
		return nil, err
	}

	if has {
		saveKey = false
	}

	s.db.Put(newKey, value)
	if saveKey {
		err = s.saveKey(newKey, epochID)
	}
	return newKey, err
}

//PutWithIndex use to set a key-value store with a given epochID and Index
func (s *Db) PutWithIndex(epochID uint64, index uint64, key string, value []byte) ([]byte, error) {
	return s.put(epochID, index, key, value, true)
}

//GetWithIndex use to get a key-value store with a given epochID and Index
func (s *Db) GetWithIndex(epochID uint64, index uint64, key string) ([]byte, error) {
	newKey := s.getUniqueKeyBytes(epochID, index, key)

	ret, err := s.db.Get(newKey)
	if err != nil {
		//debug.PrintStack()
	}
	return ret, err
}

//Put use to set a key-value store with a given epochID
func (s *Db) Put(epochID uint64, key string, value []byte) ([]byte, error) {
	return s.PutWithIndex(epochID, 0, key, value)
}

//Get use to get a key-value store with a given epochID
func (s *Db) Get(epochID uint64, key string) ([]byte, error) {
	return s.GetWithIndex(epochID, 0, key)
}

func (s *Db) saveKey(key []byte, epochID uint64) error {
	keyCount := s.getKeyCount(epochID)

	keyName := s.getKeyName(epochID, keyCount)

	_, err := s.putNoCount(0, keyName, key)
	if err != nil {
		return err
	}

	keyCount++
	_, err = s.putNoCount(0, s.getKeyCountName(epochID), Uint64ToBytes(keyCount))
	if err != nil {
		return err
	}

	return nil
}

func (s *Db) getKeyCount(epochID uint64) uint64 {
	ret, err := s.Get(0, s.getKeyCountName(epochID))
	if err != nil {
		return 0
	}
	return BytesToUint64(ret)
}

func (s *Db) getAllKeys(epochID uint64) []string {
	keyCount := s.getKeyCount(epochID)

	keys := make([]string, keyCount)
	var i uint64
	for i = 0; i < keyCount; i++ {
		keyName := s.getKeyName(epochID, i)
		ret, err := s.Get(0, keyName)
		if err != nil {
			log.Warn(err.Error())
			continue
		}

		keys[i] = string(ret)
	}

	return keys
}

func (s *Db) getKeyName(epochID uint64, keyIndex uint64) string {
	return "key_" + Uint64ToString(epochID) + "_" + Uint64ToString(keyIndex)
}

func (s *Db) getKeyCountName(epochID uint64) string {
	return "keyCount_" + Uint64ToString(epochID)
}

func (s *Db) putNoCount(epochID uint64, key string, value []byte) ([]byte, error) {
	return s.put(epochID, 0, key, value, false)
}

//DbClose use to close db file
func (s *Db) DbClose() {
	s.db.Close()
}

// GetStorageByteArray : cb is callback function. cb return true indicating like to continue, return false indicating stop
func (s *Db) GetStorageByteArray(epochID uint64) [][]byte {
	searchKey := s.getUniqueKey(epochID, 0, "")

	searchKey = strings.Split(searchKey, "_")[0]

	keys := s.getAllKeys(epochID)

	arrays := make([][]byte, len(keys))

	for i := 0; i < len(keys); i++ {
		ret, err := s.db.Get([]byte(keys[i]))
		if err != nil {
			log.Warn(err.Error())
			continue
		}
		arrays[i] = ret
	}

	return arrays
}

func (s *Db) getUniqueKey(epochID uint64, index uint64, key string) string {
	uskey := Uint64ToString(epochID) + "_" + Uint64ToString(index) + "_" + key
	return uskey
}

func (s *Db) getUniqueKeyBytes(epochID uint64, index uint64, key string) []byte {
	return []byte(s.getUniqueKey(epochID, index, key))
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
	str := big.NewInt(0).SetUint64(input).String()
	if len(str) == 0 {
		str = "00"
	}
	return str
}

//-------------------------------------------------------------------
