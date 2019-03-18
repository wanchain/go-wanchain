package randombeacon

import (
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/pos/wanpos_crypto"
	"math/big"
	"testing"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	accBn256 "github.com/wanchain/go-wanchain/accounts/keystore/bn256"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/go-wanchain/core/vm"
)

var(
	selfPrivate *accBn256.PrivateKeyBn256
	commityPrivate *accBn256.PrivateKeyBn256
	proposerGroupLen = 10
)

func TestInit(t *testing.T) {
	var epocher epochLeader.Epocher
	var key keystore.Key
	var rb RandomBeacon

	var err error
	key.PrivateKey3, err = accBn256.GenerateBn256()
	if err != nil {
		t.Error("generate bn256 fail, ", err)
	}

	rb.Init(&epocher)

	if rb.epochStage != vm.RbDkg1Stage {
		t.Error("invalid epoch stage")
	}

	if rb.epochId != maxUint64 {
		t.Error("invalid init epoch id")
	}

	if rb.statedb != nil {
		t.Error("invalid init statedb")
	}

	if rb.epocher != &epocher {
		t.Error("invalid rb epocher")
	}

	if rb.rpcClient != nil {
		t.Error("invalid rb rpc client")
	}
}

func tmpGetRBProposerGroup(epochId uint64) []bn256.G1 {
	ret := make([]bn256.G1, proposerGroupLen)
	for i := 0; i < proposerGroupLen; i++ {
		ret[i] = *commityPrivate.PublicKeyBn256.G1
	}

	return ret
}

func TestGetMyRBProposerId(t *testing.T) {
	var epocher epochLeader.Epocher
	var key keystore.Key
	var rb RandomBeacon

	var err error
	key.PrivateKey3, err = accBn256.GenerateBn256()
	if err != nil {
		t.Error("generate bn256 fail, ", err)
	}

	selfPrivate = key.PrivateKey3
	pos.Cfg().MinerKey = &key

	commityPrivate, err = accBn256.GenerateBn256()
	if err != nil {
		t.Error("generate bn256 fail, ", err)
	}

	rb.Init(&epocher)
	rb.getRBProposerGroupF = tmpGetRBProposerGroup

	ids := rb.getMyRBProposerId(0)
	println("ids len:", len(ids))
	if len(ids) != 0 {
		t.Error("invalid my proposer id")
	}

	commityPrivate = key.PrivateKey3
	ids = rb.getMyRBProposerId(0)
	println("ids len:", len(ids))
	if len(ids) != proposerGroupLen {
		t.Error("invalid my proposer id group len. expect len:", proposerGroupLen, ", acture:", len(ids))
	}

	for i := 0; i < len(ids); i++ {
		println("ids[", i, "]:", ids[i])
		if ids[i] != uint32(i) {
			t.Error("invalid my proposer id. expect:", i, ", acture:", ids[i])
			break
		}
	}
}

func TestDoGenerateDKG(t *testing.T) {
	var epocher epochLeader.Epocher
	var key keystore.Key
	var rb RandomBeacon

	var err error
	key.PrivateKey3, err = accBn256.GenerateBn256()
	if err != nil {
		t.Error("generate bn256 fail, ", err)
	}

	selfPrivate = key.PrivateKey3
	commityPrivate = selfPrivate
	pos.Cfg().MinerKey = &key

	rb.Init(&epocher)
	rb.getRBProposerGroupF = tmpGetRBProposerGroup

	epochId := uint64(0)
	proposerId := uint32(0)
	payload, err := rb.generateDKG1(epochId, proposerId)
	if err != nil {
		t.Fatal("rb generate dkg info fail. err:", err)
	}

	if payload == nil {
		t.Fatal("rb generate dkg info is nil")
	}

	// verify
	if payload.EpochId != epochId || payload.ProposerId != proposerId {
		t.Error("invalid epochId proposerId")
	}

	// Reed-Solomon code verification
	dkg1Param, err := vm.Dkg1FlatToDkg1(payload)
	if err != nil {
		t.Error("trans dkg1flat to dkg1 fail. err:", err)
	}

	pks := rb.getRBProposerGroup(epochId)
	nr := len(pks)

	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&pks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	temp := make([]bn256.G2, nr)
	for j := 0; j < nr; j++ {
		temp[j] = *dkg1Param.Commit[j]
	}

	if !wanpos.RScodeVerify(temp, x, int(pos.Cfg().PolymDegree)) {
		t.Error("reed solomon verification fail")
	}
}

//func TestGetRBDKGTxPayloadBytes(t *testing.T) {
//	var payload *vm.RbDKGTxPayload
//	payloadBuf, err := getRBDKGTxPayloadBytes(payload)
//	if err == nil {
//		t.Fatal("should retrun error while payload is nil")
//	}
//
//	if payloadBuf != nil {
//		t.Fatal("should retrun nil while payload is nil")
//	}
//
//
//}











