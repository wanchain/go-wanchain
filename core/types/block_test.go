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

package types

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	// "github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/rlp"
)

func stringToBlockNonce(input string) BlockNonce {
	raw := []byte(input)
	var nonce BlockNonce
	copy(nonce[:], raw[:])
	return nonce
}

func stringToBloom(input string) Bloom {
	raw := []byte(input)
	var bloom Bloom
	copy(bloom[:], raw[:])
	return bloom
}

func genBlockRLP() ([]byte, error) {
	// block meta from bcValidBlockTest.json, "SimpleTx"
	header := Header{
		Bloom:       stringToBloom("00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		Coinbase:    common.StringToAddress("8888f1f195afa192cfee860698584c030f4c9db1"),
		Difficulty:  big.NewInt(0x020000),
		GasLimit:    big.NewInt(0x2fefd8),
		GasUsed:     big.NewInt(0x5208),
		Nonce:       stringToBlockNonce("e6805bde6de82a7b"),
		Number:      big.NewInt(0x1),
		ParentHash:  common.HexToHash("7285abd5b24742f184ad676e31f6054663b3529bc35ea2fcad8a3e0f642a46f7"),
		UncleHash:   common.HexToHash("1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Root:        common.HexToHash("964e6c9995e7e3757e934391b4f16b50c20409ee4eb9abd4c4617cb805449b9a"),
		TxHash:      common.HexToHash("53d5b71a8fbb9590de82d69dfa4ac31923b0c8afce0d30d0d8d1e931f25030dc"),
		ReceiptHash: common.HexToHash("bc37d79753ad738a6dac4921e57392f145d8887476de3f783dfa7edae9283e52"),
		Time:        big.NewInt(0x5802053a),
		Extra:       []byte{},
		MixDigest:   common.HexToHash("80f4006497e0d03ab44e10b0aa53da639d6de4b8f3d877972dab9622a6b2274b"),
	}

	// generate block(header) hash
	// blockHash := header.Hash()
	// fmt.Println(blockHash.Hex())

	transactions := make([]*Transaction, 0, 0)

	var trans Transaction
	transactions = append(transactions, &trans)

	var block Block
	block.header = &header
	block.transactions = transactions

	blockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		fmt.Println("rlp_encode error: ", err)
		return nil, err
	}

	return blockEnc, err

}

func TestBlockEncoding(t *testing.T) {
	blockEnc, err := genBlockRLP()
	if err != nil {
		t.Fatal("gen block rlp error: ", err)
	}
	// blockEnc := common.FromHex("0xf90209f901f9a07285abd5b24742f184ad676e31f6054663b3529bc35ea2fcad8a3e0f642a46f7a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347943836303639383538346330333066346339646231a0964e6c9995e7e3757e934391b4f16b50c20409ee4eb9abd4c4617cb805449b9aa053d5b71a8fbb9590de82d69dfa4ac31923b0c8afce0d30d0d8d1e931f25030dca0bc37d79753ad738a6dac4921e57392f145d8887476de3f783dfa7edae9283e52b90100303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030308302000001832fefd8825208845802053a80a080f4006497e0d03ab44e10b0aa53da639d6de4b8f3d877972dab9622a6b2274b886536383035626465cbca80808080808080808080c0")
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("Difficulty", block.Difficulty(), big.NewInt(0x020000))
	check("GasLimit", block.GasLimit(), big.NewInt(0x2fefd8))
	check("GasUsed", block.GasUsed(), big.NewInt(0x5208))
	check("Coinbase", block.Coinbase(), common.StringToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("MixDigest", block.MixDigest(), common.HexToHash("80f4006497e0d03ab44e10b0aa53da639d6de4b8f3d877972dab9622a6b2274b"))
	check("Root", block.Root(), common.HexToHash("964e6c9995e7e3757e934391b4f16b50c20409ee4eb9abd4c4617cb805449b9a"))
	// check("Hash", block.Hash(), common.HexToHash("1ce0ee9ab21296a73828ecc2b1834769573c565fa8371dd7a9c5aeea4d164ab6"))
	check("Hash", block.Hash(), common.HexToHash("074612bb961781092fc1d8cd49dd466d5c39f25234fa6f6625c4fd3b67da4ae6"))
	// check("Nonce", block.Nonce(), stringToBlockNonce("e6805bde6de82a7b"))
	check("Nonce", block.Nonce(), stringToBlockNonce("e6805bde6de82a7b").Uint64())
	check("Time", block.Time(), big.NewInt(0x5802053a))
	check("Size", block.Size(), common.StorageSize(len(blockEnc)))

	// tx1 := NewTransaction(0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(10), big.NewInt(50000), big.NewInt(10), nil)

	// tx1, _ = tx1.WithSignature(HomesteadSigner{}, common.Hex2Bytes("9bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094f8a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b100"))
	// fmt.Println(block.Transactions()[0].Hash())
	// fmt.Println(block.Transactions()[0].Hash().Hex())
	// fmt.Println(tx1.data)
	// fmt.Println(tx1.Hash())
	check("len(Transactions)", len(block.Transactions()), 1)
	// check("Transactions[0].Hash", block.Transactions()[0].Hash(), tx1.Hash())

	ourBlockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		t.Fatal("encode error: ", err)
	}
	if !bytes.Equal(ourBlockEnc, blockEnc) {
		t.Errorf("encoded block mismatch:\ngot:  %x\nwant: %x", ourBlockEnc, blockEnc)
	}
}
