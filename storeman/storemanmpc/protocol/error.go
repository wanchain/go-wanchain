package protocol

import "errors"

var (
	ErrQuit              = errors.New("Quit")
	ErrMpcResultExist    = errors.New("Mpc Result is not exist")
	ErrContextType       = errors.New("Err Context Type is error")
	ErrTimeOut           = errors.New("Mpc Requst is TimeOut")
	ErrAddress           = errors.New("Mpc Address is not found")
	ErrChainID           = errors.New("Mpc ChainID is not Defined")
	ErrPointZero         = errors.New("Mpc Point is zero")
	ErrChainTypeError    = errors.New("Mpc transaction chaintype error")
	ErrMpcSeedOutRange   = errors.New("Mpc seeds are out range")
	ErrMpcSeedDuplicate  = errors.New("Mpc seeds have duplicate")
	ErrDecrypt           = errors.New("could not decrypt key with given passphrase")
	ErrTooLessStoreman   = errors.New("Mpc alived Storeman is not enough")
	ErrTooMoreStoreman   = errors.New("Mpc alived Storeman is too more")
	ErrFailedTxVerify    = errors.New("Mpc signing transaction verify failed")
	ErrMpcContextExist   = errors.New("Mpc Context ID is already exist")
	ErrInvalidMPCAddr    = errors.New("Invalid mpc account address")
	ErrFailSignRetVerify = errors.New("Mpc signing result verify failed")
	ErrInvalidStmAccType = errors.New("invalid storeman account type! please input 'WAN' or 'ETH' or 'BTC' ")
	ErrInvalidMpcTx      = errors.New("invalid mpc transaction")
	ErrInvalidSignedData = errors.New("invalid signed Data")
)
