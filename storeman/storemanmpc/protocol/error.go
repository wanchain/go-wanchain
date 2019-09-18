package protocol

import "errors"

var (
	ErrQuit              = errors.New("quit")
	ErrMpcResultExist    = errors.New("mpc Result is not exist")
	ErrContextType       = errors.New("err Context Type is error")
	ErrTimeOut           = errors.New("mpc Request is TimeOut")
	ErrPointZero         = errors.New("mpc Point is zero")
	ErrChainTypeError    = errors.New("mpc transaction chain type error")
	ErrMpcSeedOutRange   = errors.New("mpc seeds are out range")
	ErrMpcSeedDuplicate  = errors.New("mpc seeds have duplicate")
	ErrDecrypt           = errors.New("could not decrypt key with given password")
	ErrTooLessStoreman   = errors.New("mpc alive Storeman is not enough")
	ErrFailedTxVerify    = errors.New("mpc signing transaction verify failed")
	ErrMpcContextExist   = errors.New("mpc Context ID is already exist")
	ErrInvalidMPCAddr    = errors.New("invalid mpc account address")
	ErrFailSignRetVerify = errors.New("mpc signing result verify failed")
	ErrInvalidSignedData = errors.New("invalid signed Data")
	ErrInvalidMPCR       = errors.New("invalid signed data(R)")
	ErrInvalidMPCS       = errors.New("invalid signed data(S)")
)
