package pos

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/rlp"
)

//Db is the wanpos leveldb class
type Db struct {
	db *ethdb.LDBDatabase
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

//DbClose use to close db file
func (s *Db) DbClose() {
	s.db.Close()
}
