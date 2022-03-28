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

/*

This key store behaves as KeyStorePlain with the difference that
the private key is encrypted and on disk uses another JSON encoding.

The crypto is documented at https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition

*/

package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/scrypt"
	"io"
	"io/ioutil"
)

var (
	ErrWAddressFieldNotExist = errors.New("It seems that this account doesn't include a valid wanchain address field, please update your keyfile version")
	ErrInvalidAccountKey     = errors.New("invalid account key")
	ErrInvalidPrivateKey     = errors.New("invalid private key")
)

func (ks keyStorePassphrase) GetKeyFromKeyJson(addr common.Address, keyjson []byte, auth string) (*Key, error) {
	if len(keyjson) == 0 {
		return nil, fmt.Errorf("key content invalid")
	}

	key, err := DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

// Implements GetEncryptedKey method of keystore interface
func (ks keyStorePassphrase) GetEncryptedKey(a common.Address, filename string) (*Key, error) {
	// load the encrypted json keyfile
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := GenerateKeyWithWAddress(keyjson)
	if err != nil {
		return nil, err
	}
	key.Address = a
	return key, nil
}

// Generate a Key initialized with WAddress field
func GenerateKeyWithWAddress(keyjson []byte) (*Key, error) {
	// parse the json blob into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(keyjson, &m); err != nil {
		return nil, err
	}

	waddress, ok := m["waddress"].(string)

	if !ok || waddress == "" {
		return nil, ErrWAddressFieldNotExist
	}

	waddressRaw, err := hex.DecodeString(waddress)
	if err != nil {
		return nil, err
	}

	if len(waddressRaw) != common.WAddressLength {
		return nil, ErrWAddressInvalid
	}

	key := new(Key)
	copy(key.WAddress[:], waddressRaw)
	return key, nil
}

// EncryptOnePrivateKey encrypts a key using the specified scrypt parameters into one field of a json
// blob that can be decrypted later on.
func EncryptOnePrivateKey(privateKey *ecdsa.PrivateKey, auth string, scryptN, scryptP int) (*CryptoJSON, error) {
	if privateKey == nil {
		return nil, ErrInvalidPrivateKey
	}

	authArray := []byte(auth)

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}

	encryptKey := derivedKey[:16]
	keyBytes := math.PaddedBigBytes(privateKey.D, 32)

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}

	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := &CryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}

	return cryptoStruct, nil

}

func decryptKeyV3Item(cryptoItem CryptoJSON, auth string) (keyBytes []byte, err error) {
	if cryptoItem.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("Cipher not supported: %v", cryptoItem.Cipher)
	}

	mac, err := hex.DecodeString(cryptoItem.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(cryptoItem.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(cryptoItem.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(cryptoItem, auth)
	if err != nil {
		return nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrDecrypt
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}

	return plainText, err
}
