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
	"fmt"
	"math/big"

	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"strings"
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

	evm *vm.EVM
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

func MessageCreatesContract(msg Message) bool {
	return msg.To() == nil
}

// IntrinsicGas computes the 'intrinsic gas' for a message
// with the given data.
//
// TODO convert to uint64
//func IntrinsicGas(data []byte, contractCreation, homestead bool) *big.Int {
func IntrinsicGas(data []byte, contractCreation bool) *big.Int {
	igas := new(big.Int)
	//if contractCreation && homestead {
	if contractCreation {
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
func ApplyMessage(evm *vm.EVM, msg Message, gp *GasPool) ([]byte, *big.Int, error) {
	st := NewStateTransition(evm, msg, gp)

	ret, _, gasUsed, err := st.TransitionDb()
	return ret, gasUsed, err
}

func (self *StateTransition) from() vm.AccountRef {
	f := self.msg.From()
	if !self.state.Exist(f) {
		self.state.CreateAccount(f)
	}
	return vm.AccountRef(f)
}

func (self *StateTransition) to() vm.AccountRef {
	if self.msg == nil {
		return vm.AccountRef{}
	}
	to := self.msg.To()
	if to == nil {
		return vm.AccountRef{} // contract creation
	}

	reference := vm.AccountRef(*to)
	if !self.state.Exist(*to) {
		self.state.CreateAccount(*to)
	}
	return reference
}

func (self *StateTransition) useGas(amount uint64) error {
	if self.gas < amount {
		return vm.ErrOutOfGas
	}
	self.gas -= amount

	return nil
}

func (self *StateTransition) buyGas() error {
	mgas := self.msg.Gas()
	if mgas.BitLen() > 64 {
		return vm.ErrOutOfGas
	}

	mgval := new(big.Int).Mul(mgas, self.gasPrice)

	var (
		state  = self.state
		sender = self.from()
	)

	if state.GetBalance(sender.Address()).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}
	if err := self.gp.SubGas(mgas); err != nil {
		return err
	}
	self.gas += mgas.Uint64()

	self.initialGas.Set(mgas)
	state.SubBalance(sender.Address(), mgval)
	return nil
}

func (self *StateTransition) preCheck() error {
	msg := self.msg
	sender := self.from()

	// Make sure this transaction's nonce is correct
	if msg.CheckNonce() {
		if n := self.state.GetNonce(sender.Address()); n != msg.Nonce() {
			return fmt.Errorf("invalid nonce: have %d, expected %d", msg.Nonce(), n)
		}
	}
	return self.buyGas()
}

var (
	utilAbiDefinition = `[{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}],"name":"combine","outputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}]}]`

	utilAbi, errAbiInit = abi.JSON(strings.NewReader(utilAbiDefinition))
)

func init() {
	if errAbiInit != nil {
		panic(errAbiInit)
	}
}

func (self *StateTransition) DecodeRingSignOut(s string) (error, []*ecdsa.PublicKey, *ecdsa.PublicKey, []*big.Int, []*big.Int) {
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

func (self *StateTransition) preProcessPrivacyTx(hashInput []byte, in []byte) (callData []byte, keyimage *ecdsa.PublicKey, err error) {

	var TxDataWithRing struct {
		RingSignedData string
		CxtCallParams  []byte
	}

	err = utilAbi.Unpack(&TxDataWithRing, "combine", in[4:])
	if err != nil {
		return nil, nil, err
	}
	callData = TxDataWithRing.CxtCallParams[:]

	err, publickeys, keyimage, ws, qs := self.DecodeRingSignOut(TxDataWithRing.RingSignedData)
	if err != nil {
		return nil, nil, err
	}

	otaAXs := make([][]byte, 0, len(publickeys))
	for i := 0; i < len(publickeys); i++ {
		pkBytes := crypto.FromECDSAPub(publickeys[i])
		otaAXs = append(otaAXs, pkBytes[1:1+common.HashLength])
	}

	exit, balanceGet, unexit, err := vm.BatCheckOTAExit(self.evm.StateDB, otaAXs)
	if !exit || balanceGet == nil {
		if err != nil {
			log.Warn("verify mix ota fail. err:%s", err.Error())
		}
		if unexit != nil {
			log.Warn("invalid mix ota:%s", common.ToHex(unexit))
		}
		if balanceGet == nil {
			log.Warn("balance getting from ota is wrong! get:%s, expect:%s")
		}

		return nil, nil, err
	}

	kix := crypto.FromECDSAPub(keyimage)
	exit, _, err = vm.CheckOTAImageExit(self.evm.StateDB, kix)
	if err != nil || exit {
		return nil, nil, err
	}

	b := crypto.VerifyRingSign(hashInput, publickeys, keyimage, ws, qs)
	if !b {
		err = errors.New("ring sign is invalid!")
		return nil, nil, err
	}

	vm.AddOTAImage(self.evm.StateDB, kix, balanceGet.Bytes())
	return

}

// TransitionDb will transition the state by applying the current message and returning the result
// including the required gas for the operation as well as the used gas. It returns an error if it
// failed. An error indicates a consensus issue.
func (self *StateTransition) TransitionDb() (ret []byte, requiredGas, usedGas *big.Int, err error) {
	//txtype 2 is contract trasaction
	if self.msg.TxType() != 6 {
		if err = self.preCheck(); err != nil {
			return
		}
	} else {
		fmt.Println("txType is 2")
	}

	msg := self.msg
	sender := self.from() // err checked in preCheck

	//homestead := self.evm.ChainConfig().IsHomestead(self.evm.BlockNumber)

	contractCreation := MessageCreatesContract(msg)
	// Pay intrinsic gas
	// TODO convert to uint64
	//intrinsicGas := IntrinsicGas(self.data, contractCreation, homestead)
	intrinsicGas := IntrinsicGas(self.data, contractCreation)
	if intrinsicGas.BitLen() > 64 {
		return nil, nil, nil, vm.ErrOutOfGas
	}

	if self.msg.TxType() != 6 {
		if err = self.useGas(intrinsicGas.Uint64()); err != nil {
			return nil, nil, nil, err
		}
	} else {
		fmt.Println("txType is 2")
	}

	var (
		evm = self.evm
		// vm errors do not effect consensus and are therefor
		// not assigned to err, except for insufficient balance
		// error.
		vmerr error
	)

	if contractCreation {
		ret, _, self.gas, vmerr = evm.Create(sender, self.data, self.gas, self.value)
	} else {
		// Increment the nonce for the next transaction
		self.state.SetNonce(sender.Address(), self.state.GetNonce(sender.Address())+1)

		if self.msg.TxType() == 6 {
			pureCallData, _, err := self.preProcessPrivacyTx(sender.Address().Bytes(), self.data)
			if err != nil {
				return nil, nil, nil, err
			}
			// TODO: set gas correponding stamp value, stamp_value / gas_price
			//       and sub gas used by ring sign
			self.gas = 200000
			self.initialGas.SetUint64(200000)
			self.data = pureCallData[:]
		}
		ret, self.gas, vmerr = evm.Call(sender, self.to().Address(), self.data, self.gas, self.value)
	}

	if vmerr != nil {
		log.Debug("VM returned with error", "err", err)
		// The only possible consensus-error would be if there wasn't
		// sufficient balance to make the transfer happen. The first
		// balance transfer may never fail.
		if vmerr == vm.ErrInsufficientBalance {
			return nil, nil, nil, vmerr
		}
	}
	requiredGas = new(big.Int).Set(self.gasUsed())

	if self.msg.TxType() != 6 {

		self.refundGas()
		self.state.AddBalance(self.evm.Coinbase, new(big.Int).Mul(self.gasUsed(), self.gasPrice))

	}

	return ret, requiredGas, self.gasUsed(), err
}

func (self *StateTransition) refundGas() {
	// Return eth for remaining gas to the sender account,
	// exchanged at the original rate.
	sender := self.from() // err already checked
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(self.gas), self.gasPrice)
	self.state.AddBalance(sender.Address(), remaining)

	// Apply refund counter, capped to half of the used gas.
	uhalf := remaining.Div(self.gasUsed(), common.Big2)
	refund := math.BigMin(uhalf, self.state.GetRefund())
	self.gas += refund.Uint64()

	self.state.AddBalance(sender.Address(), refund.Mul(refund, self.gasPrice))

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	self.gp.AddGas(new(big.Int).SetUint64(self.gas))
}

func (self *StateTransition) gasUsed() *big.Int {
	return new(big.Int).Sub(self.initialGas, new(big.Int).SetUint64(self.gas))
}
