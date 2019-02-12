// Copyright 2018 Wanchain Foundation Ltd
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

	"crypto/ecdsa"
	"strings"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"golang.org/x/crypto/ripemd160"
	"fmt"
)

// RunPrecompiledContract runs and evaluates the output of a precompiled contract.
func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract, evm *EVM) (ret []byte, err error) {
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

func (c *ecrecover) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
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

func (c *sha256hash) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
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

func (c *ripemd160hash) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
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

func (c *dataCopy) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
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

func (c *bigModExp) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

var (
	// errNotOnCurve is returned if a point being unmarshalled as a bn256 elliptic
	// curve point is not on the curve.
	errNotOnCurve = errors.New("point not on elliptic curve")

	// errInvalidCurvePoint is returned if a point being unmarshalled as a bn256
	// elliptic curve point is invalid.
	errInvalidCurvePoint = errors.New("invalid elliptic curve point")

	// invalid ring signed info
	ErrInvalidRingSigned = errors.New("invalid ring signed info")
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

func (c *bn256Add) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
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

func (c *bn256ScalarMul) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
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

func (c *bn256Pairing) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

///////////////////////for wan privacy tx /////////////////////////////////////////////////////////

var (
	coinSCDefinition = `
	[{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [{"name": "OtaAddr","type":"string"},{"name": "Value","type": "uint256"}],"name": "buyCoinNote","outputs": [{"name": "OtaAddr","type":"string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","inputs": [{"name":"RingSignedData","type": "string"},{"name": "Value","type": "uint256"}],"name": "refundCoin","outputs": [{"name": "RingSignedData","type": "string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [],"name": "getCoins","outputs": [{"name":"Value","type": "uint256"}]}]`

	stampSCDefinition = `[{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [{"name":"OtaAddr","type": "string"},{"name": "Value","type": "uint256"}],"name": "buyStamp","outputs": [{"name": "OtaAddr","type": "string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","inputs": [{"name": "RingSignedData","type": "string"},{"name": "Value","type": "uint256"}],"name": "refundCoin","outputs": [{"name": "RingSignedData","type": "string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [],"name": "getCoins","outputs": [{"name": "Value","type": "uint256"}]}]`

	coinAbi, errCoinSCInit               = abi.JSON(strings.NewReader(coinSCDefinition))
	buyIdArr, refundIdArr, getCoinsIdArr [4]byte

	stampAbi, errStampSCInit = abi.JSON(strings.NewReader(stampSCDefinition))
	stBuyId                  [4]byte

	errBuyCoin    = errors.New("error in buy coin")
	errRefundCoin = errors.New("error in refund coin")

	errBuyStamp = errors.New("error in buy stamp")

	errParameters = errors.New("error parameters")
	errMethodId   = errors.New("error method id")

	errBalance = errors.New("balance is insufficient")

	errStampValue = errors.New("stamp value is not support")

	errCoinValue = errors.New("wancoin value is not support")

	ErrMismatchedValue = errors.New("mismatched wancoin value")

	ErrInvalidOTASet = errors.New("invalid OTA mix set")

	ErrOTAReused = errors.New("OTA is reused")

	StampValueSet   = make(map[string]string, 5)
	WanCoinValueSet = make(map[string]string, 10)
)

const (
	Wancoin10  = "10000000000000000000"  //10
	Wancoin20  = "20000000000000000000"  //20
	Wancoin50  = "50000000000000000000"  //50
	Wancoin100 = "100000000000000000000" //100

	Wancoin200   = "200000000000000000000"   //200
	Wancoin500   = "500000000000000000000"   //500
	Wancoin1000  = "1000000000000000000000"  //1000
	Wancoin5000  = "5000000000000000000000"  //5000
	Wancoin50000 = "50000000000000000000000" //50000

	WanStampdot001 = "1000000000000000" //0.001
	WanStampdot002 = "2000000000000000" //0.002
	WanStampdot005 = "5000000000000000" //0.005

	WanStampdot003 = "3000000000000000" //0.003
	WanStampdot006 = "6000000000000000" //0.006
	WanStampdot009 = "9000000000000000" //0.009

	WanStampdot03 = "30000000000000000"  //0.03
	WanStampdot06 = "60000000000000000"  //0.06
	WanStampdot09 = "90000000000000000"  //0.09
	WanStampdot2  = "200000000000000000" //0.2
	WanStampdot3  = "300000000000000000" //0.3
	WanStampdot5  = "500000000000000000" //0.5

)

func init() {
	if errCoinSCInit != nil || errStampSCInit != nil {
		panic("err in coin sc initialize or stamp error initialize ")
	}

	copy(buyIdArr[:], coinAbi.Methods["buyCoinNote"].Id())
	copy(refundIdArr[:], coinAbi.Methods["refundCoin"].Id())
	copy(getCoinsIdArr[:], coinAbi.Methods["getCoins"].Id())

	copy(stBuyId[:], stampAbi.Methods["buyStamp"].Id())

	svaldot001, _ := new(big.Int).SetString(WanStampdot001, 10)
	StampValueSet[svaldot001.Text(16)] = WanStampdot001

	svaldot002, _ := new(big.Int).SetString(WanStampdot002, 10)
	StampValueSet[svaldot002.Text(16)] = WanStampdot002

	svaldot005, _ := new(big.Int).SetString(WanStampdot005, 10)
	StampValueSet[svaldot005.Text(16)] = WanStampdot005

	svaldot003, _ := new(big.Int).SetString(WanStampdot003, 10)
	StampValueSet[svaldot003.Text(16)] = WanStampdot003

	svaldot006, _ := new(big.Int).SetString(WanStampdot006, 10)
	StampValueSet[svaldot006.Text(16)] = WanStampdot006

	svaldot009, _ := new(big.Int).SetString(WanStampdot009, 10)
	StampValueSet[svaldot009.Text(16)] = WanStampdot009

	svaldot03, _ := new(big.Int).SetString(WanStampdot03, 10)
	StampValueSet[svaldot03.Text(16)] = WanStampdot03

	svaldot06, _ := new(big.Int).SetString(WanStampdot06, 10)
	StampValueSet[svaldot06.Text(16)] = WanStampdot06

	svaldot09, _ := new(big.Int).SetString(WanStampdot09, 10)
	StampValueSet[svaldot09.Text(16)] = WanStampdot09

	svaldot2, _ := new(big.Int).SetString(WanStampdot2, 10)
	StampValueSet[svaldot2.Text(16)] = WanStampdot2

	svaldot3, _ := new(big.Int).SetString(WanStampdot3, 10)
	StampValueSet[svaldot3.Text(16)] = WanStampdot3

	svaldot5, _ := new(big.Int).SetString(WanStampdot5, 10)
	StampValueSet[svaldot5.Text(16)] = WanStampdot5

	cval10, _ := new(big.Int).SetString(Wancoin10, 10)
	WanCoinValueSet[cval10.Text(16)] = Wancoin10

	cval20, _ := new(big.Int).SetString(Wancoin20, 10)
	WanCoinValueSet[cval20.Text(16)] = Wancoin20

	cval50, _ := new(big.Int).SetString(Wancoin50, 10)
	WanCoinValueSet[cval50.Text(16)] = Wancoin50

	cval100, _ := new(big.Int).SetString(Wancoin100, 10)
	WanCoinValueSet[cval100.Text(16)] = Wancoin100

	cval200, _ := new(big.Int).SetString(Wancoin200, 10)
	WanCoinValueSet[cval200.Text(16)] = Wancoin200

	cval500, _ := new(big.Int).SetString(Wancoin500, 10)
	WanCoinValueSet[cval500.Text(16)] = Wancoin500

	cval1000, _ := new(big.Int).SetString(Wancoin1000, 10)
	WanCoinValueSet[cval1000.Text(16)] = Wancoin1000

	cval5000, _ := new(big.Int).SetString(Wancoin5000, 10)
	WanCoinValueSet[cval5000.Text(16)] = Wancoin5000

	cval50000, _ := new(big.Int).SetString(Wancoin50000, 10)
	WanCoinValueSet[cval50000.Text(16)] = Wancoin50000

}

type wanchainStampSC struct{}

func (c *wanchainStampSC) RequiredGas(input []byte) uint64 {
	// ota balance store gas + ota wanaddr store gas
	return params.SstoreSetGas * 2
}

func (c *wanchainStampSC) Run(in []byte, contract *Contract, env *EVM) ([]byte, error) {
	if len(in) < 4 {
		return nil, errParameters
	}

	var methodId [4]byte
	copy(methodId[:], in[:4])

	if methodId == stBuyId {
		return c.buyStamp(in[4:], contract, env)
	}

	return nil, errMethodId
}

func (c *wanchainStampSC) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	if stateDB == nil || signer == nil || tx == nil {
		return errParameters
	}

	payload := tx.Data()
	if len(payload) < 4 {
		return errParameters
	}

	var methodId [4]byte
	copy(methodId[:], payload[:4])
	if methodId == stBuyId {
		_, err := c.ValidBuyStampReq(stateDB, payload[4:], tx.Value())
		return err
	}

	return errParameters
}

func (c *wanchainStampSC) ValidBuyStampReq(stateDB StateDB, payload []byte, value *big.Int) (otaAddr []byte, err error) {
	if stateDB == nil || len(payload) == 0 || value == nil {
		return nil, errors.New("unknown error")
	}

	var StampInput struct {
		OtaAddr string
		Value   *big.Int
	}

	err = stampAbi.Unpack(&StampInput, "buyStamp", payload)
	if err != nil || StampInput.Value == nil {
		return nil, errBuyStamp
	}

	if StampInput.Value.Cmp(value) != 0 {
		return nil, ErrMismatchedValue
	}

	_, ok := StampValueSet[StampInput.Value.Text(16)]
	if !ok {
		return nil, errStampValue
	}

	wanAddr, err := hexutil.Decode(StampInput.OtaAddr)
	if err != nil {
		return nil, err
	}

	ax, err := GetAXFromWanAddr(wanAddr)
	exist, _, err := CheckOTAAXExist(stateDB, ax)
	if err != nil {
		return nil, err
	}

	if exist {
		return nil, ErrOTAReused
	}

	return wanAddr, nil
}

func (c *wanchainStampSC) buyStamp(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	wanAddr, err := c.ValidBuyStampReq(evm.StateDB, in, contract.value)
	if err != nil {
		return nil, err
	}

	add, err := AddOTAIfNotExist(evm.StateDB, contract.value, wanAddr)
	if err != nil || !add {
		return nil, errBuyStamp
	}

	addrSrc := contract.CallerAddress
	balance := evm.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0 {
		// Need check contract value in  build in value sets
		evm.StateDB.SubBalance(addrSrc, contract.value)
		return []byte{1}, nil
	} else {
		return nil, errBalance
	}
}

type wanCoinSC struct {
}

func (c *wanCoinSC) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

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

		// ringsign compute gas + ota image key store setting gas
		return ringSigDiffRequiredGas + params.SstoreSetGas

	} else {
		// ota balance store gas + ota wanaddr store gas
		return params.SstoreSetGas * 2
	}

}

