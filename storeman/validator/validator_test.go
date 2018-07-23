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
)

var (
	to1   = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	nonce = hexutil.Uint64(100)
	tx    = SendTxArgs{
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
