// Copyright 2017 The go-ethereum Authors
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

// Package accounts implements high level Ethereum account management.
package accounts

import (
	"github.com/ethereum/go-ethereum/common"
)

type WanWallet interface {
	// GetWanAddress represents the wallet to retrieve corresponding wanchain public address for a specific ordinary account/address
	GetWanAddress(account Account) (common.WAddress, error)

	// ComputeOTAPPKeys returns one-time-address pair
	ComputeOTAPPKeys(account Account, AX, AY, BX, BY string) ([]string, error)
}
