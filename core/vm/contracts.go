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
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/trie"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"math/rand"
	"fmt"
	"bytes"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"strings"
	abi "github.com/wanchain/go-wanchain/accounts/abi"
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
	WAN_CONTRACT_SEND_OTA = byte(0)

	WAN_BUY_STAMP = byte(3)
	WAN_VERIFY_STAMP = byte(4)
	WAN_STAMP_SET = byte(5)


	WAN_STAMP_DOT1 = "10000000000000000"//0.01
	WAN_STAMP_DOT2 = "20000000000000000"//0.02
	WAN_STAMP_DOT5 = "50000000000000000"//0.05

	OTA_ADDR_LEN = 128
)



type wanchainStampSC struct{

}


func (c *wanchainStampSC) RequiredGas(inputSize int) uint64 {
	return 0
}

func (c *wanchainStampSC) Run(in []byte,contract *Contract,evm *Interpreter) []byte {

	if in[0]==WAN_BUY_STAMP {
		return c.buyStamp(in[1:],contract,evm)
	} else if in[0]==WAN_VERIFY_STAMP{
		return c.verifyStamp(in[0:],contract,evm)
	}  else if in[0]==WAN_STAMP_SET{
		return c.getStamps(in[1:],contract,evm)
	}

	return  nil
}

func (c *wanchainStampSC) buyStamp(in []byte,contract *Contract,evm *Interpreter) []byte {

	otaAddr,err:= keystore.WaddrToUncompressed(in)//input is wand address
	if err!= nil {
		return nil
	}

	contractAddr := common.HexToAddress(contract.value.String())
	otaAddrKey := common.BytesToHash(otaAddr[0:64])

	// prevent rebuy
	storagedOtaAddr := evm.env.StateDB.GetStateByteArray(contractAddr, otaAddrKey)
	if storagedOtaAddr != nil && len(storagedOtaAddr) != 0 && bytes.Equal(storagedOtaAddr, in) {
		return nil
	}

	evm.env.StateDB.SetStateByteArray(contractAddr, otaAddrKey, in)


	addrSrc := contract.CallerAddress
	balance := evm.env.StateDB.GetBalance(addrSrc)
	if balance.Cmp(contract.value) >= 0{
		evm.env.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1")
	}

	return nil
}

func (c *wanchainStampSC) getStamps(in []byte,contract *Contract,evm *Interpreter) []byte {

	length := len(in)
	otaAddr := make([]byte,length)
	copy(otaAddr,in[:])

	var trie * trie.SecureTrie = nil
	stampVals := [...]string {WAN_STAMP_DOT1, WAN_STAMP_DOT2, WAN_STAMP_DOT5}
	for _, stampVal := range stampVals {
		contractAddr := common.HexToAddress(stampVal)
		otaAddrKey := common.BytesToHash(otaAddr[0:64])
		storagedOtaAddr := evm.env.StateDB.GetStateByteArray(contractAddr, otaAddrKey)
		if storagedOtaAddr != nil && len(storagedOtaAddr) != 0 {
			trie = evm.env.StateDB.StorageVmTrie(contractAddr)
			break
		}
	}

	if trie==nil {
		return nil
	}

	return getOtaSet(trie,3,otaAddr)

}

