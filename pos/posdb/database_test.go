package posdb

import (
	"encoding/hex"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
)

func TestUint64Convert(t *testing.T) {
	value := uint64(0)
	buf := Uint64ToBytes(value)
	if len(buf) != 1 || buf[0] != 0 {
		t.Fail()
	}

	buf = Uint64StringToByte("0")
	if len(buf) != 1 || buf[0] != 0 {
		t.Fail()
	}

	str := Uint64ToString(value)
	if str != "00" && str != "0" {
		t.Fail()
	}

	if StringToUint64("0") != uint64(0) {
		t.Fail()
	}

	if StringToUint64("00") != uint64(0) {
		t.Fail()
	}

	if BytesToUint64([]byte{0x00}) != uint64(0) {
		t.Fail()
	}

	str = Uint64ToString(uint64(102400000))

	v1 := StringToUint64("257")
	v2 := BytesToUint64([]byte{0x01, 0x01})
	if v1+v2 != uint64(0x202) {
		t.Fail()
	}

	v3 := StringToUint64("aaff")
	if v3 != 0 {
		t.Fail()
	}
}

func TestDbInitAll(t *testing.T) {
	DbInitAll("/tmp")
	db := NewDb("pos")
	if db == nil {
		t.Fail()
	}

	db = NewDb("rblocaldb")
	if db == nil {
		t.Fail()
	}

	db = NewDb("eplocaldb")
	if db == nil {
		t.Fail()
	}

	testCount := 1000
	keys := make([][]byte, testCount)
	for i := 0; i < 1000; i++ {
		key, _ := crypto.GenerateKey()
		keys[i] = crypto.FromECDSAPub(&key.PublicKey)
	}

	allQuit := make(chan struct{}, 3)

	go func() {
		for i := 0; i < testCount; i++ {
			db := NewDb("pos")
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
			db := NewDb("rblocaldb")
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
			db := NewDb("eplocaldb")
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
