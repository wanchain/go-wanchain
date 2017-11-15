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
	"errors"
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256"
	"github.com/wanchain/go-wanchain/params"
	"golang.org/x/crypto/ripemd160"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"strings"
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/log"
)

// RunPrecompiledContract runs and evaluates the output of a precompiled contract.
func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract,evm *EVM) (ret []byte, err error) {
	gas := p.RequiredGas(input)
	if contract.UseGas(gas) {
		return p.Run(input, contract, evm)
	}
	return nil, ErrOutOfGas
}

// ECRECOVER implemented as a native contract.
type ecrecover struct{}

func (c *ecrecover) RequiredGas(input []byte) uint64 {
	return params.EcrecoverGas
}

func (c *ecrecover) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	const ecRecoverInputLength = 128

	input = common.RightPadBytes(input, ecRecoverInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])
	v := input[63] - 27

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(input[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(input[:32], append(input[64:128], v))
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32), nil
}

// SHA256 implemented as a native contract.
type sha256hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *sha256hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Sha256PerWordGas + params.Sha256BaseGas
}
func (c *sha256hash) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	h := sha256.Sum256(input)
	return h[:], nil
}

// RIPMED160 implemented as a native contract.
type ripemd160hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *ripemd160hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Ripemd160PerWordGas + params.Ripemd160BaseGas
}
func (c *ripemd160hash) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	ripemd := ripemd160.New()
	ripemd.Write(input)
	return common.LeftPadBytes(ripemd.Sum(nil), 32), nil
}

// data copy implemented as a native contract.
type dataCopy struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *dataCopy) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.IdentityPerWordGas + params.IdentityBaseGas
}
func (c *dataCopy) Run(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	return in, nil
}

// bigModExp implements a native big integer exponential modular operation.
type bigModExp struct{}

var (
	big1      = big.NewInt(1)
	big4      = big.NewInt(4)
	big8      = big.NewInt(8)
	big16     = big.NewInt(16)
	big32     = big.NewInt(32)
	big64     = big.NewInt(64)
	big96     = big.NewInt(96)
	big480    = big.NewInt(480)
	big1024   = big.NewInt(1024)
	big3072   = big.NewInt(3072)
	big199680 = big.NewInt(199680)
)

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bigModExp) RequiredGas(input []byte) uint64 {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32))
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32))
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32))
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Retrieve the head 32 bytes of exp for the adjusted exponent length
	var expHead *big.Int
	if big.NewInt(int64(len(input))).Cmp(baseLen) <= 0 {
		expHead = new(big.Int)
	} else {
		if expLen.Cmp(big32) > 0 {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), 32))
		} else {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), expLen.Uint64()))
		}
	}
	// Calculate the adjusted exponent length
	var msb int
	if bitlen := expHead.BitLen(); bitlen > 0 {
		msb = bitlen - 1
	}
	adjExpLen := new(big.Int)
	if expLen.Cmp(big32) > 0 {
		adjExpLen.Sub(expLen, big32)
		adjExpLen.Mul(big8, adjExpLen)
	}
	adjExpLen.Add(adjExpLen, big.NewInt(int64(msb)))

	// Calculate the gas cost of the operation
	gas := new(big.Int).Set(math.BigMax(modLen, baseLen))
	switch {
	case gas.Cmp(big64) <= 0:
		gas.Mul(gas, gas)
	case gas.Cmp(big1024) <= 0:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big4),
			new(big.Int).Sub(new(big.Int).Mul(big96, gas), big3072),
		)
	default:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big16),
			new(big.Int).Sub(new(big.Int).Mul(big480, gas), big199680),
		)
	}
	gas.Mul(gas, math.BigMax(adjExpLen, big1))
	gas.Div(gas, new(big.Int).SetUint64(params.ModExpQuadCoeffDiv))

	if gas.BitLen() > 64 {
		return math.MaxUint64
	}
	return gas.Uint64()
}

func (c *bigModExp) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32)).Uint64()
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32)).Uint64()
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32)).Uint64()
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Handle a special case when both the base and mod length is zero
	if baseLen == 0 && modLen == 0 {
		return []byte{}, nil
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big.Int).SetBytes(getData(input, 0, baseLen))
		exp  = new(big.Int).SetBytes(getData(input, baseLen, expLen))
		mod  = new(big.Int).SetBytes(getData(input, baseLen+expLen, modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen)), nil
	}
	return common.LeftPadBytes(base.Exp(base, exp, mod).Bytes(), int(modLen)), nil
}