func (c *wanchainStampSC) verifyStamp(all []byte,contract *Contract,evm *Interpreter) []byte {

	addrsLen := int(all[1])
	otaLen := hexutil.BytesToShort(all[2:4])

	idx := int(otaLen) + addrsLen
	verifyHsBegin := idx //duplicate for hash verify

	pubsLen := int(all[idx])
	idx = idx + 1

	PublicKeySet := *new([]*ecdsa.PublicKey)
	W_random := *new([]*big.Int)
	Q_random := *new([]*big.Int)

	var storagedOtaAddr []byte = nil
	lenxy := int(all[idx])
	x := make([]byte,lenxy)
	copy(x,all[idx+1:])

	var stampVal string
	stampVals := [...]string {WAN_STAMP_DOT1, WAN_STAMP_DOT2, WAN_STAMP_DOT5}
	for _, stampVal = range stampVals {
		contractAddr := common.HexToAddress(stampVal)
		otaAddrKey := common.BytesToHash(x[1:])
		storagedOtaAddr = evm.env.StateDB.GetStateByteArray(contractAddr, otaAddrKey)
		if storagedOtaAddr != nil && len(storagedOtaAddr) != 0 {
			break
		}
	}

	//check if user have bought stamp
	if storagedOtaAddr == nil || len(storagedOtaAddr) == 0 {
		return nil
	}

	var i int
	contractAddr := common.HexToAddress(stampVal)
	for i = 0; i < pubsLen; i++ {
		lenxy = int(all[idx])
		idx = idx + 1

		x := make([]byte,lenxy)
		copy(x,all[idx:])

		//verify the stamp in the set is from current stamp tree
		otaAddrKey := common.BytesToHash(x[1:])
		storagedOtaAddr = evm.env.StateDB.GetStateByteArray(contractAddr, otaAddrKey)
		if storagedOtaAddr==nil || len(storagedOtaAddr)==0 {
			fmt.Print("not get stamp in the set")
			return nil
		}

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

	//txHashLen := int(all[idx])
	//idx = idx + 1
	//txhashBytes :=  make([]byte,txHashLen)
	//copy(txhashBytes,all[idx:])
	//idx = idx + txHashLen
	//
	//sigLen := all[idx]
	////idx = idx + 1 //sig data will not increase idx
	//sigBytes :=  make([]byte,sigLen)
	//copy(sigBytes,all[idx+1:])
	//res := verifyHash(all[0:verifyHsBegin],contract,evm,txhashBytes)
	//if !res {
	//	return nil
	//}

	txHashLen := int(all[idx])
	idx = idx + 1
	txhashBytes :=  make([]byte,txHashLen)
	copy(txhashBytes,all[idx:])
	idx = idx + txHashLen

	res := verifyHash(all[0:verifyHsBegin],contract,evm,txhashBytes)
	if !res {
		return nil
	}

	sendValue, ok := new(big.Int).SetString(stampVal, 10)
	if !ok {
		log.Error("get stamp value big int fail:%s", stampVal)
		return  nil
	}

	kixH := crypto.Keccak256Hash(kix)
	storagedSendValue := evm.env.StateDB.GetStateByteArray(contract.Address(), kixH)

	if storagedSendValue != nil && len(storagedSendValue) != 0 {
		return nil
	} else  {

		verifyRes := crypto.VerifyRingSign(txhashBytes,PublicKeySet,KeyImage,[]*big.Int(W_random),[]*big.Int(Q_random))
		if verifyRes {

			evm.env.StateDB.SetStateByteArray(contract.Address(), kixH, sendValue.Bytes())
			//send the value to the miner
			evm.env.StateDB.AddBalance(evm.env.Coinbase, sendValue)
			return []byte("1")

		}
	}

	return nil

}

//////////////////////////genesis coin precompile contract/////////////////////////////////////////
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

}

func (c *wanCoinSC) RequiredGas(inputSize int) uint64 {
	return params.EcrecoverGas
}

const (
	Pre0dot1 = "100000000000000000"//0.1
	Pre0dot2 = "200000000000000000"//0.2
	Pre0dot5 = "500000000000000000"//0.5
	Pre1 = 	   "1000000000000000000"//1
	Pre2 = 	   "2000000000000000000"//2
	Pre5 = 	   "5000000000000000000"//5
	Pre10 =    "10000000000000000000"//10
	Pre20 =    "20000000000000000000"//20
	Pre50 =    "50000000000000000000"//50
	Pre100 =   "100000000000000000000"//100
)

func (c *wanCoinSC) Run(in []byte,contract *Contract,evm *Interpreter) []byte {

	if in[0]==WANCOIN_BUY {
		return c.buyCoin(in[1:],contract,evm)
	} else if in[0]==WANCOIN_GET_COINS {
		return c.getCoins(in[0:],contract,evm)
	} else if in[0]==WANCOIN_REFUND {
		return c.refund(in[0:],contract,evm)
	}

	return  nil
}

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)


func (c *wanCoinSC) buyCoin(in []byte,contract *Contract,evm *Interpreter) []byte {

	otaAddr,err:= keystore.WaddrToUncompressed(in)//input is wand address
	if err!= nil {
		return nil
	}

	contractAddr := common.HexToAddress(contract.value.String())
	otaAddrKey := common.BytesToHash(otaAddr[0:64])

	// prevent rebuy
	storagedOtaAddr := evm.env.StateDB.GetStateByteArray(contractAddr, otaAddrKey)
	if storagedOtaAddr != nil && len(storagedOtaAddr) != 0 && bytes.Equal(storagedOtaAddr, otaAddr) {
		return nil
	}

	evm.env.StateDB.SetStateByteArray(contractAddr, otaAddrKey, in)

	addrSrc := contract.CallerAddress

	balance := evm.env.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0{
		evm.env.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1")
	}

	return nil
}


func (c *wanCoinSC) getCoins(all []byte,contract *Contract,evm *Interpreter) []byte {
	length := len(all)
	temp := make([]byte,length)
	copy(temp,all[:])

	contractAddr := common.HexToAddress(contract.value.String())
	trie := evm.env.StateDB.StorageVmTrie(contractAddr)
	if trie==nil {
		return nil
	}

	return getOtaSet(trie,3,temp)
}


