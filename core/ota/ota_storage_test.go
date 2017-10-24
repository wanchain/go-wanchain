package ota

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"math/big"
	"testing"
)

var (
	otaShortAddrs = []string{
		"0x022c849aefd10287bb1fb831524a83403ecefc9d546fbf73ef5e95b79c3cb5ae7602ca02565436af262a4cc9197145278d355aee79140e201e35879c5ac72f5dbd2f",
		"0x0348cc8f64f14085eb24e100db9dbd46d217a44451c571f3ebbb8a1b387e2a613c03ef64a43cc2f4498a6641dcee5afe317654d72f61971c03821a1f1b06a32a58db",
		"0x02864c100e06bcfc53ad86aecd0d14b126bc90268b5a64e267556244281d7c0288032f82c8055f947a1509885f5551804fcfb6fa084c2b0915a286747a892cdaba54",
		"0x03bfdf88c14bda519d7d348be2b3a04e9ea7888e064707ffd9bba9dc264e6d8c9f03d7ea3d3d10f39115ff00c70606cae16e9ef7dbcb533f907d3d05e88983023e5e",
		"0x02850cbb0c4b8e3930e5dd79eb7b736c38e24514f89168f87a25496658713a90a4029eccc7471db606ed4a279b4571e4a4ea2f0158ebf53e20071c85d0b2d1ec5fab",
		"0x02483128152168625de2b21b4d7ba1f8e98a160ea78361b3225695517385fc3218023f1f8f4079be98200f882cbdaabbc6cc18ceae48b44f6bf9053de09d024de9be",
		"0x0305246565268865843190a09ece7cce28c9295d11f79930ca1787f2f044e413fd02e87f1b3c3103f028a000c7bda3e09d82f56e63ac1edf157f8955c61a059aa8a8",
		"0x027037ad331a3028d9005f1eb2b78b288fcece677c380142ea5b9919f1302ed00b032a5e555c0bbb29c42b5f5e7402f35bc22bc34d0d008dac41b00ad43fdb39f6d5",
		"0x039d89050b5981bcb6de8c47cdee5365b8676698cba82ccc244cea33ff4da814d6026c94b7fa6b5ce6bb67449d2db032271081abb1dde056de4a2f31130a979e9479",
	}
)

func TestGetOtaBalance(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr = common.FromHex(otaShortAddrs[0])
	)

	t.Logf("otaShortAddr len:%d", len(otaShortAddr))
	otaAX := otaShortAddr[1:common.HashLength]
	balance, err := GetOtaBalanceFromAX(statedb, otaAX)
	if err == nil || balance != nil {
		t.Errorf("otaAX len:%d, err:%s", len(otaAX), err.Error())
	}

	otaAX = otaShortAddr[1 : 1+common.HashLength]
	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if balance != nil && balance.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("balance:%v", balance)
	}

	err = SetOTA(statedb, big.NewInt(10), otaShortAddr)
	if err != nil {
		t.Errorf("SetOTA err:%s", err.Error())
		return
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("GetOtaBalanceFromAX err:%s", err.Error())
	}

	if balance == nil || balance.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("GetOtaBalanceFromAX balance:%v", balance)
	}
}

func TestCheckOTAExit(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr = common.FromHex(otaShortAddrs[1])
		otaAX        = otaShortAddr[1 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	exit, balanceGet, err := CheckOTAExit(statedb, otaAX)
	if err != nil {
		t.Errorf("CheckOTAExit, err:%s", err.Error())
	}

	if exit || (balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0) {
		t.Errorf("exit:%d, balance:%v", exit, balanceGet)
	}

	err = SetOTA(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("SetOTA err:%s", err.Error())
	}

	exit, balanceGet, err = CheckOTAExit(statedb, otaAX)
	if err != nil {
		t.Errorf("CheckOTAExit, err:%s", err.Error())
	}
	if !exit || balanceGet == nil || balanceGet.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("ChechOTAExit, exit:%d, balanceGet:%v", exit, balanceGet)
	}
}

