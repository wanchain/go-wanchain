package posdb

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"
)

func TestUint64Convert(t *testing.T) {
	value := uint64(0)
	buf := convert.Uint64ToBytes(value)
	if len(buf) != 1 || buf[0] != 0 {
		t.Fail()
	}

	buf = convert.Uint64ToBytes(uint64(0x12345678))
	if buf[0] != 0x12 || buf[1] != 0x34 || buf[2] != 0x56 || buf[3] != 0x78 {
		t.Fail()
	}

	buf = convert.Uint64StringToByte("0")
	if len(buf) != 1 || buf[0] != 0 {
		t.Fail()
	}

	str := convert.Uint64ToString(value)
	if str != "00" && str != "0" {
		t.Fail()
	}

	if convert.StringToUint64("0") != uint64(0) {
		t.Fail()
	}

	if convert.StringToUint64("00") != uint64(0) {
		t.Fail()
	}

	if convert.BytesToUint64([]byte{0x00}) != uint64(0) {
		t.Fail()
	}

	str = convert.Uint64ToString(uint64(102400000))

	v1 := convert.StringToUint64("257")
	v2 := convert.BytesToUint64([]byte{0x01, 0x01})
	if v1+v2 != uint64(0x202) {
		t.Fail()
	}

	v3 := convert.StringToUint64("aaff")
	if v3 != 0 {
		t.Fail()
	}
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
		if !util.PkEqual(&key.PublicKey, &key.PublicKey) {
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
}
