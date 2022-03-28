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

/*
Package hexutil implements hex encoding with 0x prefix.
This encoding is used by the Ethereum RPC API to transport binary data in JSON payloads.

Encoding Rules

All hex data must have prefix "0x".

For byte slices, the hex data must be of even length. An empty byte slice
encodes as "0x".

Integers are encoded using the least amount of digits (no leading zero digits). Their
encoding may be of uneven length. The number zero encodes as "0x0".
*/
package hexutil

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common/math"
)

// PKPair2HexSlice generate byte-slice based on given public key pair
func PKPair2HexSlice(pk1 *ecdsa.PublicKey, pk2 *ecdsa.PublicKey) []string {
	return []string{
		Encode(math.PaddedBigBytes(pk1.X, 32)),
		Encode(math.PaddedBigBytes(pk1.Y, 32)),
		Encode(math.PaddedBigBytes(pk2.X, 32)),
		Encode(math.PaddedBigBytes(pk2.Y, 32)),
	}
}

func Has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}
