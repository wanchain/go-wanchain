package util

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
)

func TestGetEpochSlotID(t *testing.T) {
	epochID, slotID := GetEpochSlotID()
	fmt.Println("epochID:", epochID, " slotID:", slotID)
}

func TestPkCompress(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pk := &key.PublicKey

	buf, err := CompressPk(pk)
	if err != nil {
		t.Fail()
	}

	fmt.Println("len(pk):", len(buf))

	pkUncompress, err := UncompressPk(buf)
	if err != nil {
		t.Fail()
	}

	if hex.EncodeToString(crypto.FromECDSAPub(pk)) != hex.EncodeToString(crypto.FromECDSAPub(pkUncompress)) {
		t.Fail()
	}
}

func TestGetEpochIDFromDifficulty(t *testing.T) {
	GetEpochSlotIDFromDifficulty(nil)

	ep, sl := GetEpochSlotIDFromDifficulty(big.NewInt(3<<32 | 4<<8 | 1))
	if ep != 3 || sl != 4 {
		t.FailNow()
	}
}

func TestFromWin(t *testing.T) {
	a, _ := big.NewInt(0).SetString("83713850837138508370", 10)
	f := FromWin(a)
	fmt.Println(f)
}
