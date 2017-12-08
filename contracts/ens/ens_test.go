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

package ens

import (
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/accounts/abi/bind"
	"github.com/wanchain/go-wanchain/accounts/abi/bind/backends"
	"github.com/wanchain/go-wanchain/crypto"
)

var (
	key, _ = crypto.HexToECDSA("3efdddbf163faf1b5ec73e833b7820e87560137917773f63b7dc33e1dcb6dd24")
	name   = "my name on ENS"
	hash   = crypto.Keccak256Hash([]byte("my content"))
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

func TestENS(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(nil)
	transactOpts := bind.NewKeyedTransactor(key)
	// Workaround for bug estimating gas in the call to Register
	transactOpts.GasLimit = big.NewInt(1000000)

	ens, err := DeployENS(transactOpts, contractBackend)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// @anson
	contractBackend.SetExtra()

	contractBackend.Commit()

	_, err = ens.Register(name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	contractBackend.Commit()

	_, err = ens.SetContentHash(name, hash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	contractBackend.Commit()

	vhost, err := ens.Resolve(name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if vhost != hash {
		t.Fatalf("resolve error, expected %v, got %v", hash.Hex(), vhost.Hex())
	}
}
