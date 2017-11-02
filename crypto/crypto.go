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

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"math/big"
	Mrand "math/rand"
	"os"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto/sha3"
	"github.com/wanchain/go-wanchain/rlp"
	"fmt"
	"github.com/wanchain/go-wanchain/common/math"
)

var (
	secp256k1_N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1_halfN = new(big.Int).Div(secp256k1_N, big.NewInt(2))
)

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

// Keccak512 calculates and returns the Keccak512 hash of the input data.
func Keccak512(data ...[]byte) []byte {
	d := sha3.NewKeccak512()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// Deprecated: For backward compatibility as other packages depend on these
func Sha3Hash(data ...[]byte) common.Hash { return Keccak256Hash(data...) }

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b common.Address, nonce uint64) common.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	return common.BytesToAddress(Keccak256(data)[12:])
}

// ToECDSA creates a private key with the given D value.
func ToECDSA(prv []byte) *ecdsa.PrivateKey {
	if len(prv) == 0 {
		return nil
	}

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	priv.D = new(big.Int).SetBytes(prv)
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(prv)
	return priv
}

func FromECDSA(prv *ecdsa.PrivateKey) []byte {
	if prv == nil {
		return nil
	}
	return prv.D.Bytes()
}

func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(S256(), pub)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y)
}

// HexToECDSA parses a secp256k1 private key.
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	if len(b) != 32 {
		return nil, errors.New("invalid length, need 256 bits")
	}
	return ToECDSA(b), nil
}

// LoadECDSA loads a secp256k1 private key from the given file.
// The key data is expected to be hex-encoded.
func LoadECDSA(file string) (*ecdsa.PrivateKey, error) {
	buf := make([]byte, 64)
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	if _, err := io.ReadFull(fd, buf); err != nil {
		return nil, err
	}

	key, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}

	return ToECDSA(key), nil
}

// SaveECDSA saves a secp256k1 private key to the given file with
// restrictive permissions. The key data is saved hex-encoded.
func SaveECDSA(file string, key *ecdsa.PrivateKey) error {
	k := hex.EncodeToString(FromECDSA(key))
	return ioutil.WriteFile(file, []byte(k), 0600)
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(S256(), rand.Reader)
}

// ValidateSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	// reject upper range of s values (ECDSA malleability)
	// see discussion in secp256k1/libsecp256k1/include/secp256k1.h
	if homestead && s.Cmp(secp256k1_halfN) > 0 {
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256k1_N) < 0 && s.Cmp(secp256k1_N) < 0 && (v == 0 || v == 1)
}

func PubkeyToAddress(p ecdsa.PublicKey) common.Address {
	pubBytes := FromECDSAPub(&p)

	return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])

}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}

///////////////////////////////////以下为新加内容/////////////////////////////////////



//PublicKeyToInt for json 把公钥数组点转int数组，(0放x，1放y)
//PublicKeys[n]数组时，调用 outInt=PublicKeyToInt(PublicKeys...），
//单个公钥KeyImage时，调用outInt=PublicKeyToInt（KeyImage),    outInt[0]为返回值
func PublicKeyToInt(PublicKeys ...*ecdsa.PublicKey) []*hexutil.Big {
	n := len(PublicKeys)
	outInt := make([]*hexutil.Big, 2*n)
	for i := 0; i < n; i++ {
		outInt[2*i] = (*hexutil.Big)(PublicKeys[i].X)
		outInt[2*i+1] = (*hexutil.Big)(PublicKeys[i].Y)
	}

	return outInt
}

//IntToPublicKey from json 把int数组点转公钥数组点
func IntToPublicKey(in ...*big.Int) []*ecdsa.PublicKey {
	n := len(in)
	PublicKeys := make([]*ecdsa.PublicKey, n/2)
	for i := 0; i < n/2; i++ {
		//PublicKeys[i] = ToECDSAPub(in[i].Bytes())
		PublicKeys[i] = new(ecdsa.PublicKey)
		PublicKeys[i].X = in[2*i]
		PublicKeys[i].Y = in[2*i+1]
		PublicKeys[i].Curve = S256()
	}
	return PublicKeys
}

