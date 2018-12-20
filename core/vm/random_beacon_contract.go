package vm

import "github.com/wanchain/go-wanchain/core/types"

type RandomBeaconContract struct {}


func (c *RandomBeaconContract) RequiredGas(input []byte) uint64 {
	return 10000
}
func (c *RandomBeaconContract) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	// *********
	return nil, nil
}

func (c *RandomBeaconContract) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	// *******************
	return nil
}
