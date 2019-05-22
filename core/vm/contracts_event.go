package vm

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
)


func precompiledScAddLog(contractAddress common.Address, evm *EVM,evenID common.Hash,argsValue []common.Hash,data []byte)  error{
	topics := make([]common.Hash, 1)

	//topic[0] is for the name of event
	topics[0] = evenID
	topics = append(topics, argsValue...)

	evm.StateDB.AddLog(&types.Log{
		Address: contractAddress,
		Topics:  topics,
		Data:    data,
		// This is a non-consensus field, but assigned here because
		// core/state doesn't know the current block number.
		BlockNumber: evm.BlockNumber.Uint64(),
	})

	return nil
}