//2528 Shi TeemoGuo
func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

//2528 Shi TeemoGuo
func AesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	return aesCTRXOR(key, inText, iv)
}

//2528 add

var bigOne = big.NewInt(1)
var bigZero = big.NewInt(0)
var one = new(big.Int).SetInt64(1)

func modInverse(a, n *big.Int) (ia *big.Int, ok bool) {
	g := new(big.Int)
	x := new(big.Int)
	y := new(big.Int)
	g.GCD(x, y, a, n)
	if g.Cmp(bigOne) != 0 {
		// In this case, a and n aren't coprime and we cannot calculate
		// the inverse. This happens because the values of n are nearly
		// prime (being the product of two primes) rather than truly
		// prime.
		return
	}

	if x.Cmp(bigOne) < 0 {
		// 0 is not the multiplicative inverse of any element so, if x
		// < 1, then x is negative.
		x.Add(x, n)
	}

	return x, true
}

//2528 add
func Rsa_encrypt(c *big.Int, pub *rsa.PublicKey, m *big.Int) *big.Int {
	e := big.NewInt(int64(pub.E))
	c.Exp(m, e, pub.N)
	return c
}

func Rsa_decrypt(random io.Reader, priv *rsa.PrivateKey, c *big.Int) (m *big.Int, err error) {
	// TODO(agl): can we get away with reusing blinds?
	if c.Cmp(priv.N) > 0 {
		err = rsa.ErrDecryption
		return
	}
	if priv.N.Sign() == 0 {
		return nil, rsa.ErrDecryption
	}

	var ir *big.Int
	if random != nil {
		// Blinding enabled. Blinding involves multiplying c by r^e.
		// Then the decryption operation performs (m^e * r^e)^d mod n
		// which equals mr mod n. The factor of r can then be removed
		// by multiplying by the multiplicative inverse of r.

		var r *big.Int

		for {
			r, err = rand.Int(random, priv.N)
			if err != nil {
				return
			}
			if r.Cmp(bigZero) == 0 {
				r = bigOne
			}
			var ok bool
			ir, ok = modInverse(r, priv.N)
			if ok {
				break
			}
		}
		bigE := big.NewInt(int64(priv.E))
		rpowe := new(big.Int).Exp(r, bigE, priv.N) // N != 0
		cCopy := new(big.Int).Set(c)
		cCopy.Mul(cCopy, rpowe)
		cCopy.Mod(cCopy, priv.N)
		c = cCopy
	}

	if priv.Precomputed.Dp == nil {
		m = new(big.Int).Exp(c, priv.D, priv.N)
	} else {
		// We have the precalculated values needed for the CRT.
		m = new(big.Int).Exp(c, priv.Precomputed.Dp, priv.Primes[0])
		m2 := new(big.Int).Exp(c, priv.Precomputed.Dq, priv.Primes[1])
		m.Sub(m, m2)
		if m.Sign() < 0 {
			m.Add(m, priv.Primes[0])
		}
		m.Mul(m, priv.Precomputed.Qinv)
		m.Mod(m, priv.Primes[0])
		m.Mul(m, priv.Primes[1])
		m.Add(m, m2)

		for i, values := range priv.Precomputed.CRTValues {
			prime := priv.Primes[2+i]
			m2.Exp(c, values.Exp, prime)
			m2.Sub(m2, m)
			m2.Mul(m2, values.Coeff)
			m2.Mod(m2, prime)
			if m2.Sign() < 0 {
				m2.Add(m2, prime)
			}
			m2.Mul(m2, values.R)
			m.Add(m, m2)
		}
	}

	if ir != nil {
		// Unblind.
		m.Mul(m, ir)
		m.Mod(m, priv.N)
	}

	return
}

func randFieldElement2528(rand io.Reader) (k *big.Int, err error) {
	params := S256().Params()
	b := make([]byte, params.BitSize/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}
	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)

	return
}

