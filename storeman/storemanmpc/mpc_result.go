package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
)

type BaseMpcResult struct {
	Result     map[string][]big.Int
	byteResult map[string][]byte
}

func (result *BaseMpcResult) InitializeValue(preSetValue ...MpcValue) {
	mpcsyslog.Info("BaseMpcResult.InitializeValue begin")

	for i := 0; i < len(preSetValue); i++ {
		if preSetValue[i].Value != nil {
			result.SetValue(preSetValue[i].Key, preSetValue[i].Value)
		} else if preSetValue[i].ByteValue != nil {
			result.SetByteValue(preSetValue[i].Key, preSetValue[i].ByteValue)
		}
	}
}

func createMpcBaseMpcResult() *BaseMpcResult {
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

	mpcsyslog.Err("BaseMpcResult GetValue fail. key:%s", key)
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

	mpcsyslog.Err("GetByteValue fail, key:%s", key)
	return value, mpcprotocol.ErrQuit
}

func (mpc *BaseMpcResult) Initialize() error {
	return nil
}