func (c *wanCoinSC) Run(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(in) < 4 {
		return nil, errParameters
	}

	var methodIdArr [4]byte
	copy(methodIdArr[:], in[:4])

	if methodIdArr == buyIdArr {
		return c.buyCoin(in[4:], contract, evm)
	} else if methodIdArr == refundIdArr {
		return c.refund(in[4:], contract, evm)
	}

	return nil, errMethodId
}

func (c *wanCoinSC) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	if stateDB == nil || signer == nil || tx == nil {
		return errParameters
	}

	payload := tx.Data()
	if len(payload) < 4 {
		return errParameters
	}

	var methodIdArr [4]byte
	copy(methodIdArr[:], payload[:4])

	if methodIdArr == buyIdArr {
		_, err := c.ValidBuyCoinReq(stateDB, payload[4:], tx.Value())
		return err

	} else if methodIdArr == refundIdArr {
		from, err := types.Sender(signer, tx)
		if err != nil {
			return err
		}

		_, _, err = c.ValidRefundReq(stateDB, payload[4:], from.Bytes())
		return err
	}

	return errParameters
}

var (
	ether = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)

func (c *wanCoinSC) ValidBuyCoinReq(stateDB StateDB, payload []byte, txValue *big.Int) (otaAddr []byte, err error) {
	if stateDB == nil || len(payload) == 0 || txValue == nil {
		return nil, errors.New("unknown error")
	}

	var outStruct struct {
		OtaAddr string
		Value   *big.Int
	}

	err = coinAbi.Unpack(&outStruct, "buyCoinNote", payload)
	if err != nil || outStruct.Value == nil {
		return nil, errBuyCoin
	}

	if outStruct.Value.Cmp(txValue) != 0 {
		return nil, ErrMismatchedValue
	}

	_, ok := WanCoinValueSet[outStruct.Value.Text(16)]
	if !ok {
		return nil, errCoinValue
	}

	wanAddr, err := hexutil.Decode(outStruct.OtaAddr)
	if err != nil {
		return nil, err
	}

	ax, err := GetAXFromWanAddr(wanAddr)
	if err != nil {
		return nil, err
	}

	exist, _, err := CheckOTAAXExist(stateDB, ax)
	if err != nil {
		return nil, err
	}

	if exist {
		return nil, ErrOTAReused
	}

	return wanAddr, nil
}

