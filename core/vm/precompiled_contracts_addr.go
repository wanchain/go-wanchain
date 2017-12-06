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
	otaBalancePercent1WStorageAddr = common.HexToAddress(WanStamp0dot1)
	otaBalancePercent2WStorageAddr = common.HexToAddress(WanStamp0dot2)
	otaBalancePercent5WStorageAddr = common.HexToAddress(WanStamp0dot5)
	otaBalanceTenth1WStorageAddr   = common.HexToAddress(Wancoindot1)
	otaBalanceTenth2WStorageAddr   = common.HexToAddress(Wancoindot2)
	otaBalanceTenth5WStorageAddr   = common.HexToAddress(Wancoindot5)
	otaBalance1WStorageAddr        = common.HexToAddress(Wancoin1)
	otaBalance2WStorageAddr        = common.HexToAddress(Wancoin2)
	otaBalance5WStorageAddr        = common.HexToAddress(Wancoin5)
	otaBalance10WStorageAddr       = common.HexToAddress(Wancoin10)
	otaBalance20WStorageAddr       = common.HexToAddress(Wancoin20)
	otaBalance50WStorageAddr       = common.HexToAddress(Wancoin50)
	otaBalance100WStorageAddr      = common.HexToAddress(Wancoin100)
)

// PrecompiledContract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	RequiredGas(input []byte) uint64                                // RequiredPrice calculates the contract gas use
	Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) // Run runs the precompiled contract
	InvalidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error
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
