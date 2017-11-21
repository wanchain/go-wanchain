// INFO: copied from consensus/clique/snapshot.go , but heavily modified

package ethash

import (
	"container/list"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/common"
	"encoding/json"
)

const (
	windowRatio = 2  //
)

type Snapshot struct {
	PermissionSigners map[common.Address]struct{}

	Number uint64
	Hash   common.Hash
	UsedSigners map[common.Address]struct{}
	RecentSignersWindow *list.List
}

type plainSnapShot struct {
	PermissionSigners map[common.Address]struct{}  `json:"permissionSigners"`

	Number uint64                                  `json:"number"`
	Hash   common.Hash                             `json:"hash"`
	UsedSigners map[common.Address]struct{}        `json:"usedSigners"`
	RecentSignersWindow []common.Address           `json:"recentSignersWindow"`
}

func NewSnapshot(number uint64, hash common.Hash, signers []common.Address) *Snapshot {
	snap := &Snapshot{
		PermissionSigners: make(map[common.Address]struct{}),
		Number: number,
		Hash: hash,
		UsedSigners: make(map[common.Address]struct{}),
		RecentSignersWindow: list.New(),
	}

	for _, s := range signers{
		snap.PermissionSigners[s] = struct{}{}
	}
	return snap
}


func LoadSnapShot(db ethdb.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("ppow-"), hash[:]...))
	if err != nil {
		return nil, err
	}

	plain := new(plainSnapShot)
	if err := json.Unmarshal(blob, plain); err != nil {
		return nil, err
	}

	snap := &Snapshot{
		PermissionSigners: plain.PermissionSigners,
		Number: plain.Number,
		Hash: plain.Hash,
		UsedSigners: plain.UsedSigners,
		RecentSignersWindow: list.New(),
	}

	for _, v := range plain.RecentSignersWindow {
		snap.RecentSignersWindow.PushBack(v)
	}

	return snap, nil
}

func(s *Snapshot) store(db ethdb.Database) error {
	plain := &plainSnapShot{
		PermissionSigners: s.PermissionSigners,
		Number: s.Number,
		Hash: s.Hash,
		UsedSigners: s.UsedSigners,
		RecentSignersWindow: make([]common.Address, s.RecentSignersWindow.Len()),
	}

	for e := s.RecentSignersWindow.Front(); e!= nil; e = e.Next() {
		if _, ok := e.Value.(common.Address); ok{
			plain.RecentSignersWindow = append(plain.RecentSignersWindow, e.Value.(common.Address))
		}
	}

	blob, err := json.Marshal(plain)
	if err != nil {
		return err
	}

	return db.Put(append([]byte("ppow-"), plain.Hash[:]...), blob)
}

func(s* Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		PermissionSigners:  s.PermissionSigners,
		Number:    s.Number,
		Hash: s.Hash,
		UsedSigners: make(map[common.Address]struct{}),
		RecentSignersWindow: list.New(),
	}

	for signer := range s.UsedSigners {
		cpy.UsedSigners[signer] = struct{}{}
	}
	cpy.RecentSignersWindow.PushBackList(s.RecentSignersWindow)
	return cpy
}

func (self *Snapshot)updateSignerStatus(signer common.Address, isExist bool){
	if isExist {
		//if signer already presence
		if(self.RecentSignersWindow.Len()) > 0 {
			self.RecentSignersWindow.PushFront(signer)
			self.RecentSignersWindow.Remove(self.RecentSignersWindow.Back())
		}
	} else {
		// This is the first time the signer appear
		self.UsedSigners[signer] = struct{}{}
		preWindowLen := self.RecentSignersWindow.Len()
		newWindowLen := (len(self.UsedSigners)-1) / windowRatio
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
func(s *Snapshot) apply(header *types.Header) (*Snapshot, error){
	snap := s.copy()

	signer, err := ecrecover(header)
	if err != nil {
		return nil, err
	}

	if _, ok := snap.PermissionSigners[signer]; !ok {
		return nil, errUnauthorized
	}

	for e := snap.RecentSignersWindow.Front(); e!= nil; e = e.Next() {
		if _, ok := e.Value.(common.Address); ok{
			wSigner := e.Value.(common.Address)
			if signer == wSigner {
				return nil, errAuthorTooOften
			}
		}
	}

	_, ok := snap.UsedSigners[signer]
	snap.updateSignerStatus(signer, ok)

	return snap, nil
}

func(s *Snapshot) isLegal4Sign(signer common.Address) bool {
	if _, ok := s.PermissionSigners[signer]; !ok {
		return false
	}
	for e := s.RecentSignersWindow.Front(); e != nil; e = e.Next() {
		if _, ok := e.Value.(common.Address); ok{
			wSigner := e.Value.(common.Address)
			if signer == wSigner {
				return false
			}
		}
	}
	return true
}
