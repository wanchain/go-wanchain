package vm

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/crypto/sha3"
)

// SHA3-256 FIPS 202 standard implementation.

type sha3fips struct {
	contract *Contract
	evm      *EVM
}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *sha3fips) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Sha256PerWordGas + params.Sha256BaseGas
}

func (c *sha3fips) Run(input []byte) ([]byte, error) {
	hexStr := common.Bytes2Hex(input)
	pub, _ := hex.DecodeString(hexStr)
	h := sha3.Sum256(pub[:])
	return h[:], nil
}

func (s *sha3fips) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

// Uncompressed Public Key recovery implementation.
type ecrecoverPublicKey struct {
	contract *Contract
	evm      *EVM
}

func (c *ecrecoverPublicKey) RequiredGas(input []byte) uint64 {
	return params.EcrecoverGas
}

func (c *ecrecoverPublicKey) Run(input []byte) ([]byte, error) {
	const ecrecoverPublicKeyInputLength = 128

	input = common.RightPadBytes(input, ecrecoverPublicKeyInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])
	v := input[63]

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(input[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}
	// We must make sure not to modify the 'input', so placing the 'v' along with
	// the signature needs to be done on a new allocation
	sig := make([]byte, 65)
	copy(sig, input[64:128])
	sig[64] = v
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(input[:32], sig)
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}

	return pubKey, nil
}

func (s *ecrecoverPublicKey) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}
