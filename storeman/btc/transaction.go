package btc

import (
	"github.com/wanchain/go-wanchain/common"
	"errors"
)

const BTC_VERSION = 2

// OutPoint defines a bitcoin data type that is used to track previous
// transaction outputs.
type OutPoint struct {
	Hash  common.Hash
	Index uint32
}

// TxWitness defines the witness for a TxIn. A witness is to be interpreted as
// a slice of byte slices, or a stack with one or many elements.
type TxWitness [][]byte


// TxIn defines a bitcoin transaction input.
type TxIn struct {
	PreviousOutPoint OutPoint
	SignatureScript  []byte
	Witness          TxWitness
	Sequence         uint32
}

// TxOut defines a bitcoin transaction output.
type TxOut struct {
	Value    int64
	PkScript []byte
}

type MsgTx struct {
	Version  int32
	TxIn     []*TxIn
	TxOut    []*TxOut
	LockTime uint32
}

type TxInArgs struct {
	PreviousOutPoint OutPoint
	SignatureScript  string
	Sequence         uint32
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
}

func GetMsgTxFromMsgTxArgs(args * MsgTxArgs) (*MsgTx, error)  {
	if args == nil {
		return nil, errors.New("invalid btc MsgTxArgs")
	}

	if args.Version != BTC_VERSION {
		return nil, errors.New("invalid btc tx version")
	}

	if args.LockTime != 0 {
		return nil, errors.New("invalid btc tx lock time")
	}

	ret := &MsgTx{args.Version, make([]*TxIn, 0, len(args.TxIn)), make([]*TxOut, 0, len(args.TxOut)), args.LockTime}

	for _, txInArgs := range args.TxIn {
		scriptBytes := common.FromHex(txInArgs.SignatureScript)
		if scriptBytes == nil {
			return nil, errors.New("invalid btc TxIn signature script!")
		}

		ret.TxIn = append(ret.TxIn, &TxIn{txInArgs.PreviousOutPoint, scriptBytes, nil, txInArgs.Sequence})
	}

	for _, txOutArgs := range args.TxOut {
		scriptBytes := common.FromHex(txOutArgs.PkScript)
		if scriptBytes == nil {
			return nil, errors.New("invalid btc TxOut PkScript!")
		}

		ret.TxOut = append(ret.TxOut, &TxOut{txOutArgs.Value, scriptBytes})
	}

	if len(ret.TxIn) == 0 {
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxIn")
	}

	if len(ret.TxOut) == 0 {
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxOut")
	}

	return ret, nil
}











