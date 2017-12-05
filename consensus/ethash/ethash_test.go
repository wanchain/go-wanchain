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

	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
)

// Tests that ethash works correctly in test mode.
func TestTestMode(t *testing.T) {
	head := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}
	head.Extra = hexutil.MustDecode("0xf9b32578b4420a36f132db32b56f3831a7cc1804810524175efa012446103d1a04c9f4263a962accdb05642eabc8347ec78e21bdf0d906ba579d423ab5eb9bf02a924367ed9d4f86dfcb1c572cd9a4f80036805b6846f26ac35f2a7d7eda4a2a58f08e8ef073d4e52c506f3f288faa9db1c1e5ae0f1e70f8c38eb01bce9bcb61327532dc5a540da4cf484ae57e98bc5a465c1d2afa6b9376709a525981f53d493a46ef1eb55428b3b88a222d80d23531054ef51dbd100cf8286136659a7d63a38a154e28dbf3e0fd")

	ethash := NewTester()
	block, err := ethash.Seal(nil, types.NewBlockWithHeader(head), nil)

	// block := &core.Block{
	// 	Config:     params.WanchainChainConfig,
	// 	Nonce:      66,
	// 	ExtraData:  hexutil.MustDecode("0xf9b32578b4420a36f132db32b56f3831a7cc1804810524175efa012446103d1a04c9f4263a962accdb05642eabc8347ec78e21bdf0d906ba579d423ab5eb9bf02a924367ed9d4f86dfcb1c572cd9a4f80036805b6846f26ac35f2a7d7eda4a2a58f08e8ef073d4e52c506f3f288faa9db1c1e5ae0f1e70f8c38eb01bce9bcb61327532dc5a540da4cf484ae57e98bc5a465c1d2afa6b9376709a525981f53d493a46ef1eb55428b3b88a222d80d23531054ef51dbd100cf8286136659a7d63a38a154e28dbf3e0fd"),
	// 	GasLimit:   0x2fefd8,
	// 	Difficulty: big.NewInt(1048576),
	// 	Alloc:      jsonPrealloc(wanchainAllocJson),
	// }
	if err != nil {
		t.Fatalf("failed to seal block: %v", err)
	}
	head.Nonce = types.EncodeNonce(block.Nonce())
	head.MixDigest = block.MixDigest()
	if err := ethash.VerifySeal(nil, head); err != nil {
		t.Fatalf("unexpected verification error: %v", err)
	}
}
