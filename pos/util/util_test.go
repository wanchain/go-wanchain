package util

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/util"
)

func TestGetEpochSlotID(t *testing.T) {
	epochID, slotID := GetEpochSlotID()
	fmt.Println("epochID:", epochID, " slotID:", slotID)
}

func TestPkCompress(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pk := &key.PublicKey

	buf, err := util.CompressPk(pk)
	if err != nil {
		t.Fail()
	}

	fmt.Println("len(pk):", len(buf))

	pkUncompress, err := util.UncompressPk(buf)
	if err != nil {
		t.Fail()
	}

	if hex.EncodeToString(crypto.FromECDSAPub(pk)) != hex.EncodeToString(crypto.FromECDSAPub(pkUncompress)) {
		t.Fail()
	}
}
