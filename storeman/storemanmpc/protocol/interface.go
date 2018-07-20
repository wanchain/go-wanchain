package protocol

import "math/big"

type MpcResultInterface interface {
	Initialize() error
	SetValue(key string, value []big.Int) error
	GetValue(key string) ([]big.Int, error)
	SetByteValue(key string, value []byte) error
	GetByteValue(key string) ([]byte, error)
}
