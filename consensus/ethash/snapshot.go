package ethash

import (
	"bytes"
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/functrace"
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

func NewSnapshot(number uint64, hash common.Hash, signers []common.Address) *Snapshot {
	functrace.Enter(fmt.Sprintf("number= %d", number))
	defer functrace.Exit()

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

func LoadSnapShot(db ethdb.Database, hash common.Hash) (*Snapshot, error) {
	functrace.Enter(hash.String())
	defer functrace.Exit()

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
	functrace.Enter()
	defer functrace.Exit()

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
	functrace.Enter()
	defer functrace.Exit()

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

func (self *Snapshot) updateSignerStatus(signer common.Address, isExist bool) {
	functrace.Enter(signer.String())
	defer functrace.Exit()

	empty, _ := hexutil.Decode("0x0000000000000000000000000000000000000000")
	if bytes.Equal(empty, signer[:]) {
		functrace.Enter("dummy")
		debug.PrintStack()
		defer functrace.Exit()
	}
	if isExist {
		//if signer already presence
		if (self.RecentSignersWindow.Len()) > 0 {
			self.RecentSignersWindow.PushFront(signer)
			self.RecentSignersWindow.Remove(self.RecentSignersWindow.Back())
		}
	} else {
		// This is the first time the signer appear
		self.UsedSigners[signer] = struct{}{}
		preWindowLen := self.RecentSignersWindow.Len()
		newWindowLen := (len(self.UsedSigners) - 1) / windowRatio
		if newWindowLen > preWindowLen {
			self.RecentSignersWindow.PushFront(signer)
		} else {
			//windowLen unchanged
			if newWindowLen > 0 {
				self.RecentSignersWindow.PushFront(signer)
				self.RecentSignersWindow.Remove(self.RecentSignersWindow.Back())
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
	functrace.Enter()
	defer functrace.Exit()

	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errors.New("invaild---")
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errors.New("invaild---")
	}

	snap := s.copy()

	for _, header := range headers {
		signer, err := ecrecover(header)
		if err != nil {
			return nil, err
		}

		if _, ok := snap.PermissionSigners[signer]; !ok {
			return nil, errUnauthorized
		}

		for e := snap.RecentSignersWindow.Front(); e != nil; e = e.Next() {
			if _, ok := e.Value.(common.Address); ok {
				wSigner := e.Value.(common.Address)
				if signer == wSigner {
					return nil, errAuthorTooOften
				}
			}
		}

		_, ok := snap.UsedSigners[signer]
		snap.updateSignerStatus(signer, ok)
	}

	snap.Hash = headers[len(headers)-1].Hash()
	snap.Number = headers[len(headers)-1].Number.Uint64()

	return snap, nil
}

func (s *Snapshot) isLegal4Sign(signer common.Address) error {
	functrace.Enter(signer.String())
	defer functrace.Exit()

	if _, ok := s.PermissionSigners[signer]; !ok {
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
