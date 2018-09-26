package validator

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
)

func TestMpcDatabase(t *testing.T) {
	to1 := common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a")
	nonce := hexutil.Uint64(100)

	tx := mpcprotocol.SendTxArgs{
		From:      common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
		To:        &to1,
		Gas:       (*hexutil.Big)(big.NewInt(100)),
		GasPrice:  (*hexutil.Big)(big.NewInt(20)),
		Value:     (*hexutil.Big)(big.NewInt(50)),
		Data:      (hexutil.Bytes)([]byte("hello wanchain")),
		Nonce:     &nonce,
		ChainType: "first",
		ChainID:   (*hexutil.Big)(big.NewInt(1)),
		SignType:  "second",
	}

	var key, val []byte

	key = append(key, tx.Value.ToInt().Bytes()...)
	key = append(key, tx.Data...)
	key = crypto.Keccak256(key)

	val, err := json.Marshal(&tx)
	if err != nil {
		log.Error("error", "error", err)
	}

	dir := tmpKeyStore(t)
	defer os.RemoveAll(dir)

	err = NewDatabase(dir)
	if err != nil {
		log.Error("error", "error", err)
	}

	db, err := GetDB()
	if err != nil {
		log.Error("error", "error", err)
	}
	defer db.Close()

	err = db.Put(key, val)
	if err != nil {
		log.Error("error", "error", err)
	}

	ret, err := db.Get(key)
	if err != nil {
		log.Error("error", "error", err)
	}

	if i := bytes.Compare(val, ret); i != 0 {
		log.Error("database storeage ERROR", "error", errors.New("database storeage ERROR"))
	}

	var dec mpcprotocol.SendTxArgs
	err = json.Unmarshal(ret, &dec)
	if err != nil {
		log.Error("error", "error", err)
	}

}

func tmpKeyStore(t *testing.T) string {
	d, err := ioutil.TempDir("", "wanchain-storeman-test")
	if err != nil {
		t.Fatal(err)
	}

	return d
}
