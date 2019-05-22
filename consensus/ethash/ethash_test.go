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

package ethash

import (
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
)

// func tmpKeyStore(t *testing.T, encrypted bool) (string, *KeyStore) {
// 	d, err := ioutil.TempDir("", "wanchain-keystore-test")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	new := NewPlaintextKeyStore
// 	if encrypted {
// 		new = func(kd string) *KeyStore { return NewKeyStore(kd, veryLightScryptN, veryLightScryptP) }
// 	}
// 	return d, new(d)
// }

// Tests that ethash works correctly in test mode.
var fakedAddr = common.HexToAddress("0xf9b32578b4420a36f132db32b56f3831a7cc1804")
var fakedAccountPrivateKey, _ = crypto.HexToECDSA("f1572f76b75b40a7da72d6f2ee7fda3d1189c2d28f0a2f096347055abe344d7f")

func fakeSignerFn(signer accounts.Account, hash []byte) ([]byte, error) {
	return crypto.Sign(hash, fakedAccountPrivateKey)
}

func TestTestMode(t *testing.T) {
	head := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}
	head.Coinbase = fakedAddr

	head.Extra = make([]byte, extraSeal+extraVanity)
	sighash4Extra, err := fakeSignerFn(accounts.Account{}, sigHash(head).Bytes())
	copy(head.Extra[len(head.Extra)-extraSeal:], sighash4Extra)

	ethash := NewTester(nil)
	ethash.signer = fakedAddr
	ethash.signFn = fakeSignerFn
	block, err := ethash.Seal(nil, types.NewBlockWithHeader(head), nil)

	if err != nil {
		t.Fatalf("failed to seal block: %v", err)
	}
	head.Nonce = types.EncodeNonce(block.Nonce())
	head.MixDigest = block.MixDigest()
	if err := ethash.VerifySeal(nil, head); err != nil {
		t.Fatalf("unexpected verification error: %v", err)
	}

}
