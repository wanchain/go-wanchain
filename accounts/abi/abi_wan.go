// Copyright 2015 The go-ethereum Authors
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

package abi

// add by jacob
// Unpack input in v according to the abi specification
// ethereum native unpack* used to unpack output/log。
func (abi ABI) UnpackInput(v interface{}, name string, input []byte) (err error) {
	var args Arguments
	if method, ok := abi.Methods[name]; ok {
		args = method.Inputs
	}
	unpacked, err := args.Unpack(input)
	if err != nil {
		return err
	}
	return args.Copy(v, unpacked)
}
