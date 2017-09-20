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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/pborman/uuid"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"math/big"
	"errors"
)

const (
	version = 3
)
//r@zy: 修改为新的结构
type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
	PrivateKey2  *ecdsa.PrivateKey

	// lzh add (AX,AY,Bx,BY,checksum)
	WAddress common.WAddress
}

type keyStore interface {
	// Loads and decrypts the key from disk.
	GetKey(addr common.Address, filename string, auth string) (*Key, error)
	// Loads the encrypt key from disk
    GetKeyEncrypt(addr common.Address, filename string) (*Key, error)
	// Writes and encrypts the key.
	StoreKey(filename string, k *Key, auth string) error
	// Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string
}

type plainKeyJSON struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privatekey"`
	Id         string `json:"id"`
	Version    int    `json:"version"`
}

type encryptedKeyJSONV3 struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Crypto2 cryptoJSON `json:"crypto2"`
	Id      string     `json:"id"`
	Version int        `json:"version"`
	WAddress string    `json:"waddress"`
}

type encryptedKeyJSONV1 struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Version string     `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

type scryptParamsJSON struct {
	N     int    `json:"n"`
	R     int    `json:"r"`
	P     int    `json:"p"`
	DkLen int    `json:"dklen"`
	Salt  string `json:"salt"`
}

func (k *Key) MarshalJSON() (j []byte, err error) {
	jStruct := plainKeyJSON{
		hex.EncodeToString(k.Address[:]),
		hex.EncodeToString(crypto.FromECDSA(k.PrivateKey)),
		k.Id.String(),
		version,
	}
	j, err = json.Marshal(jStruct)
	return j, err
}

func (k *Key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(plainKeyJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return err
	}

	u := new(uuid.UUID)
	*u = uuid.Parse(keyJSON.Id)
	k.Id = *u
	addr, err := hex.DecodeString(keyJSON.Address)
	if err != nil {
		return err
	}

	privkey, err := hex.DecodeString(keyJSON.PrivateKey)
	if err != nil {
		return err
	}

	k.Address = common.BytesToAddress(addr)
	k.PrivateKey = crypto.ToECDSA(privkey)

	return nil
}

// lzh add
func CheckSum16(data []byte) uint16 {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)

	for length > 1 {
		sum += (uint32(data[index])<<8) + (uint32(data[index+1]))
		index += 2
		length -= 2
	}

	if length > 0 {
		sum += uint32(data[index])
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum += (sum >> 16)

	return uint16(^sum)
}

func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey, privateKeyECDSA2 *ecdsa.PrivateKey) *Key {
	id := uuid.NewRandom()
	key := &Key{
		Id:         id,
		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
		PrivateKey2: privateKeyECDSA2,
	}

	// lzh add
	if err := updateWaddress(key); err != nil {
		return nil
	}

	return key
}

// lzh add
func updateWaddress(k * Key) error  {
	publicBytes := [4][]byte {
		k.PrivateKey.PublicKey.X.Bytes(),
		k.PrivateKey.PublicKey.Y.Bytes(),
		k.PrivateKey2.PublicKey.X.Bytes(),
		k.PrivateKey2.PublicKey.Y.Bytes(),
	}

	offset := 0
	for i := 0; i < len(publicBytes); i++ {
		copy(k.WAddress[offset:], publicBytes[i])
		offset += len(publicBytes[i])

		if offset >= common.WAddressLength {
			return errors.New("update waddress fail! invalid public key len! over waddress len WAddressLength!")
		}
	}

	if offset != common.WAddressLength - 2 {
		return errors.New("update waddress fail! invalid public key len! public key total len is not WAddressLength-2!")
	}

	sum := CheckSum16(k.WAddress[0:offset])
	k.WAddress[offset] = uint8(sum >> 8)
	k.WAddress[offset+1] = uint8(sum & 0xff)
	return nil
}

// lzh add
func checkWaddressValid(k * Key) bool {

	var tmpWaddress common.WAddress

	publicBytes := [4][]byte {
		k.PrivateKey.PublicKey.X.Bytes(),
		k.PrivateKey.PublicKey.Y.Bytes(),
		k.PrivateKey2.PublicKey.X.Bytes(),
		k.PrivateKey2.PublicKey.Y.Bytes(),
	}

	offset := 0
	for i := 0; i < len(publicBytes); i++ {
		copy(tmpWaddress[offset:], publicBytes[i])
		offset += len(publicBytes[i])

		if offset >= common.WAddressLength {
			return false
		}
	}

	if offset != common.WAddressLength - 2 {
		return false
	}

	sum := CheckSum16(tmpWaddress[0:offset])
	tmpWaddress[offset] = uint8(sum >> 8)
	tmpWaddress[offset+1] = uint8(sum & 0xff)

	return k.WAddress == tmpWaddress
}

// lzh add
func (k *Key)GetTwoPublicKeyRawStrs() ([]string, error) {
	if CheckSum16(k.WAddress[:]) != 0 {
		return nil, errors.New("invalid waddress! check sum is not zero!")
	}

	ret := hexutil.FourBigIntToHexSlice(k.WAddress[0:32], k.WAddress[32:64], k.WAddress[64:96], k.WAddress[96:128])
	return ret, nil
}

// lzh add
func (k *Key)GetTwoPublicKey() (*ecdsa.PublicKey, *ecdsa.PublicKey, error)  {
	if k.PrivateKey != nil && k.PrivateKey2 != nil {
		return &k.PrivateKey.PublicKey, &k.PrivateKey2.PublicKey, nil
	}

	if CheckSum16(k.WAddress[:]) != 0 {
		return nil, nil, errors.New("invalid waddress! check sum is not zero!")
	}

	pk1 := new(ecdsa.PublicKey)
	pk2 := new(ecdsa.PublicKey)

	initPublicKeyFromWaddress(pk1, pk2, &k.WAddress)

	return pk1, pk2, nil
}

// lzh add
func generatePublicKeyFromWadress(waddress * common.WAddress) (* ecdsa.PublicKey, *ecdsa.PublicKey, error)  {
	if CheckSum16(waddress[:]) != 0 {
		return nil, nil, errors.New("invalid waddress! check sum is not zero!")
	}

	pk1 := new(ecdsa.PublicKey)
	pk2 := new(ecdsa.PublicKey)

	initPublicKeyFromWaddress(pk1, pk2, waddress)

	return pk1, pk2, nil
}

// lzh add
func initPublicKeyFromWaddress(pk1, pk2 * ecdsa.PublicKey, waddress * common.WAddress)  {
	pk1.Curve = crypto.S256()
	pk2.Curve = crypto.S256()

	pk1.X = new(big.Int).SetBytes(waddress[0:32])
	pk1.Y = new(big.Int).SetBytes(waddress[32:64])
	pk2.X = new(big.Int).SetBytes(waddress[64:96])
	pk2.Y = new(big.Int).SetBytes(waddress[96:128])
}


//// lzh add (ecdsa public key AX --> AY)
//func GetEllipticYFromX(curve *elliptic.CurveParams, x *big.Int, positive bool) *big.Int  {
//	// y² = x³ - 3x + b
//	y2 := new(big.Int).Mul(x, x)
//	y2.Mul(y2, x)
//
//	threeX := new(big.Int).Lsh(x, 1)
//	threeX.Add(threeX, x)
//
//	y2.Sub(y2, threeX)
//	y2.Add(y2, curve.B)
//
//	// ⌊√y2⌋ --> y
//	y := new(big.Int).Sqrt(y2)
//
//
//	if positive && y.Cmp(new(big.Int)) < 0 {
//		y.Sub(new(big.Int), y)
//	}
//
//	return y
//}
//
//func TestGetEllipticYFromX() {
//	xStr := "d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf"
//	yStr := "6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70"
//
//	x, ok := new(big.Int).SetString(xStr, 16)
//	if !ok {
//		return
//	}
//
//	strstr := common.Bytes2Hex(x.Bytes())
//
//	exY, ok := new(big.Int).SetString(yStr, 16)
//	if !ok {
//		return
//	}
//
//	strstr = common.Bytes2Hex(exY.Bytes())
//
//	y := GetEllipticYFromX(crypto.S256().Params(), x, false)
//	strstr = common.Bytes2Hex(y.Bytes())
//
//	if y.Cmp(exY) == 0 {
//		log.Info("TestGetEllipticYFromX suc!", strstr)
//	} else {
//		log.Info("TestGetEllipticYFromX fail!", strstr)
//	}
//}

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
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	var privateKeyECDSA2 *ecdsa.PrivateKey
	privateKeyECDSA2, err = ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	key := newKeyFromECDSA(privateKeyECDSA, privateKeyECDSA2)
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

	var privateKeyECDSA2 *ecdsa.PrivateKey
	privateKeyECDSA2, err = ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}
	return newKeyFromECDSA(privateKeyECDSA, privateKeyECDSA2), nil
}

func storeNewKey(ks keyStore, rand io.Reader, auth string) (*Key, accounts.Account, error) {
	key, err := newKey(rand)
	if err != nil {
		return nil, accounts.Account{}, err
	}
	a := accounts.Account{Address: key.Address, URL: accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(keyFileName(key.Address))}}
	if err := ks.StoreKey(a.URL.Path, key, auth); err != nil {
		zeroKey(key.PrivateKey)
		return nil, a, err
	}
	return key, a, err
}

func writeKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyAddr common.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

