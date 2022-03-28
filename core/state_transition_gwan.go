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

//
//func IntrinsicGas_gwan(data []byte, to *common.Address, homestead bool) *big.Int {
//	contractCreation := to == nil
//
//	igas := new(big.Int)
//	if contractCreation && homestead {
//		igas.SetUint64(params.TxGasContractCreation)
//	} else {
//		igas.SetUint64(params.TxGas)
//	}
//	if len(data) > 0 {
//		var nz int64
//		for _, byt := range data {
//			if byt != 0 {
//				nz++
//			}
//		}
//		m := big.NewInt(nz)
//		m.Mul(m, new(big.Int).SetUint64(params.TxDataNonZeroGas))
//		igas.Add(igas, m)
//		m.SetInt64(int64(len(data)) - nz)
//		m.Mul(m, new(big.Int).SetUint64(params.TxDataZeroGas))
//		igas.Add(igas, m)
//	}
//
//	// reduce gas used for pos tx
//	if vm.IsPosPrecompiledAddr(to) {
//		igas = igas.Div(igas, big.NewInt(10))
//	}
//
//	return igas
//}
