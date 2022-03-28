package types

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// deriveSigner makes a *best* guess about which signer to use.
//func deriveSigner(V *big.Int) Signer {
//	if V.Sign() != 0 && isProtectedV(V) {
//		return NewEIP155Signer(deriveChainId(V))
//	} else {
//		return HomesteadSigner{}
//	}
//}

////////////////////////////////////for privacy tx ///////////////////////
func NewOTATransaction(nonce uint64, to common.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	return newOTATransaction(nonce, &to, amount, gasLimit, gasPrice, data)
}

func newOTATransaction(nonce uint64, to *common.Address, amount, gasLimit, gasPrice *big.Int, data []byte) *Transaction {
	if to == nil {
		return nil
	}
	return NewWanTransaction(uint64(WanPrivTxType), nonce, *to, amount, gasLimit.Uint64(), gasPrice, data)
}

const (
	JUPITER_TX = 0xffffffff
)

func IsNormalTransaction(txType uint64) bool {
	return txType == WanLegacyTxType || txType == 0 || txType == 2 || txType == JUPITER_TX || txType == 0xff // some of old tx used , which is allowed.
}
func IsPosTransaction(txType uint64) bool {
	return txType == WanPosTxType
}
func IsPrivacyTransaction(txType uint64) bool {
	return txType == WanPrivTxType
}

//func IsValidTransactionType(txType uint64) bool {
//	return (txType == NORMAL_TX || txType == PRIVACY_TX || txType == POS_TX || txType == JUPITER_TX)
//}

func IsEthereumTx(chainId uint64) bool {
	return (chainId > 100)
}
