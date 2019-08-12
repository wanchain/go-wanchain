package convert

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
)

func TestUint64Convert(t *testing.T) {
	value := uint64(0)
	buf := Uint64ToBytes(value)
	if len(buf) != 1 || buf[0] != 0 {
		t.Fail()
	}

	buf = Uint64ToBytes(uint64(0x12345678))
	if buf[0] != 0x12 || buf[1] != 0x34 || buf[2] != 0x56 || buf[3] != 0x78 {
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

func TestOtherConvert(t *testing.T) {
	biArray := make([]*big.Int, 100)
	for i := 0; i < 100; i++ {
		biArray[i] = big.NewInt(int64(i))
	}

	buf2Array := BigIntArrayToByteArray(biArray)

	biArray2 := ByteArrayToBigIntArray(buf2Array)
	for i := 0; i < 100; i++ {
		if biArray2[i].String() != biArray[i].String() {
			t.FailNow()
		}
	}

	keys := make([]*ecdsa.PublicKey, 100)
	for i := 0; i < 100; i++ {
		key, _ := crypto.GenerateKey()
		keys[i] = &key.PublicKey
	}

	bufKey := PkArrayToByteArray(keys)
	pk2 := ByteArrayToPkArray(bufKey)
	for i := 0; i < 100; i++ {
		if hex.EncodeToString(crypto.FromECDSAPub(pk2[i])) != hex.EncodeToString(crypto.FromECDSAPub(keys[i])) {
			t.FailNow()
		}
	}
}