///calc [x]Hash(P)
func xScalarHashP(x []byte, pub *ecdsa.PublicKey) (I *ecdsa.PublicKey) {
	KeyImg := new(ecdsa.PublicKey)
	I = new(ecdsa.PublicKey)
	KeyImg.X, KeyImg.Y = S256().ScalarMult(pub.X, pub.Y, Keccak256(FromECDSAPub(pub))) //Hash(P)
	I.X, I.Y = S256().ScalarMult(KeyImg.X, KeyImg.Y, x)
	I.Curve = S256()
	return
}

//明文，私钥x，公钥组，(P的公钥放在第0位,0....n)  环签名
//2528 Pengbo add Shi TeemoGuo revise
func RingSign(M []byte, x *big.Int, PublicKeys []*ecdsa.PublicKey) ([]*ecdsa.PublicKey, *ecdsa.PublicKey, []*big.Int, []*big.Int) {
	n := len(PublicKeys)
	//	fmt.Println(n)
	//n := 10
	I := xScalarHashP(x.Bytes(), PublicKeys[0]) //Key Image
	s := Mrand.Intn(n)                          //s位放主签名公钥
	if s > 0 {
		//s = s - 1
		PublicKeys[0], PublicKeys[s] = PublicKeys[s], PublicKeys[0] //交换位置
	}
	//fmt.Println("s=", s)

	//PublicKeys[n] = &P.PublicKey//P的公钥放在第n位,0....n

	var (
		q = make([]*big.Int, n)
		w = make([]*big.Int, n)
		//q = *new([10]*big.Int)
		//w = *new([10]*big.Int)
	)
	SumC := new(big.Int).SetInt64(0)
	Lpub := new(ecdsa.PublicKey)
	d := sha3.NewKeccak256()
	d.Write(M)
	//hash(M,Li,Ri)
	for i := 0; i < n; i++ {
		q[i], _ = randFieldElement2528(rand.Reader)
		w[i], _ = randFieldElement2528(rand.Reader)

		Lpub.X, Lpub.Y = S256().ScalarBaseMult(q[i].Bytes()) //[qi]G
		if i != s {
			Ppub := new(ecdsa.PublicKey)
			Ppub.X, Ppub.Y = S256().ScalarMult(PublicKeys[i].X, PublicKeys[i].Y, w[i].Bytes()) //[wi]Pi
			Lpub.X, Lpub.Y = S256().Add(Lpub.X, Lpub.Y, Ppub.X, Ppub.Y)                        //[qi]G+[wi]Pi

			SumC.Add(SumC, w[i])
			SumC.Mod(SumC, secp256k1_N)
		}
		//fmt.Printf("L%d\t%x\n", i, FromECDSAPub(Lpub))
		d.Write(FromECDSAPub(Lpub))
	}
	Rpub := new(ecdsa.PublicKey)
	for i := 0; i < n; i++ {
		Rpub = xScalarHashP(q[i].Bytes(), PublicKeys[i]) //[qi]HashPi
		if i != s {
			Ppub := new(ecdsa.PublicKey)
			Ppub.X, Ppub.Y = S256().ScalarMult(I.X, I.Y, w[i].Bytes())  //[wi]I
			Rpub.X, Rpub.Y = S256().Add(Rpub.X, Rpub.Y, Ppub.X, Ppub.Y) //[qi]HashPi+[wi]I
		}
		//fmt.Printf("R%d\t%x\n", i, FromECDSAPub(Rpub))

		d.Write(FromECDSAPub(Rpub))
	}
	Cs := new(big.Int).SetBytes(d.Sum(nil)) //hash(m,Li,Ri)

	Cs.Sub(Cs, SumC)
	Cs.Mod(Cs, secp256k1_N)

	tmp := new(big.Int).Mul(Cs, x)
	Rs := new(big.Int).Sub(q[s], tmp)
	Rs.Mod(Rs, secp256k1_N)
	w[s] = Cs
	q[s] = Rs

	return PublicKeys, I, w, q
}

