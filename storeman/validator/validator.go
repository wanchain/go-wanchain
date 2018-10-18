package validator

import (
	"encoding/json"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
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
		log.Info("validator.init, add notice func id", "id", common.ToHex(funcId[:]))
	}
}

func ValidateTx(signer mpccrypto.MPCTxSigner, leaderTxRawData []byte, leaderTxLeaderHashBytes []byte) bool {
	log.Info("ValidateTx, begin")
	mpcsyslog.Info("ValidateTx, leaderTxLeaderHashBytes:%s, leaderTxRawData:%s", common.ToHex(leaderTxLeaderHashBytes), common.ToHex(leaderTxRawData))

	var leaderTx types.Transaction
	err := rlp.DecodeBytes(leaderTxRawData, &leaderTx)
	if err != nil {
		mpcsyslog.Err("ValidateTx leader tx data decode fail. err:%s", err.Error())
		log.Error("ValidateTx leader tx data decode fail", "error", err)
		return false
	}

	log.Info("ValidateTx", "leaderTx", leaderTx)
	mpcsyslog.Info("ValidateTx, leaderTxData:%s", common.ToHex(leaderTx.Data()))
	isNotice, err := IsNoticeTransaction(leaderTx.Data())
	if err != nil {
		log.Error("ValidateTx, check notice transaction fail", "err", err)
		mpcsyslog.Err("ValidateTx, check notice transaction fail, err:", err.Error())
	} else if isNotice {
		log.Info("ValidateTx, is notice transaction, skip validating")
		mpcsyslog.Info("ValidateTx, is notice transaction, skip validating")
		return true
	}

	keysBytes := make([]byte, 0)
	keysBytes = append(keysBytes, leaderTx.Value().Bytes()...)
	keysBytes = append(keysBytes, leaderTx.Data()...)

	key := crypto.Keccak256(keysBytes)
	log.Info("ValidateTx", "key", common.ToHex(key))
	mpcsyslog.Info("mpc ValidateTx. key:%s", common.ToHex(key))

	followerDB, err := GetDB()
	if err != nil {
		log.Error("ValidateTx leader get database fail", "error", err)
		mpcsyslog.Err("ValidateTx leader get database fail. err:%s", err.Error())
		return false
	}

	_, err = waitKeyFromDB([][]byte{key})
	if err != nil {
		log.Error("ValidateTx, check has fail", "error", err)
		mpcsyslog.Err("ValidateTx, check has fail. err:%s", err.Error())
		return false
	}

	followerTxRawData, err := followerDB.Get(key)
	if err != nil {
		log.Error("ValidateTx, getting followerTxRawData fail", "error", err)
		mpcsyslog.Err("ValidateTx, getting followerTxRawData fail. err:%s", err.Error())
		return false
	}

	log.Info("ValidateTx, followerTxRawData is got")
	mpcsyslog.Info("ValidateTx, followerTxRawData is got")

	var followerRawTx mpcprotocol.SendTxArgs
	err = json.Unmarshal(followerTxRawData, &followerRawTx)
	if err != nil {
		log.Error("ValidateTx, follower tx data decode fail", "error", err)
		mpcsyslog.Err("ValidateTx, follower tx data decode fail. err:%s", err.Error())
		return false
	}

	followerCreatedTx := types.NewTransaction(leaderTx.Nonce(), *followerRawTx.To, followerRawTx.Value.ToInt(),
		leaderTx.Gas(), leaderTx.GasPrice(), followerRawTx.Data)
	followerCreatedHash := signer.Hash(followerCreatedTx)
	leaderTxLeaderHash := common.BytesToHash(leaderTxLeaderHashBytes)

	if followerCreatedHash == leaderTxLeaderHash {
		log.Info("ValidateTx, validate success")
		mpcsyslog.Info("ValidateTx, validate success")
		return true
	} else {
		log.Error("ValidateTx, leader tx hash is not same with follower tx hash", "leaderTxLeaderHash",
			leaderTxLeaderHash, "followerCreatedHash", followerCreatedHash)
		mpcsyslog.Err("ValidateTx, leader tx hash is not same with follower tx hash. leaderTxLeaderHash:%s, followerCreatedHash:%s",
			leaderTxLeaderHash.String(), followerCreatedHash.String())
		return false
	}
}


