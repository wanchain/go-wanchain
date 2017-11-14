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

package core

import (
	"errors"
	"math/big"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"strings"
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/common/hexutil"

)

var (
	Big0                         = big.NewInt(0)
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
)

/*
The State Transitioning Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all all the necessary work to work out a valid new state root.

1) Nonce handling
2) Pre pay gas
3) Create a new state object if the recipient is \0*32
4) Value transfer
== If contract creation ==
  4a) Attempt to run transaction data
  4b) If valid, use result as code for the new state object
== end ==
5) Run Script section
6) Derive new state root
*/
type StateTransition struct {
	gp         *GasPool
	msg        Message
	gas        uint64
	gasPrice   *big.Int
	initialGas *big.Int
	value      *big.Int
	data       []byte
	state      vm.StateDB
	evm        *vm.EVM
}

// Message represents a message sent to a contract.
type Message interface {
	From() common.Address
	//FromFrontier() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() *big.Int
	Value() *big.Int

	Nonce() uint64
	CheckNonce() bool
	Data() []byte

	TxType() uint64
}

// IntrinsicGas computes the 'intrinsic gas' for a message
// with the given data.
//
// TODO convert to uint64
func IntrinsicGas(data []byte, contractCreation, homestead bool) *big.Int {
	igas := new(big.Int)
	if contractCreation && homestead {
		igas.SetUint64(params.TxGasContractCreation)
	} else {
		igas.SetUint64(params.TxGas)
	}
	if len(data) > 0 {
		var nz int64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		m := big.NewInt(nz)
		m.Mul(m, new(big.Int).SetUint64(params.TxDataNonZeroGas))
		igas.Add(igas, m)
		m.SetInt64(int64(len(data)) - nz)
		m.Mul(m, new(big.Int).SetUint64(params.TxDataZeroGas))
		igas.Add(igas, m)
	}
	return igas
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(evm *vm.EVM, msg Message, gp *GasPool) *StateTransition {
	return &StateTransition{
		gp:         gp,
		evm:        evm,
		msg:        msg,
		gasPrice:   msg.GasPrice(),
		initialGas: new(big.Int),
		value:      msg.Value(),
		data:       msg.Data(),
		state:      evm.StateDB,
	}
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.
func ApplyMessage(evm *vm.EVM, msg Message, gp *GasPool) ([]byte, *big.Int, bool, error) {
	st := NewStateTransition(evm, msg, gp)

	ret, _, gasUsed, failed, err := st.TransitionDb()
	return ret, gasUsed, failed, err
}

func (st *StateTransition) from() vm.AccountRef {
	f := st.msg.From()
	if !st.state.Exist(f) {
		st.state.CreateAccount(f)
	}
	return vm.AccountRef(f)
}

func (st *StateTransition) to() vm.AccountRef {
	if st.msg == nil {
		return vm.AccountRef{}
	}
	to := st.msg.To()
	if to == nil {
		return vm.AccountRef{} // contract creation
	}

	reference := vm.AccountRef(*to)
	if !st.state.Exist(*to) {
		st.state.CreateAccount(*to)
	}
	return reference
}

func (st *StateTransition) useGas(amount uint64) error {
	if st.gas < amount {
		return vm.ErrOutOfGas
	}
	st.gas -= amount

	return nil
}

func (st *StateTransition) buyGas() error {
	mgas := st.msg.Gas()
	if mgas.BitLen() > 64 {
		return vm.ErrOutOfGas
	}

	mgval := new(big.Int).Mul(mgas, st.gasPrice)

	var (
		state  = st.state
		sender = st.from()
	)

	if state.GetBalance(sender.Address()).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}

	if err := st.gp.SubGas(mgas); err != nil {
		return err
	}
	st.gas += mgas.Uint64()

	st.initialGas.Set(mgas)
	state.SubBalance(sender.Address(), mgval)
	return nil
}

func (st *StateTransition) preCheck() error {
	msg := st.msg
	sender := st.from()

	// Make sure this transaction's nonce is correct
	if msg.CheckNonce() {
		nonce := st.state.GetNonce(sender.Address())
		if nonce < msg.Nonce() {
			return ErrNonceTooHigh
		} else if nonce > msg.Nonce() {
			return ErrNonceTooLow
		}
	}
	return st.buyGas()
}



// TransitionDb will transition the state by applying the current message and returning the result
// including the required gas for the operation as well as the used gas. It returns an error if it
// failed. An error indicates a consensus issue.
func (st *StateTransition) TransitionDb() (ret []byte, requiredGas, usedGas *big.Int, failed bool, err error) {

	if st.msg.TxType() != 6 {
		if err = st.preCheck(); err != nil {
			return
		}
	}

	msg := st.msg
	sender := st.from() // err checked in preCheck

	homestead := st.evm.ChainConfig().IsHomestead(st.evm.BlockNumber)
	contractCreation := msg.To() == nil


	// Pay intrinsic gas
	// TODO convert to uint64
	intrinsicGas := IntrinsicGas(st.data, contractCreation, homestead)
	if intrinsicGas.BitLen() > 64 {
		return nil, nil, nil, false, vm.ErrOutOfGas
	}

	if st.msg.TxType() == 6 {
		pureCallData, stampGas, err := st.preProcessPrivacyTx(sender.Address().Bytes(), st.data)
		if err != nil {
			return nil, nil, nil, false, err
		}
		st.gas = stampGas
		st.initialGas.SetUint64(stampGas)
		st.data = pureCallData[:]
	}

	if err = st.useGas(intrinsicGas.Uint64()); err != nil {
		return nil, nil, nil, false, err
	}


	var (
		evm = st.evm
		// vm errors do not effect consensus and are therefor
		// not assigned to err, except for insufficient balance
		// error.
		vmerr error
	)
	if contractCreation {
		ret, _, st.gas, vmerr = evm.Create(sender, st.data, st.gas, st.value)
	} else {
		// Increment the nonce for the next transaction
		st.state.SetNonce(sender.Address(), st.state.GetNonce(sender.Address())+1)
		ret, st.gas, vmerr = evm.Call(sender, st.to().Address(), st.data, st.gas, st.value)
	}

	if vmerr != nil {
		log.Debug("VM returned with error", "err", vmerr)
		// The only possible consensus-error would be if there wasn't
		// sufficient balance to make the transfer happen. The first
		// balance transfer may never fail.
		if vmerr == vm.ErrInsufficientBalance {
			return nil, nil, nil, false, vmerr
		}
	}
	requiredGas = new(big.Int).Set(st.gasUsed())

	if st.msg.TxType() != 6 {
		st.refundGas()
		st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(st.gasUsed(), st.gasPrice))
	}

	return ret, requiredGas, st.gasUsed(), vmerr != nil, err
}

