package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	"github.com/wanchain/go-wanchain/log"
)

type BaseMpcResult struct {
	Result     map[string][]big.Int
	byteResult map[string][]byte
}

func (result *BaseMpcResult) InitializeValue(preSetValue ...MpcValue) {
	log.Warn("-----------------BaseMpcResult.InitializeValue begin")

	for i := 0; i < len(preSetValue); i++ {
		log.Warn("-----------------BaseMpcResult.InitializeValue", "value", preSetValue[i].String())

		if preSetValue[i].Value != nil {
			result.SetValue(preSetValue[i].Key, preSetValue[i].Value)
		} else if preSetValue[i].ByteValue != nil {
			result.SetByteValue(preSetValue[i].Key, preSetValue[i].ByteValue)
		}
	}
}

func createMpcBaseMpcResult() *BaseMpcResult {
	log.Warn("-----------------createMpcBaseMpcResult begin")

	return &BaseMpcResult{make(map[string][]big.Int), make(map[string][]byte)}
}

func (mpc *BaseMpcResult) SetValue(key string, value []big.Int) error {
	mpc.Result[key] = value
	return nil
}

func (mpc *BaseMpcResult) GetValue(key string) ([]big.Int, error) {
	value, exist := mpc.Result[key]
	if exist {
		return value, nil
	}

	log.Error("BaseMpcResult GetValue fail.", "key", key)
	return value, mpcprotocol.ErrMpcResultExist
}

func (mpc *BaseMpcResult) SetByteValue(key string, value []byte) error {
	mpc.byteResult[key] = value
	return nil
}

func (mpc *BaseMpcResult) GetByteValue(key string) ([]byte, error) {
	value, exist := mpc.byteResult[key]
	if exist {
		return value, nil
	}

	log.Error("-----------------GetByteValue fail", "key", key)
	return value, mpcprotocol.ErrQuit
}

func (mpc *BaseMpcResult) Initialize() error {
	log.Warn("-----------------BaseMpcResult.Initialize begin")
	return nil
}
