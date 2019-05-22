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

package types

import (
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	// "github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rlp"
)

func TestEIP155Signing(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	signer := NewEIP155Signer(big.NewInt(18))
	tx, err := SignTx(NewTransaction(0, addr, new(big.Int), new(big.Int), new(big.Int), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(signer, tx)
	if err != nil {
		t.Fatal(err)
	}
	if from != addr {
		t.Errorf("exected from and address to be equal. Got %x want %x", from, addr)
	}
}

func TestEIP155ChainId(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	signer := NewEIP155Signer(big.NewInt(18))
	tx, err := SignTx(NewTransaction(0, addr, new(big.Int), new(big.Int), new(big.Int), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}
	if !tx.Protected() {
		t.Fatal("expected tx to be protected")
	}

	if tx.ChainId().Cmp(signer.chainId) != 0 {
		t.Error("expected chainId to be", signer.chainId, "got", tx.ChainId())
	}

	tx = NewTransaction(0, addr, new(big.Int), new(big.Int), new(big.Int), nil)
	tx, err = SignTx(tx, HomesteadSigner{}, key)
	if err != nil {
		t.Fatal(err)
	}

	if tx.Protected() {
		t.Error("didn't expect tx to be protected")
	}

	if tx.ChainId().Sign() != 0 {
		t.Error("expected chain id to be 0 got", tx.ChainId())
	}
}

func TestEIP155SigningVitalik(t *testing.T) {
	// Test vectors come from http://vitalik.ca/files/eip155_testvec.txt
	for i, test := range []struct {
		txRlp, addr string
	}{
		{"f86c018083030d4083015f90945f2630e44c6f71c6c70a482dbb07fc6e64b983e8880de0b6b3a76400008026a030e57d842cd469ecb6c823e77c99a45f63e58db6c9c70f0457619dd47063e0eca046791c76d504b283dfcc535b0136bb8c9dfd1511b72df847eb47dc4509733414", "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"},
		{"f86c018083030d4083033450942c7b4615292c30c4b2f8063ec07e234aea6df5df8829a2241af62c00008026a09df8a039434fe43291fc35f034ffc1ac36d0e17fc60cda732c32b6d7ed4ad2b0a0528108626c13666919699c4d89b9f37ee466bd2822bf08ce104e81051cea29ac", "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"},
		{"f86a01806483033450942c7b4615292c30c4b2f8063ec07e234aea6df5df89056bc75e2d631000008025a0e4053a6da040c3640ba2597fa22fce6f639126b9cd36013dae6ecd8493f60db4a0041a82bd8d05f6533e627fb501fbefad1c1001745a9572a39f187fb0b683f092", "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"},
		{"f8680180648203e8942c7b4615292c30c4b2f8063ec07e234aea6df5df8806f05b59d3b200008026a0ac63642e2767020d4e73a5a04a03675249a9de6ec3af57fdc98302ba4899d9d5a07146b8581f8f7e27609e73bfdafde6eb7cfce08814a894c755194de1142497f6", "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"},
		{"f8680180018207d0942c7b4615292c30c4b2f8063ec07e234aea6df5df8814d1120d7b1600008025a0f660863cd240808f8951e203051966be3fb459be41e82a62e761284ca36e5f0aa0191e7c66405fe939ffd3e5b5192b63ea807002277f0ac8a22e02f5522ac9ac95", "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e"},
	} {
		signer := NewEIP155Signer(big.NewInt(1))

		var tx *Transaction
		err := rlp.DecodeBytes(common.Hex2Bytes(test.txRlp), &tx)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		from, err := Sender(signer, tx)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		addr := common.HexToAddress(test.addr)
		if from != addr {
			t.Errorf("%d: expected %x got %x", i, addr, from)
		}

	}
}

func TestChainId(t *testing.T) {
	key, _ := defaultTestKey()

	tx := NewTransaction(0, common.Address{}, new(big.Int), new(big.Int), new(big.Int), nil)

	var err error
	tx, err = SignTx(tx, NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Sender(NewEIP155Signer(big.NewInt(2)), tx)
	if err != ErrInvalidChainId {
		t.Error("expected error:", ErrInvalidChainId)
	}

	_, err = Sender(NewEIP155Signer(big.NewInt(1)), tx)
	if err != nil {
		t.Error("expected no error")
	}
}
