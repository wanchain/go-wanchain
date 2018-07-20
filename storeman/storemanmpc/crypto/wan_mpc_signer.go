package crypto

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rlp"
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
		log.Error("wan mpc sign fail", "sign error", err)
		return nil, common.Address{}, err
	}

	tx, err = tx.WithSignature(sign.signer, sig)
	if err != nil {
		log.Error("wan mpc sign fail", "with signature error", err)
		return nil, common.Address{}, err
	}

	log.Debug("wan mpc sign", "Tx", tx.String())
	from, err := types.Sender(sign.signer, tx)
	if err != nil {
		log.Error("wan mpc sign fail", "get sender error", err)
		return nil, common.Address{}, err
	}

	log.Debug("wan mpc sign", "from", from)
	txSign, err := rlp.EncodeToBytes(tx)
	if err != nil {
		log.Error("wan mpc sign fail", "rlp encode error", err)
		return nil, common.Address{}, err
	}

	log.Debug("wan mpc sign success", "signed tx raw", common.ToHex(txSign))
	return txSign, from, nil
}

func CreateWanMPCTxSigner(chainID *big.Int) *WanMPCTxSigner {
	return &WanMPCTxSigner{signer: types.NewEIP155Signer(chainID)}
}