var (
	// errNotOnCurve is returned if a point being unmarshalled as a bn256 elliptic
	// curve point is not on the curve.
	errNotOnCurve = errors.New("point not on elliptic curve")

	// errInvalidCurvePoint is returned if a point being unmarshalled as a bn256
	// elliptic curve point is invalid.
	errInvalidCurvePoint = errors.New("invalid elliptic curve point")
)

// newCurvePoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newCurvePoint(blob []byte) (*bn256.G1, error) {
	p, onCurve := new(bn256.G1).Unmarshal(blob)
	if !onCurve {
		return nil, errNotOnCurve
	}
	gx, gy, _, _ := p.CurvePoints()
	if gx.Cmp(bn256.P) >= 0 || gy.Cmp(bn256.P) >= 0 {
		return nil, errInvalidCurvePoint
	}
	return p, nil
}

// newTwistPoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newTwistPoint(blob []byte) (*bn256.G2, error) {
	p, onCurve := new(bn256.G2).Unmarshal(blob)
	if !onCurve {
		return nil, errNotOnCurve
	}
	x2, y2, _, _ := p.CurvePoints()
	if x2.Real().Cmp(bn256.P) >= 0 || x2.Imag().Cmp(bn256.P) >= 0 ||
		y2.Real().Cmp(bn256.P) >= 0 || y2.Imag().Cmp(bn256.P) >= 0 {
		return nil, errInvalidCurvePoint
	}
	return p, nil
}

// bn256Add implements a native elliptic curve point addition.
type bn256Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Add) RequiredGas(input []byte) uint64 {
	return params.Bn256AddGas
}

func (c *bn256Add) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	x, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	y, err := newCurvePoint(getData(input, 64, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.Add(x, y)
	return res.Marshal(), nil
}

// bn256ScalarMul implements a native elliptic curve scalar multiplication.
type bn256ScalarMul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256ScalarMul) RequiredGas(input []byte) uint64 {
	return params.Bn256ScalarMulGas
}

func (c *bn256ScalarMul) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	p, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.ScalarMult(p, new(big.Int).SetBytes(getData(input, 64, 32)))
	return res.Marshal(), nil
}

var (
	// true32Byte is returned if the bn256 pairing check succeeds.
	true32Byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// false32Byte is returned if the bn256 pairing check fails.
	false32Byte = make([]byte, 32)

	// errBadPairingInput is returned if the bn256 pairing input is invalid.
	errBadPairingInput = errors.New("bad elliptic curve pairing size")
)

// bn256Pairing implements a pairing pre-compile for the bn256 curve
type bn256Pairing struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Pairing) RequiredGas(input []byte) uint64 {
	return params.Bn256PairingBaseGas + uint64(len(input)/192)*params.Bn256PairingPerPointGas
}

func (c *bn256Pairing) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	// Handle some corner cases cheaply
	if len(input)%192 > 0 {
		return nil, errBadPairingInput
	}
	// Convert the input into a set of coordinates
	var (
		cs []*bn256.G1
		ts []*bn256.G2
	)
	for i := 0; i < len(input); i += 192 {
		c, err := newCurvePoint(input[i : i+64])
		if err != nil {
			return nil, err
		}
		t, err := newTwistPoint(input[i+64 : i+192])
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
		ts = append(ts, t)
	}
	// Execute the pairing checks and return the results
	if bn256.PairingCheck(cs, ts) {
		return true32Byte, nil
	}
	return false32Byte, nil
}


///////////////////////for wan privacy tx /////////////////////////////////////////////////////////

/////////////////////////////////////added by jqg ///////////////////////////////////


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

	errBuyCoin = errors.New("error in buy coin")
	errRefundCoin = errors.New("error in refund coin")

	errBuyStamp = errors.New("error in buy stamp")

	errParameters = errors.New("error parameters")
	errMethodId = errors.New("error method id")

	errBalance = errors.New("balance is insufficient")

	errStampValue = errors.New("stamp value is not support")

	errCoinValue = errors.New("wancoin value is not support")

	StampValueSet = make (map[string]string,5)
	WanCoinValueSet = make (map[string]string,10)

)

