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

package vm

import (
	"crypto/sha256"
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"golang.org/x/crypto/ripemd160"
    "math/rand"
	"bytes"
	"strconv"
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/trie"
)

// Precompiled contract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	RequiredGas(inputSize int) uint64 // RequiredPrice calculates the contract gas use
	Run(input []byte,contract *Contract,evm *Interpreter) []byte          // Run runs the precompiled contract
}

// Precompiled contains the default set of ethereum contracts
var PrecompiledContracts = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}): &ecrecover{},
	common.BytesToAddress([]byte{2}): &sha256hash{},
	common.BytesToAddress([]byte{3}): &ripemd160hash{},
	common.BytesToAddress([]byte{4}): &dataCopy{},
	common.BytesToAddress([]byte{5}): &wanchainStampSC{},
	common.BytesToAddress([]byte{6}): &wanCoinSC{},
}

// RunPrecompile runs and evaluate the output of a precompiled contract defined in contracts.go
func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract,evm *Interpreter) (ret []byte, err error) {
	gas := p.RequiredGas(len(input))
	if contract.UseGas(gas) {
		
		ret = p.Run(input,contract,evm)
		if ret!= nil {
			return ret, nil
		} else {
			return nil, ErrOutOfGas
		}

	} else {
		return nil, ErrOutOfGas
	}
}

// ECRECOVER implemented as a native contract
type ecrecover struct{}

func (c *ecrecover) RequiredGas(inputSize int) uint64 {
	return params.EcrecoverGas
}

func (c *ecrecover) Run(in []byte,contract *Contract,evm *Interpreter) []byte {
	const ecRecoverInputLength = 128

	in = common.RightPadBytes(in, ecRecoverInputLength)
	// "in" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(in[64:96])
	s := new(big.Int).SetBytes(in[96:128])
	v := in[63] - 27

	// tighter sig s values in homestead only apply to tx sigs
	if !allZero(in[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		log.Trace("ECRECOVER error: v, r or s value invalid")
		return nil
	}
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(in[:32], append(in[64:128], v))
	// make sure the public key is a valid one
	if err != nil {
		log.Trace("ECRECOVER failed", "err", err)
		return nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32)
}

// SHA256 implemented as a native contract
type sha256hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *sha256hash) RequiredGas(inputSize int) uint64 {
	return uint64(inputSize+31)/32*params.Sha256WordGas + params.Sha256Gas
}
func (c *sha256hash) Run(in []byte,contract *Contract,evm *Interpreter) []byte {
	h := sha256.Sum256(in)
	return h[:]
}

// RIPMED160 implemented as a native contract
type ripemd160hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *ripemd160hash) RequiredGas(inputSize int) uint64 {
	return uint64(inputSize+31)/32*params.Ripemd160WordGas + params.Ripemd160Gas
}
func (c *ripemd160hash) Run(in []byte,contract *Contract,evm *Interpreter) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(in)
	return common.LeftPadBytes(ripemd.Sum(nil), 32)
}

// data copy implemented as a native contract
type dataCopy struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *dataCopy) RequiredGas(inputSize int) uint64 {
	return uint64(inputSize+31)/32*params.IdentityWordGas + params.IdentityGas
}

func (c *dataCopy) Run(in []byte,contract *Contract,evm *Interpreter) []byte {
	return in
}

/////////////////////////////////////added by jqg ///////////////////////////////////
//in structure
//the first byte is the ac
/*  byte[0]: 0->buy stamp
 * 			 1->get stampSet
 *			 2->refund
 *  byte[1]: if action is stampSet, this is the set number
 *  byte[2:]:the OTA-Address
 */

const (
	ACT_BUY_STAMP    = byte(0)
	ACT_GET_STAMP    = byte(1)
	ACT_REFUND_STAMP = byte(2)

	ONE_WAN_STAMP   = byte(1)
	TWO_WAN_STAMP   = byte(2)
	THREE_WAN_STAMP = byte(3)

)

type wanchainStampSC struct{
	wanchainStamps map[*big.Int]map[int][]byte
	wanchainStampValue map[int] *big.Int
}