func (c *wanCoinSC) buyCoin(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	otaAddr, err := c.ValidBuyCoinReq(evm.StateDB, in, contract.value)
	if err != nil {
		return nil, err
	}

	add, err := AddOTAIfNotExist(evm.StateDB, contract.value, otaAddr)
	if err != nil || !add {
		return nil, errBuyCoin
	}

	addrSrc := contract.CallerAddress
	balance := evm.StateDB.GetBalance(addrSrc)

	if balance.Cmp(contract.value) >= 0 {
		// Need check contract value in  build in value sets
		evm.StateDB.SubBalance(addrSrc, contract.value)
		return []byte{1}, nil
	} else {
		return nil, errBalance
	}
}

func (c *wanCoinSC) ValidRefundReq(stateDB StateDB, payload []byte, from []byte) (image []byte, value *big.Int, err error) {
	if stateDB == nil || len(payload) == 0 || len(from) == 0 {
		return nil, nil, errors.New("unknown error")
	}

	var RefundStruct struct {
		RingSignedData string
		Value          *big.Int
	}

	err = coinAbi.Unpack(&RefundStruct, "refundCoin", payload)
	if err != nil || RefundStruct.Value == nil {
		return nil, nil, errRefundCoin
	}

	ringSignInfo, err := FetchRingSignInfo(stateDB, from, RefundStruct.RingSignedData)
	if err != nil {
		return nil, nil, err
	}

	if ringSignInfo.OTABalance.Cmp(RefundStruct.Value) != 0 {
		return nil, nil, ErrMismatchedValue
	}

	kix := crypto.FromECDSAPub(ringSignInfo.KeyImage)
	exist, _, err := CheckOTAImageExist(stateDB, kix)
	if err != nil {
		return nil, nil, err
	}

	if exist {
		return nil, nil, ErrOTAReused
	}

	return kix, RefundStruct.Value, nil

}

