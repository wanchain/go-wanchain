package btc

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/txscript"
	"github.com/wanchain/go-wanchain/log"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"math/big"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/accounts/keystore"
)

const BTC_VERSION = 1

type OutPointArg struct {
	Hash  string
	Index uint32
}


type TxInArgs struct {
	PreviousOutPoint OutPointArg
	SignatureScript  string			// hex string
	Sequence         uint32
	PkScript		 string			// hex string
}

type TxOutArgs struct {
	Value    uint64
	PkScript string
}

type MsgTxArgs struct {
	Version  uint32
	TxIn     []TxInArgs
	TxOut    []TxOutArgs
	LockTime uint32
	From     common.Address
}

func (msg *MsgTxArgs)Cmp(arg *MsgTxArgs) bool {
	if arg == nil {
		return msg == nil
	}

	if msg == nil {
		return false
	}

	if msg.Version != arg.Version {
		return false
	}

	if msg.LockTime != arg.LockTime {
		return false
	}

	if msg.From != arg.From {
		return false
	}

	if len(msg.TxIn) != len(arg.TxIn) {
		return false
	}

	if len(msg.TxOut) != len(arg.TxOut) {
		return false
	}

	for i, txIn := range msg.TxIn {
		if txIn != arg.TxIn[i] {
			return false
		}
	}

	for i, txOut := range msg.TxOut {
		if txOut != msg.TxOut[i] {
			return false
		}
	}

	return true
}

func GetMsgTxFromMsgTxArgs(args * MsgTxArgs) (*wire.MsgTx, error)  {
	if args == nil {
		return nil, errors.New("invalid btc MsgTxArgs")
	}

	if args.Version != BTC_VERSION {
		return nil, errors.New("invalid btc tx version")
	}

	ret := &wire.MsgTx{int32(args.Version), make([]*wire.TxIn, 0, len(args.TxIn)), make([]*wire.TxOut, 0, len(args.TxOut)), args.LockTime}

	for _, txInArgs := range args.TxIn {
		scriptBytes := common.FromHex(txInArgs.SignatureScript)
		if scriptBytes == nil {
			return nil, errors.New("invalid btc TxIn signature script!")
		}

		inTxId, err := chainhash.NewHashFromStr(txInArgs.PreviousOutPoint.Hash)
		if err != nil {
			return nil, errors.New("invalid btc TxInId!")
		}

		previousOutPoint := wire.OutPoint{*inTxId, txInArgs.PreviousOutPoint.Index}
		ret.TxIn = append(ret.TxIn, &wire.TxIn{previousOutPoint, scriptBytes, nil, txInArgs.Sequence})
	}

	for _, txOutArgs := range args.TxOut {
		scriptBytes := common.FromHex(txOutArgs.PkScript)
		if scriptBytes == nil {
			return nil, errors.New("invalid btc TxOut PkScript!")
		}

		ret.TxOut = append(ret.TxOut, &wire.TxOut{int64(txOutArgs.Value), scriptBytes})
	}

	if len(ret.TxIn) == 0 {
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxIn")
	}

	if len(ret.TxOut) == 0 {
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxOut")
	}

	return ret, nil
}


func GetHashedForEachTxIn(args *MsgTxArgs) ([]common.Hash, error) {
	log.Warn("-----------------GetHashedForEachTxIn begin")
	tx, err := GetMsgTxFromMsgTxArgs(args)
	if err != nil {
		return nil, err
	}

	hashes := []common.Hash{}
	for i := 0; i < len(args.TxIn); i++ {
		log.Warn("-----------------GetHashedForEachTxIn", "i", i, "pkScript", args.TxIn[i].PkScript)
		hash, err := txscript.CalcSignatureHash(common.FromHex(args.TxIn[i].PkScript), txscript.SigHashAll, tx, i)
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


func RecoverPublicKey(sighash common.Hash, R, S, Vb *big.Int) (error) {
	log.Warn("-----------------RecoverPublicKey begin", "R", common.ToHex(R.Bytes()), "S", common.ToHex(S.Bytes()), "V", common.ToHex(Vb.Bytes()))

	if Vb.BitLen() > 8 {
		return errors.New("invalid sign")
	}
	//V := byte(Vb.Uint64() - 27)

	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)

	vBytes := Vb.Bytes()
	if len(vBytes) > 0 {
		sig[64] = vBytes[0]
	}

	pubKey, err := crypto.SigToPub(sighash[:], sig)
	if err != nil {
		log.Error("-----------------RecoverPublicKey fail", "err", err)
	}

	pubKeyCompressed := keystore.ECDSAPKCompression(pubKey)
	log.Warn("-----------------RecoverPublicKey", "pubKeyCompressed", common.ToHex(pubKeyCompressed))
	addr := crypto.PubkeyToRipemd160(pubKey)
	log.Warn("-----------------RecoverPublicKey", "hash160", common.ToHex(addr[:]))

	return nil
}





