package slotleader

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"

	"github.com/btcsuite/btcd/btcec"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"
)

var (
	addrsCount = posconfig.EpochLeaderCount
	epAddrs    = make([]common.Address, addrsCount)
	epPks      = make([]*ecdsa.PublicKey, addrsCount)
)

type TestSelectLead struct{}

func (t *TestSelectLead) SelectLeadersLoop(epochID uint64) error { return nil }
func (t *TestSelectLead) GetEpochLeaders(epochID uint64) [][]byte {
	buf := make([][]byte, len(epPks))
	for i := 0; i < len(epPks); i++ {
		buf[i] = crypto.FromECDSAPub(epPks[i])
	}
	return buf
}
func (t *TestSelectLead) GetProposerBn256PK(epochID uint64, idx uint64, addr common.Address) []byte {
	return nil
}

func generateTestAddrs() {
	for i := 0; i < addrsCount; i++ {
		key, _ := crypto.GenerateKey()
		epAddrs[i] = crypto.PubkeyToAddress(key.PublicKey)
		epPks[i] = &key.PublicKey
	}
}

func TestLoop(t *testing.T) {
	posdb.GetDb().DbInit("test")

	generateTestAddrs()
	testInitSlotleader()
	util.SetEpocherInst(&TestSelectLead{})

	key := &keystore.Key{}
	key.PrivateKey, _ = crypto.GenerateKey()
	key.PrivateKey.PublicKey = *epPks[0]

	epochIDStart := time.Now().Second()

	for i := 0; i < posconfig.SlotCount; i++ {
		s.Loop(&rpc.Client{}, key, util.GetEpocherInst(), uint64(epochIDStart+0), uint64(i))
	}

	for i := 0; i < posconfig.SlotCount; i++ {
		s.Loop(&rpc.Client{}, key, util.GetEpocherInst(), uint64(epochIDStart+1), uint64(i))
	}
}

func TestGenerateCommitmentSuccess(t *testing.T) {
	posdb.GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	fmt.Println("priv len:", len(crypto.FromECDSA(privKey)))
	fmt.Println("pk len:", len(crypto.FromECDSAPub(&privKey.PublicKey)))
	fmt.Println("pk: ", hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey)))

	pkCompress := btcec.PublicKey(privKey.PublicKey)
	fmt.Println("compressed pk: :", hex.EncodeToString(pkCompress.SerializeCompressed()), "len: ", len(pkCompress.SerializeCompressed()))

	epochID := uint64(8192)
	payload, err := slot.generateCommitment(&privKey.PublicKey, epochID, 0)
	if err != nil {
		t.Fail()
	}

	if payload == nil {
		t.Fail()
	}

	fmt.Println("payload len:", len(payload), " data: ", hex.EncodeToString(payload))

	alpha, err := slot.getAlpha(epochID, 0)
	if alpha == nil || err != nil {
		t.Fail()
	}

	epID, selfIndex, _, err := vm.RlpUnpackStage1DataForTx(payload)
	if err != nil {
		t.Fail()
	}
	var output [][]byte
	rlp.DecodeBytes(payload, &output)

	fmt.Println("epochID: ", epID)
	fmt.Println("selfIndex: ", selfIndex)
	fmt.Println("Alpha: ", alpha)
}

func TestGenerateCommitmentFailed(t *testing.T) {
	posdb.GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()

	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	epochID := uint64(1)

	_, err = slot.generateCommitment(nil, epochID, 0)
	if err == nil {
		t.Fail()
	}

	// _, err = slot.GenerateCommitment(&privKey.PublicKey, 0)
	// if err == nil {
	// 	t.Fail()
	// }

	privKey.PublicKey.X = nil
	privKey.PublicKey.Y = nil
	_, err = slot.generateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey.PublicKey.Curve = nil
	_, err = slot.generateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey2, _ := crypto.GenerateKey()

	privKey.X = privKey2.X
	_, err = slot.generateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}
}

func TestStartStage1Work(t *testing.T) {
	TestLoop(t)

	err := s.startStage1Work()
	if err != nil {
		t.FailNow()
	}
}
