package crypto

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/rlp"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
)

type WanMPCTxSigner struct {
	signer types.Signer
}

func (sign *WanMPCTxSigner) Hash(tx *types.Transaction) common.Hash {
	return sign.signer.Hash(tx)
}

func (sign *WanMPCTxSigner) SignTransaction(tx *types.Transaction, R *big.Int, S *big.Int, V *big.Int) ([]byte, common.Address, error) {
	sig, err := TransSignature(R, S, V)
	if err != nil {
		mpcsyslog.Err("wan mpc sign fail, sign error:%s", err.Error())
		return nil, common.Address{}, err
	}

	tx, err = tx.WithSignature(sign.signer, sig)
	if err != nil {
		mpcsyslog.Err("wan mpc sign fail, with signature error:%s", err.Error())
		return nil, common.Address{}, err
	}

	from, err := types.Sender(sign.signer, tx)
	if err != nil {
		mpcsyslog.Err("wan mpc sign fail, get sender error:%s", err.Error())
		return nil, common.Address{}, err
	}

	txSign, err := rlp.EncodeToBytes(tx)
	if err != nil {
		mpcsyslog.Err("wan mpc sign fail, rlp encode error:%s", err.Error())
		return nil, common.Address{}, err
	}

	mpcsyslog.Info("wan mpc sign success, signed tx raw:%s", common.ToHex(txSign))
	return txSign, from, nil
}

func CreateWanMPCTxSigner(chainID *big.Int) *WanMPCTxSigner {
	return &WanMPCTxSigner{signer: types.NewEIP155Signer(chainID)}
}
