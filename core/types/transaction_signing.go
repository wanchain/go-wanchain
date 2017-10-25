// Copyright 2016 The go-ethereum Authors
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

package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/common/hexutil"
)

var ErrInvalidChainId = errors.New("invalid chaid id for signer")

// sigCache is used to cache the derived sender and contains
// the signer used to derive it.
type sigCache struct {
	signer Signer
	from   common.Address
}

// MakeSigner returns a Signer based on the given chain config and block number.
func MakeSigner(config *params.ChainConfig, blockNumber *big.Int) Signer {
	var signer Signer
	switch {
	case config.IsEIP155(blockNumber):
		signer = NewEIP155Signer(config.ChainId)
	case config.IsHomestead(blockNumber):
		signer = HomesteadSigner{}
	default:
		signer = FrontierSigner{}
	}
	return signer
}

//cr@zy-OTA: TODO:不知道在一次性地址的交易情况下是不是需要有改动

// TeemoGuo revise: 扩充函数参数，增加OTA交易类型的签名，todo 外部增加扫链程序，提供SignTx的参数PublicKeys
// SignTx signs the transaction using the given signer and private key
// TODO: Additional parameters added on SignTx, causes a conflict with test case in bench_test.go
//jqg  keys [] string defining is for transfer OTA private key

func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey, keys [] string) (*Transaction, error) {

	txh := s.Hash(tx)

	if len(tx.Data()) == 0 || (tx.data.Txtype != 0 &&tx.Txtype()!=2) {
		sig, err := crypto.Sign(txh[:], prv)
		if err != nil {
			return nil, err
		}
		return s.WithSignature(tx, sig)

	} else {//OTA类型交易环签名

		var otaPrivD *big.Int
		var privReplace *ecdsa.PrivateKey

		if tx.Data()[0] == WANCOIN_REFUND || tx.Data()[0] == WAN_CONTRACT_OTA {

			if tx.Data()[0] == WAN_CONTRACT_OTA {
				var err error
				privReplace, err = crypto.GenerateKey()
				if err != nil {
					return nil, err
				}
			} else {
				//refund do not need change the sender because the balance of the ota need to send to sender
				privReplace = prv
			}

			//used sender address and part of data as the rign sign hash check
			var addr common.Address
			pubkey := crypto.FromECDSAPub(&privReplace.PublicKey)
			//caculate the address for replaced pub
			copy(addr[:], crypto.Keccak256(pubkey[1:])[12:])

			txData := tx.Data();
			hashBytes := make([]byte,len(addr[:])+ len(txData))//the use addr and the tx.data[0:4] as the hash input for ring sig verify

			copy(hashBytes,addr[:])
			copy(hashBytes[common.AddressLength:],txData)

			//this hash is used to veriy the sender
			verifyHash := common.BytesToHash(hashBytes)


			otaPrivD,_ = new (big.Int).SetString(keys[0],16)
			keysLen := len(keys)

			publicKeys := make([]*ecdsa.PublicKey, 0)
			for i:=1;i<keysLen;i++ {
				x  := keys[i][0:64]
				xb,_:= new(big.Int).SetString(x,16)
				y := keys[i][64:]
				yb,_:= new(big.Int).SetString(y,16)
				tk:= &ecdsa.PublicKey{X: xb, Y: yb}
				publicKeys = append(publicKeys,tk)
			}

			PublicKeys, KeyImage, w_random, q_random := crypto.RingSign(verifyHash.Bytes(), otaPrivD, publicKeys)
			crypto.VerifyRingSign(verifyHash.Bytes(),PublicKeys,KeyImage,w_random,q_random)
			var idx int
			pubsLen := len(PublicKeys)
			orgLen := len(tx.data.Payload) //value

			puklen := len(crypto.FromECDSAPub(PublicKeys[0])) + 1 //length + value
			rndlen := len(w_random[0].Bytes()) + 1                //length + value
			verifyHashLen := len(verifyHash.Bytes()) + 1                       //length + value

			all := make([]byte, orgLen+pubsLen*(puklen+rndlen+rndlen )+puklen+verifyHashLen+1) //one public key,w,q ramdom is 2 segment

			//copy orginal data
			copy(all, tx.data.Payload)
				//record the
			idx = orgLen
			all[idx] = byte(pubsLen)
			idx = idx + 1

			var i int
			for i = 0; i < pubsLen; i++ {

				pubk := PublicKeys[i]
				byteArray := crypto.FromECDSAPub(pubk)
				lenxy := len(byteArray)
				all[idx] = byte(lenxy)
				idx = idx + 1
				copy(all[idx:], byteArray)
				idx = idx + lenxy

				w := w_random[i].Bytes()
				lenw := len(w)
				all[idx] = byte(lenw)
				idx = idx + 1
				copy(all[idx:], w)
				idx = idx + lenw

				q := q_random[i].Bytes()
				lenq := len(q)
				all[idx] = byte(lenq)
				idx = idx + 1
				copy(all[idx:], q)
				idx = idx + lenq
			}

			byteArray := crypto.FromECDSAPub(KeyImage)
			lenkixy := len(byteArray)

			all[idx] = byte(lenkixy)
			idx = idx + 1
			copy(all[idx:], byteArray)
			idx = idx + lenkixy

			all[idx] = byte(len(verifyHash.Bytes()))
			idx = idx + 1
			verifyhashBytes := verifyHash.Bytes()
			copy(all[idx:], verifyhashBytes)
			idx = idx + len(verifyHash.Bytes())


			tx.data.Payload = all

			//tx data is changed, so it is need to hash tx again
			txh = s.Hash(tx)
		}

		if tx.Data()[0] == WAN_CONTRACT_OTA {

			sig, err := crypto.Sign(txh[:], privReplace)
			if err != nil {
				return nil, err
			}

			tx.data.AccountNonce = 0

			tx, err = s.WithSignature(tx, sig)
			return tx, nil

		} else {

			sig, err := crypto.Sign(txh[:], prv)
			if err != nil {
				return nil, err
			}

			tx, err = s.WithSignature(tx, sig)
			return tx, nil
		}
	}
}

