package validator

import (
	"encoding/json"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rlp"
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	"os"
	"testing"
	"strings"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/storeman/btc"
	"fmt"
)

var (
	to1   = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	nonce = hexutil.Uint64(100)
	tx    = mpcprotocol.SendTxArgs{
		From:      common.HexToAddress("0xbb9003ca8226f411811dd16a3f1a2c1b3f71825d"),
		To:        &to1,
		Gas:       (*hexutil.Big)(big.NewInt(100)),
		GasPrice:  (*hexutil.Big)(big.NewInt(20)),
		Value:     (*hexutil.Big)(big.NewInt(50)),
		Data:      (hexutil.Bytes)([]byte("hello wanchain")),
		Nonce:     &nonce,
		ChainType: "WAN", //ETH
		ChainID:   (*hexutil.Big)(big.NewInt(1)),
		SignType:  "hash", //nil
	}

	btcTx0 = btc.MsgTxArgs {
		Version : 1,
		TxIn : []btc.TxInArgs{
			{btc.OutPointArg{"0000000000000000000000000000000000000000000000000000000000001111", 0}, "0x0012345678", 0, "0x76a914fd1c8e80f5dea6295ea3f82d8b103e3cf7d04b9288ac"},
		},
		TxOut: []btc.TxOutArgs{
			{Value:1110000, PkScript: "0x00a91491c6e41ae47789e7a98cd5625f27f0473e5b0d1d88ac"},
			{Value:1110001, PkScript: "0x00a914000000000000000000000000000000000000001188ac"},
		},
		LockTime: 0,
		From : common.HexToAddress("0x0000000000000000000000000000000000000011"),
	}
	btcTx1 = btc.MsgTxArgs {
		Version : 1,
		TxIn : []btc.TxInArgs{
				{btc.OutPointArg{"0000000000000000000000000000000000000000000000000000000000000001", 0}, "0x0012345678", 0, "0x76a914fd1c8e80f5dea6295ea3f82d8b103e3cf7d04b9288ac"},
			},
		TxOut: []btc.TxOutArgs{
				{Value:1000000, PkScript: "0x76a91491c6e41ae47789e7a98cd5625f27f0473e5b0d1d88ac"},
				{Value:1000001, PkScript: "0x76a914000000000000000000000000000000000000001188ac"},
			},
		LockTime: 0,
		From : common.HexToAddress("0x0000000000000000000000000000000000000011"),
	}
	btcTx2 = btc.MsgTxArgs {
		Version : 1,
		TxIn : []btc.TxInArgs{
			{btc.OutPointArg{"0000000000000000000000000000000000000000000000000000000000000001", 0}, "0x0012345678", 0, "0x76a914fd1c8e80f5dea6295ea3f82d8b103e3cf7d04b9288ac"},
		},
		TxOut: []btc.TxOutArgs{
			{Value:1000000, PkScript: "0x76a91491c6e41ae47789e7a98cd5625f27f0473e5b0d1d88ac"},
			{Value:1000022, PkScript: "0x76a914000000000000000000000000000000000000001188ac"},
		},
		LockTime: 0,
		From : common.HexToAddress("0x0000000000000000000000000000000000000011"),
	}
	btcTx3 = btc.MsgTxArgs {
		Version : 1,
		TxIn : []btc.TxInArgs{
			{btc.OutPointArg{"0000000000000000000000000000000000000000000000000000000000000001", 0}, "0x0012345678", 0, "0x76a914fd1c8e80f5dea6295ea3f82d8b103e3cf7d04b9288ac"},
		},
		TxOut: []btc.TxOutArgs{
			{Value:1000000, PkScript: "0x76a91491c6e41ae47789e7a98cd5625f27f0473e5b0d1d88ac"},
			{Value:1000022, PkScript: "0x76a914000000000000000000000000000000000000001188ac"},
		},
		LockTime: 0,
		From : common.HexToAddress("0x0000000000000000000000000000000000002211"),
	}
	btcTx4 = btc.MsgTxArgs {
		Version : 1,
		TxIn : []btc.TxInArgs{
			{btc.OutPointArg{"0000000000000000000000000000000000000000000000000000000000000001", 0}, "0x0012345678", 0, "0x76a914fd1c8e80f5dea6295ea3f82d8b103e3cf7d04b9288ac"},
		},
		TxOut: []btc.TxOutArgs{
			{Value:1000000, PkScript: "0x76a91491c6e41ae47789e7a98cd5625f27f0473e5b0d1d88ac"},
		},
		LockTime: 0,
		From : common.HexToAddress("0x0000000000000000000000000000000000000011"),
	}

	noticeSCDefine = `[{"constant":false,"inputs":[{"name":"stmWanAddr","type":"address"},{"name":"userBtcAddr","type":"address"},{"name":"xHash","type":"bytes32"},{"name":"txHash","type":"bytes32"},{"name":"lockedTimestamp","type":"uint256"}],"name":"btc2wbtcLockNotice","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"stmBtcAddr","type":"address"},{"name":"userWanAddr","type":"address"},{"name":"userBtcAddr","type":"address"},{"name":"xHash","type":"bytes32"},{"name":"txHash","type":"bytes32"},{"name":"lockedTimestamp","type":"uint256"}],"name":"wbtc2btcLockNotice","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
)



func CreateMPCTxSigner(ChainType string, ChainID *big.Int) (mpccrypto.MPCTxSigner, error) {
	if ChainType == "WAN" {
		return mpccrypto.CreateWanMPCTxSigner(ChainID), nil
	} else if ChainType == "ETH" {
		return mpccrypto.CreateEthMPCTxSigner(ChainID), nil
	}

	return nil, mpcprotocol.ErrChainTypeError
}

func TestValidateTxWan(t *testing.T) {
	//create emu tx in database
	dir := tmpKeyStore(t)
	defer os.RemoveAll(dir)
	var key, val []byte

	key = append(key, tx.Value.ToInt().Bytes()...)
	key = append(key, tx.Data...)
	key = crypto.Keccak256(key)

	val, err := json.Marshal(&tx)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	err = NewDatabase(dir)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	db, err := GetDB()
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}
	defer db.Close()

	err = db.Put(key, val)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	//create em tx from leader
	trans := types.NewTransaction(uint64(*tx.Nonce), *tx.To, (*big.Int)(tx.Value), (*big.Int)(tx.Gas), (*big.Int)(tx.GasPrice), tx.Data)
	signer, err := CreateMPCTxSigner(tx.ChainType, tx.ChainID.ToInt())
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	txHash := signer.Hash(trans).Bytes()
	txbytes, err := rlp.EncodeToBytes(trans)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	checkres := ValidateTx(signer, txbytes, txHash)

	if !checkres {
		log.Error("error", "error", err)
		t.Error("error", "error", err)
	}
}

func TestValidateTxEth(t *testing.T) {
	//create emu tx in database
	dir := tmpKeyStore(t)
	defer os.RemoveAll(dir)
	var key, val []byte

	key = append(key, tx.Value.ToInt().Bytes()...)
	key = append(key, tx.Data...)
	key = crypto.Keccak256(key)

	val, err := json.Marshal(&tx)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	err = NewDatabase(dir)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	db, err := GetDB()
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}
	defer db.Close()

	err = db.Put(key, val)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	tx.SignType = "ETH"
	//create em tx from leader
	trans := types.NewTransaction(uint64(*tx.Nonce), *tx.To, (*big.Int)(tx.Value), (*big.Int)(tx.Gas), (*big.Int)(tx.GasPrice), tx.Data)
	signer, err := CreateMPCTxSigner(tx.ChainType, tx.ChainID.ToInt())
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	txHash := signer.Hash(trans).Bytes()
	txbytes, err := rlp.EncodeToBytes(trans)
	if err != nil {
		log.Error("error", "error", err)
		t.Error(err)
	}

	checkres := ValidateTx(signer, txbytes, txHash)

	if !checkres {
		log.Error("error", "error", err)
		t.Error("error", "error", err)
	}

}

func TestIsNoticeTransaction(t *testing.T) {
	noticeABI, _ := abi.JSON(strings.NewReader(noticeSCDefine))
	data1, _ := noticeABI.Pack(
		"btc2wbtcLockNotice",
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
		common.HexToHash("0x03"),
		common.HexToHash("0x04"),
		big.NewInt(1))

	data2, _ := noticeABI.Pack(
		"wbtc2btcLockNotice",
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
		common.HexToAddress("0x03"),
		common.HexToHash("0x04"),
		common.HexToHash("0x05"),
		big.NewInt(2))

	data3, _ := noticeABI.Pack(
		"btc2wbtcLockNotice2",
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
		common.HexToHash("0x03"),
		common.HexToHash("0x04"),
		big.NewInt(3))

	data4, _ := noticeABI.Pack(
		"wbtc2btcLockNotice2",
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
		common.HexToAddress("0x03"),
		common.HexToHash("0x04"),
		common.HexToHash("0x05"),
		big.NewInt(4))

	fmt.Println("data1:", common.ToHex(data1[:4]))
	fmt.Println("data2:", common.ToHex(data2[:4]))

	if is, err := IsNoticeTransaction(data1); err != nil || !is {
		t.Error("check btc2wbtcLockNotice transaction fail")
	}
	if is, err := IsNoticeTransaction(data2); err != nil || !is  {
		t.Error("check wbtc2btcLockNotice transaction fail")
	}
	if is, err := IsNoticeTransaction(data3); err == nil && is  {
		t.Error("check btc2wbtcLockNotice2 transaction fail")
	}
	if is, err := IsNoticeTransaction(data4); err == nil && is  {
		t.Error("check wbtc2btcLockNotice2 transaction fail")
	}
}

func TestAddValidMpcBtcTx(t *testing.T) {
	dir := tmpKeyStore(t)
	defer os.RemoveAll(dir)
	err := NewDatabase(dir)
	if err != nil {
		t.Fatal("create database fail, err:", err)
	}

	err = AddValidMpcBtcTx(&btcTx0)
	if err != nil {
		t.Fatal("AddValidMpcBtcTx fail. err:", err)
	}

	_, key := GetKeyFromBtcTx(&btcTx0)
	sdb, err := GetDB()
	if err != nil {
		t.Fatal("GetDB fail. err", err)
	}

	ret, err := sdb.Get(key)
	if err != nil {
		t.Fatal("get value from db by key fail. err", err)
	}

	var tx btc.MsgTxArgs
	err = json.Unmarshal(ret, &tx)
	if err != nil {
		t.Fatal("json unmarshal fail. err", err)
	}

	if !tx.Cmp(&btcTx0) {
		t.Fatal("getting tx data doesn't equal to original data")
	}
}

func TestValidateBtcTx(t *testing.T)  {
	dir := tmpKeyStore(t)
	defer os.RemoveAll(dir)
	err := NewDatabase(dir)
	if err != nil {
		t.Fatal("create database fail, err:", err)
	}

	bValid := ValidateBtcTx(&btcTx1)
	if bValid {
		t.Fatal("validateBtcTx fail, expect return false")
	}

	err = AddValidMpcBtcTx(&btcTx1)
	if err != nil {
		t.Fatal("AddValidMpcBtcTx fail. err:", err)
	}

	bValid = ValidateBtcTx(&btcTx1)
	if !bValid {
		t.Fatal("validateBtcTx fail, expect return true")
	}

	bValid = ValidateBtcTx(&btcTx2)
	if !bValid {
		t.Fatal("validateBtcTx fail, expect return true")
	}

	bValid = ValidateBtcTx(&btcTx3)
	if bValid {
		t.Fatal("validateBtcTx fail, expect return false")
	}

	bValid = ValidateBtcTx(&btcTx4)
	if !bValid {
		t.Fatal("validateBtcTx fail, expect return true")
	}
}