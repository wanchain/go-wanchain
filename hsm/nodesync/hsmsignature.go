package nodesync

import (
	"sync"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/accounts"
)

var (
	mu sync.RWMutex
)

func SignHash(acc accounts.Account, hash []byte) ([]byte, error) {
	// Look up the key to sign with and abort if it cannot be found
	mu.RLock()
	defer mu.RUnlock()

	// Sign the hash using plain ECDSA operations
	return crypto.Sign(hash,NodeSignKey)
}