//zhangy
func SignTx_zy(tx *Transaction, s Signer, prv *ecdsa.PrivateKey, PublicKeys []*ecdsa.PublicKey) (*Transaction, error) {
	h := s.Hash(tx)
	if tx.data.Txtype != 0 &&tx.Txtype() != 2{
		sig, err := crypto.Sign(h[:], prv)
		if err != nil {
			return nil, err
		}
		return s.WithSignature(tx, sig)
	} else {//OTA类型交易环签名

		//tx.data.PublicKeys = PublicKeys
		// need help:为了测试先请吧环签名里面用于混淆的publickeys写死用几个测试用，暂时不从外面动态获取
		sig, err := crypto.Sign(h[:], prv)
		if err != nil {
			return nil, err
		}
		tx, err = s.WithSignature(tx, sig)

		// lzh modify
		testPublicKeys := make([]*ecdsa.PublicKey, 0)
		//testPublicKeys := *new([]*ecdsa.PublicKey)
		for i:=0; i< 10; i++{
			testPublicKeys = append(testPublicKeys, &prv.PublicKey)
		}

		PublicKeys, KeyImage, w_random, q_random := crypto.RingSign(h[:], prv.D, testPublicKeys)
		cpy := &Transaction{data: tx.data}

		cpy.data.PublicKeys = crypto.PublicKeyToInt(PublicKeys...)

		// lzh modify
		W_random := make([]*hexutil.Big, 0)
		Q_random := make([]*hexutil.Big, 0)
		//W_random := *new([]*hexutil.Big)
		//Q_random := *new([]*hexutil.Big)

		for i := 0; i < len(PublicKeys); i++ {
			w := w_random[i]
			q := q_random[i]

			W_random = append(W_random, (*hexutil.Big)(w))
			Q_random = append(Q_random, (*hexutil.Big)(q))
		}

		keyImage := crypto.PublicKeyToInt(KeyImage)


		cpy.data.KeyImage = keyImage
		cpy.data.W_random = W_random
		cpy.data.Q_random = Q_random

		return cpy, nil
	}	
}

// Sender derives the sender from the tx using the signer derivation
// functions.

// Sender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
// TeemoGuo revise: 环签名下无法知道发送者，如何修改程序 todo
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	pubkey, err := signer.PublicKey(tx)
	if err != nil {
		return common.Address{}, err
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pubkey[1:])[12:])
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

type Signer interface {
	// Hash returns the rlp encoded hash for signatures
	Hash(tx *Transaction) common.Hash
	// PubilcKey returns the public key derived from the signature
	PublicKey(tx *Transaction) ([]byte, error)
	// WithSignature returns a copy of the transaction with the given signature.
	// The signature must be encoded in [R || S || V] format where V is 0 or 1.
	WithSignature(tx *Transaction, sig []byte) (*Transaction, error)
	// Checks for equality on the signers
	Equal(Signer) bool
}

// EIP155Transaction implements TransactionInterface using the
// EIP155 rules
type EIP155Signer struct {
	HomesteadSigner

	chainId, chainIdMul *big.Int
}

func NewEIP155Signer(chainId *big.Int) EIP155Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP155Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

func (s EIP155Signer) Equal(s2 Signer) bool {
	eip155, ok := s2.(EIP155Signer)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}

