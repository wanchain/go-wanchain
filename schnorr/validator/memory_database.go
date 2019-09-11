package validator

import (
	"errors"
	"sync"

	"github.com/wanchain/go-wanchain/common"
)

// Memory implementation
type memDB struct {
	lock sync.RWMutex
	db   map[string][]byte
}

func (db *memDB) Put(key, val []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = val

	return nil
}

func (db *memDB) Has(key []byte) (bool, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	_, ok := db.db[string(key)]

	return ok, nil
}

func (db *memDB) Get(key []byte) ([]byte, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}

	return nil, errors.New("not found")
}

func (db *memDB) Close() {}

func (db *memDB) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}
