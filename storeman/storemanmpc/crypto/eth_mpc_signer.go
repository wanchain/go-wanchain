package crypto

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto/ethtrans"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
)

type EthMPCTxSigner struct {
	ethtrans.Signer
}

func CreateEthMPCTxSigner(chainID *big.Int) *EthMPCTxSigner {
	return &EthMPCTxSigner{Signer: ethtrans.NewEIP155Signer(chainID)}
}

func (sign *EthMPCTxSigner) Hash(tx *types.Transaction) common.Hash {
	return sign.Signer.Hash(ethtrans.NewTransactionFromWan(tx))
}

func (sign *EthMPCTxSigner) SignTransaction(tx1 *types.Transaction, R *big.Int, S *big.Int, V *big.Int) ([]byte, common.Address, error) {
	sig, err := TransSignature(R, S, V)
	if err != nil {
		log.SyslogErr("eth mpc sign fail", "sign error", err.Error())
		return nil, common.Address{}, err
	}

	tx := ethtrans.NewTransactionFromWan(tx1)
	tx, err = tx.WithSignature(sign.Signer, sig)
	if err != nil {
		log.SyslogErr("eth mpc sign fail", "with signature error", err.Error())
		return nil, common.Address{}, err
	}

	from, err := ethtrans.Sender(sign.Signer, tx)
	if err != nil {
		log.SyslogErr("eth mpc sign fail", "get sender error", err.Error())
		return nil, common.Address{}, err
	}

	txSign, err := rlp.EncodeToBytes(tx)
	if err != nil {
		log.SyslogErr("eth mpc sign fail", "rlp encode error", err.Error())
		return nil, common.Address{}, err
	}

	log.SyslogInfo("eth mpc sign success", "signed tx raw", common.ToHex(txSign))
	return txSign, from, nil
}