func (c *wanchainStampSC) init()  {
	c.wanchainStamps =  make(map[*big.Int]map[int][]byte)
	c.wanchainStampValue = make(map[int] *big.Int)
	for i:=ONE_WAN_STAMP;i<=THREE_WAN_STAMP;i++ {
		val := strconv.Itoa(int(i)) + "00000000000000000"
		c.wanchainStampValue[int(i)] = new(big.Int).SetBytes([]byte(val))
		c.wanchainStamps[new(big.Int).SetBytes([]byte(val))] = make(map[int][]byte)
	}

}

func (c *wanchainStampSC) RequiredGas(inputSize int) uint64 {
	return params.EcrecoverGas
}

func (c *wanchainStampSC) Run(in []byte,contract *Contract,evm *Interpreter) []byte {
	if c.wanchainStamps==nil {
		c.init();
	}

	if in[0]==ACT_BUY_STAMP {
		return c.buyStamp(in[1:],contract,evm)
	} else if in[0]==ACT_GET_STAMP {
		return c.getStamps(in[1:],contract,evm)
	} else {
		return c.refund(in[1:],contract,evm)
	}

	return  nil
}


func (c *wanchainStampSC) buyStamp(in []byte,contract *Contract,evm *Interpreter) []byte {

	length := len(in)
	temp := make([]byte,length-1)
	copy(temp,in[1:])

	mapRef := c.wanchainStamps[contract.value]
	if mapRef == nil {
		return nil
	}

	elNum := len(mapRef)
	mapRef[elNum+1]= temp


	addrSrc := contract.CallerAddress

	balance := evm.env.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0{
		evm.env.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1")
	}

	return nil
}

func (c *wanchainStampSC) getStamps(in []byte,contract *Contract,evm *Interpreter) []byte {

	num := int(in[0])
	var mapLen int

	stamps := []byte("")
	stampSet := make([][]byte, num)

	mapRef := c.wanchainStamps[contract.value]
	if mapRef == nil {
		return nil
	}

	for i:=0;i<num;i++ {
		rnd := rand.Intn(mapLen)
		stampSet[i] = mapRef[rnd]
	}

	return bytes.Join(stampSet, stamps)
}

func (c *wanchainStampSC) refund(in []byte,contract *Contract,evm *Interpreter) []byte {
	return nil
}
///////////////////////////////////////////////////////////////////
/*  byte[0]: 0->buy
 *			 1->refund
 *
 *  byte[2]: if action is stampSet, this is the set number
 *  byte[3:]:the OTA-Address
 */

const (
	WANCOIN_BUY    = byte(0)
	WANCOIN_GET_COINS = byte(1)
	WANCOIN_REFUND = byte(2)
)

type wanCoinSC struct{
	vmtrie *trie.SecureTrie
}

func (c *wanCoinSC) RequiredGas(inputSize int) uint64 {
	return params.EcrecoverGas
}

const (
	pre0dot1 = "10000000000000000"//0.1
	pre0dot2 = "20000000000000000"//0.2
	pre0dot5 = "50000000000000000"//0.5
	pre1 = 	   "100000000000000000"//1
	pre2 = 	   "200000000000000000"//2
	pre5 = 	   "500000000000000000"//5
	pre10 =    "1000000000000000000"//10
	pre20 =    "2000000000000000000"//20
	pre50 =    "5000000000000000000"//50
	pre100 =   "50000000000000000000"//100
)

func (c *wanCoinSC) init(in []byte,contract *Contract,evm *Interpreter){
	common.StringToAddress("wanchainCoinSc")
	c.vmtrie = evm.env.StateDB.StorageVmTrie(contract.Address())

}

func (c *wanCoinSC) Run(in []byte,contract *Contract,evm *Interpreter) []byte {
	if c.vmtrie== nil {
		c.init(nil,contract,evm)
	}

	if in[0]==WANCOIN_BUY {
		return c.buyCoin(in[1:],contract,evm)
	} else if in[0]==WANCOIN_GET_COINS {
		return c.getCoins(in[1:],contract,evm)
	} else if in[0]==WANCOIN_REFUND {
		return c.refund(in[0:],contract,evm)
	}

	return  nil
}

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)


func (c *wanCoinSC) buyCoin(in []byte,contract *Contract,evm *Interpreter) []byte {

	length := len(in)
	temp := make([]byte,length)
	copy(temp,in[:])

	val := contract.value.Bytes()

	//common.BytesToAddress([]byte{1,2}
	trie := c.vmtrie
	if trie==nil {
		return nil
	}

	err :=trie.TryUpdate(temp,val)
	if err!=nil {
		return nil
	}
	trie.Commit()

	addrSrc := contract.CallerAddress

	balance := evm.env.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0{
		evm.env.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1")
	}

	return nil
}