func ValidateBtcTx(args *btc.MsgTxArgs) bool {
	if args == nil {
		return false
	}

	log.Info("ValidateBtcTx, begin", "txInfo", args.String())
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
			log.Error("ValidateBtcTx, check has fail", "error", "invalid tx out pkscript")
			mpcsyslog.Err("ValidateBtcTx, check has fail. err:invalid tx out pkscript")
			return false
		}
	}

	keyWithoutTxin, keyWithTxin := GetKeyFromBtcTx(args)
	log.Info("ValidateBtcTx", "keyWithoutTxin", common.ToHex(keyWithoutTxin), "keyWithTxin", common.ToHex(keyWithTxin))
	mpcsyslog.Info("ValidateBtcTx, keyWithoutTxin:%s, keyWithTxin:%s", common.ToHex(keyWithoutTxin), common.ToHex(keyWithTxin))

	key, err := waitKeyFromDB([][]byte{keyWithTxin, keyWithoutTxin})
	if err != nil {
		log.Error("ValidateBtcTx, check has fail", "error", err)
		mpcsyslog.Err("ValidateBtcTx, check has fail. err:%s", err.Error())
		return false
	} else {
		log.Info("ValidateBtcTx, key is got", "key", common.ToHex(key))
		mpcsyslog.Info("ValidateBtcTx, key is got, key:" + common.ToHex(key))
		return true
	}
}

func waitKeyFromDB(keys [][]byte) ([]byte, error) {
	log.Info("waitKeyFromDB, begin")
	mpcsyslog.Info("waitKeyFromDB, begin")

	for i, key := range keys {
		log.Info("waitKeyFromDB", "i", i, "key", common.ToHex(key))
		mpcsyslog.Info("waitKeyFromDB, i:%d, key:%s", i, common.ToHex(key))
	}

	db, err := GetDB()
	if err != nil {
		log.Error("ValidateBtcTx get database fail", "error", err)
		mpcsyslog.Err("waitKeyFromDB get database fail. err:%s", err.Error())
		return nil, err
	}

	start := time.Now()
	for {
		for _, key := range keys {
			isExist, err := db.Has(key)
			if err != nil {
				log.Info("waitKeyFromDB, fail", "err", err)
				mpcsyslog.Err("waitKeyFromDB fail, err:%s", err.Error())
				return nil, err
			} else if isExist {
				log.Info("waitKeyFromDB, got it", "key", key)
				mpcsyslog.Info("waitKeyFromDB, got it, key:%s", common.ToHex(key))
				return key, nil
			}
		}

		if time.Now().Sub(start) >= mpcprotocol.MPCTimeOut {
			mpcsyslog.Info("waitKeyFromDB, time out")
			log.Info("waitKeyFromDB, time out")
			return nil, errors.New("waitKeyFromDB, time out")
		}

		time.Sleep(200 * time.Microsecond)
	}

	return nil, errors.New("waitKeyFromDB, unknown error")
}


