package posdb

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	//"github.com/ethereum/go-ethereum/pos/util"
)

//PkEqual only can use in same curve. return whether the two points equal
func PkEqual(pk1, pk2 *ecdsa.PublicKey) bool {
	if pk1 == nil || pk2 == nil {
		return false
	}

	if hex.EncodeToString(pk1.X.Bytes()) == hex.EncodeToString(pk2.X.Bytes()) &&
		hex.EncodeToString(pk1.Y.Bytes()) == hex.EncodeToString(pk2.Y.Bytes()) {
		return true
	}
	return false
}

func TestDbInitAll(t *testing.T) {
	os.RemoveAll("/tmp/gwan")
	DbInitAll("/tmp")
	db := NewDb(posconfig.PosLocalDB)
	if db == nil {
		t.Fail()
	}

	db = NewDb(posconfig.RbLocalDB)
	if db == nil {
		t.Fail()
	}

	db = NewDb(posconfig.EpLocalDB)
	if db == nil {
		t.Fail()
	}

	testCount := 1000
	keys := make([][]byte, testCount)
	for i := 0; i < 1000; i++ {
		key, _ := crypto.GenerateKey()
		keys[i] = crypto.FromECDSAPub(&key.PublicKey)
		if !PkEqual(&key.PublicKey, &key.PublicKey) {
			t.Fail()
		}
	}

	allQuit := make(chan struct{}, 3)

	go func() {
		for i := 0; i < testCount; i++ {
			db := NewDb(posconfig.PosLocalDB)
			db.PutWithIndex(0, uint64(i), "", keys[i])
		}

		for i := 0; i < testCount; i++ {
			value, err := db.GetWithIndex(0, uint64(i), "")
			if hex.EncodeToString(value) != hex.EncodeToString(keys[i]) || err != nil {
				t.Fail()
			}
		}

		bufs := db.GetStorageByteArray(0)
		for i := 0; i < testCount; i++ {
			if hex.EncodeToString(bufs[i]) != hex.EncodeToString(keys[i]) {
				t.Fail()
			}
		}

		allQuit <- struct{}{}
	}()

	go func() {
		for i := 0; i < testCount; i++ {
			db := NewDb(posconfig.RbLocalDB)
			db.PutWithIndex(0, uint64(i), "", keys[i])
		}

		for i := 0; i < testCount; i++ {
			value, err := db.GetWithIndex(0, uint64(i), "")
			if hex.EncodeToString(value) != hex.EncodeToString(keys[i]) || err != nil {
				t.Fail()
			}
		}

		bufs := db.GetStorageByteArray(0)
		for i := 0; i < testCount; i++ {
			if hex.EncodeToString(bufs[i]) != hex.EncodeToString(keys[i]) {
				t.Fail()
			}
		}

		allQuit <- struct{}{}
	}()

	go func() {
		for i := 0; i < testCount; i++ {
			db := NewDb(posconfig.EpLocalDB)
			db.PutWithIndex(0, uint64(i), "", keys[i])
		}

		for i := 0; i < testCount; i++ {
			value, err := db.GetWithIndex(0, uint64(i), "")
			if hex.EncodeToString(value) != hex.EncodeToString(keys[i]) || err != nil {
				t.Fail()
			}
		}

		bufs := db.GetStorageByteArray(0)
		for i := 0; i < testCount; i++ {
			if hex.EncodeToString(bufs[i]) != hex.EncodeToString(keys[i]) {
				t.Fail()
			}
		}

		allQuit <- struct{}{}
	}()

	select {
	case <-allQuit:
	}
	select {
	case <-allQuit:
	}
	select {
	case <-allQuit:
	}

	db = NewDb("test")
	db.Put(0, "hello", []byte{1, 2, 3})
	buf, err := db.Get(0, "hello")
	if buf[0] != 1 || buf[1] != 2 || buf[2] != 3 || err != nil {
		t.Fail()
	}

	db = GetDb()
	db.Put(0, "hello", []byte{3, 4, 5})
	buf, err = db.Get(0, "hello")
	if buf[0] != 3 || buf[1] != 4 || buf[2] != 5 || err != nil {
		t.Fail()
	}

	db.DbClose()
}

func TestInfomationGet(t *testing.T) {
	buf := GetRBProposerGroup(0)
	fmt.Println(buf)
	buf2 := GetStakerInfoBytes(0, common.Address{})
	fmt.Println(buf2)
	buf4 := GetEpochLeaderGroup(0)
	fmt.Println(buf4)
}
