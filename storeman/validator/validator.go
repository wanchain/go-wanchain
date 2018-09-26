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
)

var noticeFuncIds [][4]byte

func init() {
	noticeFuncDefs := []string{
		"btc2wbtcLockNotice(address,address,bytes32,bytes32,uint256)",
		"wbtc2btcLockNotice(address,address,bytes32,bytes32,uint256)",}

	var funcId [4]byte
	for _, funcDef := range noticeFuncDefs {
		copy(funcId[:], crypto.Keccak256([]byte(funcDef))[:4])
		noticeFuncIds = append(noticeFuncIds, funcId)
		log.Debug("validator.init, add notice func id", "id", common.ToHex(funcId[:]))
	}
}

func ValidateTx(signer mpccrypto.MPCTxSigner, leaderTxRawData []byte, leaderTxLeaderHashBytes []byte) bool {
	var leaderTx types.Transaction
	err := rlp.DecodeBytes(leaderTxRawData, &leaderTx)
	if err != nil {
		mpcsyslog.Err("ValidateTx leader tx data decode fail. err:%s", err.Error())
		log.Error("ValidateTx leader tx data decode fail", "error", err)
		return false
	}

	isNotice, err := IsNoticeTransaction(leaderTx.Data())
	if err != nil {
		log.Error("ValidateTx, check notice transaction fail", "err", err)
	} else if isNotice {
		log.Info("ValidateTx, is notice transaction, skip validating")
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
		mpcsyslog.Err("ValidateTx leader get database fail. err:%s", err.Error())
		log.Error("ValidateTx leader get database fail", "error", err)
		return false
	}

	_, err = waitKeyFromDB([][]byte{key})
	if err != nil {
		mpcsyslog.Err("ValidateTx, check has fail. err:%s", err.Error())
		log.Error("ValidateTx, check has fail", "error", err)
		return false
	}

	followerTxRawData, err := followerDB.Get(key)
	if err != nil {
		mpcsyslog.Err("ValidateTx, getting followerTxRawData fail. err:%s", err.Error())
		log.Error("ValidateTx, getting followerTxRawData fail", "error", err)
		return false
	}

	mpcsyslog.Info("ValidateTx, followerTxRawData is got")
	log.Info("ValidateTx, followerTxRawData is got")

	var followerRawTx mpcprotocol.SendTxArgs
	err = json.Unmarshal(followerTxRawData, &followerRawTx)
	if err != nil {
		mpcsyslog.Err("ValidateTx, follower tx data decode fail. err:%s", err.Error())
		log.Error("ValidateTx, follower tx data decode fail", "error", err)
		return false
	}

	followerCreatedTx := types.NewTransaction(leaderTx.Nonce(), *followerRawTx.To, followerRawTx.Value.ToInt(),
		leaderTx.Gas(), leaderTx.GasPrice(), followerRawTx.Data)
	followerCreatedHash := signer.Hash(followerCreatedTx)
	leaderTxLeaderHash := common.BytesToHash(leaderTxLeaderHashBytes)

	if followerCreatedHash == leaderTxLeaderHash {
		mpcsyslog.Info("ValidateTx, validate success")
		log.Info("ValidateTx, validate success")
		return true
	} else {
		mpcsyslog.Err("ValidateTx, leader tx hash is not same with follower tx hash. leaderTxLeaderHash:%s, followerCreatedHash:%s",
			leaderTxLeaderHash.String(), followerCreatedHash.String())
		log.Error("ValidateTx, leader tx hash is not same with follower tx hash", "leaderTxLeaderHash",
			leaderTxLeaderHash, "followerCreatedHash", followerCreatedHash)
		return false
	}
}


func ValidateBtcTx(args *btc.MsgTxArgs) bool {
	if args == nil {
		return false
	}

	keyWithoutTxin, keyWithTxin := GetKeyFromBtcTx(args)
	log.Info("-----------------GetKeyFromBtcTx", "keyWithoutTxin", common.ToHex(keyWithoutTxin))
	log.Info("-----------------GetKeyFromBtcTx", "keyWithTxin", common.ToHex(keyWithTxin))

	key, err := waitKeyFromDB([][]byte{keyWithTxin, keyWithoutTxin})
	if err != nil {
		mpcsyslog.Err("ValidateBtcTx, check has fail. err:%s", err.Error())
		log.Error("ValidateBtcTx, check has fail", "error", err)
		return false
	} else {
		mpcsyslog.Info("ValidateBtcTx, key is got, key:" + common.ToHex(key))
		log.Info("ValidateBtcTx, key is got", "key", common.ToHex(key))
		return true
	}
}

func waitKeyFromDB(keys [][]byte) ([]byte, error) {
	db, err := GetDB()
	if err != nil {
		mpcsyslog.Err("waitKeyFromDB get database fail. err:%s", err.Error())
		log.Error("ValidateBtcTx get database fail", "error", err)
		return nil, err
	}

	start := time.Now()
	for {
		for _, key := range keys {
			isExist, err := db.Has(key)
			if err != nil {
				return nil, err
			} else if isExist {
				return key, nil
			}
		}

		if time.Now().Sub(start) >= mpcprotocol.MPCTimeOut {
			mpcsyslog.Info("ValidateBtcTx time out")
			log.Info("ValidateBtcTx time out")
			return nil, errors.New("time out")
		}

		time.Sleep(200 * time.Microsecond)
	}

	return nil, errors.New("unknown")
}


func GetKeyFromBtcTx(args *btc.MsgTxArgs) (keyWithoutTxIn []byte, keyWithTxIn []byte) {
	keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(args.Version)).Bytes()...)
	keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(args.LockTime)).Bytes()...)

	for _, out := range args.TxOut {
		keyWithoutTxIn = append(keyWithoutTxIn, big.NewInt(int64(out.Value)).Bytes()...)
		keyWithoutTxIn = append(keyWithoutTxIn, []byte(out.PkScript)...)
	}

	keyWithTxIn = make([]byte, len(keyWithoutTxIn))
	copy(keyWithTxIn, keyWithoutTxIn)
	log.Info("-----------------GetKeyFromBtcTx", "keyWithTxin", common.ToHex(keyWithTxIn))
	log.Info("-----------------GetKeyFromBtcTx", "keyWithoutTxIn", common.ToHex(keyWithoutTxIn))
	for _, in := range args.TxIn {
		log.Warn("-----------------GetKeyFromBtcTx, add txIn info to key")
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
	if len(payload) < 4 {
		return false, errors.New("invalid payload length")
	}

	var callFuncId [4]byte
	copy(callFuncId[:], payload[:4])
	log.Debug("IsNoticeTransaction", "callFuncId", common.ToHex(callFuncId[:]))
	for _, noticeFuncId := range noticeFuncIds {
		if callFuncId == noticeFuncId {
			return true, nil
		}
	}

	return false, nil
}


func AddValidMpcTx(tx *mpcprotocol.SendTxArgs) error {
	log.Warn("-----------------AddValidMpcTx begin", "tx", tx)

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
	log.Warn("-----------------AddValidMpcBtcTx begin", "args", args)
	msgTx, err := btc.GetMsgTxFromMsgTxArgs(args)
	if err != nil {
		return err
	}

	log.Warn("-----------------AddValidMpcBtcTx", "msgTx", msgTx)
	for _, txIn := range msgTx.TxIn {
		log.Warn("-----------------AddValidMpcBtcTx, msgTx", "TxIn", *txIn)
	}
	for _, txOut := range msgTx.TxOut {
		log.Warn("-----------------AddValidMpcBtcTx, msgTx", "TxOut", *txOut)
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