func GetKeyFromBtcTx(args *btc.MsgTxArgs) (keyWithoutTxIn []byte, keyWithTxIn []byte) {
	log.Info("GetKeyFromBtcTx, begin")

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
	log.Info("GetKeyFromBtcTx", "keyWithTxin", common.ToHex(keyWithTxIn), "keyWithoutTxIn", common.ToHex(keyWithoutTxIn))
	for _, in := range args.TxIn {
		log.Info("GetKeyFromBtcTx, add txIn info to key", "txInPreOutHash", in.PreviousOutPoint.Hash, "index", in.PreviousOutPoint.Index)
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

func IsNoticeTransaction(payload []byte) (bool, error) {
	log.Info("IsNoticeTransaction, begin", "payload", common.ToHex(payload))
	mpcsyslog.Info("IsNoticeTransaction, payload:%s", common.ToHex(payload))
	if len(payload) < 4 {
		return false, errors.New("invalid payload length")
	}

	var callFuncId [4]byte
	copy(callFuncId[:], payload[:4])
	log.Debug("IsNoticeTransaction", "callFuncId", common.ToHex(callFuncId[:]))
	for _, noticeFuncId := range noticeFuncIds {
		if callFuncId == noticeFuncId {
			log.Info("IsNoticeTransaction, is notice")
			mpcsyslog.Info("IsNoticeTransaction, is notice")
			return true, nil
		}
	}

	log.Info("IsNoticeTransaction, is not notice")
	mpcsyslog.Info("IsNoticeTransaction, is not notice")
	return false, nil
}


func AddValidMpcTx(tx *mpcprotocol.SendTxArgs) error {
	log.Info("AddValidMpcTx begin", "txInfo", tx)
	mpcsyslog.Info("AddValidMpcTx, data:%s", common.ToHex([]byte(tx.Data)))

	var key, val []byte
	if tx.Value == nil {
		err := errors.New("tx.Value field is required")
		log.Error("AddValidMpcTx, invalid input", "error", err)
		mpcsyslog.Err("AddValidMpcTx, invalid input. err:%s", err.Error())
		return err
	}

	if tx.Data == nil {
		err := errors.New("tx.Data should not be empty")
		log.Error("AddValidMpcTx, invalid input", "error", err)
		mpcsyslog.Err("AddValidMpcTx, invalid input. err:%s", err.Error())
		return err
	}

	key = append(key, tx.Value.ToInt().Bytes()...)
	key = append(key, tx.Data...)
	key = crypto.Keccak256(key)

	val, err := json.Marshal(&tx)
	if err != nil {
		log.Error("AddValidMpcTx, marshal fail", "error", err)
		mpcsyslog.Err("AddValidMpcTx, marshal fail. err:%s", err.Error())
		return err
	}

	return addKeyValueToDB(key, val)
}

func AddValidMpcBtcTx(args *btc.MsgTxArgs) error {
	log.Info("AddValidMpcBtcTx, begin", "txInfo", args.String())
	mpcsyslog.Info("AddValidMpcBtcTx, begin, txInfo:%s", args.String())

	msgTx, err := btc.GetMsgTxFromMsgTxArgs(args)
	if err != nil {
		return err
	}

	for _, txIn := range msgTx.TxIn {
		log.Info("AddValidMpcBtcTx, msgTx", "TxIn", *txIn)
		mpcsyslog.Info("AddValidMpcBtcTx, txInPreOutHash:%s, txInIndex:%d", txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index)
	}
	for _, txOut := range msgTx.TxOut {
		log.Info("AddValidMpcBtcTx, msgTx", "TxOut", *txOut)
		mpcsyslog.Info("AddValidMpcBtcTx, txOutPkScript:%s", common.ToHex(txOut.PkScript))
	}

	_, key := GetKeyFromBtcTx(args)
	val, err := json.Marshal(args)
	if err != nil {
		log.Error("AddValidMpcBtcTx, marshal fail", "error", err)
		mpcsyslog.Err("AddValidMpcBtcTxRaw, marshal fail. err:%s", err.Error())
		return err
	}

	return addKeyValueToDB(key, val)
}

func addKeyValueToDB(key, value []byte) error {
	log.Info("addKeyValueToDB, begin", "key", common.ToHex(key))
	mpcsyslog.Info("addKeyValueToDB, begin, key:", common.ToHex(key))
	sdb, err := GetDB()
	if err != nil {
		log.Error("addKeyValueToDB, getting storeman database fail", "error", err)
		mpcsyslog.Err("addKeyValueToDB, getting storeman database fail. err:%s", err.Error())
		return err
	}

	err = sdb.Put(key, value)
	if err != nil {
		log.Error("addKeyValueToDB, getting storeman database fail", "error", err)
		mpcsyslog.Err("addKeyValueToDB, getting storeman database fail. err:%s", err.Error())
		return err
	}

	log.Info("addKeyValueToDB", "key", common.ToHex(key))
	mpcsyslog.Info("addKeyValueToDB. key:%s", common.ToHex(key))
	ret, err := sdb.Get(key)
	if err != nil {
		log.Error("addKeyValueToDB, getting storeman database fail", "error", err)
		mpcsyslog.Err("addKeyValueToDB, getting storeman database fail. err:%s", err.Error())
		return err
	}

	log.Info("addKeyValueToDB succeed to get data from leveldb after putting key-val pair", "ret", string(ret))
	mpcsyslog.Info("addKeyValueToDB succeed to get data from leveldb after putting key-val pair. ret:%s", string(ret))
	return nil
}
