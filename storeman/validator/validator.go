package validator

import (
	"encoding/json"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/rlp"
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"time"
	"github.com/wanchain/go-wanchain/storeman/btc"
	"math/big"
	"errors"
	"github.com/btcsuite/btcd/txscript"
	"bytes"
)

var noticeFuncIds [][4]byte

func init() {
	noticeFuncDefs := []string{
		"btc2wbtcLockNotice(address,address,bytes32,bytes32,uint256)",
		"wbtc2btcLockNotice(address,address,address,bytes32,bytes32,uint256)",}

	var funcId [4]byte
	for _, funcDef := range noticeFuncDefs {
		copy(funcId[:], crypto.Keccak256([]byte(funcDef))[:4])
		noticeFuncIds = append(noticeFuncIds, funcId)
		mpcsyslog.Info("validator.init, add notice func id, id:%s", common.ToHex(funcId[:]))
	}
}

func ValidateTx(signer mpccrypto.MPCTxSigner, from common.Address, chainType string, chainId *big.Int, leaderTxRawData []byte, leaderTxLeaderHashBytes []byte) bool {
	mpcsyslog.Info("ValidateTx, from:%s, chainType:%s, chainId:%s, leaderTxLeaderHashBytes:%s, leaderTxRawData:%s",
		from.String(), chainType, chainId.String(), common.ToHex(leaderTxLeaderHashBytes), common.ToHex(leaderTxRawData))

	var leaderTx types.Transaction
	err := rlp.DecodeBytes(leaderTxRawData, &leaderTx)
	if err != nil {
		mpcsyslog.Err("ValidateTx leader tx data decode fail. err:%s", err.Error())
		return false
	}

	mpcsyslog.Info("ValidateTx, leaderTxData:%s", common.ToHex(leaderTx.Data()))
	isNotice, err := IsNoticeTransaction(leaderTx.Data())
	if err != nil {
		mpcsyslog.Err("ValidateTx, check notice transaction fail, err:", err.Error())
	} else if isNotice {
		mpcsyslog.Info("ValidateTx, is notice transaction, skip validating")
		return true
	}

	key := GetKeyFromTx(&from, leaderTx.To(), leaderTx.Value(), leaderTx.Data(), &chainType, chainId)
	mpcsyslog.Info("mpc ValidateTx. key:%s", common.ToHex(key))

	followerDB, err := GetDB()
	if err != nil {
		mpcsyslog.Err("ValidateTx leader get database fail. err:%s", err.Error())
		return false
	}

	_, err = waitKeyFromDB([][]byte{key})
	if err != nil {
		mpcsyslog.Err("ValidateTx, check has fail. err:%s", err.Error())
		return false
	}

	followerTxRawData, err := followerDB.Get(key)
	if err != nil {
		mpcsyslog.Err("ValidateTx, getting followerTxRawData fail. err:%s", err.Error())
		return false
	}

	mpcsyslog.Info("ValidateTx, followerTxRawData is got")

	var followerRawTx mpcprotocol.SendTxArgs
	err = json.Unmarshal(followerTxRawData, &followerRawTx)
	if err != nil {
		mpcsyslog.Err("ValidateTx, follower tx data decode fail. err:%s", err.Error())
		return false
	}

	followerCreatedTx := types.NewTransaction(leaderTx.Nonce(), *followerRawTx.To, followerRawTx.Value.ToInt(),
		leaderTx.Gas(), leaderTx.GasPrice(), followerRawTx.Data)
	followerCreatedHash := signer.Hash(followerCreatedTx)
	leaderTxLeaderHash := common.BytesToHash(leaderTxLeaderHashBytes)

	if followerCreatedHash == leaderTxLeaderHash {
		mpcsyslog.Info("ValidateTx, validate success")
		return true
	} else {
		mpcsyslog.Err("ValidateTx, leader tx hash is not same with follower tx hash. leaderTxLeaderHash:%s, followerCreatedHash:%s",
			leaderTxLeaderHash.String(), followerCreatedHash.String())
		return false
	}
}