func (c *wanCoinSC) refund(all []byte, contract *Contract, evm *EVM) ([]byte, error) {
	kix, value, err := c.ValidRefundReq(evm.StateDB, all, contract.CallerAddress.Bytes())
	if err != nil {
		fmt.Println("failed refund")
		fmt.Println(evm.BlockNumber)
		return nil, err
	}

	err = AddOTAImage(evm.StateDB, kix, value.Bytes())
	if err != nil {
		return nil, err
	}

	addrSrc := contract.CallerAddress
	evm.StateDB.AddBalance(addrSrc, value)
	return []byte{1}, nil

}

func DecodeRingSignOut(s string) (error, []*ecdsa.PublicKey, *ecdsa.PublicKey, []*big.Int, []*big.Int) {
	ss := strings.Split(s, "+")
	if len(ss) < 4 {
		return ErrInvalidRingSigned, nil, nil, nil, nil
	}

	ps := ss[0]
	k := ss[1]
	ws := ss[2]
	qs := ss[3]

	pa := strings.Split(ps, "&")
	publickeys := make([]*ecdsa.PublicKey, 0)
	for _, pi := range pa {

		publickey := crypto.ToECDSAPub(common.FromHex(pi))
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return ErrInvalidRingSigned, nil, nil, nil, nil
		}

		publickeys = append(publickeys, publickey)
	}

	keyimgae := crypto.ToECDSAPub(common.FromHex(k))
	if keyimgae == nil || keyimgae.X == nil || keyimgae.Y == nil {
		return ErrInvalidRingSigned, nil, nil, nil, nil
	}

	wa := strings.Split(ws, "&")
	w := make([]*big.Int, 0)
	for _, wi := range wa {
		bi, err := hexutil.DecodeBig(wi)
		if bi == nil || err != nil {
			return ErrInvalidRingSigned, nil, nil, nil, nil
		}

		w = append(w, bi)
	}

	qa := strings.Split(qs, "&")
	q := make([]*big.Int, 0)
	for _, qi := range qa {
		bi, err := hexutil.DecodeBig(qi)
		if bi == nil || err != nil {
			return ErrInvalidRingSigned, nil, nil, nil, nil
		}

		q = append(q, bi)
	}

	if len(publickeys) != len(w) || len(publickeys) != len(q) {
		return ErrInvalidRingSigned, nil, nil, nil, nil
	}

	return nil, publickeys, keyimgae, w, q
}

