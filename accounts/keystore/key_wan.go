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

package keystore

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/btcsuite/btcd/btcec"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/google/uuid"
)

var (
	ErrWAddressInvalid = errors.New("invalid wanchain address")
)

type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
	// add a second privkey for privary
	PrivateKey2 *ecdsa.PrivateKey
	// compact wanchain address format
	WAddress common.WAddress
}

type keyStore interface {
	// Loads and decrypts the key from disk.
	GetKey(addr common.Address, filename string, auth string) (*Key, error)
	// Writes and encrypts the key.
	StoreKey(filename string, k *Key, auth string) error
	// Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string

	// add by Jacob begin
	// Decrypts the key from keyjson
	GetKeyFromKeyJson(addr common.Address, keyjson []byte, auth string) (*Key, error)
	// Loads an encrypted keyfile from disk
	GetEncryptedKey(addr common.Address, filename string) (*Key, error)
	// add by Jacob end
}

type encryptedKeyJSONV3 struct {
	Address  string     `json:"address"`
	Crypto   CryptoJSON `json:"crypto"`
	Crypto2  CryptoJSON `json:"crypto2"` // add by Jacob
	Id       string     `json:"id"`
	Version  int        `json:"version"`
	WAddress string     `json:"waddress"` // add by Jacob

}

// Used to import and export raw keypair
type keyPair struct {
	D  string `json:"privateKey"`
	D1 string `json:"privateKey1"`
}

type AwsKmsInfo struct {
	AKID      string
	SecretKey string
	Region    string
}

func newKeyFromECDSA(sk1, sk2 *ecdsa.PrivateKey) *Key {
	id, _ := uuid.NewRandom()
	key := &Key{
		Id:          id,
		Address:     crypto.PubkeyToAddress(sk1.PublicKey),
		PrivateKey:  sk1,
		PrivateKey2: sk2,
	}

	updateWaddress(key)
	return key
}

// updateWaddress adds WAddress field to the Key struct
func updateWaddress(k *Key) {
	k.WAddress = *GenerateWaddressFromPK(&k.PrivateKey.PublicKey, &k.PrivateKey2.PublicKey)
}

// ECDSAPKCompression serializes a public key in a 33-byte compressed format from btcec
func ECDSAPKCompression(p *ecdsa.PublicKey) []byte {
	const pubkeyCompressed byte = 0x2
	b := make([]byte, 0, 33)
	format := pubkeyCompressed
	if p.Y.Bit(0) == 1 {
		format |= 0x1
	}
	b = append(b, format)
	b = append(b, math.PaddedBigBytes(p.X, 32)...)
	return b
}

// NewKeyForDirectICAP generates a key whose address fits into < 155 bits so it can fit
// into the Direct ICAP spec. for simplicity and easier compatibility with other libs, we
// retry until the first byte is 0.
func NewKeyForDirectICAP(rand io.Reader) *Key {
	randBytes := make([]byte, 64*2)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	reader := bytes.NewReader(randBytes)
	sk1, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}

	sk2, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	key := newKeyFromECDSA(sk1, sk2)
	if !strings.HasPrefix(key.Address.Hex(), "0x00") {
		return NewKeyForDirectICAP(rand)
	}
	return key
}

func newKey(rand io.Reader) (*Key, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}

	privateKeyECDSA2, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}
	return newKeyFromECDSA(privateKeyECDSA, privateKeyECDSA2), nil
}

// GeneratePKPairFromWAddress represents the keystore to retrieve public key-pair from given WAddress
func GeneratePKPairFromWAddress(w []byte) (*ecdsa.PublicKey, *ecdsa.PublicKey, error) {
	if len(w) != common.WAddressLength {
		return nil, nil, ErrWAddressInvalid
	}

	tmp := make([]byte, 33)
	copy(tmp[:], w[:33])
	curve := btcec.S256()
	PK1, err := btcec.ParsePubKey(tmp, curve)
	if err != nil {
		return nil, nil, err
	}

	copy(tmp[:], w[33:])
	PK2, err := btcec.ParsePubKey(tmp, curve)
	if err != nil {
		return nil, nil, err
	}

	return (*ecdsa.PublicKey)(PK1), (*ecdsa.PublicKey)(PK2), nil
}

func GenerateWaddressFromPK(A *ecdsa.PublicKey, B *ecdsa.PublicKey) *common.WAddress {
	var tmp common.WAddress
	copy(tmp[:33], ECDSAPKCompression(A))
	copy(tmp[33:], ECDSAPKCompression(B))
	return &tmp
}

func WaddrFromUncompressedRawBytes(raw []byte) (*common.WAddress, error) {
	if len(raw) != 32*2*2 {
		return nil, errors.New("invalid uncompressed wan address len")
	}

	pub := make([]byte, 65)
	pub[0] = 0x004
	copy(pub[1:], raw[:64])
	A := crypto.ToECDSAPub(pub)
	copy(pub[1:], raw[64:])
	B := crypto.ToECDSAPub(pub)
	return GenerateWaddressFromPK(A, B), nil
}

func WaddrToUncompressedRawBytes(waddr []byte) ([]byte, error) {
	if len(waddr) != common.WAddressLength {
		return nil, ErrWAddressInvalid
	}

	A, B, err := GeneratePKPairFromWAddress(waddr)
	if err != nil {
		return nil, err
	}

	u := make([]byte, 32*2*2)
	ax := math.PaddedBigBytes(A.X, 32)
	ay := math.PaddedBigBytes(A.Y, 32)
	bx := math.PaddedBigBytes(B.X, 32)
	by := math.PaddedBigBytes(B.Y, 32)
	copy(u[0:], ax[:32])
	copy(u[32:], ay[:32])
	copy(u[64:], bx[:32])
	copy(u[96:], by[:32])

	return u, nil
}

// LoadECDSAPair loads a secp256k1 private key pair from the given file
func LoadECDSAPair(file string) (*ecdsa.PrivateKey, *ecdsa.PrivateKey, error) {
	// read the given file including private key pair
	kp := keyPair{}

	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}

	err = json.Unmarshal(raw, &kp)
	if err != nil {
		return nil, nil, err
	}

	// Decode the key pair
	d, err := hex.DecodeString(kp.D)
	if err != nil {
		return nil, nil, err
	}
	d1, err := hex.DecodeString(kp.D1)
	if err != nil {
		return nil, nil, err
	}

	// Generate ecdsa private keys
	sk, err := crypto.ToECDSA(d)
	if err != nil {
		return nil, nil, err
	}

	sk1, err := crypto.ToECDSA(d1)
	if err != nil {
		return nil, nil, err
	}

	return sk, sk1, err
}

// ExportECDSAPair returns an ecdsa-private-key pair
// func ExportECDSAPair(d, d1, fp string) error {
// 	kp := keyPair{
// 		D:  d,
// 		D1: d1,
// 	}
// 	log.Info("Exporting ECDSA Prikave-Key-Pair", "file", fp)
// 	fh, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
// 	if err != nil {
// 		return err
// 	}
// 	defer fh.Close()

// 	var fileWriter io.Writer = fh
// 	err = json.NewEncoder(fileWriter).Encode(kp)
// 	return err
// }

// func ExportECDSAPairStr(d, d1 string) (string, error) {
// 	kp := keyPair{
// 		D:  d,
// 		D1: d1,
// 	}
// 	r, err := json.Marshal(kp)
// 	if err != nil {
// 		return "", err
// 	}

// 	return string(r), err
// }
