package core

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var (
	utilAbiDefinition = `[{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}],"name":"combine","outputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}]}]`

	utilAbi, errAbiInit = abi.JSON(strings.NewReader(utilAbiDefinition))

	TokenAbi = utilAbi
)

func init() {
	if errAbiInit != nil {
		panic(errAbiInit)
	}
}

type PrivacyTxInfo struct {
	PublicKeys         []*ecdsa.PublicKey
	KeyImage           *ecdsa.PublicKey
	W_Random           []*big.Int
	Q_Random           []*big.Int
	CallData           []byte
	StampBalance       *big.Int
	StampTotalGas      uint64
	GasLeftSubRingSign uint64
}

func FetchPrivacyTxInfo(stateDB vm.StateDB, hashInput []byte, in []byte, gasPrice *big.Int) (info *PrivacyTxInfo, err error) {
	if len(in) < 4 {
		return nil, vm.ErrInvalidRingSigned
	}

	var TxDataWithRing struct {
		RingSignedData string
		CxtCallParams  []byte
	}

	TxDataWithRingI, err := utilAbi.Unpack("combine", in[4:])
	if err != nil {
		return
	}

	TxDataWithRing.RingSignedData = TxDataWithRingI[0].(string)
	TxDataWithRing.CxtCallParams = TxDataWithRingI[1].([]byte)
	ringSignInfo, err := vm.FetchRingSignInfo(stateDB, hashInput, TxDataWithRing.RingSignedData)
	if err != nil {
		return
	}

	stampGasBigInt := new(big.Int).Div(ringSignInfo.OTABalance, gasPrice)
	if stampGasBigInt.BitLen() > 64 {
		return nil, vm.ErrOutOfGas
	}

	StampTotalGas := stampGasBigInt.Uint64()
	mixLen := len(ringSignInfo.PublicKeys)
	ringSigDiffRequiredGas := params.RequiredGasPerMixPub * (uint64(mixLen))

	// ringsign compute gas + ota image key store setting gas
	preSubGas := ringSigDiffRequiredGas + params.SstoreSetGas
	if StampTotalGas < preSubGas {
		return nil, vm.ErrOutOfGas
	}

	GasLeftSubRingSign := StampTotalGas - preSubGas
	info = &PrivacyTxInfo{
		ringSignInfo.PublicKeys,
		ringSignInfo.KeyImage,
		ringSignInfo.W_Random,
		ringSignInfo.Q_Random,
		TxDataWithRing.CxtCallParams[:],
		ringSignInfo.OTABalance,
		StampTotalGas,
		GasLeftSubRingSign,
	}

	return info, nil
}

func ValidPrivacyTx(stateDB vm.StateDB, hashInput []byte, in []byte, gasPrice *big.Int,
	intrGas *big.Int, txValue *big.Int, gasLimit *big.Int) error {
	if intrGas == nil || intrGas.BitLen() > 64 {
		return vm.ErrOutOfGas
	}

	if txValue.Sign() != 0 {
		return vm.ErrInvalidPrivacyValue
	}

	if gasPrice == nil || gasPrice.Cmp(common.Big0) <= 0 {
		return vm.ErrInvalidGasPrice
	}

	info, err := FetchPrivacyTxInfo(stateDB, hashInput, in, gasPrice)
	if err != nil {
		return err
	}

	if info.StampTotalGas > gasLimit.Uint64() {
		return ErrGasLimit
	}

	kix := crypto.FromECDSAPub(info.KeyImage)
	exist, _, err := vm.CheckOTAImageExist(stateDB, kix)
	if err != nil {
		return err
	} else if exist {
		return errors.New("stamp has been spended")
	}

	if info.GasLeftSubRingSign < intrGas.Uint64() {
		return vm.ErrOutOfGas
	}

	return nil
}

func PreProcessPrivacyTx(stateDB vm.StateDB, hashInput []byte, in []byte, gasPrice *big.Int, txValue *big.Int) (callData []byte, totalUseableGas uint64, evmUseableGas uint64, err error) {
	if txValue.Sign() != 0 {
		return nil, 0, 0, vm.ErrInvalidPrivacyValue
	}

	info, err := FetchPrivacyTxInfo(stateDB, hashInput, in, gasPrice)
	if err != nil {
		return nil, 0, 0, err
	}

	kix := crypto.FromECDSAPub(info.KeyImage)
	exist, _, err := vm.CheckOTAImageExist(stateDB, kix)
	if err != nil || exist {
		return nil, 0, 0, err
	}

	vm.AddOTAImage(stateDB, kix, info.StampBalance.Bytes())

	return info.CallData, info.StampTotalGas, info.GasLeftSubRingSign, nil
}
