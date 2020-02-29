package posdb

import (
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wanchain/go-wanchain/common"

	"github.com/wanchain/go-wanchain/rlp"

	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util/convert"
)

//Db is the wanpos leveldb class
type Db struct {
	db *ethdb.LDBDatabase
}

var (
	dbInstMap = make(map[string]*Db)
	//RANDOMBEACON_DB_KEY = "PosRandomBeacon"

	mu sync.RWMutex
)

func NewDb(fileName string) *Db {
	mu.Lock()
	defer mu.Unlock()
	db := GetDbByName(fileName)
	if db != nil {
		return db
	}

	dbInst := &Db{db: nil}

	dbInst.DbInit(fileName)

	dbInstMap[fileName] = dbInst

	return dbInst
}

var dbInstance *Db

func init() {
	dbInstance = NewDb("")
}

// DbInitAll init all db files
func DbInitAll(pathname string) {
	posconfig.Cfg().Dbpath = pathname
	dbInstance = NewDb(posconfig.PosLocalDB)
	NewDb(posconfig.RbLocalDB)
	NewDb(posconfig.EpLocalDB)
}

//GetDb can get a Db instance to use
func GetDb() *Db {
	return dbInstance
}

func GetDbByName(name string) *Db {
	return dbInstMap[name]
}

//DbInit use to init leveldb in this object, user should not use this. It is automate called in init().
func (s *Db) DbInit(dbPath string) {
	var dirname string
	var err error
	if dbPath == "" {
		//Remove old unuse tmp files and directorys.
		files, _ := filepath.Glob(filepath.Join(os.TempDir(), "wanpos_tmpdb_*"))
		for i := 0; i < len(files); i++ {
			os.RemoveAll(files[i])
		}

		dirname, err = ioutil.TempDir(os.TempDir(), "wanpos_tmpdb_")
		if err != nil {
			panic("failed to create wanpos_tmpdb file: " + err.Error())
		}
	} else {
		nameIdx := strings.LastIndex(dbPath, string(os.PathSeparator))
		if nameIdx < 0 {
			dirname = path.Join(posconfig.Cfg().Dbpath, "gwan")
			dirname = path.Join(dirname, dbPath)
		} else {
			dbPath = path.Join(dbPath, "gwan")
			dirname = path.Join(dbPath, "wanposdb")
		}
	}

	if s.db != nil {
		s.db.Close()
	}

	inst, ok := dbInstMap[dbPath]

	if ok {
		inst.DbClose()
	}

	s.db, err = ethdb.NewLDBDatabase(dirname, 0, 256)
	if err != nil {
		panic("failed to create wanpos_tmpdb database: " + dbPath + "_" + err.Error())
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
	_, err = s.putNoCount(0, s.getKeyCountName(epochID), convert.Uint64ToBytes(keyCount))
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
	return convert.BytesToUint64(ret)
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
	return "key_" + convert.Uint64ToString(epochID) + "_" + convert.Uint64ToString(keyIndex)
}

func (s *Db) getKeyCountName(epochID uint64) string {
	return "keyCount_" + convert.Uint64ToString(epochID)
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
	uskey := convert.Uint64ToString(epochID) + "_" + convert.Uint64ToString(index) + "_" + key
	return uskey
}

func (s *Db) getUniqueKeyBytes(epochID uint64, index uint64, key string) []byte {
	return []byte(s.getUniqueKey(epochID, index, key))
}

// TODO duplicated with epochLeader
type Proposer struct {
	PubSec256     []byte
	PubBn256      []byte
	Probabilities *big.Int
}

func GetRBProposerGroup(epochId uint64) [][]byte {
	db := NewDb(posconfig.RbLocalDB)
	if db == nil {
		// todo : os.Exit ??
		log.SyslogErr("GetRBProposerGroup create db error")
		return nil
	}

	proposersArray := db.GetStorageByteArray(epochId)
	length := len(proposersArray)
	g1s := make([][]byte, length, length)

	for i := 0; i < length; i++ {
		proposer := Proposer{}
		err := rlp.DecodeBytes(proposersArray[i], &proposer)
		if err != nil {
			log.Error("can't rlp decode:", err)
		}
		g1s[i] = proposer.PubBn256
	}

	return g1s

}

func GetStakerInfoBytes(epochId uint64, addr common.Address) []byte {
	db := NewDb(posconfig.StakerLocalDB)
	if db == nil {
		log.SyslogErr("GetStakerInfo create db error")
		return nil
	}
	stakerBytes, err := db.GetWithIndex(epochId, 0, common.ToHex(addr[:]))
	if err != nil {
		return nil
	}

	return stakerBytes
}

func GetEpochLeaderGroup(epochId uint64) [][]byte {
	db := NewDb(posconfig.EpLocalDB)
	if db == nil {
		log.SyslogErr("GetEpochLeaderGroup create db error")
		return nil
	}

	proposersArray := db.GetStorageByteArray(epochId)
	length := len(proposersArray)
	pks := make([][]byte, length, length)

	for i := 0; i < length; i++ {
		proposer := Proposer{}
		err := rlp.DecodeBytes(proposersArray[i], &proposer)
		if err != nil {
			log.Error("can't rlp decode:", err)
		}
		pks[i] = proposer.PubSec256
	}

	return pks

}

//-------------------------------------------------------------------
