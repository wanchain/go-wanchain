// Copyright (c) 2013 Kyle Isom <kyle@tyrfingr.is>
// Copyright (c) 2012 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package ecies

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"hash"
)

////////////////////////////////////////////////////////////////////////////////////////
func concatKDF2(password, salt []byte, iter, keyLen int, h func() hash.Hash) []byte {

	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var buf [4]byte
	dk := make([]byte, 0, numBlocks*hashLen)
	U := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		// N.B.: || means concatenation, ^ means XOR
		// for each block T_i = U_1 ^ U_2 ^ ... ^ U_iter
		// U_1 = PRF(password, salt || uint(i))
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf[:4])
		dk = prf.Sum(dk)
		T := dk[len(dk)-hashLen:]
		copy(U, T)

		// U_n = PRF(password, U_(n-1))
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(U)
			U = U[:0]
			U = prf.Sum(U)
			for x := range U {
				T[x] ^= U[x]
			}
		}
	}

	return dk[:keyLen]
}
func EncryptWithRandom(rbprv *PrivateKey, pub *PublicKey, iv []byte, m, s1, s2 []byte) (ct []byte, err error) {

	params := pub.Params
	if params == nil {
		if params = ParamsFromCurve(pub.Curve); params == nil {
			err = ErrUnsupportedECIESParameters
			return
		}
	}

	shared, err := rbprv.GenerateShared(pub, 16, 16)
	if err != nil {
		return
	}
	sharedStr := common.Bytes2Hex(shared)

	fmt.Println("GenerateShared=" + sharedStr)

	hash := sha256.New()

	K := concatKDF2([]byte(sharedStr), []byte(" "), 2, 64, sha256.New)

	fmt.Println("concatKDF=" + common.Bytes2Hex(K))

	Ke := K[:16]
	Km := K[16:32]

	hash.Write(Km)
	Km = hash.Sum(nil)

	fmt.Println("macKey=" + common.Bytes2Hex(Km))
	hash.Reset()

	em := AES_CBC_Encrypt(m, Ke, iv)
	if len(em) <= params.BlockSize {
		return
	}
	fmt.Println("encrypt message=" + common.Bytes2Hex(em))

	d := hmacSha256(em, Km)

	fmt.Println("mac=" + common.Bytes2Hex(d))

	empub := elliptic.Marshal(pub.Curve, rbprv.PublicKey.X, rbprv.PublicKey.Y)

	ct = make([]byte, (((len(empub)+len(iv)+len(em)+len(d))/32 + 1) * 32))

	copy(ct, empub)
	copy(ct[len(empub):], iv)

	copy(ct[len(empub)+len(iv):], em)

	copy(ct[len(empub)+len(iv)+len(em):], d)

	fmt.Println("eccEncWhole=" + common.Bytes2Hex(ct))

	return
}

func AES_CBC_Encrypt(plainText []byte, key []byte, iv []byte) []byte {

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	plainText = Padding(plainText, block.BlockSize())

	blockMode := cipher.NewCBCEncrypter(block, iv)

	cipherText := make([]byte, len(plainText))
	blockMode.CryptBlocks(cipherText, plainText)

	return cipherText
}

func Padding(plainText []byte, blockSize int) []byte {

	n := blockSize - len(plainText)%blockSize

	temp := bytes.Repeat([]byte{byte(n)}, n)
	plainText = append(plainText, temp...)
	return plainText
}

func UnPadding(cipherText []byte) []byte {

	end := cipherText[len(cipherText)-1]

	cipherText = cipherText[:len(cipherText)-int(end)]
	return cipherText
}

func AES_CBC_Decrypt(cipherText []byte, key []byte, iv []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	blockMode := cipher.NewCBCDecrypter(block, iv)

	plainText := make([]byte, len(cipherText))
	blockMode.CryptBlocks(plainText, cipherText)

	plainText = UnPadding(plainText)
	return plainText
}

func hmacSha256(data []byte, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return h.Sum(nil)
}