const (
	Wancoindot1 = "100000000000000000"    //0.1
	Wancoindot2 = "200000000000000000"    //0.2
	Wancoindot5 = "500000000000000000"    //0.5
	Wancoin1     = "1000000000000000000"   //1
	Wancoin2     = "2000000000000000000"   //2
	Wancoin5     = "5000000000000000000"   //5
	Wancoin10    = "10000000000000000000"  //10
	Wancoin20    = "20000000000000000000"  //20
	Wancoin50    = "50000000000000000000"  //50
	Wancoin100   = "100000000000000000000" //100

	WanStamp0dot1 = "10000000000000000" //0.01
	WanStamp0dot2 = "20000000000000000" //0.02
	WanStamp0dot5 = "50000000000000000"  //0.05
)



func init() {
	if errCoinSCInit != nil || errStampSCInit != nil {
		// TODO: refact panic
	}

	copy(buyIdArr[:], coinAbi.Methods["buyCoinNote"].Id())
	copy(refundIdArr[:], coinAbi.Methods["refundCoin"].Id())
	copy(getCoinsIdArr[:], coinAbi.Methods["getCoins"].Id())

	copy(stBuyId[:], stampAbi.Methods["buyStamp"].Id())

	sval01,_:= new(big.Int).SetString(WanStamp0dot1,10)
	StampValueSet[sval01.Text(16)] = WanStamp0dot1

	sval02,_:= new(big.Int).SetString(WanStamp0dot2,10)
	StampValueSet[sval02.Text(16)] = WanStamp0dot2

	sval05,_:= new(big.Int).SetString(WanStamp0dot5,10)
	StampValueSet[sval05.Text(16)] = WanStamp0dot5


	cval01,_:= new(big.Int).SetString(Wancoindot1,10)
	WanCoinValueSet[cval01.Text(16)] = Wancoindot1

	cval02,_:= new(big.Int).SetString(Wancoindot2,10)
	WanCoinValueSet[cval02.Text(16)] = Wancoindot2

	cval05,_:= new(big.Int).SetString(Wancoindot5,10)
	WanCoinValueSet[cval05.Text(16)] = Wancoindot5

	cval1,_:= new(big.Int).SetString(Wancoin1,10)
	WanCoinValueSet[cval1.Text(16)] = Wancoin1

	cval2,_:= new(big.Int).SetString(Wancoin2,10)
	WanCoinValueSet[cval2.Text(16)] = Wancoin2

	cval5,_:= new(big.Int).SetString(Wancoin5,10)
	WanCoinValueSet[cval5.Text(16)] = Wancoin5

	cval10,_:= new(big.Int).SetString(Wancoin10,10)
	WanCoinValueSet[cval10.Text(16)] = Wancoin10

	cval20,_:= new(big.Int).SetString(Wancoin20,10)
	WanCoinValueSet[cval20.Text(16)] = Wancoin20

	cval50,_:= new(big.Int).SetString(Wancoin50,10)
	WanCoinValueSet[cval50.Text(16)] = Wancoin50

	cval100,_:= new(big.Int).SetString(Wancoin100,10)
	WanCoinValueSet[cval100.Text(16)] = Wancoin100
}

type wanchainStampSC struct {}

func (c *wanchainStampSC) RequiredGas(input []byte) uint64 {
	return params.CreateDataGas
}

func (c *wanchainStampSC) Run(in []byte, contract *Contract, env *EVM)([]byte, error) {
	if in==nil || len(in)<4 {
		return nil,errParameters
	}

	var methodId [4]byte
	copy(methodId[:], in[:4])

	if methodId == stBuyId {
		return c.buyStamp(in[4:], contract, env)
	}

	return nil,errMethodId
}

func (c *wanchainStampSC) buyStamp(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var StampInput struct {
		OtaAddr string
		Value   *big.Int
	}

	err := stampAbi.Unpack(&StampInput, "buyStamp", in)
	if err != nil {
		return nil,errBuyStamp
	}

	_,ok := StampValueSet[contract.value.Text(16)]
	if !ok {
		return nil,errStampValue
	}

	wanAddr, err := hexutil.Decode(StampInput.OtaAddr)
	if err != nil {
		return nil,errBuyStamp
	}

	add, err := AddOTAIfNotExit(evm.StateDB, contract.value, wanAddr)
	if err != nil || !add {
		return nil,errBuyStamp
	}

	addrSrc := contract.CallerAddress

	balance := evm.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0 {
		// Need check contract value in  build in value sets
		evm.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1"),nil
	} else {

		return nil,errBalance
	}
}


