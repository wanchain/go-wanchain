// Copyright 2018 Wanchain Foundation Ltd

package vm

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"math/big"
)

// Precompiled contracts address or
// Reserved contracts address.
// Should prevent overwriting to them.
var (
	ecrecoverPrecompileAddr      = common.BytesToAddress([]byte{1})
	sha256hashPrecompileAddr     = common.BytesToAddress([]byte{2})
	ripemd160hashPrecompileAddr  = common.BytesToAddress([]byte{3})
	dataCopyPrecompileAddr       = common.BytesToAddress([]byte{4})
	bigModExpPrecompileAddr      = common.BytesToAddress([]byte{5})
	bn256AddPrecompileAddr       = common.BytesToAddress([]byte{6})
	bn256ScalarMulPrecompileAddr = common.BytesToAddress([]byte{7})
	bn256PairingPrecompileAddr   = common.BytesToAddress([]byte{8})

	wanCoinPrecompileAddr  = common.BytesToAddress([]byte{100})
	wanStampPrecompileAddr = common.BytesToAddress([]byte{200})

	otaBalanceStorageAddr = common.BytesToAddress(big.NewInt(300).Bytes())
	otaImageStorageAddr   = common.BytesToAddress(big.NewInt(301).Bytes())

	// 0.01wan --> "0x0000000000000000000000010000000000000000"
	otaBalancePercentdot001WStorageAddr = common.HexToAddress(WanStampdot001)
	otaBalancePercentdot002WStorageAddr = common.HexToAddress(WanStampdot002)
	otaBalancePercentdot005WStorageAddr = common.HexToAddress(WanStampdot005)
	
	otaBalancePercentdot003WStorageAddr = common.HexToAddress(WanStampdot003)
	otaBalancePercentdot006WStorageAddr = common.HexToAddress(WanStampdot006)
	otaBalancePercentdot009WStorageAddr = common.HexToAddress(WanStampdot009)

	otaBalancePercentdot03WStorageAddr = common.HexToAddress(WanStampdot03)
	otaBalancePercentdot06WStorageAddr = common.HexToAddress(WanStampdot06)
	otaBalancePercentdot09WStorageAddr = common.HexToAddress(WanStampdot09)
	otaBalancePercentdot2WStorageAddr = common.HexToAddress(WanStampdot2)
	otaBalancePercentdot5WStorageAddr = common.HexToAddress(WanStampdot5)

	otaBalance10WStorageAddr       = common.HexToAddress(Wancoin10)
	otaBalance20WStorageAddr       = common.HexToAddress(Wancoin20)
	otaBalance50WStorageAddr       = common.HexToAddress(Wancoin50)
	otaBalance100WStorageAddr      = common.HexToAddress(Wancoin100)

	otaBalance200WStorageAddr       = common.HexToAddress(Wancoin200)
	otaBalance500WStorageAddr       = common.HexToAddress(Wancoin500)
	otaBalance1000WStorageAddr      = common.HexToAddress(Wancoin1000)
	otaBalance5000WStorageAddr      = common.HexToAddress(Wancoin5000)
	otaBalance50000WStorageAddr     = common.HexToAddress(Wancoin50000)
)

// PrecompiledContract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	RequiredGas(input []byte) uint64                                // RequiredPrice calculates the contract gas use
	Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) // Run runs the precompiled contract
	ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error
}

// PrecompiledContractsHomestead contains the default set of pre-compiled Ethereum
// contracts used in the Frontier and Homestead releases.
var PrecompiledContractsHomestead = map[common.Address]PrecompiledContract{
	ecrecoverPrecompileAddr:     &ecrecover{},
	sha256hashPrecompileAddr:    &sha256hash{},
	ripemd160hashPrecompileAddr: &ripemd160hash{},
	dataCopyPrecompileAddr:      &dataCopy{},

	wanCoinPrecompileAddr:  &wanCoinSC{},
	wanStampPrecompileAddr: &wanchainStampSC{},
}

// PrecompiledContractsByzantium contains the default set of pre-compiled Ethereum
// contracts used in the Byzantium release.
var PrecompiledContractsByzantium = map[common.Address]PrecompiledContract{
	ecrecoverPrecompileAddr:      &ecrecover{},
	sha256hashPrecompileAddr:     &sha256hash{},
	ripemd160hashPrecompileAddr:  &ripemd160hash{},
	dataCopyPrecompileAddr:       &dataCopy{},
	bigModExpPrecompileAddr:      &bigModExp{},
	bn256AddPrecompileAddr:       &bn256Add{},
	bn256ScalarMulPrecompileAddr: &bn256ScalarMul{},
	bn256PairingPrecompileAddr:   &bn256Pairing{},

	wanCoinPrecompileAddr:  &wanCoinSC{},
	wanStampPrecompileAddr: &wanchainStampSC{},
}
