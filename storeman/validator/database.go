package validator

import (
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
)

var dbInstance Database

type Database interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Has(key []byte) (bool, error)
	Close()
}

// Leveldb implementation
type storemanDB struct {
	fn string
	db *leveldb.DB
}

func NewDatabase(file string) error {

	if dbInstance != nil {
		return nil
	}

	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: 5,
	})

	if err != nil {
		return err
	}

	dbInstance = &storemanDB{
		fn: file,
		db: db,
	}

	return nil
}

func (db *storemanDB) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *storemanDB) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

func (db *storemanDB) Get(key []byte) ([]byte, error) {
	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	return dat, nil
}

func (db *storemanDB) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *storemanDB) Close() {
	err := db.db.Close()
	if err == nil {
		mpcsyslog.Info("Storeman database closed")
	} else {
		mpcsyslog.Err("Failed to close database. err:%s", err.Error())
	}

	dbInstance = nil
}

// GetDB returns singleton of Database implementation
func GetDB() (Database, error) {
	if dbInstance == nil {
		mpcsyslog.Err("get storeman database error")
		return nil, errors.New("get storeman database error")
	}

	return dbInstance, nil
}
