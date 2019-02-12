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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pborman/uuid"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/randentropy"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

type keyStorePassphrase struct {
	keysDirPath string
	scryptN     int
	scryptP     int
}

var (
	ErrWAddressFieldNotExist = errors.New("It seems that this account doesn't include a valid wanchain address field, please update your keyfile version")
	ErrWAddressInvalid       = errors.New("invalid wanchain address")
	ErrInvalidAccountKey     = errors.New("invalid account key")
	ErrInvalidPrivateKey     = errors.New("invalid private key")
)

func (ks keyStorePassphrase) GetKey(addr common.Address, filename, auth string) (*Key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
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

func (ks keyStorePassphrase) StoreKey(filename string, key *Key, auth string) error {
	keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	return writeKeyFile(filename, keyjson)
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

func (ks keyStorePassphrase) JoinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	} else {
		return filepath.Join(ks.keysDirPath, filename)
	}
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func EncryptKey(key *Key, auth string, scryptN, scryptP int) ([]byte, error) {
	if key == nil {
		return nil, ErrInvalidAccountKey
	}

	cryptoStruct, err := EncryptOnePrivateKey(key.PrivateKey, auth, scryptN, scryptP)
	if err != nil {
		return nil, err
	}

	cryptoStruct2, err := EncryptOnePrivateKey(key.PrivateKey2, auth, scryptN, scryptP)
	if err != nil {
		return nil, err
	}

	encryptedKeyJSONV3 := encryptedKeyJSONV3{
		key.Address.Hex()[2:],
		*cryptoStruct,
		*cryptoStruct2,
		key.Id.String(),
		version,
		hex.EncodeToString(key.WAddress[:]),
	}
	return json.Marshal(encryptedKeyJSONV3)
}

// EncryptOnePrivateKey encrypts a key using the specified scrypt parameters into one field of a json
// blob that can be decrypted later on.
func EncryptOnePrivateKey(privateKey *ecdsa.PrivateKey, auth string, scryptN, scryptP int) (*cryptoJSON, error) {
	if privateKey == nil {
		return nil, ErrInvalidPrivateKey
	}

	authArray := []byte(auth)
	salt := randentropy.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}

	encryptKey := derivedKey[:16]
	keyBytes := math.PaddedBigBytes(privateKey.D, 32)

	iv := randentropy.GetEntropyCSPRNG(aes.BlockSize) // 16

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

	cryptoStruct := &cryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}

	return cryptoStruct, nil

}

// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptKey(keyjson []byte, auth string) (*Key, error) {
	// Parse the json into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(keyjson, &m); err != nil {
		return nil, err
	}
	// Depending on the version try to parse one way or another
	var (
		keyBytes, keyBytes2, keyId []byte
		err                        error
		waddressStr                *string
	)
	if version, ok := m["version"].(string); ok && version == "1" {
		k := new(encryptedKeyJSONV1)
		if err := json.Unmarshal(keyjson, k); err != nil {
			return nil, err
		}

		keyBytes, keyId, err = decryptKeyV1(k, auth)
		key, err := crypto.ToECDSA(keyBytes)
		if err != nil || key == nil {
			return nil, err
		}

		return &Key{
			Id:         uuid.UUID(keyId),
			Address:    crypto.PubkeyToAddress(key.PublicKey),
			PrivateKey: key,
		}, nil
	} else {
		k := new(encryptedKeyJSONV3)
		if err := json.Unmarshal(keyjson, k); err != nil {
			return nil, err
		}
		keyBytes, keyBytes2, keyId, err = decryptKeyV3(k, auth)
		if err != nil {
			return nil, err
		}

		waddressStr = &k.WAddress
	}

	key, err := crypto.ToECDSA(keyBytes)
	if err != nil || key == nil {
		return nil, ErrInvalidPrivateKey
	}

	key2, err := crypto.ToECDSA(keyBytes2)
	if err != nil || key2 == nil {
		return nil, ErrInvalidPrivateKey
	}

	waddressRaw, err := hex.DecodeString(*waddressStr)
	if err != nil {
		return nil, err
	}

	var waddress common.WAddress
	copy(waddress[:], waddressRaw)

	return &Key{
		Id:          uuid.UUID(keyId),
		Address:     crypto.PubkeyToAddress(key.PublicKey),
		PrivateKey:  key,
		PrivateKey2: key2,
		WAddress:    waddress,
	}, nil
}

func decryptKeyV3(keyProtected *encryptedKeyJSONV3, auth string) (keyBytes []byte, keyBytes2 []byte, keyId []byte, err error) {
	if keyProtected.Version != version {
		return nil, nil, nil, fmt.Errorf("Version not supported: %v", keyProtected.Version)
	}

	keyId = uuid.Parse(keyProtected.Id)

	plainText, err := decryptKeyV3Item(keyProtected.Crypto, auth)
	if err != nil {
		return nil, nil, nil, err
	}

	plainText2, err2 := decryptKeyV3Item(keyProtected.Crypto2, auth)
	if err2 != nil {
		if "" == keyProtected.Crypto2.Cipher {
			plainText2 = make([]byte, 0)
		} else {
			return nil, nil, nil, err2
		}
	}

	return plainText, plainText2, keyId, err
}

func decryptKeyV3Item(cryptoItem cryptoJSON, auth string) (keyBytes []byte, err error) {
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

func decryptKeyV1(keyProtected *encryptedKeyJSONV1, auth string) (keyBytes []byte, keyId []byte, err error) {
	keyId = uuid.Parse(keyProtected.Id)
	mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
	if err != nil {
		return nil, nil, err
	}

	iv, err := hex.DecodeString(keyProtected.Crypto.CipherParams.IV)
	if err != nil {
		return nil, nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return nil, nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.Crypto, auth)
	if err != nil {
		return nil, nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, nil, ErrDecrypt
	}

	plainText, err := aesCBCDecrypt(crypto.Keccak256(derivedKey[:16])[:16], cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyId, err
}

func getKDFKey(cryptoJSON cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])

	if cryptoJSON.KDF == keyHeaderKDF {
		n := ensureInt(cryptoJSON.KDFParams["n"])
		r := ensureInt(cryptoJSON.KDFParams["r"])
		p := ensureInt(cryptoJSON.KDFParams["p"])
		return scrypt.Key(authArray, salt, n, r, p, dkLen)

	} else if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

// TODO: can we do without this when unmarshalling dynamic JSON?
// why do integers in KDF params end up as float64 and not int after
// unmarshal?
func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}