//VerifyRingSign 验证环签名
func VerifyRingSign(M []byte, PublicKeys []*ecdsa.PublicKey, I *ecdsa.PublicKey, c []*big.Int, r []*big.Int) bool {

	ret := false
	n := len(PublicKeys)

	fmt.Printf("R'%d\t%x\n", 0, M)
	for i:=0;i<n;i++ {
		fmt.Printf("R'%d\t%x\n", 1, FromECDSAPub(PublicKeys[i]))
	}

	fmt.Printf("R'%d\t%x\n", 2, FromECDSAPub(I))

	for i:=0;i<n;i++ {
		fmt.Printf("R'%d\t%x\n", 3, c[i].Bytes())
	}

	for i:=0;i<n;i++ {
		fmt.Printf("R'%d\t%x\n", 4, r[i].Bytes())
	}

	SumC := new(big.Int).SetInt64(0)
	Lpub := new(ecdsa.PublicKey)
	d := sha3.NewKeccak256()
	d.Write(M)

	//hash(M,Li,Ri)
	for i := 0; i < n; i++ {
		Lpub.X, Lpub.Y = S256().ScalarBaseMult(r[i].Bytes()) //[ri]G

		Ppub := new(ecdsa.PublicKey)
		Ppub.X, Ppub.Y = S256().ScalarMult(PublicKeys[i].X, PublicKeys[i].Y, c[i].Bytes()) //[ci]Pi
		Lpub.X, Lpub.Y = S256().Add(Lpub.X, Lpub.Y, Ppub.X, Ppub.Y)                        //[ri]G+[ci]Pi
		SumC.Add(SumC, c[i])
		SumC.Mod(SumC, secp256k1_N)
		d.Write(FromECDSAPub(Lpub))
		fmt.Printf("L'%d\t%x\n", i, FromECDSAPub(Lpub))
	}
	Rpub := new(ecdsa.PublicKey)
	for i := 0; i < n; i++ {
		Rpub = xScalarHashP(r[i].Bytes(), PublicKeys[i]) //[qi]HashPi
		Ppub := new(ecdsa.PublicKey)
		Ppub.X, Ppub.Y = S256().ScalarMult(I.X, I.Y, c[i].Bytes())  //[wi]I
		Rpub.X, Rpub.Y = S256().Add(Rpub.X, Rpub.Y, Ppub.X, Ppub.Y) //[qi]HashPi+[wi]I
		fmt.Printf("R'%d\t%x\n", i, FromECDSAPub(Rpub))

		d.Write(FromECDSAPub(Rpub))
	}
	hash := new(big.Int).SetBytes(d.Sum(nil)) //hash(m,Li,Ri)
	fmt.Printf("hash'%d\t%x\n", 0, hash.Bytes())

	hash.Mod(hash, secp256k1_N)
	fmt.Printf("hash'%d\t%x\n", 2,hash.Bytes())

	fmt.Printf("SumC'%d\t%x\n", 3, SumC.Bytes())
	if hash.Cmp(SumC) == 0 {
		ret = true
	}
	return ret
}

// 2528 Pengbo add TeemoGuo revise: A1=[hash([r]B)]G+A
func generateA1(r []byte, A *ecdsa.PublicKey, B *ecdsa.PublicKey) ecdsa.PublicKey {
	A1 := new(ecdsa.PublicKey)
	A1.X, A1.Y = S256().ScalarMult(B.X, B.Y, r)   //A1=[r]B
	A1Bytes := Keccak256(FromECDSAPub(A1))        //hash([r]B)
	A1.X, A1.Y = S256().ScalarBaseMult(A1Bytes)   //[hash([r]B)]G
	A1.X, A1.Y = S256().Add(A1.X, A1.Y, A.X, A.Y) //A1=[hash([r]B)]G+A
	A1.Curve = S256()
	return *A1
}

