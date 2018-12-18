package vm

import (
	"encoding/hex"
	"fmt"

	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/rlp"
)

type wanSlotLeaderCommitment struct {
}

//RequiredPrice calculates the contract gas use
func (s *wanSlotLeaderCommitment) RequiredGas(input []byte) uint64 {
	return 0
}

func (s *wanSlotLeaderCommitment) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if input == nil {
		return nil, errParameters
	}

	compressedPublicKeyLen := 33

	if len(input) < compressedPublicKeyLen*2 {
		return nil, errParameters
	}

	var params [][]byte
	err := rlp.DecodeBytes(input, &params)
	if err != nil {
		return nil, err
	}

	fmt.Println("epochID: ", hex.EncodeToString(params[0]))
	fmt.Println("pk: ", hex.EncodeToString(params[1]))
	fmt.Println("mi: ", hex.EncodeToString(params[2]))

	return nil, nil
}

func (s *wanSlotLeaderCommitment) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}