func (st *StateTransition) refundGas() {
	// Return eth for remaining gas to the sender account,
	// exchanged at the original rate.
	sender := st.from() // err already checked
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
	st.state.AddBalance(sender.Address(), remaining)

	// Apply refund counter, capped to half of the used gas.
	uhalf := remaining.Div(st.gasUsed(), common.Big2)
	refund := math.BigMin(uhalf, st.state.GetRefund())
	st.gas += refund.Uint64()

	st.state.AddBalance(sender.Address(), refund.Mul(refund, st.gasPrice))

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(new(big.Int).SetUint64(st.gas))
}

func (st *StateTransition) gasUsed() *big.Int {
	return new(big.Int).Sub(st.initialGas, new(big.Int).SetUint64(st.gas))
}


///////////////////////added for privacy tx /////////////////////////////
var (
	utilAbiDefinition = `[{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}],"name":"combine","outputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}]}]`

	utilAbi, errAbiInit = abi.JSON(strings.NewReader(utilAbiDefinition))
)

func init() {
	if errAbiInit != nil {
		panic(errAbiInit)
	}
}

func (st *StateTransition) DecodeRingSignOut(s string) (error, []*ecdsa.PublicKey, *ecdsa.PublicKey, []*big.Int, []*big.Int) {
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

func (st *StateTransition) preProcessPrivacyTx(hashInput []byte, in []byte) (callData []byte, stampGas uint64, err error) {

	var TxDataWithRing struct {
		RingSignedData string
		CxtCallParams  []byte
	}

	err = utilAbi.Unpack(&TxDataWithRing, "combine", in[4:])
	if err != nil {
		return nil, 0, err
	}
	callData = TxDataWithRing.CxtCallParams[:]

	err, publickeys, keyimage, ws, qs := st.DecodeRingSignOut(TxDataWithRing.RingSignedData)
	if err != nil {
		return nil, 0, err
	}

	otaAXs := make([][]byte, 0, len(publickeys))
	for i := 0; i < len(publickeys); i++ {
		pkBytes := crypto.FromECDSAPub(publickeys[i])
		otaAXs = append(otaAXs, pkBytes[1:1+common.HashLength])
	}

	exit, balanceGet, unexit, err := vm.BatCheckOTAExit(st.evm.StateDB, otaAXs)
	if !exit || balanceGet == nil {
		if err != nil {
			log.Warn("verify mix ota fail", "err", err.Error())
		}
		if unexit != nil {
			log.Warn("invalid mix ota", "invalid ota", common.ToHex(unexit))
		}
		if balanceGet == nil {
			log.Warn("balance getting from ota is wrong!")
		}

		return nil, 0, err
	}

	kix := crypto.FromECDSAPub(keyimage)
	exit, _, err = vm.CheckOTAImageExit(st.evm.StateDB, kix)
	if err != nil || exit {
		return nil, 0, err
	}

	b := crypto.VerifyRingSign(hashInput, publickeys, keyimage, ws, qs)
	if !b {
		err = errors.New("ring sign is invalid!")
		return nil, 0, err
	}

	vm.AddOTAImage(st.evm.StateDB, kix, balanceGet.Bytes())

	stampGas = new(big.Int).Div(balanceGet,st.gasPrice).Uint64()

	mixLen := len(publickeys)
	ringSigDiffRequiredGas := params.RequiredGasPerMixPub * (uint64(mixLen))

	stampGas -= ringSigDiffRequiredGas


	return

}