func CompareA1(b []byte, A *ecdsa.PublicKey, S1 *ecdsa.PublicKey, A1 *ecdsa.PublicKey) bool {
	A1n := generateA1(b, A, S1)
	if A1.X.Cmp(A1n.X) == 0 && A1.Y.Cmp(A1n.Y) == 0 {
		return true
	}
	return false
}

// 2528 Pengbo add TeemoGuo revise
func generateOneTimeKey2528(A *ecdsa.PublicKey, B *ecdsa.PublicKey) (A1 *ecdsa.PublicKey, R *ecdsa.PublicKey, err error) {
	RPrivateKey, err := GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	R = &RPrivateKey.PublicKey
	A1 = new(ecdsa.PublicKey)
	//*A1 = generateA1(RPrivateKey.D.Bytes(), A, B)
	// anson modifies
	*A1 = generateA1(math.PaddedBigBytes(RPrivateKey.D, 32), A, B)
	return A1, R, err
}

func GenerateOneTimeKey(AX string, AY string, BX string, BY string) (ret []string, err error) {
	bytesAX, err := hexutil.Decode(AX)
	if err != nil {
		return
	}
	bytesAY, err := hexutil.Decode(AY)
	if err != nil {
		return
	}
	bytesBX, err := hexutil.Decode(BX)
	if err != nil {
		return
	}
	bytesBY, err := hexutil.Decode(BY)
	if err != nil {
		return
	}
	bnAX := new(big.Int).SetBytes(bytesAX)
	bnAY := new(big.Int).SetBytes(bytesAY)
	bnBX := new(big.Int).SetBytes(bytesBX)
	bnBY := new(big.Int).SetBytes(bytesBY)

	pa := &ecdsa.PublicKey{X: bnAX, Y: bnAY}
	pb := &ecdsa.PublicKey{X: bnBX, Y: bnBY}

	generatedA1, generatedR, err := generateOneTimeKey2528(pa, pb)
	return hexutil.TwoPublicKeyToHexSlice(generatedA1, generatedR), nil
}

func GenerteOTAPrivateKey(privateKey *ecdsa.PrivateKey, privateKey2 *ecdsa.PrivateKey, AX string, AY string, BX string, BY string) (retPub *ecdsa.PublicKey, retPriv1 *ecdsa.PrivateKey, retPriv2 *ecdsa.PrivateKey, err error) {
	bytesAX, err := hexutil.Decode(AX)
	if err != nil {
		return
	}
	bytesAY, err := hexutil.Decode(AY)
	if err != nil {
		return
	}
	bytesBX, err := hexutil.Decode(BX)
	if err != nil {
		return
	}
	bytesBY, err := hexutil.Decode(BY)
	if err != nil {
		return
	}
	bnAX := new(big.Int).SetBytes(bytesAX)
	bnAY := new(big.Int).SetBytes(bytesAY)
	bnBX := new(big.Int).SetBytes(bytesBX)
	bnBY := new(big.Int).SetBytes(bytesBY)

	retPub = &ecdsa.PublicKey{X: bnAX, Y: bnAY}
	pb := &ecdsa.PublicKey{X: bnBX, Y: bnBY}
    retPriv1, retPriv2, err = GenerateOneTimePrivateKey2528(privateKey, privateKey2, retPub, pb)
	return
}

func GenerateOneTimePrivateKey2528(privateKey *ecdsa.PrivateKey, privateKey2 *ecdsa.PrivateKey, destPubA *ecdsa.PublicKey, destPubB *ecdsa.PublicKey) (retPriv1 *ecdsa.PrivateKey, retPriv2 *ecdsa.PrivateKey, err error) {
	pub := new(ecdsa.PublicKey)
	pub.X, pub.Y = S256().ScalarMult(destPubB.X, destPubB.Y, privateKey2.D.Bytes()) //[b]R
	k := new(big.Int).SetBytes(Keccak256(FromECDSAPub(pub)))                        //hash([b]R)
	k.Add(k, privateKey.D)                                                          //hash([b]R)+a
	k.Mod(k, S256().Params().N)                                                     //mod to feild N

	retPriv1 = new(ecdsa.PrivateKey)
	retPriv2 = new(ecdsa.PrivateKey)

	retPriv1.D = k
	retPriv2.D = new(big.Int).SetInt64(0)
	return retPriv1, retPriv2, nil
}


