package protocol

import "errors"

var (
	ErrQuit              = errors.New("quit")
	ErrMpcResultExist    = errors.New("mpc Result is not exist")
	ErrContextType       = errors.New("err Context Type is error")
	ErrTimeOut           = errors.New("mpc Request is TimeOut")
	ErrAddress           = errors.New("mpc Address is not found")
	ErrChainID           = errors.New("mpc ChainID is not Defined")
	ErrPointZero         = errors.New("mpc Point is zero")
	ErrChainTypeError    = errors.New("mpc transaction chain type error")
	ErrMpcSeedOutRange   = errors.New("mpc seeds are out range")
	ErrMpcSeedDuplicate  = errors.New("mpc seeds have duplicate")
	ErrDecrypt           = errors.New("could not decrypt key with given password")
	ErrTooLessStoreman   = errors.New("mpc alive Storeman is not enough")
	ErrTooMoreStoreman   = errors.New("mpc alive Storeman is too more")
	ErrFailedTxVerify    = errors.New("mpc signing transaction verify failed")
	ErrMpcContextExist   = errors.New("mpc Context ID is already exist")
	ErrInvalidMPCAddr    = errors.New("invalid mpc account address")
	ErrFailSignRetVerify = errors.New("mpc signing result verify failed")
	ErrInvalidStmAccType = errors.New("invalid storeman account type! please input 'WAN' or 'ETH' or 'BTC' ")
	ErrInvalidMpcTx      = errors.New("invalid mpc transaction")
	ErrInvalidSignedData = errors.New("invalid signed Data")
	ErrInvalidMPCR       = errors.New("invalid signed data(R)")
	ErrInvalidMPCS       = errors.New("invalid signed data(S)")
)
