// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"bytes"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
)

type StorageByteArray map[common.Hash][]byte

// stateObject represents an Ethereum account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateObject struct {
	address  common.Address
	addrHash common.Hash // hash of ethereum address of the accountGetStateByteArray
	data     types.StateAccount
	db       *StateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage cache of original entries to dedup rewrites, reset for every transaction
	pendingStorage Storage // Storage entries that need to be flushed to disk, at the end of an entire block
	dirtyStorage   Storage // Storage entries that have been modified in the current transaction execution
	fakeStorage    Storage // Fake storage which constructed by caller for debugging purpose.

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	deleted   bool

	// new
	originStorageByteArray  StorageByteArray // Storage cache of original entries to dedup rewrites, reset for every transaction
	dirtyStorageByteArray   StorageByteArray
	pendingStorageByteArray StorageByteArray
}

func (self *stateObject) GetStateByteArray(db Database, key common.Hash) []byte {
	value, exists := self.dirtyStorageByteArray[key]
	if exists {
		return value
	}

	return self.GetCommittedStateByteArray(db, key)
}

// GetCommittedStateByteArray retrieves a value from the committed account storage trie.
func (s *stateObject) GetCommittedStateByteArray(db Database, key common.Hash) []byte {
	// If we have a pending write or clean cached, return that
	if value, pending := s.pendingStorageByteArray[key]; pending {
		return value
	}
	if value, cached := s.originStorageByteArray[key]; cached {
		return value
	}
	// If no live objects are available, attempt to use snapshots
	var (
		enc   []byte
		err   error
		meter *time.Duration
	)
	readStart := time.Now()
	if metrics.EnabledExpensive {
		// If the snap is 'under construction', the first lookup may fail. If that
		// happens, we don't want to double-count the time elapsed. Thus this
		// dance with the metering.
		defer func() {
			if meter != nil {
				*meter += time.Since(readStart)
			}
		}()
	}
	if s.db.snap != nil {
		if metrics.EnabledExpensive {
			meter = &s.db.SnapshotStorageReads
		}
		// If the object was destructed in *this* block (and potentially resurrected),
		// the storage has been cleared out, and we should *not* consult the previous
		// snapshot about any storage values. The only possible alternatives are:
		//   1) resurrect happened, and new slot values were set -- those should
		//      have been handles via pendingStorage above.
		//   2) we don't have new values, and can deliver empty response back
		if _, destructed := s.db.snapDestructs[s.addrHash]; destructed {
			return []byte{}
		}
		enc, err = s.db.snap.Storage(s.addrHash, crypto.Keccak256Hash(key.Bytes()))
	}
	// If the snapshot is unavailable or reading from it fails, load from the database.
	if s.db.snap == nil || err != nil {
		if meter != nil {
			// If we already spent time checking the snapshot, account for it
			// and reset the readStart
			*meter += time.Since(readStart)
			readStart = time.Now()
		}
		if metrics.EnabledExpensive {
			meter = &s.db.StorageReads
		}
		if enc, err = s.getTrie(db).TryGet(key.Bytes()); err != nil {
			s.setError(err)
			return []byte{}
		}
	}

	// TODO: why not rlp.split for []byte
	s.originStorageByteArray[key] = enc
	return enc
}

func (self *stateObject) SetStateByteArray(db Database, key common.Hash, value []byte) {
	// If the new value is the same as old, don't set
	prev := self.GetStateByteArray(db, key)
	if bytes.Equal(prev, value) {
		return
	}

	self.db.journal.append(storageByteArrayChange{
		account:  &self.address,
		key:      key,
		prevalue: self.GetStateByteArray(db, key),
	})
	self.setStateByteArray(key, value)

}

func (self *stateObject) setStateByteArray(key common.Hash, value []byte) {
	self.dirtyStorageByteArray[key] = value
}

func (self StorageByteArray) Copy() StorageByteArray {
	cpy := make(StorageByteArray)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

func (s *stateObject) deepCopy(db *StateDB) *stateObject {
	stateObject := newObject(db, s.address, s.data)
	if s.trie != nil {
		stateObject.trie = db.db.CopyTrie(s.trie)
	}
	stateObject.code = s.code
	stateObject.dirtyStorage = s.dirtyStorage.Copy()
	stateObject.originStorage = s.originStorage.Copy()
	stateObject.pendingStorage = s.pendingStorage.Copy()
	stateObject.suicided = s.suicided
	stateObject.dirtyCode = s.dirtyCode
	stateObject.deleted = s.deleted

	stateObject.originStorageByteArray = s.originStorageByteArray.Copy()
	stateObject.dirtyStorageByteArray = s.dirtyStorageByteArray.Copy()
	stateObject.pendingStorageByteArray = s.pendingStorageByteArray.Copy()

	return stateObject
}

// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	emptyHash := common.Hash{}
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 &&
		(bytes.Equal(s.data.CodeHash, emptyCodeHash) || bytes.Equal(s.data.CodeHash, emptyHash[:])) &&
		(s.data.Root == emptyHash || s.data.Root == emptyRoot) &&
		len(s.dirtyStorage) == 0 && len(s.dirtyStorageByteArray) == 0
}
