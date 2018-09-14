package btc

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/txscript"
	"github.com/wanchain/go-wanchain/log"
)

const BTC_VERSION = 2

type TxInArgs struct {
	PreviousOutPoint wire.OutPoint
	SignatureScript  string			// hex string
	Sequence         uint32
	PubKeyScrip		 string			// hex string
}

type TxOutArgs struct {
	Value    int64
	PkScript string
}

type MsgTxArgs struct {
	Version  int32
	TxIn     []TxInArgs
	TxOut    []TxOutArgs
	LockTime uint32
	From     common.Address
}

func GetMsgTxFromMsgTxArgs(args * MsgTxArgs) (*wire.MsgTx, error)  {
	if args == nil {
		return nil, errors.New("invalid btc MsgTxArgs")
	}

	if args.Version != BTC_VERSION {
		return nil, errors.New("invalid btc tx version")
	}

	if args.LockTime != 0 {
		return nil, errors.New("invalid btc tx lock time")
	}

	ret := &wire.MsgTx{args.Version, make([]*wire.TxIn, 0, len(args.TxIn)), make([]*wire.TxOut, 0, len(args.TxOut)), args.LockTime}

	for _, txInArgs := range args.TxIn {
		scriptBytes := common.FromHex(txInArgs.SignatureScript)
		if scriptBytes == nil {
			return nil, errors.New("invalid btc TxIn signature script!")
		}

		ret.TxIn = append(ret.TxIn, &wire.TxIn{txInArgs.PreviousOutPoint, scriptBytes, nil, txInArgs.Sequence})
	}

	for _, txOutArgs := range args.TxOut {
		scriptBytes := common.FromHex(txOutArgs.PkScript)
		if scriptBytes == nil {
			return nil, errors.New("invalid btc TxOut PkScript!")
		}

		ret.TxOut = append(ret.TxOut, &wire.TxOut{txOutArgs.Value, scriptBytes})
	}

	if len(ret.TxIn) == 0 {
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxIn")
	}

	if len(ret.TxOut) == 0 {
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxOut")
	}

	return ret, nil
}

//func PrintBtcTx(args * MsgTxArgs) {
//	if args == nil {
//		return
//	}
//
//
//}

func GetHashedForEachTxIn(args *MsgTxArgs) ([]common.Hash, error) {
	log.Warn("-----------------GetHashedForEachTxIn begin")
	tx, err := GetMsgTxFromMsgTxArgs(args)
	if err != nil {
		return nil, err
	}

	hashes := []common.Hash{}
	for i := 0; i < len(args.TxIn); i++ {
		hash, err := txscript.CalcSignatureHash(common.FromHex(args.TxIn[i].PubKeyScrip), txscript.SigHashAll, tx, i)
		if err != nil {
			log.Error("GetHashedForEachTxIn, CalcSignatureHash fail.", "err", err)
			return nil, err
		}

		log.Warn("-----------------GetHashedForEachTxIn", "i", i, "hash", common.ToHex(hash))
		hashes = append(hashes, common.BytesToHash(hash))
	}

	log.Warn("-----------------GetHashedForEachTxIn succeed")
	return hashes, nil
}









