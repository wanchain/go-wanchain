// Copyright 2018 Wanchain Foundation Ltd
// Copyright 2017 The go-ethereum Authors
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
// The code is mostly inspired by POA of ethereum

package ethash

import (
	"container/list"
	"encoding/json"
	"errors"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/ethdb"
	"strings"
	"fmt"
)

const (
	windowRatio = 2 //
)

type Snapshot struct {
	PermissionSigners map[common.Address]struct{}

	Number              uint64
	Hash                common.Hash
	UsedSigners         map[common.Address]struct{}
	RecentSignersWindow *list.List
}

type plainSnapShot struct {
	PermissionSigners map[common.Address]struct{} `json:"permissionSigners"`

	Number              uint64                      `json:"number"`
	Hash                common.Hash                 `json:"hash"`
	UsedSigners         map[common.Address]struct{} `json:"usedSigners"`
	RecentSignersWindow []common.Address            `json:"recentSignersWindow"`
}

func newSnapshot(number uint64, hash common.Hash, signers []common.Address) *Snapshot {
	snap := &Snapshot{
		PermissionSigners:   make(map[common.Address]struct{}),
		Number:              number,
		Hash:                hash,
		UsedSigners:         make(map[common.Address]struct{}),
		RecentSignersWindow: list.New(),
	}

	for _, s := range signers {
		snap.PermissionSigners[s] = struct{}{}
	}

	return snap
}

func loadSnapShot(db ethdb.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("ppow-"), hash[:]...))
	if err != nil {
		return nil, err
	}

	plain := new(plainSnapShot)
	if err := json.Unmarshal(blob, plain); err != nil {
		return nil, err
	}

	snap := &Snapshot{
		PermissionSigners:   plain.PermissionSigners,
		Number:              plain.Number,
		Hash:                plain.Hash,
		UsedSigners:         plain.UsedSigners,
		RecentSignersWindow: list.New(),
	}

	for _, v := range plain.RecentSignersWindow {
		snap.RecentSignersWindow.PushBack(v)
	}

	return snap, nil
}

func (s *Snapshot) store(db ethdb.Database) error {
	plain := &plainSnapShot{
		PermissionSigners:   s.PermissionSigners,
		Number:              s.Number,
		Hash:                s.Hash,
		UsedSigners:         s.UsedSigners,
		RecentSignersWindow: make([]common.Address, 0),
	}

	for e := s.RecentSignersWindow.Front(); e != nil; e = e.Next() {
		if _, ok := e.Value.(common.Address); ok {
			plain.RecentSignersWindow = append(plain.RecentSignersWindow, e.Value.(common.Address))
		}
	}

	blob, err := json.Marshal(plain)
	if err != nil {
		return err
	}

	return db.Put(append([]byte("ppow-"), plain.Hash[:]...), blob)
}

func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		PermissionSigners:   s.PermissionSigners,
		Number:              s.Number,
		Hash:                s.Hash,
		UsedSigners:         make(map[common.Address]struct{}),
		RecentSignersWindow: list.New(),
	}

	for signer := range s.UsedSigners {
		cpy.UsedSigners[signer] = struct{}{}
	}
	cpy.RecentSignersWindow.PushBackList(s.RecentSignersWindow)
	return cpy
}

func (s *Snapshot) updateSignerStatus(signer common.Address, isExist bool) {
	if isExist {
		//if signer already presence
		if (s.RecentSignersWindow.Len()) > 0 {
			s.RecentSignersWindow.PushFront(signer)
			s.RecentSignersWindow.Remove(s.RecentSignersWindow.Back())
		}
	} else {
		// This is the first time the signer appear
		s.UsedSigners[signer] = struct{}{}
		preWindowLen := s.RecentSignersWindow.Len()
		newWindowLen := (len(s.UsedSigners) - 1) / windowRatio
		if newWindowLen > preWindowLen {
			s.RecentSignersWindow.PushFront(signer)
		} else {
			//windowLen unchanged
			if newWindowLen > 0 {
				s.RecentSignersWindow.PushFront(signer)
				s.RecentSignersWindow.Remove(s.RecentSignersWindow.Back())
			}
		}
	}
}

// apply creates a new authorization snapshot by applying the given headers to
// the original one.
// PermissionSigners is the full set of signers who can sign blocks
// UsedSigners is the set of signers who had sign blocks
// RecentSignersWindow is the set who can not sign next block
// len(RecentSignersWindow) = (len(UsedSigners)-1)/2
// so when n > 2, hacker should got (n / 2 + 1) key to reorg chain?
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errors.New("invalid applied headers")
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errors.New("invalid applied header to snapshot")
	}

	snap := s.copy()

	for _, header := range headers {
		signer, err := ecrecover(header)
		if err != nil || 0 != strings.Compare(signer.String(), header.Coinbase.String()) {
			return nil, err
		}

		err = snap.isLegal4Sign(signer)
		if err != nil {
			return nil, err
		}

		_, ok := snap.UsedSigners[signer]
		snap.updateSignerStatus(signer, ok)
	}

	snap.Hash = headers[len(headers)-1].Hash()
	snap.Number = headers[len(headers)-1].Number.Uint64()

	return snap, nil
}

func (s *Snapshot) isLegal4Sign(signer common.Address) error {
	if _, ok := s.PermissionSigners[signer]; !ok {
		fmt.Println(common.ToHex(signer[:]))
		return errUnauthorized
	}

	for e := s.RecentSignersWindow.Front(); e != nil; e = e.Next() {
		if _, ok := e.Value.(common.Address); ok {
			wSigner := e.Value.(common.Address)
			if signer == wSigner {
				return errAuthorTooOften
			}
		}
	}
	return nil
}
