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

// Package keystore implements encrypted storage of secp256k1 private keys.
//
// Keys are stored as encrypted JSON files according to the Web3 Secret Storage specification.
// See https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition for more information.
package keystore

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"io/ioutil"
	"time"
)

func (ks *KeyStore) ImportECDSA(priv1, priv2 *ecdsa.PrivateKey, passphrase string) (accounts.Account, error) {
	ks.importMu.Lock()
	defer ks.importMu.Unlock()

	key := newKeyFromECDSA(priv1, priv2)
	if ks.cache.hasAddress(key.Address) {
		return accounts.Account{
			Address: key.Address,
		}, ErrAccountAlreadyExists
	}
	return ks.importKey(key, passphrase)
}

// Update changes the passphrase of an existing account.
func (ks *KeyStore) Update(a accounts.Account, passphrase, newPassphrase string) error {
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}
	if key.PrivateKey2 == nil {
		sk2, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		key.PrivateKey2 = sk2
	}
	updateWaddress(key)
	return ks.storage.StoreKey(a.URL.Path, key, newPassphrase)
}

func (ks *KeyStore) ComputeOTAPPKeys(a accounts.Account, AX, AY, BX, BY string) ([]string, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}

	pub1, priv1, priv2, err := crypto.GenerteOTAPrivateKey(unlockedKey.PrivateKey, unlockedKey.PrivateKey2, AX, AY, BX, BY)

	pub1X := hexutil.Encode(common.LeftPadBytes(pub1.X.Bytes(), 32))
	pub1Y := hexutil.Encode(common.LeftPadBytes(pub1.Y.Bytes(), 32))
	priv1D := hexutil.Encode(common.LeftPadBytes(priv1.D.Bytes(), 32))
	priv2D := hexutil.Encode(common.LeftPadBytes(priv2.D.Bytes(), 32))

	return []string{pub1X, pub1Y, priv1D, priv2D}, err
}

func (ks *KeyStore) UnlockMemKey(a accounts.Account, keyjson []byte, passphrase string) error {
	return ks.TimedUnlockMemKey(a, keyjson, passphrase, 0)
}

func (ks *KeyStore) TimedUnlockMemKey(a accounts.Account, keyjson []byte, passphrase string, timeout time.Duration) error {
	a, key, err := ks.getDecryptedKeyFromMemJson(a, keyjson, passphrase)
	if err != nil {
		return err
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()
	u, found := ks.unlocked[a.Address]
	if found {
		if u.abort == nil {
			// The address was unlocked indefinitely, so unlocking
			// it with a timeout would be confusing.
			zeroKey(key.PrivateKey)
			return nil
		}
		// Terminate the expire goroutine and replace it below.
		close(u.abort)
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go ks.expire(a.Address, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	ks.unlocked[a.Address] = u
	return nil
}

func (ks *KeyStore) getDecryptedKeyFromMemJson(a accounts.Account, keyjson []byte, auth string) (accounts.Account, *Key, error) {
	key, err := ks.storage.GetKeyFromKeyJson(a.Address, keyjson, auth)
	return a, key, err
}

// getEncryptedKey loads an encrypted keyfile from the disk
func (ks *KeyStore) getEncryptedKey(a accounts.Account) (accounts.Account, *Key, error) {
	a, err := ks.Find(a)
	if err != nil {
		return a, nil, err
	}
	key, err := ks.storage.GetEncryptedKey(a.Address, a.URL.Path)
	if err != nil {
		return a, nil, err
	}
	return a, key, nil

}

// TODO: temp add, for quickly print public keys, maybe removed later
func (ks *KeyStore) GetKey(a accounts.Account, passphrase string) (*Key, error) {
	keyJSON, err := ioutil.ReadFile(a.URL.Path)
	if err != nil {
		return nil, err
	}
	key, err := DecryptKey(keyJSON, passphrase)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// GetWanAddress represents the keystore to retrieve corresponding wanchain public address for a specific ordinary account/address
func (ks *KeyStore) GetWanAddress(account accounts.Account) (common.WAddress, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[account.Address]
	if !found {
		_, ksen, err := ks.getEncryptedKey(account)
		if err != nil {
			return common.WAddress{}, ErrLocked
		}
		return ksen.WAddress, nil
	}

	ret := unlockedKey.WAddress
	return ret, nil
}

func (ks keyStorePlain) GetEncryptedKey(a common.Address, filename string) (*Key, error) {
	return nil, nil
}

func (ks keyStorePlain) GetKeyFromKeyJson(addr common.Address, keyjson []byte, auth string) (*Key, error) {
	key := new(Key)
	if err := json.Unmarshal(keyjson, key); err != nil {
		return nil, err
	}
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have address %x, want %x", key.Address, addr)
	}
	return key, nil
}

func (w *keystoreWallet) GetUnlockedKey(address common.Address) (*Key, error) {
	value, ok := w.keystore.unlocked[address]
	if !ok {
		return nil, errors.New("can not found a unlock key of: " + address.Hex())
	}

	return value.Key, nil
}
