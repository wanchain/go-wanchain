package validator

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/btcsuite/btcd/txscript"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/storeman/btc"
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	"time"
)

var noticeFuncIds [][4]byte

func init() {
	noticeFuncDefs := []string{
		"btc2wbtcLockNotice(address,address,bytes32,bytes32,uint256)",
		"wbtc2btcLockNotice(address,address,address,bytes32,bytes32,uint256)"}

	var funcId [4]byte
	for _, funcDef := range noticeFuncDefs {
		copy(funcId[:], crypto.Keccak256([]byte(funcDef))[:4])
		noticeFuncIds = append(noticeFuncIds, funcId)
		log.SyslogInfo("validator.init, add notice func id", "id", common.ToHex(funcId[:]))
	}
}

func ValidateTx(signer mpccrypto.MPCTxSigner, from common.Address, chainType string, chainId *big.Int, leaderTxRawData []byte, leaderTxLeaderHashBytes []byte) bool {
	log.SyslogInfo("ValidateTx",
		"from", from.String(),
		"chainType", chainType,
		"chainId", chainId.String(),
		"leaderTxLeaderHashBytes", common.ToHex(leaderTxLeaderHashBytes),
		"leaderTxRawData", common.ToHex(leaderTxRawData))

	var leaderTx types.Transaction
	err := rlp.DecodeBytes(leaderTxRawData, &leaderTx)
	if err != nil {
		log.SyslogErr("ValidateTx leader tx data decode fail", "err", err.Error())
		return false
	}

	log.SyslogInfo("ValidateTx", "leaderTxData", common.ToHex(leaderTx.Data()))
	isNotice, err := IsNoticeTransaction(leaderTx.Data())
	if err != nil {
		log.SyslogErr("ValidateTx, check notice transaction fail", "err", err.Error())
	} else if isNotice {
		log.SyslogInfo("ValidateTx, is notice transaction, skip validating")
		return true
	}

	key := GetKeyFromTx(&from, leaderTx.To(), leaderTx.Value(), leaderTx.Data(), &chainType, chainId)
	log.SyslogInfo("mpc ValidateTx", "key", common.ToHex(key))

	followerDB, err := GetDB()
	if err != nil {
		log.SyslogErr("ValidateTx leader get database fail", "err", err.Error())
		return false
	}

	_, err = waitKeyFromDB([][]byte{key})
	if err != nil {
		log.SyslogErr("ValidateTx, check has fail", "err", err.Error())
		return false
	}

	followerTxRawData, err := followerDB.Get(key)
	if err != nil {
		log.SyslogErr("ValidateTx, getting followerTxRawData fail", "err", err.Error())
		return false
	}

	log.SyslogInfo("ValidateTx, followerTxRawData is got")

	var followerRawTx mpcprotocol.SendTxArgs
	err = json.Unmarshal(followerTxRawData, &followerRawTx)
	if err != nil {
		log.SyslogErr("ValidateTx, follower tx data decode fail", "err", err.Error())
		return false
	}

	followerCreatedTx := types.NewTransaction(leaderTx.Nonce(), *followerRawTx.To, followerRawTx.Value.ToInt(),
		leaderTx.Gas(), leaderTx.GasPrice(), followerRawTx.Data)
	followerCreatedHash := signer.Hash(followerCreatedTx)
	leaderTxLeaderHash := common.BytesToHash(leaderTxLeaderHashBytes)

	if followerCreatedHash == leaderTxLeaderHash {
		log.SyslogInfo("ValidateTx, validate success")
		return true
	} else {
		log.SyslogErr("ValidateTx, leader tx hash is not same with follower tx hash",
			"leaderTxLeaderHash", leaderTxLeaderHash.String(),
			"followerCreatedHash", followerCreatedHash.String())
		return false
	}
}

func ValidateBtcTx(args *btc.MsgTxArgs) bool {
	if args == nil {
		return false
	}

	log.SyslogInfo("ValidateBtcTx, begin", "txInfo", args.String())

	var txOutScript [25]byte
	txOutScript[0] = txscript.OP_DUP
	txOutScript[1] = txscript.OP_HASH160
	txOutScript[2] = 0x14
	copy(txOutScript[3:23], args.From[:])
	txOutScript[23] = txscript.OP_EQUALVERIFY
	txOutScript[24] = txscript.OP_CHECKSIG

	for i := 1; i < len(args.TxOut); i++ {
		log.SyslogInfo("ValidateBtcTx", "outScript", args.TxOut[i].PkScript)

		if !bytes.Equal(txOutScript[:], common.FromHex(args.TxOut[i].PkScript)) {
			log.SyslogErr("ValidateBtcTx, check has fail", "err", "invalid tx out pkscript")
			return false
		}
	}

	keyWithoutTxin, keyWithTxin := GetKeyFromBtcTx(args)
	log.SyslogInfo("ValidateBtcTx", "keyWithoutTxin", common.ToHex(keyWithoutTxin), "keyWithTxin", common.ToHex(keyWithTxin))

	key, err := waitKeyFromDB([][]byte{keyWithTxin, keyWithoutTxin})
	if err != nil {
		log.SyslogErr("ValidateBtcTx, check has fail", "err", err.Error())
		return false
	} else {
		log.SyslogInfo("ValidateBtcTx, key is got", "key", common.ToHex(key))
		return true
	}
}