func ValidateBtcTx(args *btc.MsgTxArgs) bool {
	if args == nil {
		return false
	}

	mpcsyslog.Info("ValidateBtcTx, begin, txInfo", args.String())

	var txOutScript [25]byte
	txOutScript[0] = txscript.OP_DUP
	txOutScript[1] = txscript.OP_HASH160
	txOutScript[2] = 0x14
	copy(txOutScript[3:23], args.From[:])
	txOutScript[23] = txscript.OP_EQUALVERIFY
	txOutScript[24] = txscript.OP_CHECKSIG

	for i := 1; i < len(args.TxOut); i++ {
		mpcsyslog.Info("ValidateBtcTx, outScript:%s", args.TxOut[i].PkScript)

		if !bytes.Equal(txOutScript[:], common.FromHex(args.TxOut[i].PkScript)) {
			mpcsyslog.Err("ValidateBtcTx, check has fail. err:invalid tx out pkscript")
			return false
		}
	}

	keyWithoutTxin, keyWithTxin := GetKeyFromBtcTx(args)
	mpcsyslog.Info("ValidateBtcTx, keyWithoutTxin:%s, keyWithTxin:%s", common.ToHex(keyWithoutTxin), common.ToHex(keyWithTxin))

	key, err := waitKeyFromDB([][]byte{keyWithTxin, keyWithoutTxin})
	if err != nil {
		mpcsyslog.Err("ValidateBtcTx, check has fail. err:%s", err.Error())
		return false
	} else {
		mpcsyslog.Info("ValidateBtcTx, key is got, key:" + common.ToHex(key))
		return true
	}
}

func waitKeyFromDB(keys [][]byte) ([]byte, error) {
	mpcsyslog.Info("waitKeyFromDB, begin")

	for i, key := range keys {
		mpcsyslog.Info("waitKeyFromDB, i:%d, key:%s", i, common.ToHex(key))
	}

	db, err := GetDB()
	if err != nil {
		mpcsyslog.Err("waitKeyFromDB get database fail. err:%s", err.Error())
		return nil, err
	}

	start := time.Now()
	for {
		for _, key := range keys {
			isExist, err := db.Has(key)
			if err != nil {
				mpcsyslog.Err("waitKeyFromDB fail, err:%s", err.Error())
				return nil, err
			} else if isExist {
				mpcsyslog.Info("waitKeyFromDB, got it, key:%s", common.ToHex(key))
				return key, nil
			}
		}

		if time.Now().Sub(start) >= mpcprotocol.MPCTimeOut {
			mpcsyslog.Info("waitKeyFromDB, time out")
			return nil, errors.New("waitKeyFromDB, time out")
		}

		time.Sleep(200 * time.Microsecond)
	}

	return nil, errors.New("waitKeyFromDB, unknown error")
}


func GetKeyFromBtcTx(args *btc.MsgTxArgs) (keyWithoutTxIn []byte, keyWithTxIn []byte) {
	keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(args.Version)).Bytes()...)
	keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(args.LockTime)).Bytes()...)

	for _, out := range args.TxOut {
		keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(out.Value)).Bytes()...)
		keyWithoutTxIn = append(keyWithoutTxIn, []byte(out.PkScript)...)
		mpcsyslog.Info("GetKeyFromBtcTx, out.PkScript:%s", out.PkScript)
		break
	}

	keyWithTxIn = make([]byte, len(keyWithoutTxIn))
	copy(keyWithTxIn, keyWithoutTxIn)
	mpcsyslog.Info("GetKeyFromBtcTx, keyWithTxin:%s, keyWithoutTxIn:%s", common.ToHex(keyWithTxIn), common.ToHex(keyWithoutTxIn))
	for _, in := range args.TxIn {
		mpcsyslog.Info("GetKeyFromBtcTx, txInPreOutHash:%s, txInIndex:%d", in.PreviousOutPoint.Hash, in.PreviousOutPoint.Index)
		keyWithTxIn = append(keyWithTxIn, in.PreviousOutPoint.Hash[:]...)
		keyWithTxIn = append(keyWithTxIn, big.NewInt(int64(in.PreviousOutPoint.Index)).Bytes()...)
		keyWithTxIn = append(keyWithTxIn, []byte(in.PkScript)...)
		keyWithTxIn = append(keyWithTxIn, big.NewInt(int64(in.Sequence)).Bytes()...)
	}

	keyWithoutTxIn = crypto.Keccak256(keyWithoutTxIn)
	keyWithTxIn = crypto.Keccak256(keyWithTxIn)

	return keyWithoutTxIn, keyWithTxIn
}

