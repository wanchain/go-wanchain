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

	"github.com/btcsuite/btcd/btcec"
	"github.com/pborman/uuid"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	"math/big"
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
	PrivateKey  *ecdsa.PrivateKey
	PrivateKey2 *ecdsa.PrivateKey

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
	Address  string     `json:"address"`
	Crypto   cryptoJSON `json:"crypto"`
	Crypto2  cryptoJSON `json:"crypto2"`
	Id       string     `json:"id"`
	Version  int        `json:"version"`
	WAddress string     `json:"waddress"`
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

func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey, privateKeyECDSA2 *ecdsa.PrivateKey) *Key {
	id := uuid.NewRandom()
	key := &Key{
		Id:          id,
		Address:     crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey:  privateKeyECDSA,
		PrivateKey2: privateKeyECDSA2,
	}

	updateWaddress(key)

	return key
}

// SerializeCompressed serializes a public key in a 33-byte compressed format. from btcec.
func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}
func PubkeySerializeCompressed(p *ecdsa.PublicKey) []byte {
	const pubkeyCompressed byte = 0x2
	b := make([]byte, 0, 33)
	format := pubkeyCompressed
	if isOdd(p.Y) {
		format |= 0x1
	}
	b = append(b, format)
	b = append(b, p.X.Bytes()...)
	return b
}
func (k *Key) GenerateWaddress() common.WAddress {
	var tmpWaddress common.WAddress
	copy(tmpWaddress[0:33], PubkeySerializeCompressed(&k.PrivateKey.PublicKey))
	copy(tmpWaddress[33:66], PubkeySerializeCompressed(&k.PrivateKey2.PublicKey))
	return tmpWaddress
}
func GenerateWaddressFromPubkey(Pub1, Pub2 *ecdsa.PublicKey) common.WAddress {
	var tmpWaddress common.WAddress
	copy(tmpWaddress[0:33], PubkeySerializeCompressed(Pub1))
	copy(tmpWaddress[33:66], PubkeySerializeCompressed(Pub2))
	return tmpWaddress
}

// lzh add
func updateWaddress(k *Key) {
	k.WAddress = k.GenerateWaddress()
}

// lzh add
func checkWaddressValid(k *Key) bool {
	return k.WAddress == k.GenerateWaddress()
}

// lzh add
func (k *Key) GetTwoPublicKeyRawStrs() ([]string, error) {
	PK1, PK2, err := k.GetTwoPublicKey()
	if err != nil {
		return nil, err
	}
	ret := hexutil.TwoPublicKeyToHexSlice(PK1, PK2)
	return ret, nil
}

// lzh add
func (k *Key) GetTwoPublicKey() (*ecdsa.PublicKey, *ecdsa.PublicKey, error) {
	return GeneratePublicKeyFromWadress(k.WAddress[:])
}

// lzh add
func GeneratePublicKeyFromWadress(waddr []byte) (*ecdsa.PublicKey, *ecdsa.PublicKey, error) {
	pb := make([]byte, 33)
	copy(pb[0:33], waddr[0:33])
	curve := btcec.S256()
	pk1, err := btcec.ParsePubKey(pb, curve)
	if err != nil {
		return nil, nil, err
	}
	copy(pb[0:33], waddr[33:66])
	pk2, err2 := btcec.ParsePubKey(pb, curve)
	if err2 != nil {
		return nil, nil, err2
	}
	return (*ecdsa.PublicKey)(pk1), (*ecdsa.PublicKey)(pk2), nil
}

func WaddrFromUncompressed(waddr []byte, raw []byte) error {
	pub := make([]byte, 65)
	pub[0] = 0x04
	copy(pub[1:], raw[0:64])
	A := crypto.ToECDSAPub(pub)
	copy(pub[1:], raw[64:])
	B := crypto.ToECDSAPub(pub)
	wd := GenerateWaddressFromPubkey(A, B)
	copy(waddr, wd[:])
	return nil
}

func ToWaddr(raw []byte)([]byte, error) {
	pub := make([]byte, 65)
	pub[0] = 0x04
	copy(pub[1:], raw[0:64])
	A := crypto.ToECDSAPub(pub)
	copy(pub[1:], raw[64:])
	B := crypto.ToECDSAPub(pub)
	wd := GenerateWaddressFromPubkey(A, B)
	return wd[:],nil
}

func WaddrToUncompressed(waddr []byte) ( []byte, error) {
	A, B, err := GeneratePublicKeyFromWadress(waddr)
	if err != nil {
		return nil,err
	}

	u := make([]byte,128)
	temp :=  A.X.Bytes()
	copy(u[0:],temp[0:32])
	temp =  A.Y.Bytes()
	copy(u[32:],temp[0:32])
	temp =  B.X.Bytes()
	copy(u[64:],temp[0:32])
	temp =  B.Y.Bytes()
	copy(u[96:],temp[0:32])

	return u,nil
}

func WaddrToUncompressedFromString(waddr string) ( []byte, error) {
	waddrBytes, _ := hexutil.Decode(waddr)
	A, B, err := GeneratePublicKeyFromWadress(waddrBytes)
	if err != nil {
		return nil,err
	}

	u := make([]byte,128)
	temp :=  A.X.Bytes()
	copy(u[0:],temp[0:32])
	temp =  A.Y.Bytes()
	copy(u[32:],temp[0:32])
	temp =  B.X.Bytes()
	copy(u[64:],temp[0:32])
	temp =  B.Y.Bytes()
	copy(u[96:],temp[0:32])

	return u,nil
}


// lzh add
func initPublicKeyFromWaddress(pk1, pk2 *ecdsa.PublicKey, waddress *common.WAddress) error {

	PK1, PK2, err := GeneratePublicKeyFromWadress(waddress[:])
	if err != nil {
		return err
	}
	pk1.Curve = crypto.S256()
	pk2.Curve = crypto.S256()

	pk1.X = PK1.X
	pk1.Y = PK1.Y
	pk2.X = PK2.X
	pk2.Y = PK2.Y

	return nil
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