func TestBatCheckOTAExit(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddrBytes = [][]byte{
			common.FromHex(otaShortAddrs[1]),
			common.FromHex(otaShortAddrs[2]),
			common.FromHex(otaShortAddrs[3]),
			common.FromHex(otaShortAddrs[4]),
		}

		balanceSet = big.NewInt(10)
	)

	otaAXs := make([][]byte, 0, 4)
	for _, otaShortAddr := range otaShortAddrBytes {
		otaAXs = append(otaAXs, otaShortAddr[1:1+common.HashLength])
	}

	exit, balanceGet, unexitotaAx, err := BatCheckOTAExit(statedb, otaAXs)
	if exit || (balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0) {
		t.Errorf("exit:%d, balanceGet:%v", exit, balanceGet)
	}

	if unexitotaAx == nil {
		t.Errorf("unexitotaAX is nil!")
	}

	if common.ToHex(unexitotaAx) != common.ToHex(otaAXs[0]) {
		t.Errorf("unexitotaAx:%s, expect:%s", common.ToHex(unexitotaAx), common.ToHex(otaAXs[0]))
	}

	if err != nil {
		t.Logf("err:%s", err.Error())
	}

	for _, otaShortAddr := range otaShortAddrBytes {
		err = SetOTA(statedb, balanceSet, otaShortAddr)
		if err != nil {
			t.Errorf("err:%s", err.Error())
		}
	}

	exit, balanceGet, unexitotaAx, err = BatCheckOTAExit(statedb, otaAXs)
	if !exit || (balanceGet != nil && balanceSet.Cmp(balanceGet) != 0) {
		t.Errorf("exit:%d, balanceGet:%v", exit, balanceGet)
	}

	if unexitotaAx != nil {
		t.Errorf("unexitota:%s", common.ToHex(unexitotaAx))
	}

	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	unexitotaShortAddr := common.FromHex(otaShortAddrs[5])
	unexitotaAXSet := unexitotaShortAddr[1 : 1+common.HashLength]
	otaAXs = append(otaAXs, unexitotaAXSet)
	exit, balanceGet, unexitotaAx, err = BatCheckOTAExit(statedb, otaAXs)
	if exit || (balanceGet != nil && balanceSet.Cmp(balanceGet) == 0) {
		t.Errorf("exit:%d, balanceGet:%v", exit, balanceGet)
	}

	if unexitotaAx != nil {
		t.Logf("unexitotaAx:%s", common.ToHex(unexitotaAx))
	}
	if err != nil {
		t.Logf("err:%s", err.Error())
	}

	err = SetOTA(statedb, big.NewInt(0).Add(balanceSet, big.NewInt(10)), unexitotaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	exit, balanceGet, unexitotaAx, err = BatCheckOTAExit(statedb, otaAXs)
	if exit || (balanceGet != nil && balanceSet.Cmp(balanceGet) == 0) {
		t.Errorf("exit:%d, balanceGet:%v", exit, balanceGet)
	}

	if exit || (balanceGet != nil && balanceSet.Cmp(balanceGet) == 0) {
		t.Errorf("exit:%d, balanceGet:%v", exit, balanceGet)
	}

	if err != nil {
		t.Logf("err:%s", err.Error())
	}

	if unexitotaAx == nil {
		t.Errorf("unexitota is nil!")
	}

	if common.ToHex(unexitotaAx) != common.ToHex(unexitotaAXSet) {
		t.Errorf("unexitota:%s, expect:%s", common.ToHex(unexitotaAx), common.ToHex(unexitotaAXSet))
	}

}