func GetKeyFromTx(from *common.Address, to *common.Address, value *big.Int, data []byte, chainType *string, chainId *big.Int) []byte {
	key := make([]byte, 0)
	key = append(key, from.Bytes()...)
	key = append(key, to.Bytes()...)
	key = append(key, value.Bytes()...)
	key = append(key, data...)
	key = append(key, []byte(*chainType)...)
	key = append(key, chainId.Bytes()...)

	return crypto.Keccak256(key)
}

func IsNoticeTransaction(payload []byte) (bool, error) {
	mpcsyslog.Info("IsNoticeTransaction, payload:%s", common.ToHex(payload))
	if len(payload) < 4 {
		return false, errors.New("invalid payload length")
	}

	var callFuncId [4]byte
	copy(callFuncId[:], payload[:4])
	mpcsyslog.Info("IsNoticeTransaction, callFuncId:%s", common.ToHex(callFuncId[:]))
	for _, noticeFuncId := range noticeFuncIds {
		if callFuncId == noticeFuncId {
			mpcsyslog.Info("IsNoticeTransaction, is notice")
			return true, nil
		}
	}

	mpcsyslog.Info("IsNoticeTransaction, is not notice")
	return false, nil
}


func AddValidMpcTx(tx *mpcprotocol.SendTxArgs) error {
	mpcsyslog.Info("AddValidMpcTx, txInfo:%s", tx.String())

	var key, val []byte
	if tx.Value == nil {
		err := errors.New("tx.Value field is required")
		mpcsyslog.Err("AddValidMpcTx, invalid input. err:%s", err.Error())
		return err
	}

	if tx.Data == nil {
		err := errors.New("tx.Data should not be empty")
		mpcsyslog.Err("AddValidMpcTx, invalid input. err:%s", err.Error())
		return err
	}

	key = GetKeyFromTx(&tx.From, tx.To, (*big.Int)(tx.Value), tx.Data, &tx.ChainType, (*big.Int)(tx.ChainID))

	val, err := json.Marshal(&tx)
	if err != nil {
		mpcsyslog.Err("AddValidMpcTx, marshal fail. err:%s", err.Error())
		return err
	}

	return addKeyValueToDB(key, val)
}

func AddValidMpcBtcTx(args *btc.MsgTxArgs) error {
	mpcsyslog.Info("AddValidMpcBtcTx, begin, txInfo:%s", args.String())

	msgTx, err := btc.GetMsgTxFromMsgTxArgs(args)
	if err != nil {
		return err
	}

	for _, txIn := range msgTx.TxIn {
		mpcsyslog.Info("AddValidMpcBtcTx, txInPreOutHash:%s, txInIndex:%d", txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index)
	}
	for _, txOut := range msgTx.TxOut {
		mpcsyslog.Info("AddValidMpcBtcTx, txOutPkScript:%s", common.ToHex(txOut.PkScript))
	}

	_, key := GetKeyFromBtcTx(args)
	val, err := json.Marshal(args)
	if err != nil {
		mpcsyslog.Err("AddValidMpcBtcTxRaw, marshal fail. err:%s", err.Error())
		return err
	}

	return addKeyValueToDB(key, val)
}

func addKeyValueToDB(key, value []byte) error {
	mpcsyslog.Info("addKeyValueToDB, begin, key:", common.ToHex(key))
	sdb, err := GetDB()
	if err != nil {
		mpcsyslog.Err("addKeyValueToDB, getting storeman database fail. err:%s", err.Error())
		return err
	}

	err = sdb.Put(key, value)
	if err != nil {
		mpcsyslog.Err("addKeyValueToDB, getting storeman database fail. err:%s", err.Error())
		return err
	}

	mpcsyslog.Info("addKeyValueToDB. key:%s", common.ToHex(key))
	ret, err := sdb.Get(key)
	if err != nil {
		mpcsyslog.Err("addKeyValueToDB, getting storeman database fail. err:%s", err.Error())
		return err
	}

	mpcsyslog.Info("addKeyValueToDB succeed to get data from leveldb after putting key-val pair. ret:%s", string(ret))
	return nil
}
