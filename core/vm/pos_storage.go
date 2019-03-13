package vm

import (
	"github.com/wanchain/go-wanchain/common"
	"errors"
)

//store information to the list
func StoreInfo(statedb StateDB,listAddr common.Address,pubHash common.Hash, info []byte) error {
	if statedb == nil {
		return ErrUnknown
	}

	if pubHash == (common.Hash{})  {
		return errors.New("public key hash is not right")
	}

	statedb.SetStateByteArray(listAddr,pubHash, info)

	return nil
}

//get stored info
func GetInfo(statedb StateDB,listAddr common.Address,pubHash common.Hash) ([]byte, error) {
	if statedb == nil {
		return nil, ErrUnknown
	}

	if pubHash == (common.Hash{})  {
		return nil,errors.New("public key hash is not right")
	}

	info := statedb.GetStateByteArray(listAddr, pubHash)

	return info, nil
}

//store information to the list
func UpdateInfo(statedb StateDB,listAddr common.Address,pubHash common.Hash, info []byte) error {
	return StoreInfo(statedb,listAddr,pubHash,info)
}

