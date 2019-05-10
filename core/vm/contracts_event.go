package vm

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
)

const maxTopicLengh  = 4

func precompiledScMakeLog(contract *Contract, evm *EVM,evenName string,argsValue [][]byte,data []byte)  error{

	size := len(argsValue)

	if size > maxTopicLengh {
		return errors.New("topic is too long,max topic is 4")
	}

	topics := make([]common.Hash, size + 1)
	if len(evenName) > common.HashLength {
		topics[0] = crypto.Keccak256Hash([]byte(evenName))
	} else {
		topics[0] = common.StringToHash(evenName)
	}

	for i := 1; i < size; i++ {
		if len(argsValue) > common.HashLength {
			topics[i] = crypto.Keccak256Hash(argsValue[i-1])
		} else {
			topics[i] = common.BytesToHash(argsValue[i-1])
		}
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