/////////////////////////////////////////jia added////////////////////////////////////////////////////////////////
const (
	// alphabet is the modified base58 alphabet used by Bitcoin.
	alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	alphabetIdx0 = '1'
)

var b58 = [256]byte{
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 0, 1, 2, 3, 4, 5, 6,
	7, 8, 255, 255, 255, 255, 255, 255,
	255, 9, 10, 11, 12, 13, 14, 15,
	16, 255, 17, 18, 19, 20, 21, 255,
	22, 23, 24, 25, 26, 27, 28, 29,
	30, 31, 32, 255, 255, 255, 255, 255,
	255, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 255, 44, 45, 46,
	47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
}

var bigRadix = big.NewInt(58)
//var bigZero = big.NewInt(0)

func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)

	return h
}
var FactoidPrefix = []byte{0x6c, 0x12}
var WangLuMagicBigInt = new(big.Int).SetBytes(Hex2Bytes("9da26fc2e1d6ad9fdd46138906b0104ae68a65d8"))


// Decode decodes a modified base58 string to a byte slice.
func Decode(b string) []byte {
	answer := big.NewInt(0)
	j := big.NewInt(1)

	scratch := new(big.Int)
	for i := len(b) - 1; i >= 0; i-- {
		tmp := b58[b[i]]
		if tmp == 255 {
			return []byte("")
		}
		scratch.SetInt64(int64(tmp))
		scratch.Mul(j, scratch)
		answer.Add(answer, scratch)
		j.Mul(j, bigRadix)
	}

	tmpval := answer.Bytes()

	var numZeros int
	for numZeros = 0; numZeros < len(b); numZeros++ {
		if b[numZeros] != alphabetIdx0 {
			break
		}
	}
	flen := numZeros + len(tmpval)
	val := make([]byte, flen, flen)
	copy(val[numZeros:], tmpval)

	return val
}

// Encode encodes a byte slice to a modified base58 string.
func Encode(b []byte) string {
	x := new(big.Int)
	x.SetBytes(b)

	answer := make([]byte, 0, len(b)*136/100)
	for x.Cmp(bigZero) > 0 {
		mod := new(big.Int)
		x.DivMod(x, bigRadix, mod)
		answer = append(answer, alphabet[mod.Int64()])
	}

	// leading zero bytes
	for _, i := range b {
		if i != 0 {
			break
		}
		answer = append(answer, alphabetIdx0)
	}

	// reverse
	alen := len(answer)
	for i := 0; i < alen/2; i++ {
		answer[i], answer[alen-1-i] = answer[alen-1-i], answer[i]
	}

	return string(answer)
}

func getPreFixedBigInt() *big.Int{
	baseBigInt := new(big.Int)
	baseBigInt.SetBytes(Hex2Bytes("ffffffffffffffffffffffffffffffffffffffff"))
	fmt.Println("baseBegInt: " + baseBigInt.String())
	xdecimal := big.NewInt(58)
	base := big.NewInt(58)
	for base.Cmp(baseBigInt) <= 0 {
		base = base.Mul(base, xdecimal)
	}
	LWangLu := big.NewInt(19)
	LWangLu.Mul(LWangLu, base)
	WWangLu := big.NewInt(58 * 29)
	WWangLu.Mul(WWangLu, base)
	retBigInt := big.NewInt(0)
	retBigInt.Add(retBigInt, LWangLu)
	retBigInt.Add(retBigInt, WWangLu)
	fmt.Println("retBigInt: " + retBigInt.String())
	fmt.Println("retBigInt hex: " + hex.EncodeToString(retBigInt.Bytes()))
	return retBigInt
}

func otaAddress(address common.Address) string{

	result := Encode(append(FactoidPrefix,Hex2Bytes(address.Hex())...))


	return result
}