func (c *wanCoinSC) getCoins(in []byte,contract *Contract,evm *Interpreter) []byte {

	num := int(in[0])

	//var mapLen int

	coins := []byte("")
	coinSet := make([][]byte, num)
	//val := string(contract.value.Bytes())
	//mapRef := //c.wanCoin[val]
	//if mapRef == nil {
	//	return nil
	//}

	for i:=0;i<num;i++ {
		//rnd := rand.Intn(mapLen)
		//coinSet[i] = mapRef[rnd]
	}

	return bytes.Join(coinSet, coins)
}



func (c *wanCoinSC) refund(all []byte,contract *Contract,evm *Interpreter) []byte {

	valLen := int(all[1])
	otaLen := int(all[2]<<8|all[3])
	otaAddrBytes := all[4:otaLen]

	refundValBytes := all[otaLen:otaLen+valLen]

	trie := c.vmtrie

	if trie==nil {
		return nil
	}

	sendValueBytes,err :=trie.TryGet(otaAddrBytes)
	if err!=nil {
		return nil
	}

	if !bytes.Equal(refundValBytes,sendValueBytes) {
		return nil
	}

	idx := otaLen + valLen
	pubsLen := int(all[idx])
	idx = idx + 1

	PublicKeySet := *new([]*ecdsa.PublicKey)
	W_random := *new([]*big.Int)
	Q_random := *new([]*big.Int)


	var i int
	for i = 0; i < pubsLen; i++ {
		lenxy := int(all[idx])
		idx = idx + 1

		x := make([]byte,lenxy)
		copy(x,all[idx:])
		puk := crypto.ToECDSAPub(x)
		PublicKeySet = append(PublicKeySet, puk)//convert []byte to public key
		idx = idx + lenxy


		lenw :=  int(all[idx])
		idx = idx + 1

		w := make([]byte,lenw)
		copy(w,all[idx:])
		rndw := new (big.Int).SetBytes(w)
		W_random = append(W_random, rndw) //convert []byte to random
		idx = idx + lenw



		lenq :=  int(all[idx])
		idx = idx + 1

		q := make([]byte,lenq)
		copy(q,all[idx:])
		rndq := new (big.Int).SetBytes(q)
		Q_random = append(Q_random, rndq)//convert []byte to random
		idx = idx + lenq
	}

	lenkixy := int(all[idx])
	idx = idx + 1

	kix := make([]byte,lenkixy)
	copy(kix,all[idx:])
	KeyImage := crypto.ToECDSAPub(kix)
	idx = idx + lenkixy

	txHashLen := all[idx]
	idx = idx + 1
	txhashBytes :=  make([]byte,txHashLen)
	copy(txhashBytes,all[idx:])

	imageValue,erri :=trie.TryGet(kix)

	if len(imageValue)!=0&&erri==nil {
		return nil
	} else  {

	   trie.Update(kix,sendValueBytes)
		//func VerifyRingSign(M []byte, PublicKeys []*ecdsa.PublicKey, I *ecdsa.PublicKey, c []*big.Int, r []*big.Int) bool
	   verifyRes := crypto.VerifyRingSign(txhashBytes,PublicKeySet,KeyImage,[]*big.Int(W_random),[]*big.Int(Q_random))

		if verifyRes {
			vb := new (big.Int)
			vb.SetBytes(refundValBytes)

			addrSrc := contract.CallerAddress
			evm.env.StateDB.AddBalance(addrSrc, vb)
			//evm.env.Transfer(evm.env.StateDB,contract.Address() ,addrSrc,(*hexutil.Big)vb)

			return []byte("1")

		}
	}

	return nil

}


//func saveOtaAddress (all []byte,contract *Contract,evm *Interpreter) {
//
//	//d := memory.Get(mStart.Int64(), mSize.Int64())
//	trie := evm.env.StateDB.
//
//	evm.env.StateDB.AddLog(&types.Log{
//		Address: contract.Address(),
//		Topics:  topics,
//		Data:    d,
//		// This is a non-consensus field, but assigned here because
//		// core/state doesn't know the current block number.
//		BlockNumber: evm.env.BlockNumber.Uint64(),
//	})
//
//}




