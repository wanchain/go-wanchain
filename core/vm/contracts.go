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

	"bytes"
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"golang.org/x/crypto/ripemd160"
	"strings"
)

// Precompiled contract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	RequiredGas(inputSize int) uint64                              // RequiredPrice calculates the contract gas use
	Run(input []byte, contract *Contract, evm *Interpreter) []byte // Run runs the precompiled contract
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
func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract, evm *Interpreter) (ret []byte, err error) {

	gas := p.RequiredGas(len(input))
	if contract.UseGas(gas) {

		ret = p.Run(input, contract, evm)
		if ret != nil {
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

func (c *ecrecover) Run(in []byte, contract *Contract, evm *Interpreter) []byte {
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
func (c *sha256hash) Run(in []byte, contract *Contract, evm *Interpreter) []byte {
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
func (c *ripemd160hash) Run(in []byte, contract *Contract, evm *Interpreter) []byte {
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

func (c *dataCopy) Run(in []byte, contract *Contract, evm *Interpreter) []byte {

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

	WAN_BUY_STAMP    = byte(3)
	WAN_VERIFY_STAMP = byte(4)
	WAN_STAMP_SET    = byte(5)

	WAN_STAMP_DOT1 = "10000000000000000" //0.01
	WAN_STAMP_DOT2 = "20000000000000000" //0.02
	WAN_STAMP_DOT5 = "50000000000000000" //0.05

	OTA_ADDR_LEN = 128
)

var (
	coinSCDefinition = `
	[
  {
    "constant": false,
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {
        "name": "OtaAddr",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ],
    "name": "buyCoinNote",
    "outputs": [
      {
        "name": "OtaAddr",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ]
  },
  {
    "constant": false,
    "type": "function",
    "inputs": [
      {
        "name": "RingSignedData",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ],
    "name": "refundCoin",
    "outputs": [
      {
        "name": "RingSignedData",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ]
  },
  {
    "constant": false,
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [],
    "name": "getCoins",
    "outputs": [
      {
        "name": "Value",
        "type": "uint256"
      }
    ]
  }
]`
	stampSCDefinition = `
	[
  {
    "constant": false,
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {
        "name": "OtaAddr",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ],
    "name": "buyStamp",
    "outputs": [
      {
        "name": "OtaAddr",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ]
  },
  {
    "constant": false,
    "type": "function",
    "inputs": [
      {
        "name": "RingSignedData",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ],
    "name": "refundCoin",
    "outputs": [
      {
        "name": "RingSignedData",
        "type": "string"
      },
      {
        "name": "Value",
        "type": "uint256"
      }
    ]
  },
  {
    "constant": false,
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [],
    "name": "getCoins",
    "outputs": [
      {
        "name": "Value",
        "type": "uint256"
      }
    ]
  }
]`

	coinAbi, errCoinSCInit               = abi.JSON(strings.NewReader(coinSCDefinition))
	buyIdArr, refundIdArr, getCoinsIdArr [4]byte

	stampAbi, errStampSCInit = abi.JSON(strings.NewReader(stampSCDefinition))
	stBuyId                  [4]byte
)

func init() {
	if errCoinSCInit != nil || errStampSCInit != nil {
		// TODO: refact panic
	}

	copy(buyIdArr[:], coinAbi.Methods["buyCoinNote"].Id())
	copy(refundIdArr[:], coinAbi.Methods["refundCoin"].Id())
	copy(getCoinsIdArr[:], coinAbi.Methods["getCoins"].Id())

	copy(stBuyId[:], stampAbi.Methods["buyStamp"].Id())
}

type wanchainStampSC struct {
}

func (c *wanchainStampSC) RequiredGas(inputSize int) uint64 {
	return 0
}

func (c *wanchainStampSC) Run(in []byte, contract *Contract, evm *Interpreter) []byte {
	if in==nil || len(in)<4 {
		return nil
	}
	
	var methodId [4]byte
	copy(methodId[:], in[:4])

	if methodId == stBuyId {
		return c.buyStamp(in[4:], contract, evm)
	}

	return nil
}

func (c *wanchainStampSC) buyStamp(in []byte, contract *Contract, evm *Interpreter) []byte {
	var StampInput struct {
		OtaAddr string
		Value   *big.Int
	}

	err := stampAbi.Unpack(&StampInput, "buyStamp", in)
	if err != nil {
		return nil
	}

	wanAddr, err := hexutil.Decode(StampInput.OtaAddr)
	if err != nil {
		return nil
	}

	add, err := AddOTAIfNotExit(evm.env.StateDB, contract.value, wanAddr)
	if err != nil || !add {
		return nil
	}

	addrSrc := contract.CallerAddress

	balance := evm.env.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0 {
		// Need check contract value in  build in value sets
		evm.env.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1")
	}

	return nil
}

func (c *wanchainStampSC) getStamps(in []byte, contract *Contract, evm *Interpreter) []byte {
	setEleNum := 3
	otaAX := in[:common.HashLength]
	otaWanAddrs, _, err := GetOTASet(evm.env.StateDB, otaAX, setEleNum)
	if err != nil {
		return nil
	}

	retBuf := make([]byte, 0, common.WAddressLength*setEleNum)
	for _, otaWanAddr := range otaWanAddrs {
		retBuf = append(retBuf, otaWanAddr...)
	}

	return retBuf
}

//////////////////////////genesis coin precompile contract/////////////////////////////////////////
/*  byte[0]: 0->buy
 *			 1->refund
 *
 *  byte[2]: if action is stampSet, this is the set number
 *  byte[3:]:the OTA-Address
 */
const (
	WANCOIN_BUY       = byte(0)
	WANCOIN_GET_COINS = byte(1)
	WANCOIN_REFUND    = byte(2)
)

type wanCoinSC struct {
}

func (c *wanCoinSC) RequiredGas(inputSize int) uint64 {
	return params.EcrecoverGas
}

const (
	Pre0dot1 = "100000000000000000"    //0.1
	Pre0dot2 = "200000000000000000"    //0.2
	Pre0dot5 = "500000000000000000"    //0.5
	Pre1     = "1000000000000000000"   //1
	Pre2     = "2000000000000000000"   //2
	Pre5     = "5000000000000000000"   //5
	Pre10    = "10000000000000000000"  //10
	Pre20    = "20000000000000000000"  //20
	Pre50    = "50000000000000000000"  //50
	Pre100   = "100000000000000000000" //100
)

func (c *wanCoinSC) Run(in []byte, contract *Contract, evm *Interpreter) []byte {
	if in==nil || len(in)<4 {
		return nil
	}
	
	var methodIdArr [4]byte
	copy(methodIdArr[:], in[:4])

	if methodIdArr == buyIdArr {
		return c.buyCoin(in[4:], contract, evm)
	} else if methodIdArr == getCoinsIdArr {
		return c.getCoins(in[4:], contract, evm)
	} else if methodIdArr == refundIdArr {
		return c.refund(in[4:], contract, evm)
	}

	return nil
}

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

func (c *wanCoinSC) buyCoin(in []byte, contract *Contract, evm *Interpreter) []byte {
	var outStruct struct {
		OtaAddr string
		Value   *big.Int
	}

	err := coinAbi.Unpack(&outStruct, "buyCoinNote", in)
	if err != nil {
		return nil
	}

	wanAddr, err := hexutil.Decode(outStruct.OtaAddr)
	if err != nil {
		return nil
	}

	add, err := AddOTAIfNotExit(evm.env.StateDB, contract.value, wanAddr)
	if err != nil || !add {
		return nil
	}

	addrSrc := contract.CallerAddress

	balance := evm.env.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0 {
		// Need check contract value in  build in value sets
		evm.env.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1")
	}

	return nil
}

func (c *wanCoinSC) getCoins(all []byte, contract *Contract, evm *Interpreter) []byte {
	setEleNum := 3
	otaAX := all[:common.HashLength]
	otaWanAddrs, _, err := GetOTASet(evm.env.StateDB, otaAX, setEleNum)
	if err != nil {
		return nil
	}

	retBuf := make([]byte, 0, common.WAddressLength*setEleNum)
	for _, otaWanAddr := range otaWanAddrs {
		retBuf = append(retBuf, otaWanAddr...)
	}

	return retBuf
}

func DecodeRingSignOut(s string) (error, []*ecdsa.PublicKey, *ecdsa.PublicKey, []*big.Int, []*big.Int) {
	ss := strings.Split(s, "+")
	ps := ss[0]
	k := ss[1]
	ws := ss[2]
	qs := ss[3]

	pa := strings.Split(ps, "&")
	publickeys := make([]*ecdsa.PublicKey, 0)
	for _, pi := range pa {
		publickeys = append(publickeys, crypto.ToECDSAPub(common.FromHex(pi)))
	}
	keyimgae := crypto.ToECDSAPub(common.FromHex(k))
	wa := strings.Split(ws, "&")
	w := make([]*big.Int, 0)
	for _, wi := range wa {
		bi, _ := hexutil.DecodeBig(wi)
		w = append(w, bi)
	}
	qa := strings.Split(qs, "&")
	q := make([]*big.Int, 0)
	for _, qi := range qa {
		bi, _ := hexutil.DecodeBig(qi)
		q = append(q, bi)
	}
	return nil, publickeys, keyimgae, w, q
}

func (c *wanCoinSC) refund(all []byte, contract *Contract, evm *Interpreter) []byte {
	var RefundStruct struct {
		RingSignedData string
		Value          *big.Int
	}

	err := coinAbi.Unpack(&RefundStruct, "refundCoin", all)
	if err != nil {
		return nil
	}

	err, publickeys, keyimage, ws, qs := DecodeRingSignOut(RefundStruct.RingSignedData)
	if err != nil {
		return nil
	}

	otaAXs := make([][]byte, 0, len(publickeys))
	for i := 0; i < len(publickeys); i++ {
		pkBytes := crypto.FromECDSAPub(publickeys[i])
		otaAXs = append(otaAXs, pkBytes[1:1+common.HashLength])
	}

	exit, balanceGet, unexit, err := BatCheckOTAExit(evm.env.StateDB, otaAXs)
	if !exit || balanceGet == nil || balanceGet.Cmp(RefundStruct.Value) != 0 {
		if err != nil {
			log.Warn("verify mix ota fail. err:%s", err.Error())
		}
		if unexit != nil {
			log.Warn("invalid mix ota:%s", common.ToHex(unexit))
		}
		if balanceGet != nil && balanceGet.Cmp(RefundStruct.Value) != 0 {
			log.Warn("balance getting from ota is wrong! get:%s, expect:%s",
				balanceGet.String(), RefundStruct.Value.String())
		}

		return nil
	}

	kix := crypto.FromECDSAPub(keyimage)
	exit, _, err = CheckOTAImageExit(evm.env.StateDB, kix)
	if err != nil || exit {
		return nil
	}

	b := crypto.VerifyRingSign(contract.CallerAddress.Bytes(), publickeys, keyimage, ws, qs)
	if !b {
	} else { // For test
		AddOTAImage(evm.env.StateDB, kix, RefundStruct.Value.Bytes())

		addrSrc := contract.CallerAddress
		evm.env.StateDB.AddBalance(addrSrc, RefundStruct.Value)
		return []byte("1")
	}

	return nil
}

func verifyHash(all []byte, contract *Contract, evm *Interpreter, hashOrig []byte) bool {

	from := contract.caller.Address()
	hashBytes := make([]byte, len(from[:])+len(all)) //the use addr and the tx.data[0:4] as the hash input for ring sig verify
	copy(hashBytes, from[:])
	copy(hashBytes[common.AddressLength:], all)
	//this hash is used to veriy the sender
	hcal := common.BytesToHash(hashBytes).Bytes()

	if bytes.Equal(hashOrig, hcal) {
		return true
	}

	return false

}
