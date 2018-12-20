package posdb

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
)

func TestWanposDbSuccess(t *testing.T) {
	GetDb().DbInit("test")

	//Test for database put/get with epochID
	//Put
	for i := 0; i < 2000; i++ {
		alphaI := big.NewInt(int64(i)).Bytes()
		epochID := uint64(100000000 + i)
		GetDb().Put(epochID, "alpha", alphaI)
	}

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