func TestSetOTA(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr = common.FromHex(otaShortAddrs[3])
		otaAX        = otaShortAddr[1 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	t.Logf("otaShortAddr len:%d", len(otaShortAddr))

	err := SetOTA(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	balance, err := GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if balance == nil || balance.Cmp(balanceSet) != 0 {
		t.Errorf("balance:%v", balance)
	}
}

func TestAddOTAIfNotExit(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr = common.FromHex(otaShortAddrs[4])
		otaAX        = otaShortAddr[1 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	t.Logf("otaShortAddr len:%d", len(otaShortAddr))

	add, err := AddOTAIfNotExit(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if !add {
		t.Errorf("add is false!")
	}

	add, err = AddOTAIfNotExit(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if add {
		t.Errorf("add is true!")
	}

	balance, err := GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if balance == nil || balance.Cmp(balanceSet) != 0 {
		t.Errorf("balance:%v", balance)
	}
}

func TestGetOTAInfoFromAX(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr = common.FromHex(otaShortAddrs[4])
		otaAX        = otaShortAddr[1 : 1+common.HashLength]
		balanceSet   = big.NewInt(10)
	)

	otaShortAddrGet, balanceGet, err := GetOTAInfoFromAX(statedb, otaAX)
	if otaShortAddrGet != nil {
		t.Errorf("otaShortAddrGet is not nil.")
	}

	if balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("balance is not 0! balance:%s", balanceGet.String())
	}

	if err == nil {
		t.Errorf("err is nil!")
	}

	err = SetOTA(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	otaShortAddrGet, balanceGet, err = GetOTAInfoFromAX(statedb, otaAX)
	if otaShortAddrGet == nil {
		t.Errorf("otaShortAddrGet is nil!")
	}

	if common.ToHex(otaShortAddrGet) != common.ToHex(otaShortAddr) {
		t.Errorf("otaShortAddrGet:%s, expect:%s", common.ToHex(otaShortAddrGet), common.ToHex(otaShortAddr))
	}

	if balanceGet == nil {
		t.Errorf("balanceGet is nil!")
	}

	if balanceSet.Cmp(balanceGet) != 0 {
		t.Errorf("balanceGet:%v, expect:%v", balanceGet, balanceSet)
	}

}

func TestGetOTASet(t *testing.T) {
	t.Logf("TestGetOTASet begin..")
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr      = common.FromHex(otaShortAddrs[6])
		otaShortAddrBytes = [][]byte{
			common.FromHex(otaShortAddrs[1]),
			common.FromHex(otaShortAddrs[2]),
			common.FromHex(otaShortAddrs[3]),
			common.FromHex(otaShortAddrs[4]),
		}
		otaAX      = otaShortAddr[1 : 1+common.HashLength]
		balanceSet = big.NewInt(10)
	)

	otaShortAddrBytesGet, balanceGet, err := GetOTASet(statedb, otaAX, 3)
	if err == nil {
		t.Errorf("err is nil!")
	}

	if otaShortAddrBytesGet != nil {
		t.Errorf("otaShortAddrBytesGet is not nil!")
	}

	if balanceGet != nil && balanceGet.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("balanceGet is not 0! balanceGet:%s", balanceGet.String())
	}

	err = SetOTA(statedb, balanceSet, otaShortAddr)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	for _, otaShortAddrTmp := range otaShortAddrBytes {
		t.Logf("otaShortAddrTmp len:%d", len(otaShortAddrTmp))
		err = SetOTA(statedb, balanceSet, otaShortAddrTmp)
		if err != nil {
			t.Errorf("err:%s", err.Error())
		}
	}

	// mem database Iterator doesnt work. unit test alwayse fail!!

	//setLen := 3
	//otaShortAddrBytesGet, balanceGet, err = GetOTASet(statedb, otaAX, setLen)
	//if err != nil {
	//	t.Errorf("err:%s", err.Error())
	//}
	//
	//if otaShortAddrBytesGet == nil {
	//	t.Errorf("otaShortAddrBytesGet is nil!")
	//}
	//
	//if len(otaShortAddrBytesGet) != setLen {
	//	t.Errorf("otaShortAddrBytesGet len is wrong! len:%d, expect:%d", len(otaShortAddrBytesGet), setLen)
	//}
	//
	//for _, otaShortAddrGet := range otaShortAddrBytesGet {
	//	otaAXGet := otaShortAddrGet[1 : 1+common.HashLength]
	//	otaShortAddrReGet, balanceReGet, err := GetOTAInfoFromAX(statedb, otaAXGet)
	//	if err != nil {
	//		t.Errorf("err:%s", err.Error())
	//	}
	//
	//	if common.ToHex(otaShortAddrReGet) != common.ToHex(otaShortAddrGet) {
	//		t.Errorf("otaShortAddrReGet:%s, expect:%s", common.ToHex(otaShortAddrReGet), common.ToHex(otaShortAddrGet))
	//	}
	//
	//	if balanceReGet == nil {
	//		t.Errorf("balanceReGet is nil!")
	//	}
	//
	//	if balanceReGet.Cmp(balanceSet) != 0 {
	//		t.Errorf("balanceReGet:%s, expect:%s", balanceReGet.String(), balanceSet.String())
	//	}
	//}
}

func TestCheckOTAImageExit(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr = common.FromHex(otaShortAddrs[7])
		balanceSet   = big.NewInt(10)
	)

	otaImage := crypto.Keccak256(otaShortAddr)
	otaImageValue := balanceSet.Bytes()

	exit, otaImageValueGet, err := CheckOTAImageExit(statedb, otaImage)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if exit {
		t.Errorf("exit is true!")
	}

	if otaImageValueGet != nil && len(otaImageValueGet) != 0 {
		t.Errorf("otaImageValueGet is not empoty!")
	}

	err = AddOTAImage(statedb, otaImage, otaImageValue)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	exit, otaImageValueGet, err = CheckOTAImageExit(statedb, otaImage)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	if !exit {
		t.Errorf("exit is false!")
	}

	if otaImageValueGet == nil || common.ToHex(otaImageValueGet) != common.ToHex(otaImageValue) {
		t.Errorf("otaImageValueGet:%s, expect:%s", common.ToHex(otaImageValueGet), common.ToHex(otaImageValue))
	}
}

func TestAddOTAImage(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)

		otaShortAddr = common.FromHex(otaShortAddrs[7])
		balanceSet   = big.NewInt(10)
	)

	otaImage := crypto.Keccak256(otaShortAddr)
	otaImageValue := balanceSet.Bytes()

	err := AddOTAImage(statedb, otaImage, otaImageValue)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}

	err = AddOTAImage(statedb, otaImage, otaImageValue)
	if err != nil {
		t.Errorf("err:%s", err.Error())
	}
}