func (c *wanCoinSC) refund(all []byte,contract *Contract,evm *Interpreter) []byte {

	valLen := int(all[1])
	otaLen := hexutil.BytesToShort(all[2:4])

	refundValBytes := all[otaLen:int(otaLen)+valLen]

	vb := new (big.Int)
	vb.SetBytes(refundValBytes)
	otaContainerAddr := common.HexToAddress(vb.String())


	idx := int(otaLen) + valLen
	verifyHsBegin := idx //duplicate for hash verify

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

		otaAddrKey := common.BytesToHash(x[1:])
		storagedOtaAddr := evm.env.StateDB.GetStateByteArray(otaContainerAddr, otaAddrKey)
		if storagedOtaAddr == nil || len(storagedOtaAddr) == 0 {
			fmt.Print("not get coin in the set")
			return nil
		}

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

	txHashLen := int(all[idx])
	idx = idx + 1
	txhashBytes :=  make([]byte,txHashLen)
	copy(txhashBytes,all[idx:])
	idx = idx + txHashLen

	res := verifyHash(all[0:verifyHsBegin],contract,evm,txhashBytes)
	if !res {
		return nil
	}

	kixH := crypto.Keccak256Hash(kix)
	storagedRefundVal := evm.env.StateDB.GetStateByteArray(contract.Address(), kixH)
	if storagedRefundVal != nil && len(storagedRefundVal) != 0 {
		return nil
	} else  {

	   verifyRes := crypto.VerifyRingSign(txhashBytes,PublicKeySet,KeyImage,[]*big.Int(W_random),[]*big.Int(Q_random))
	   if verifyRes {

		    evm.env.StateDB.SetStateByteArray(contract.Address(), kixH, refundValBytes)
			addrSrc := contract.CallerAddress
			evm.env.StateDB.AddBalance(addrSrc, vb)
			return []byte("1")

		}
	}

	return nil
}



func getOtaSet(dataTrie *trie.SecureTrie,stampNUm int, otaAddr []byte) []byte {
	if dataTrie == nil {
		return nil
	}

	stampSet := make([]byte,stampNUm*OTA_ADDR_LEN)
	rnd := rand.Intn(100) + 1

	it := trie.NewIterator(dataTrie.NodeIterator(nil))
	count :=0
	i := 0
	for {

		for it.Next() {
			count ++
			if count %rnd == 0&&i<stampNUm {
				idx := i*OTA_ADDR_LEN
				copy(stampSet[idx:],it.Value) //key is the ota address,value is the dump value
				i++
			}

			if i >= stampNUm{
				return  stampSet
			}
		}

		it = trie.NewIterator(dataTrie.NodeIterator(nil))
	}

	return nil
}


func verifySig(hashData []byte,publicKeys []*ecdsa.PublicKey,sigData []byte) bool {

	allHash := common.BytesToHash(hashData)
	pubLen := len(publicKeys)
	pubKey, err := crypto.SigToPub(allHash[:], sigData)
	if err != nil {
		return false
	}

	i := 0
	for i = 0;i < pubLen; i++ {
		if bytes.Equal(crypto.FromECDSAPub(pubKey), crypto.FromECDSAPub(publicKeys[i])) {
			return  true
		}
	}

	return false

}

func verifyHash(all []byte,contract *Contract,evm *Interpreter,hashOrig []byte) bool {

	from := contract.caller.Address()
	hashBytes := make([]byte,len(from[:])+ len(all))//the use addr and the tx.data[0:4] as the hash input for ring sig verify
	copy(hashBytes,from[:])
	copy(hashBytes[common.AddressLength:],all)
	//this hash is used to veriy the sender
	hcal := common.BytesToHash(hashBytes).Bytes()

	if bytes.Equal(hashOrig,hcal) {
		return true
	}

	return  false

}


func verifyContractOtaSender(callData []byte,sig []byte) bool {

	inputAbi, err := abi.JSON(strings.NewReader(`[{"constant":false,"inputs":[{"name":"_from","type":"string"},{"name":"_to","type":"string"},{"name":"_value","type":"uint256"}],"name":"otatransfer","outputs":[{"name":"From","type":"string"},{"name":"To","type":"string"},{"name":"Value","type":"uint256"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_receiver","type":"address"},{"name":"_amount","type":"uint256"}],"name":"mint","outputs":[],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":true,"inputs":[{"name":"_owner","type":"string"}],"name":"otabalanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[],"name":"MINT_CALLER","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":false,"inputs":[{"name":"_receiver","type":"string"},{"name":"_amount","type":"uint256"}],"name":"otamint","outputs":[],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"}]`))

	var outStruct struct{
		From string
		To string
		Value *big.Int
	}

	err = inputAbi.Unpack(&outStruct, "otatransfer", callData[4:])
	if err != nil {
		return false
	}

	callHash := common.BytesToHash(callData)
	pubKey, err := crypto.SigToPub(callHash[:], sig)
	if err != nil {
		return false
	}

	pubBytes := crypto.FromECDSAPub(pubKey)
	senderBytes,_:= keystore.WaddrToUncompressed( common.Hex2Bytes(outStruct.From[2:]))

	if bytes.Equal(pubBytes[1:65],senderBytes[0:64]) {
		return true
	}

	return false

}

func VerifyContractOtaSender(callData []byte,sig []byte) bool {
	return verifyContractOtaSender(callData,sig)
}