func waitKeyFromDB(keys [][]byte) ([]byte, error) {
	log.SyslogInfo("waitKeyFromDB, begin")

	for i, key := range keys {
		log.SyslogInfo("waitKeyFromDB", "i", i, "key", common.ToHex(key))
	}

	db, err := GetDB()
	if err != nil {
		log.SyslogErr("waitKeyFromDB get database fail", "err", err.Error())
		return nil, err
	}

	start := time.Now()
	for {
		for _, key := range keys {
			isExist, err := db.Has(key)
			if err != nil {
				log.SyslogErr("waitKeyFromDB fail", "err", err.Error())
				return nil, err
			} else if isExist {
				log.SyslogInfo("waitKeyFromDB, got it", "key", common.ToHex(key))
				return key, nil
			}
		}

		if time.Now().Sub(start) >= mpcprotocol.MPCTimeOut {
			log.SyslogInfo("waitKeyFromDB, time out")
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
		log.SyslogInfo("GetKeyFromBtcTx", "out.PkScript", out.PkScript)
		break
	}

	keyWithTxIn = make([]byte, len(keyWithoutTxIn))
	copy(keyWithTxIn, keyWithoutTxIn)
	log.SyslogInfo("GetKeyFromBtcTx", "keyWithTxin", common.ToHex(keyWithTxIn), "keyWithoutTxIn", common.ToHex(keyWithoutTxIn))
	for _, in := range args.TxIn {
		log.SyslogInfo("GetKeyFromBtcTx", "txInPreOutHash", in.PreviousOutPoint.Hash, "txInIndex", in.PreviousOutPoint.Index)
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
	log.SyslogInfo("IsNoticeTransaction", "payload", common.ToHex(payload))
	if len(payload) < 4 {
		return false, errors.New("invalid payload length")
	}

	var callFuncId [4]byte
	copy(callFuncId[:], payload[:4])
	log.SyslogInfo("IsNoticeTransaction", "callFuncId", common.ToHex(callFuncId[:]))
	for _, noticeFuncId := range noticeFuncIds {
		if callFuncId == noticeFuncId {
			log.SyslogInfo("IsNoticeTransaction, is notice")
			return true, nil
		}
	}

	log.SyslogInfo("IsNoticeTransaction, is not notice")
	return false, nil
}

func AddValidMpcTx(tx *mpcprotocol.SendTxArgs) error {
	log.SyslogInfo("AddValidMpcTx", "txInfo", tx.String())

	var key, val []byte
	if tx.Value == nil {
		err := errors.New("tx.Value field is required")
		log.SyslogErr("AddValidMpcTx, invalid input", "err", err.Error())
		return err
	}

	if tx.Data == nil {
		err := errors.New("tx.Data should not be empty")
		log.SyslogErr("AddValidMpcTx, invalid input", "err", err.Error())
		return err
	}

	key = GetKeyFromTx(&tx.From, tx.To, (*big.Int)(tx.Value), tx.Data, &tx.ChainType, (*big.Int)(tx.ChainID))

	val, err := json.Marshal(&tx)
	if err != nil {
		log.SyslogErr("AddValidMpcTx, marshal fail", "err", err.Error())
		return err
	}

	return addKeyValueToDB(key, val)
}

func AddValidData(data *mpcprotocol.SendData) error {
	log.SyslogInfo("AddValidData", "txInfo", data.String())
	if len(data.Data) != common.HashLength {
		log.SyslogErr(mpcprotocol.ErrInvalidSignedData.Error())
		return mpcprotocol.ErrInvalidSignedData
	}
	val, err := json.Marshal(&data)
	if err != nil {
		log.SyslogErr("AddValidData, marshal fail", "err", err.Error())
		return err
	}

	return addKeyValueToDB(data.Data[:], val)
}

func AddValidMpcBtcTx(args *btc.MsgTxArgs) error {
	log.SyslogInfo("AddValidMpcBtcTx, begin", "txInfo", args.String())

	msgTx, err := btc.GetMsgTxFromMsgTxArgs(args)
	if err != nil {
		return err
	}

	for _, txIn := range msgTx.TxIn {
		log.SyslogInfo("AddValidMpcBtcTx", "txInPreOutHash", txIn.PreviousOutPoint.Hash.String(), "txInIndex", txIn.PreviousOutPoint.Index)
	}
	for _, txOut := range msgTx.TxOut {
		log.SyslogInfo("AddValidMpcBtcTx", "txOutPkScript", common.ToHex(txOut.PkScript))
	}

	_, key := GetKeyFromBtcTx(args)
	val, err := json.Marshal(args)
	if err != nil {
		log.SyslogErr("AddValidMpcBtcTxRaw, marshal fail", "err", err.Error())
		return err
	}

	return addKeyValueToDB(key, val)
}

func addKeyValueToDB(key, value []byte) error {
	log.SyslogInfo("addKeyValueToDB, begin", "key:", common.ToHex(key))
	sdb, err := GetDB()
	if err != nil {
		log.SyslogErr("addKeyValueToDB, getting storeman database fail", "err", err.Error())
		return err
	}

	err = sdb.Put(key, value)
	if err != nil {
		log.SyslogErr("addKeyValueToDB, getting storeman database fail", "err", err.Error())
		return err
	}

	log.SyslogInfo("addKeyValueToDB", "key", common.ToHex(key))
	ret, err := sdb.Get(key)
	if err != nil {
		log.SyslogErr("addKeyValueToDB, getting storeman database fail", "err", err.Error())
		return err
	}

	log.SyslogInfo("addKeyValueToDB succeed to get data from leveldb after putting key-val pair", "ret", string(ret))
	return nil
}