// TeemoGuo revise: 环签名下无法知道发送者，如何修改程序 todo
func (s EIP155Signer) PublicKey(tx *Transaction) ([]byte, error) {
	// if the transaction is not protected fall back to homestead signer
	if !tx.Protected() {
		return (HomesteadSigner{}).PublicKey(tx)
	}

	if tx.ChainId().Cmp(s.chainId) != 0 {
		return nil, ErrInvalidChainId
	}

	V := byte(new(big.Int).Sub(tx.data.V, s.chainIdMul).Uint64() - 35)
	if !crypto.ValidateSignatureValues(V, tx.data.R, tx.data.S, true) {
		return nil, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	R, S := tx.data.R.Bytes(), tx.data.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(R):32], R)
	copy(sig[64-len(S):64], S)
	sig[64] = V

	// recover the public key from the signature
	hash := s.Hash(tx)
	pub, err := crypto.Ecrecover(hash[:], sig)
	if err != nil {
		return nil, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return nil, errors.New("invalid public key")
	}
	return pub, nil
}

// WithSignature returns a new transaction with the given signature. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
// wlcomment:
func (s EIP155Signer) WithSignature(tx *Transaction, sig []byte) (*Transaction, error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}

	cpy := &Transaction{data: tx.data}
	cpy.data.R = new(big.Int).SetBytes(sig[:32])
	cpy.data.S = new(big.Int).SetBytes(sig[32:64])
	cpy.data.V = new(big.Int).SetBytes([]byte{sig[64]})
	if s.chainId.Sign() != 0 {
		cpy.data.V = big.NewInt(int64(sig[64] + 35))
		cpy.data.V.Add(cpy.data.V, s.chainIdMul)
	}

	//cr@zy
	//if nonce == 0 在这里对一次性地址交易特殊处理
	//TeemoGuo: 已经在SignTx函数中做了处理
	return cpy, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
// TeemoGuo revise: OTA交易格式下如何修改程序 todo
func (s EIP155Signer) Hash(tx *Transaction) common.Hash {
	//cr@zy: 为OneTimeAddressTx的hash
	//if tx.nonce == 0
	return rlpHash([]interface{}{
		tx.data.Txtype,
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
		s.chainId, uint(0), uint(0),
	})
}

// HomesteadTransaction implements TransactionInterface using the
// homestead rules.
type HomesteadSigner struct{ FrontierSigner }

func (s HomesteadSigner) Equal(s2 Signer) bool {
	_, ok := s2.(HomesteadSigner)
	return ok
}

// WithSignature returns a new transaction with the given signature. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (hs HomesteadSigner) WithSignature(tx *Transaction, sig []byte) (*Transaction, error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for snature: got %d, want 65", len(sig)))
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R = new(big.Int).SetBytes(sig[:32])
	cpy.data.S = new(big.Int).SetBytes(sig[32:64])
	cpy.data.V = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return cpy, nil
}

// TeemoGuo revise: OTA交易格式下如何修改程序 todo
func (hs HomesteadSigner) PublicKey(tx *Transaction) ([]byte, error) {
	if tx.data.V.BitLen() > 8 {
		return nil, ErrInvalidSig
	}
	V := byte(tx.data.V.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, tx.data.R, tx.data.S, true) {
		return nil, ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := tx.data.R.Bytes(), tx.data.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	// recover the public key from the snature
	hash := hs.Hash(tx)
	pub, err := crypto.Ecrecover(hash[:], sig)
	if err != nil {
		return nil, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return nil, errors.New("invalid public key")
	}
	return pub, nil
}

type FrontierSigner struct{}

func (s FrontierSigner) Equal(s2 Signer) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

// WithSignature returns a new transaction with the given signature. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (fs FrontierSigner) WithSignature(tx *Transaction, sig []byte) (*Transaction, error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for snature: got %d, want 65", len(sig)))
	}
	cpy := &Transaction{data: tx.data}
	cpy.data.R = new(big.Int).SetBytes(sig[:32])
	cpy.data.S = new(big.Int).SetBytes(sig[32:64])
	cpy.data.V = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return cpy, nil
}

// TeemoGuo revise: OTA交易格式下如何修改程序 todo
// Hash returns the hash to be sned by the sender.
// It does not uniquely identify the transaction.
func (fs FrontierSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.data.Txtype,
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
	})
}

// TeemoGuo revise: OTA交易格式下如何修改程序 todo
func (fs FrontierSigner) PublicKey(tx *Transaction) ([]byte, error) {
	if tx.data.V.BitLen() > 8 {
		return nil, ErrInvalidSig
	}

	V := byte(tx.data.V.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, tx.data.R, tx.data.S, false) {
		return nil, ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := tx.data.R.Bytes(), tx.data.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	// recover the public key from the snature
	hash := fs.Hash(tx)
	pub, err := crypto.Ecrecover(hash[:], sig)
	if err != nil {
		return nil, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return nil, errors.New("invalid public key")
	}
	return pub, nil
}

// deriveChainId derives the chain id from the given v parameter
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}
