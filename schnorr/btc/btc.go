package btc

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"math/big"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/log"
	"strconv"
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

func (txIn * TxInArgs) String() string {
	var ret string
	ret += "PreOutHash: " + txIn.PreviousOutPoint.Hash + ", "
	ret += "PreOutIndex: " + strconv.Itoa(int(txIn.PreviousOutPoint.Index)) + ", "
	ret += "SignatureScript: " + txIn.SignatureScript + ", "
	ret += "Sequence: " + strconv.Itoa(int(txIn.Sequence)) + ", "
	ret += "PkScript: " + txIn.PkScript
	return ret
}

func (txOut * TxOutArgs) String() string {
	var ret string
	ret += "value: " + strconv.Itoa(int(txOut.Value)) + ", "
	ret += "PkScript: " + txOut.PkScript
	return ret
}

func (msg *MsgTxArgs) String() string {
	var ret string
	ret += "version: " + strconv.Itoa(int(msg.Version)) + ", "
	ret += "locktime: " + strconv.Itoa(int(msg.LockTime)) + ", "
	ret += "from: " + msg.From.String() + ", "
	for i, txIn := range msg.TxIn {
		ret += "TxIn" + strconv.Itoa(i) + ": {" + txIn.String() + "}, "
	}

	for i, txOut := range msg.TxOut {
		ret += "TxOut" + strconv.Itoa(i) + ": {" + txOut.String() + "}, "
	}

	return ret
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
	log.SyslogInfo("GetMsgTxFromMsgTxArgs, begin")
	if args == nil {
		log.SyslogErr("GetMsgTxFromMsgTxArgs, invalid btc MsgTxArgs")
		return nil, errors.New("invalid btc MsgTxArgs")
	}

	if args.Version != BTC_VERSION {
		log.SyslogErr("GetMsgTxFromMsgTxArgs, invalid btc tx version", "version", args.Version)
		return nil, errors.New("invalid btc tx version")
	}

	ret := &wire.MsgTx{int32(args.Version), make([]*wire.TxIn, 0, len(args.TxIn)), make([]*wire.TxOut, 0, len(args.TxOut)), args.LockTime}

	for _, txInArgs := range args.TxIn {
		scriptBytes := common.FromHex(txInArgs.SignatureScript)
		if scriptBytes == nil {
			scriptBytes = *new([]byte)
		}

		inTxId, err := chainhash.NewHashFromStr(txInArgs.PreviousOutPoint.Hash)
		if err != nil {
			log.SyslogErr("GetMsgTxFromMsgTxArgs, invalid btc TxInId", "id", txInArgs.PreviousOutPoint.Hash)
			return nil, errors.New("invalid btc TxInId!")
		}

		previousOutPoint := wire.OutPoint{*inTxId, txInArgs.PreviousOutPoint.Index}
		ret.TxIn = append(ret.TxIn, &wire.TxIn{previousOutPoint, scriptBytes, nil, txInArgs.Sequence})
	}

	for _, txOutArgs := range args.TxOut {
		scriptBytes := common.FromHex(txOutArgs.PkScript)
		if scriptBytes == nil {
			log.SyslogErr("GetMsgTxFromMsgTxArgs, invalid btc TxOut PkScript", "script", txOutArgs.PkScript)
			return nil, errors.New("invalid btc TxOut PkScript!")
		}

		ret.TxOut = append(ret.TxOut, &wire.TxOut{int64(txOutArgs.Value), scriptBytes})
	}

	if len(ret.TxOut) == 0 {
		log.SyslogErr("GetMsgTxFromMsgTxArgs, invalid btc MsgTxArgs, doesn't have TxOut")
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxOut")
	}

	log.SyslogInfo("GetMsgTxFromMsgTxArgs, succeed")
	return ret, nil
}


func GetHashedForEachTxIn(args *MsgTxArgs) ([]common.Hash, error) {
	log.SyslogInfo("GetHashedForEachTxIn, begin")
	tx, err := GetMsgTxFromMsgTxArgs(args)
	if err != nil {
		return nil, err
	}

	hashes := []common.Hash{}
	for i := 0; i < len(args.TxIn); i++ {
		log.SyslogInfo("GetHashedForEachTxIn", "i", i, "TxInPkScript", args.TxIn[i].PkScript)
		hash, err := txscript.CalcSignatureHash(common.FromHex(args.TxIn[i].PkScript), txscript.SigHashAll, tx, i)
		if err != nil {
			log.SyslogErr("GetHashedForEachTxIn, CalcSignatureHash fail", "err", err.Error())
			return nil, err
		}

		log.SyslogInfo("GetHashedForEachTxIn", "i", i, "hash", common.ToHex(hash))
		hashes = append(hashes, common.BytesToHash(hash))
	}

	log.SyslogInfo("GetHashedForEachTxIn, succeed")
	return hashes, nil
}


func RecoverPublicKey(sighash common.Hash, R, S, Vb *big.Int) (common.Address, error) {
	log.SyslogInfo("RecoverPublicKey", "Hash", sighash.String(), "R", common.ToHex(R.Bytes()), "S", common.ToHex(S.Bytes()), "V", common.ToHex(Vb.Bytes()))

	if Vb.BitLen() > 8 {
		log.SyslogErr("RecoverPublicKey, invalid sign")
		return common.Address{}, errors.New("invalid sign")
	}

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
		log.SyslogErr("RecoverPublicKey, fail", "err", err.Error())
		return common.Address{}, err
	}

	pubKeyCompressed := keystore.ECDSAPKCompression(pubKey)
	hash160 := crypto.PubkeyToRipemd160(pubKey)
	log.SyslogInfo("RecoverPublicKey", "pubKeyCompressed", common.ToHex(pubKeyCompressed), "hash160", common.ToHex(hash160[:]))
	return hash160, nil
}