type wanCoinSC struct {
}

func (c *wanCoinSC) RequiredGas(input []byte) uint64 {
	var methodIdArr [4]byte
	copy(methodIdArr[:], input[:4])

    if methodIdArr == refundIdArr {

		var RefundStruct struct {
			RingSignedData string
			Value          *big.Int
		}

		err := coinAbi.Unpack(&RefundStruct, "refundCoin", input[4:])
		if err != nil {
			return params.RequiredGasPerMixPub
		}

		err, publickeys, _, _, _ := DecodeRingSignOut(RefundStruct.RingSignedData)
		if err != nil {
			return params.RequiredGasPerMixPub
		}

		mixLen := len(publickeys)
		ringSigDiffRequiredGas := params.RequiredGasPerMixPub * (uint64(mixLen))

		return ringSigDiffRequiredGas

	} else {
		return params.CreateDataGas
	}

}


func (c *wanCoinSC) Run(in []byte, contract *Contract, evm *EVM) ([]byte, error){
	if in==nil || len(in)<4 {
		return nil,errParameters
	}

	var methodIdArr [4]byte
	copy(methodIdArr[:], in[:4])

	if methodIdArr == buyIdArr {
		return c.buyCoin(in[4:], contract, evm)
	} else if methodIdArr == refundIdArr {
		return c.refund(in[4:], contract, evm)
	}

	return nil,errMethodId
}

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

func (c *wanCoinSC) buyCoin(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var outStruct struct {
		OtaAddr string
		Value   *big.Int
	}

	err := coinAbi.Unpack(&outStruct, "buyCoinNote", in)
	if err != nil {
		return nil,errBuyCoin
	}

	_,ok := WanCoinValueSet[contract.value.Text(16)]
	if !ok {
		return nil,errStampValue
	}

	wanAddr, err := hexutil.Decode(outStruct.OtaAddr)
	if err != nil {
		return nil,errBuyCoin
	}

	add, err := AddOTAIfNotExit(evm.StateDB, contract.value, wanAddr)
	if err != nil || !add {
		return nil,errBuyCoin
	}

	addrSrc := contract.CallerAddress

	balance := evm.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0 {
		// Need check contract value in  build in value sets
		evm.StateDB.SubBalance(addrSrc, contract.value)
		return []byte("1"),nil
	} else {
		return nil,errBalance
	}
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

func (c *wanCoinSC) refund(all []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var RefundStruct struct {
		RingSignedData string
		Value          *big.Int
	}

	err := coinAbi.Unpack(&RefundStruct, "refundCoin", all)
	if err != nil {
		return nil,errRefundCoin
	}

	err, publickeys, keyimage, ws, qs := DecodeRingSignOut(RefundStruct.RingSignedData)
	if err != nil {
		return nil,errRefundCoin
	}

	otaAXs := make([][]byte, 0, len(publickeys))
	for i := 0; i < len(publickeys); i++ {
		pkBytes := crypto.FromECDSAPub(publickeys[i])
		otaAXs = append(otaAXs, pkBytes[1:1+common.HashLength])
	}

	exit, balanceGet, unexit, err := BatCheckOTAExit(evm.StateDB, otaAXs)
	if !exit || balanceGet == nil || balanceGet.Cmp(RefundStruct.Value) != 0 {
		if err != nil {
			log.Warn("verify mix ota fail", "err", err.Error())
		}
		if unexit != nil {
			log.Warn("invalid mix ota", "invalid ota", common.ToHex(unexit))
		}
		if balanceGet != nil && balanceGet.Cmp(RefundStruct.Value) != 0 {
			log.Warn("balance getting from ota is wrong", "get", balanceGet.String(),
				"expect", RefundStruct.Value.String())
		} else {
			return nil,errBalance
		}
	}

	kix := crypto.FromECDSAPub(keyimage)
	exit, _, err = CheckOTAImageExit(evm.StateDB, kix)
	if err != nil || exit {
		return nil,errRefundCoin
	}

	b := crypto.VerifyRingSign(contract.CallerAddress.Bytes(), publickeys, keyimage, ws, qs)
	if b {

		AddOTAImage(evm.StateDB, kix, RefundStruct.Value.Bytes())

		addrSrc := contract.CallerAddress
		evm.StateDB.AddBalance(addrSrc, RefundStruct.Value)
		return []byte("1"),nil
	} else {
		return nil,errRefundCoin
	}

}
