package posdb

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/crypto"
)

func TestWanposDbSuccess(t *testing.T) {
	GetDb().DbInit("test")

	//Test for database put/get with epochID
	//Put

	t1 := time.Now()
	for i := 0; i < 2000; i++ {
		alphaI := big.NewInt(int64(i)).Bytes()
		epochID := uint64(100000000 + i)
		GetDb().Put(epochID, "alpha", alphaI)
	}

	t2 := time.Since(t1)
	fmt.Println("Put:", t2)

	t1 = time.Now()
	//Get and verify
	for i := 0; i < 2000; i++ {
		alphaI := big.NewInt(int64(i)).Bytes()
		epochID := uint64(100000000 + i)
		ret, err := GetDb().Get(epochID, "alpha")
		if err != nil {
			fmt.Println(err.Error())
			t.Fail()
		}

		if hex.EncodeToString(alphaI) != hex.EncodeToString(ret) {
			t.Fail()
		}
	}

	t2 = time.Since(t1)
	fmt.Println("Get:", t2)

	//Test for database put/get with epochID and index
	//Put
	for i := 0; i < 2000; i++ {
		epochID := uint64(100000000 + i)
		for index := 0; index < 100; index++ {
			alphaI := big.NewInt(int64(i + index)).Bytes()
			GetDb().PutWithIndex(epochID, uint64(index), "alpha", alphaI)
		}
	}

	//Get and verify
	for i := 0; i < 2000; i++ {
		epochID := uint64(100000000 + i)

		for index := 0; index < 100; index++ {
			alphaI := big.NewInt(int64(i + index)).Bytes()
			ret, err := GetDb().GetWithIndex(epochID, uint64(index), "alpha")
			if err != nil {
				fmt.Println(err.Error())
				t.Fail()
			}
			if hex.EncodeToString(alphaI) != hex.EncodeToString(ret) {
				t.Fail()
			}
		}
	}
}

func TestWanposDbLoad(t *testing.T) {
	GetDb().DbInit("test")

	//Test for database put/get with epochID

	//Get and verify
	for i := 0; i < 2000; i++ {
		alphaI := big.NewInt(int64(i)).Bytes()
		epochID := uint64(100000000 + i)
		ret, err := GetDb().Get(epochID, "alpha")
		if err != nil {
			fmt.Println(err.Error())
			t.Fail()
		}

		if hex.EncodeToString(alphaI) != hex.EncodeToString(ret) {
			t.Fail()
		}
	}

	//Test for database put/get with epochID and index

	//Get and verify
	for i := 0; i < 2000; i++ {
		epochID := uint64(100000000 + i)
		for index := 0; index < 100; index++ {
			alphaI := big.NewInt(int64(i + index)).Bytes()
			ret, err := GetDb().GetWithIndex(epochID, uint64(index), "alpha")
			if err != nil {
				fmt.Println(err.Error())
				t.Fail()
			}
			if hex.EncodeToString(alphaI) != hex.EncodeToString(ret) {
				t.Fail()
			}
		}
	}
}

func TestWanposDbFail(t *testing.T) {
	GetDb().DbInit("test")

	alpha := big.NewInt(1)

	epochID := uint64(2000)

	GetDb().Put(epochID, "alpha", alpha.Bytes())

	epochID2 := uint64(100000000 + 99999992001)

	alphaGet, err := GetDb().Get(epochID2, "alpha")

	if err.Error() != "leveldb: not found" {
		t.Fail()
	}

	if hex.EncodeToString(alphaGet) == hex.EncodeToString(alpha.Bytes()) {
		t.Fail()
	}
}

func TestGetStorageByteArray(t *testing.T) {
	GetDb().DbInit("test")

	keys := make([][]byte, 0)

	for i := 0; i < 100; i++ {
		for m := 0; m < 200; m++ {
			key, _ := crypto.GenerateKey()
			GetDb().PutWithIndex(uint64(i), uint64(m), "", crypto.FromECDSAPub(&key.PublicKey))
			keys = append(keys, crypto.FromECDSAPub(&key.PublicKey))
		}
	}

	fmt.Println("keys count:", len(keys))

	for i := 0; i < 100; i++ {
		values := GetDb().GetStorageByteArray(uint64(i))
		fmt.Println("values count: ", len(values))
		for m := 0; m < len(values); m++ {
			if hex.EncodeToString(values[m]) != hex.EncodeToString(keys[i*200+m]) {
				t.Fail()
			}
		}
	}
}

func TestUintToBytes(t *testing.T) {
	buf := Uint64ToBytes(0)

	fmt.Println(len(buf))
	fmt.Println(buf)

	str := hex.EncodeToString(buf)
	fmt.Println(str)
}
