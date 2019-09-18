package validator

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
)

func TestMpcDatabase(t *testing.T) {

	data := mpcprotocol.SendData{
		PKBytes: []byte("pkbytes"),
		Data:    []byte("wanchain"),
	}

	var key, val []byte
	key = crypto.Keccak256(data.Data[:])
	val, err := json.Marshal(&data)
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

	var dec mpcprotocol.SendData
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
