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

package core

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"math"
	"math/big"
)

// InvalidPrivacyTx remove invalidate privacy transactions
func (l *txList) InvalidPrivacyTx(stateDB vm.StateDB, signer types.Signer, gasLimit *big.Int) types.Transactions {
	removed := l.txs.Filter(func(tx *types.Transaction) bool {
		if !types.IsPrivacyTransaction(uint64(tx.Type())) {
			return false
		}

		from, err := types.Sender(signer, tx)
		if err != nil {
			return true
		}

		//intrGas, _ := IntrinsicGas(tx.Data(), tx.To(), true)
		intrGas, _ := IntrinsicGasWan(tx.Data(), nil, false, true, false, tx.To()) //todo check the parameters again.
		err = ValidPrivacyTx(stateDB, from.Bytes(), tx.Data(), tx.GasPrice(), big.NewInt(0).SetUint64(intrGas), tx.Value(), gasLimit)

		return err != nil
	})

	var invalids types.Transactions
	if l.strict && len(removed) > 0 {
		lowest := uint64(math.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce(); lowest > nonce {
				lowest = nonce
			}
		}
		invalids = l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > lowest })
	}

	// Privacy transaction's sender is not real sender, just a hash info.
	// So, no need to move invalid transactions to queue for later.
	// Just remove all of invalid transactions.
	return append(removed, invalids...)
}

// InvalidPosTx remove invalidate pos transactions
func (l *txList) InvalidPosRBTx(stateDB vm.StateDB, signer types.Signer) (types.Transactions, types.Transactions) {
	removed := l.txs.Filter(func(tx *types.Transaction) bool {
		if !types.IsPosTransaction(uint64(tx.Type())) || (*tx.To()) != vm.GetRBAddress() {
			return false
		}

		from, err := types.Sender(signer, tx)
		if err != nil {
			return true
		}

		err = vm.ValidPosRBTx(stateDB, from, tx.Data())
		return err != nil
	})

	var invalids types.Transactions
	if l.strict && len(removed) > 0 {
		lowest := uint64(math.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce(); lowest > nonce {
				lowest = nonce
			}
		}
		invalids = l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > lowest })
	}

	return removed, invalids
}

// InvalidPosTx remove invalidate pos transactions
func (l *txList) InvalidPosELTx(stateDB vm.StateDB, signer types.Signer) (types.Transactions, types.Transactions) {
	removed := l.txs.Filter(func(tx *types.Transaction) bool {
		if !types.IsPosTransaction(uint64(tx.Type())) || (*tx.To()) != vm.GetSlotLeaderSCAddress() {
			return false
		}

		from, err := types.Sender(signer, tx)
		if err != nil {
			return true
		}

		err = vm.ValidPosELTx(stateDB, from, tx.Data())
		return err != nil
	})

	var invalids types.Transactions
	if l.strict && len(removed) > 0 {
		lowest := uint64(math.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce(); lowest > nonce {
				lowest = nonce
			}
		}
		invalids = l.txs.Filter(func(tx *types.Transaction) bool { return tx.Nonce() > lowest })
	}

	return removed, invalids
}