type RingSignInfo struct {
	PublicKeys []*ecdsa.PublicKey
	KeyImage   *ecdsa.PublicKey
	W_Random   []*big.Int
	Q_Random   []*big.Int
	OTABalance *big.Int
}

func FetchRingSignInfo(stateDB StateDB, hashInput []byte, ringSignedStr string) (info *RingSignInfo, err error) {
	if stateDB == nil || hashInput == nil {
		return nil, errParameters
	}

	infoTmp := new(RingSignInfo)

	err, infoTmp.PublicKeys, infoTmp.KeyImage, infoTmp.W_Random, infoTmp.Q_Random = DecodeRingSignOut(ringSignedStr)
	if err != nil {
		return nil, err
	}

	otaLongs := make([][]byte, 0, len(infoTmp.PublicKeys))
	for i := 0; i < len(infoTmp.PublicKeys); i++ {
		otaLongs = append(otaLongs, keystore.ECDSAPKCompression(infoTmp.PublicKeys[i]))
	}

	exist, balanceGet, _, err := BatCheckOTAExist(stateDB, otaLongs)
	if err != nil {

		log.Error("verify mix ota fail", "err", err.Error())
		return nil, err
	}

	if !exist {
		return nil, ErrInvalidOTASet
	}

	infoTmp.OTABalance = balanceGet

	valid := crypto.VerifyRingSign(hashInput, infoTmp.PublicKeys, infoTmp.KeyImage, infoTmp.W_Random, infoTmp.Q_Random)
	if !valid {
		return nil, ErrInvalidRingSigned
	}

	return infoTmp, nil
}

func GetSupportWanCoinOTABalances() []*big.Int {
	cval10, _ := new(big.Int).SetString(Wancoin10, 10)
	cval20, _ := new(big.Int).SetString(Wancoin20, 10)
	cval50, _ := new(big.Int).SetString(Wancoin50, 10)
	cval100, _ := new(big.Int).SetString(Wancoin100, 10)

	cval200, _ := new(big.Int).SetString(Wancoin200, 10)
	cval500, _ := new(big.Int).SetString(Wancoin500, 10)
	cval1000, _ := new(big.Int).SetString(Wancoin1000, 10)
	cval5000, _ := new(big.Int).SetString(Wancoin5000, 10)
	cval50000, _ := new(big.Int).SetString(Wancoin50000, 10)

	wancoinBalances := []*big.Int{
		cval10,
		cval20,
		cval50,
		cval100,

		cval200,
		cval500,
		cval1000,
		cval5000,
		cval50000,
	}

	return wancoinBalances
}

func GetSupportStampOTABalances() []*big.Int {

	svaldot09, _ := new(big.Int).SetString(WanStampdot09, 10)
	svaldot2, _ := new(big.Int).SetString(WanStampdot2, 10)
	svaldot5, _ := new(big.Int).SetString(WanStampdot5, 10)

	stampBalances := []*big.Int{
		//svaldot03,
		//svaldot06,
		svaldot09,
		svaldot2,
		svaldot5,
	}

	return stampBalances
}
