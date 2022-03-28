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

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"github.com/ethereum/go-ethereum/pos/util"
	"github.com/ethereum/go-ethereum/trie"
)

func (self *StateDB) GetStateByteArray(a common.Address, b common.Hash) []byte {
	stateObject := self.getStateObject(a)
	if stateObject != nil {
		return stateObject.GetStateByteArray(self.db, b)
	}
	return nil
}
func (self *StateDB) SetStateByteArray(addr common.Address, key common.Hash, value []byte) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetStateByteArray(self.db, key, value)
	}
}



// cb is callback function. cb return true indicating like to continue, return false indicating stop
func (db *StateDB) ForEachStorageByteArray(addr common.Address, cb func(key common.Hash, value []byte) bool) {

	so := db.getStateObject(addr)
	if so == nil {
		return
	}
	for h, value := range so.dirtyStorageByteArray {
		if !cb(h, value) {
			return
		}
	}


	for h, value := range so.pendingStorageByteArray {
		if !cb(h, value) {
			return
		}
	}


	it := trie.NewIterator(so.getTrie(db.db).NodeIterator(nil))
	for it.Next() {
		// ignore cached values
		key := common.BytesToHash(db.trie.GetKey(it.Key))
		_, ok1 := so.dirtyStorageByteArray[key]
		_, ok2 := so.pendingStorageByteArray[key]
		if !ok1 && !ok2 {
			if !cb(key, it.Value) {
				return
			}
		}
	}
}


// cb is callback function. cb return true indicating like to continue, return false indicating stop
func (db *StateDB) ForEachStorageByteArray2(addr common.Address, cb func(key common.Hash, value []byte) bool) {

	epochid,_ := util.GetCurrentBlkEpochSlotID()
	if epochid < posconfig.Cfg().MercuryEpochId {
		db.ForEachStorageByteArrayBeforeFork(addr, cb)

	} else {

		so := db.getStateObject(addr)
		if so == nil {
			return
		}

		// When iterating over the storage check the cache first
		for h, value := range so.dirtyStorageByteArray {
			if !cb(h, value) {
				return
			}
		}

		it := trie.NewIterator(so.getTrie(db.db).NodeIterator(nil))
		for it.Next() {
			// ignore cached values
			key := common.BytesToHash(db.trie.GetKey(it.Key))
			if _, ok := so.dirtyStorageByteArray[key]; !ok {
				if !cb(key, it.Value) {
					return
				}
			}
		}
	}
}

// cb is callback function. cb return true indicating like to continue, return false indicating stop
func (db *StateDB) ForEachStorageByteArrayBeforeFork(addr common.Address, cb func(key common.Hash, value []byte) bool) {
	so := db.getStateObject(addr)
	if so == nil {
		return
	}

	// When iterating over the storage check the cache first
	for h, value := range so.pendingStorageByteArray {
		if !cb(h, value) {
			return
		}
	}

	it := trie.NewIterator(so.getTrie(db.db).NodeIterator(nil))
	for it.Next() {
		// ignore cached values
		key := common.BytesToHash(db.trie.GetKey(it.Key))
		//if _, ok := so.dirtyStorageByteArray[key]; !ok {
		//	if !cb(key, it.Value) {
		//		return
		//	}
		//}
		//if _, ok := so.originStorage[key]; !ok {
		//	if !cb(key, it.Value) {
		//		return
		//	}
		//}
		//if _, ok := so.dirtyStorage[key]; !ok {
		//	if !cb(key, it.Value) {
		//		return
		//	}
		//}
		if _, ok := so.pendingStorage[key]; !ok {
			if !cb(key, it.Value) {
				return
			}
		}
		//if _, ok := so.pendingStorage[key]; !ok {
		//	if !cb(key, it.Value) {
		//		return
		//	}
		//}
	}
}
