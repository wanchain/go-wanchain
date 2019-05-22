package vm

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
)

const maxTopicLengh  = 4



func precompiledScMakeLog(contract *Contract, evm *EVM,evenName string,argsValue [][]byte,data []byte)  error{

	size := len(argsValue)

	if size > maxTopicLengh {
		return errors.New("topic is too long,max topic is 4")
	}

	topics := make([]common.Hash, size + 1)

	//topic[0] is for the name of event
	topics[0] = common.StringToHash(evenName)

	//topic[1:n] is for the parameters
	for i := 1; i < size + 1; i++ {
		topics[i] = common.BytesToHash(argsValue[i-1])
	}

	evm.StateDB.AddLog(&types.Log{
		Address: contract.Address(),
		Topics:  topics,
		Data:    data,
		// This is a non-consensus field, but assigned here because
		// core/state doesn't know the current block number.
		BlockNumber: evm.BlockNumber.Uint64(),
	})


	return nil
}
