package randombeacon

import (
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	accBn256 "github.com/wanchain/go-wanchain/accounts/keystore/bn256"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/rbselection"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"
)

var (
	//selfPrivate      *accBn256.PrivateKeyBn256 = &accBn256.PrivateKeyBn256{}
	//commityPrivate   *accBn256.PrivateKeyBn256

	selfPrivate      = &accBn256.PrivateKeyBn256{}
	commityPrivate   = &accBn256.PrivateKeyBn256{}
	hbase            = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))
	ens              = make([][]*bn256.G1, 0)
	commit           [][]bn256.G2
)



func initKeystore(rb *RandomBeacon) error {
	var key keystore.Key
	var err error

	key.PrivateKey2, err = crypto.GenerateKey()
	if err != nil {
		return err
	}

	posconfig.Cfg().MinerKey = &key

	selfPrivate.D = posconfig.Cfg().GetMinerBn256SK()
	selfPrivate.G1 = posconfig.Cfg().GetMinerBn256PK()

	commityPrivate = selfPrivate

	return nil
}

func callRBLoop(rb *RandomBeacon, wg *sync.WaitGroup) {
	defer func(){
		wg.Done()
	}()

	fmt.Println("callRBLoop begin")
	if rb == nil {
		return
	}

	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
		epochId = uint64(0)
		slotId = uint64(0)
		rc = new(rpc.Client)
	)

	for ;; {
		err := rb.Loop(statedb, rc, epochId, slotId)
		if err != nil {
			fmt.Println("callRbLoop break loop, err:", err)
			break
		}
		epochId++
		slotId++
		time.Sleep(time.Second*10)
	}
}

func tmpGetRBProposerGroup(epochId uint64) []bn256.G1 {
	ret := make([]bn256.G1, posconfig.RandomProperCount)
	for i := 0; i < posconfig.RandomProperCount; i++ {
		ret[i] = *commityPrivate.PublicKeyBn256.G1
	}

	return ret
}


var (
	dkg1sCallTimes = 0
	dkg2sCallTimes = 0
	sigsCallTimes = 0
)

func DoDKG1sSuc() error {
	dkg1sCallTimes++
	return nil
}

func DoDKG1sFail() error {
	dkg1sCallTimes++
	return errors.New("fail")
}

func DoDKG2sSuc() error {
	dkg2sCallTimes++
	return nil
}

func DoDKG2sFail() error {
	dkg2sCallTimes++
	return errors.New("fail")
}

func DoSIGsSuc() error {
	sigsCallTimes++
	return nil
}

func DoSIGsFail() error {
	sigsCallTimes++
	return errors.New("fail")
}


func tmpGetEnsFunc(db vm.StateDB, epochId uint64, proposerId uint32) ([]*bn256.G1, error) {
	return ens[proposerId], nil
}

func tmpGetRBM(db vm.StateDB, epochId uint64) ([]byte, error) {
	epochIdBigInt := big.NewInt(int64(epochId + 1))
	buf := epochIdBigInt.Bytes()
	return crypto.Keccak256(buf), nil
}

func tmpGetCji(db vm.StateDB, epochId uint64, proposerId uint32) ([]*bn256.G2, error) {
	ret := make([]*bn256.G2, len(commit[proposerId]))
	for i, _ := range commit[proposerId] {
		ret[i] = &commit[proposerId][i]
	}

	return ret, nil
}

func TestRandomBeacon_GetMyRBProposerId(t *testing.T) {
	var epocher epochLeader.Epocher
	var key keystore.Key
	var rb RandomBeacon

	var err error

	// commityPrivate
	key.PrivateKey2, err = crypto.GenerateKey()
	if err != nil {
		t.Error("generate sec256 fail, ", err)
	}

	posconfig.Cfg().MinerKey = &key
	commityPrivate.D	 = posconfig.Cfg().GetMinerBn256SK()
	commityPrivate.G1 	 = posconfig.Cfg().GetMinerBn256PK()

	// commityPrivate not equal selfPrivate
	key.PrivateKey2, err = crypto.GenerateKey()
	if err != nil {
		t.Error("generate sec256 fail, ", err)
	}

	posconfig.Cfg().MinerKey = &key

	selfPrivate.D = posconfig.Cfg().GetMinerBn256SK()
	selfPrivate.G1 = posconfig.Cfg().GetMinerBn256PK()

	rb.Init(&epocher)
	rb.getRBProposerGroupF = tmpGetRBProposerGroup

	rb.myPropserIds = rb.getMyRBProposerId(0)
	println("ids len:", len(rb.myPropserIds))
	if len(rb.myPropserIds) != 0 {
		t.Error("invalid my proposer id")
	}
	// commityPrivate equal selfPrivate
	commityPrivate = selfPrivate
	rb.myPropserIds = rb.getMyRBProposerId(0)
	println("ids len:", len(rb.myPropserIds))
	if len(rb.myPropserIds) != posconfig.RandomProperCount {
		t.Error("invalid my proposer id group len. expect len:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
	}

	for i := 0; i < len(rb.myPropserIds); i++ {
		println("ids[", i, "]:", rb.myPropserIds[i])
		if rb.myPropserIds[i] != uint32(i) {
			t.Error("invalid my proposer id. expect:", i, ", acture:", rb.myPropserIds[i])
			break
		}
	}
}

func TestRandomBeacon_updateEpochId(t *testing.T) {
	rb := GetRandonBeaconInst()
	if rb == nil {
		t.Error("invalid random beacon instance")
	}

	err := initKeystore(rb)
	if err != nil {
		t.Error("init keystore fail. err:", err.Error())
	}

	rb.getRBProposerGroupF = tmpGetRBProposerGroup

	epochId := uint64(100)
	rb.updateEpochId(epochId)
	if rb.epochId != epochId {
		t.Error("invalid epochid, expect:", epochId, ", acture:", rb.epochId)
	}

	if len(rb.myPropserIds) != posconfig.RandomProperCount {
		t.Error("invalid my proposer id length, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
	}

	for i := 0; i < posconfig.RandomProperCount; i++ {
		if rb.myPropserIds[i] != uint32(i) {
			t.Error("invalid proposer id, expect:", i, ", acture:", rb.myPropserIds[i])
		}
	}

	if rb.epochStage != vm.RbDkg1Stage {
		t.Error("invalid epoch stage, expect:", vm.RbDkg1Stage, ", acture:", rb.epochStage)
	}

	if len(rb.polys) != 0 {
		t.Error("invalid polys len, expect:", 0, ", acture:", len(rb.polys))
	}

	if rb.taskTags != nil {
		t.Error("invalid tasdTags, expect nil")
	}
}

func TestRandomBeacon_updateStage(t *testing.T) {
	rb := GetRandonBeaconInst()
	if rb == nil {
		t.Error("invalid random beacon instance")
	}

	err := initKeystore(rb)
	if err != nil {
		t.Error("init keystore fail. err:", err.Error())
	}

	stage := vm.RbDkg2Stage
	rb.getRBProposerGroupF = tmpGetRBProposerGroup
	rb.updateStage(stage)

	if rb.epochStage != stage {
		t.Error("invalid epoch stage, except:", stage, ", acture:", rb.epochStage)
	}

	if rb.taskTags != nil {
		t.Error("invalid taskTags, except nil")
	}
}

func TestRandomBeacon_Init(t *testing.T) {
	var epocher epochLeader.Epocher
	//var key keystore.Key
	var rb RandomBeacon

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

	rb.Init(&epocher)
}

func TestRandomBeacon_DoGenerateDKG1(t *testing.T) {
	var epocher epochLeader.Epocher
	var key keystore.Key
	var rb RandomBeacon

	var err error

	key.PrivateKey2, err = crypto.GenerateKey()
	if err != nil {
		t.Error("generate sec256 fail, ", err)
	}

	posconfig.Cfg().MinerKey = &key

	selfPrivate.D = posconfig.Cfg().GetMinerBn256SK()
	selfPrivate.G1 = posconfig.Cfg().GetMinerBn256PK()

	commityPrivate = selfPrivate

	rb.Init(&epocher)
	rb.getRBProposerGroupF = tmpGetRBProposerGroup
	rb.getCji = tmpGetCji

	rb.epochId = uint64(0)

	// pks
	rb.proposerPks = rb.getRBProposerGroupF(rb.epochId)
	nr := len(rb.proposerPks)
	rb.myPropserIds = rb.getMyRBProposerId(rb.epochId)

	// x
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&rb.proposerPks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	commit = make([][]bn256.G2, nr)
	// generate every dkg1 and verify it
	for proposerId := 0; proposerId < nr; proposerId++ {
		payload, err := rb.generateDKG1(uint32(proposerId))
		if err != nil {
			t.Fatal("rb generate dkg info fail. err:", err)
		}

		if payload == nil {
			t.Fatal("rb generate dkg info is nil")
		}

		// verify
		if payload.EpochId != rb.epochId || payload.ProposerId != uint32(proposerId) {
			t.Error("invalid epochId proposerId")
		}

		// Reed-Solomon code verification
		dkg1, err := vm.Dkg1FlatToDkg1(payload)
		if err != nil {
			t.Error("trans dkg1flat to dkg1 fail. err:", err)
		}

		commit[proposerId] = make([]bn256.G2, nr)
		for j := 0; j < nr; j++ {
			commit[proposerId][j] = *dkg1.Commit[j]
		}

		if !rbselection.RScodeVerify(commit[proposerId], x, int(posconfig.Cfg().PolymDegree)) {
			t.Error("reed solomon verification fail")
		}

	}
}

func TestRandomBeacon_GenerateDKG2(t *testing.T) {
	var epocher epochLeader.Epocher
	var key keystore.Key
	var rb RandomBeacon

	dkg1s := make([]*vm.RbDKG1TxPayload, 0)

	var err error

	key.PrivateKey2, err = crypto.GenerateKey()
	if err != nil {
		t.Error("generate sec256 fail, ", err)
	}

	posconfig.Cfg().MinerKey = &key

	selfPrivate.D = posconfig.Cfg().GetMinerBn256SK()
	selfPrivate.G1 = posconfig.Cfg().GetMinerBn256PK()

	commityPrivate = selfPrivate

	rb.Init(&epocher)
	rb.getRBProposerGroupF = tmpGetRBProposerGroup
	rb.getCji = tmpGetCji

	epochId := uint64(0)
	rb.epochId = epochId

	// pks
	rb.proposerPks = rb.getRBProposerGroupF(epochId)
	nr := len(rb.proposerPks)

	// x
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&rb.proposerPks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	commit = make([][]bn256.G2, nr)

	// generate every dkg1 and verify it
	for proposerId := 0; proposerId < nr; proposerId++ {
		dkg1Flat, err := rb.generateDKG1(uint32(proposerId))
		if err != nil {
			t.Fatal("rb generate dkg1 info fail. err:", err)
		}

		if dkg1Flat == nil {
			t.Fatal("rb generate dkg1 info is nil")
		}

		dkg1, err := vm.Dkg1FlatToDkg1(dkg1Flat)
		if err != nil {
			t.Fatal("trans dkg1flat to dkg1 fail. err:", err)
		}

		commit[proposerId] = make([]bn256.G2, nr)
		for j := 0; j < nr; j++ {
			commit[proposerId][j] = *dkg1.Commit[j]
		}

		dkg1s = append(dkg1s, dkg1)
	}

	for proposerId := 0; proposerId < nr; proposerId++ {
		dkg2Flat, err := rb.generateDKG2(uint32(proposerId))
		if err != nil {
			t.Fatal("rb generate dkg2 fail. err:", err)
		}

		if dkg2Flat == nil {
			t.Fatal("rb generate dkg2 info is nil")
		}

		// verify
		// EnShare, Commit, Proof has the same size
		// check same size
		if nr != len(dkg2Flat.EnShare) {
			t.Fatal("dkg2 params have different length")
		}

		dkg2, err := vm.Dkg2FlatToDkg2(dkg2Flat)
		if err != nil {
			t.Fatal("transf dkg2flat to dkg2 fail, err:", err)
		}

		// proof verification
		for j := 0; j < nr; j++ {
			// get send public Key
			if !rbselection.VerifyDLEQ(dkg2.Proof[j], rb.proposerPks[j], *hbase, *dkg2.EnShare[j], *(dkg1s[proposerId].Commit[j])) {
				t.Fatal("dkg2 DLEQ verify fail")
			}
		}
	}
}

func TestRandomBeacon_GenerateSIG(t *testing.T) {
	var epocher epochLeader.Epocher
	var key keystore.Key
	var rb RandomBeacon

	dkg1s := make([]*vm.RbDKG1TxPayload, 0)
	dkg2s := make([]*vm.RbDKG2TxPayload, 0)

	var err error

	key.PrivateKey2, err = crypto.GenerateKey()
	if err != nil {
		t.Error("generate sec256 fail, ", err)
	}

	posconfig.Cfg().MinerKey = &key

	selfPrivate.D = posconfig.Cfg().GetMinerBn256SK()
	selfPrivate.G1 = posconfig.Cfg().GetMinerBn256PK()

	commityPrivate = selfPrivate

	rb.Init(&epocher)
	rb.getRBProposerGroupF = tmpGetRBProposerGroup
	rb.getEns = tmpGetEnsFunc
	rb.getRBM = tmpGetRBM
	rb.getCji = tmpGetCji

	rb.epochId = uint64(0)

	// pks
	rb.proposerPks = rb.getRBProposerGroupF(rb.epochId)
	nr := len(rb.proposerPks)

	// x
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&rb.proposerPks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	commit = make([][]bn256.G2, nr)

	// generate every dkg1 and verify it
	for proposerId := 0; proposerId < nr; proposerId++ {
		dkg1Flat, err := rb.generateDKG1(uint32(proposerId))
		if err != nil {
			t.Fatal("rb generate dkg1 info fail. err:", err)
		}

		if dkg1Flat == nil {
			t.Fatal("rb generate dkg1 info is nil")
		}

		dkg1, err := vm.Dkg1FlatToDkg1(dkg1Flat)
		if err != nil {
			t.Fatal("trans dkg1flat to dkg1 fail. err:", err)
		}

		commit[proposerId] = make([]bn256.G2, nr)
		for j := 0; j < nr; j++ {
			commit[proposerId][j] = *dkg1.Commit[j]
		}

		dkg1s = append(dkg1s, dkg1)
	}

	for proposerId := 0; proposerId < nr; proposerId++ {
		dkg2Flat, err := rb.generateDKG2(uint32(proposerId))
		if err != nil {
			t.Fatal("rb generate dkg2 fail. err:", err)
		}

		if dkg2Flat == nil {
			t.Fatal("rb generate dkg2 info is nil")
		}

		dkg2, err := vm.Dkg2FlatToDkg2(dkg2Flat)
		if err != nil {
			t.Fatal("transf dkg2flat to dkg2 fail, err:", err)
		}

		dkg2s = append(dkg2s, dkg2)
		ens = append(ens, dkg2.EnShare)
	}

	for proposerId := 0; proposerId < nr; proposerId++ {
		sig, err := rb.generateSIG(uint32(proposerId))
		if err != nil {
			t.Fatal("generate sig fail. err:", err)
		}

		// Verification
		M, err := tmpGetRBM(rb.statedb, rb.epochId)
		if err != nil {
			t.Fatal("getRBM error, err:", err)
		}

		m := new(big.Int).SetBytes(M)
		var gpkshare bn256.G2

		for id := 0; id < nr; id++ {
			gpkshare.Add(&gpkshare, dkg1s[id].Commit[proposerId])
		}

		mG := new(bn256.G1).ScalarBaseMult(m)
		pair1 := bn256.Pair(sig.GSignShare, hbase)
		pair2 := bn256.Pair(mG, &gpkshare)
		if pair1.String() != pair2.String() {
			t.Fatal("verify sig result pair fail")
		}
	}
}

func TestRandomBeacon_doLoop(t *testing.T) {
	posconfig.SelfTestMode = true
	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
		epochId    = uint64(0)
		slotId     = uint64(0)
		rc         = new(rpc.Client)
		epocher    epochLeader.Epocher
		actureDkg1sCallTimes = 0
		actureDkg2sCallTimes = 0
		actureSIGsCallTimes = 0
	)

	rb := GetRandonBeaconInst()
	if rb == nil {
		t.Error("invalid random beacon instance")
	}

	rb.Init(&epocher)
	rb.getRBProposerGroupF = tmpGetRBProposerGroup
	rb.fDoDKG1s = DoDKG1sSuc
	rb.fDoDKG2s = DoDKG2sSuc
	rb.fDoSIGs = DoSIGsSuc

	{
		private, err := accBn256.GenerateBn256()
		if err != nil {
			t.Error("generate Bn256 fail")
		}

		commityPrivate = private

		var key keystore.Key
		key.PrivateKey2, err = crypto.GenerateKey()
		if err != nil {
			t.Error("generate sec256 fail, ", err)
		}
		posconfig.Cfg().MinerKey = &key

		selfPrivate.D = posconfig.Cfg().GetMinerBn256SK()
		selfPrivate.G1 = posconfig.Cfg().GetMinerBn256PK()

		err = rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != 0 {
			t.Error("invalid my propserIds len, expect:", 0, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != 0 || rb.epochStage != vm.RbDkg1Stage {
			t.Error("invalid random beacon state, epochId:", rb.epochId, ", stage:", rb.epochStage)
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		err := initKeystore(rb)
		if err != nil {
			t.Error("init keystore fail. err:", err.Error())
		}

		epochId++
		err = rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg1Stage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}

	}


	{
		slotId++

		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg2Stage {
			t.Error("invalid random beacon state")
		}

		actureDkg1sCallTimes++
		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}

	}

	{
		epochId++
		slotId = 0
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg1Stage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}

	}

	{
		posconfig.SelfTestMode = false
		slotId++
		rb.fDoDKG1s = DoDKG1sFail
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err == nil {
			t.Error("doLoop success. expect fail")
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg1Stage {
			t.Error("invalid random beacon state")
		}

		actureDkg1sCallTimes++
		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
		posconfig.SelfTestMode = true
		rb.fDoDKG1s = DoDKG1sSuc
	}

	{
		slotId++
		posconfig.SelfTestMode = true
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg2Stage {
			t.Error("invalid random beacon state")
		}

		actureDkg1sCallTimes++
		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}


	{
		slotId++
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg2Stage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		slotId = 40
		posconfig.SelfTestMode = true
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg2Stage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		slotId = posconfig.Cfg().Dkg2Begin + 1
		rb.fDoDKG2s = DoDKG2sFail

		posconfig.SelfTestMode = false

		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err == nil {
			t.Error("doLoop success. expect fail")
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg2Stage {
			t.Error("invalid random beacon state")
		}

		actureDkg2sCallTimes++
		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}

		rb.fDoDKG2s = DoDKG2sSuc
	}

	{
		slotId++
		posconfig.SelfTestMode = true
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbSignStage {
			t.Error("invalid random beacon state")
		}

		actureDkg2sCallTimes++
		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		slotId++
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbSignStage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		slotId = 80
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbSignStage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		posconfig.SelfTestMode = false
		slotId = posconfig.Cfg().SignBegin + 1
		rb.fDoSIGs = DoSIGsFail
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err == nil {
			t.Error("doLoop success. expect fail")
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbSignStage {
			t.Error("invalid random beacon state")
		}

		actureSIGsCallTimes++
		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}

		rb.fDoSIGs = DoSIGsSuc
		posconfig.SelfTestMode = true
	}

	{
		slotId++
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbSignConfirmStage {
			t.Error("invalid random beacon state")
		}

		actureSIGsCallTimes++
		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		slotId++
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbSignConfirmStage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		epochId++
		slotId = 0
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err != nil {
			t.Error("doLoop fail. err:", err)
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId || rb.epochStage != vm.RbDkg1Stage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	{
		epochId--
		slotId = 0
		err := rb.doLoop(statedb, rc, epochId, slotId)
		if err == nil {
			t.Error("doLoop success. expect fail")
		}

		if len(rb.myPropserIds) != posconfig.RandomProperCount {
			t.Error("invalid my propserIds len, expect:", posconfig.RandomProperCount, ", acture:", len(rb.myPropserIds))
		}

		if rb.epochId != epochId + 1 || rb.epochStage != vm.RbDkg1Stage {
			t.Error("invalid random beacon state")
		}

		if dkg1sCallTimes != actureDkg1sCallTimes || dkg2sCallTimes != actureDkg2sCallTimes || sigsCallTimes != actureSIGsCallTimes {
			t.Error("invalid stage work run times")
		}
	}

	posconfig.SelfTestMode = false
}

func TestPolyMap_storePolys(t *testing.T) {
	poly1 := make(rbselection.Polynomial, 0)
	poly2 := make(rbselection.Polynomial, 0)
	poly1 = append(poly1, *big.NewInt(11))
	poly1 = append(poly1, *big.NewInt(12))
	poly2 = append(poly2, *big.NewInt(21))
	poly2 = append(poly2, *big.NewInt(22))

	rb := RandomBeacon{}
	rb.polys = make(PolyMap)
	rb.polys[1] = PolyInfo{poly1, big.NewInt(1)}
	rb.polys[2] = PolyInfo{poly2, big.NewInt(2)}

	err := rb.storePolys()
	if err != nil {
		t.Error(err)
	}
}

func TestPolyMap_loadPolys(t *testing.T) {
	TestPolyMap_storePolys(t)

	rb := RandomBeacon{}
	rb.polys = make(PolyMap)
	err := rb.loadPolys()
	if err != nil {
		t.Error(err)
	}

	poly1 := make(rbselection.Polynomial, 0)
	poly2 := make(rbselection.Polynomial, 0)
	poly1 = append(poly1, *big.NewInt(11))
	poly1 = append(poly1, *big.NewInt(12))
	poly2 = append(poly2, *big.NewInt(21))
	poly2 = append(poly2, *big.NewInt(22))

	polys := make(PolyMap)
	polys[1] = PolyInfo{poly1, big.NewInt(1)}
	polys[2] = PolyInfo{poly2, big.NewInt(2)}

	if len(polys) != len(rb.polys) {
		t.Error("rlp decode PolyMap fail, invalid len")
	}

	for k, v := range polys {
		v2, ok := rb.polys[k]
		if !ok {
			t.Error("rlp decode PolyMap fail, invalid key")
		}

		if v.s.Cmp(v2.s) != 0 {
			t.Error("rlp decode PolyMap fail, invalid value")
		}

		if len(v.poly) != len(v2.poly) {
			t.Error("rlp decode PolyMap fail, invalid poly len")
		}

		for i := 0; i < len(v.poly); i++ {
			if v.poly[i].Cmp(&v2.poly[i]) != 0 {
				t.Error("rlp decode PolyMap fail, invalid poly value")
			}
		}
	}
}

func TestPolyMap_DecodeRLP(t *testing.T) {
	poly1 := make(rbselection.Polynomial, 0)
	poly2 := make(rbselection.Polynomial, 0)
	poly1 = append(poly1, *big.NewInt(11))
	poly1 = append(poly1, *big.NewInt(12))
	poly2 = append(poly2, *big.NewInt(21))
	poly2 = append(poly2, *big.NewInt(22))

	polys1 := make(PolyMap)
	polys1[1] = PolyInfo{poly1, big.NewInt(1)}
	polys1[2] = PolyInfo{poly2, big.NewInt(2)}

	b, err := rlp.EncodeToBytes(&polys1)
	if err != nil {
		t.Error("rlp encode PolyMap fail, err:", err)
	}

	polys2 := make(PolyMap)
	err = rlp.DecodeBytes(b, &polys2)
	if err != nil && err != io.EOF {
		t.Error("rlp decode PolyMap fail, err:", err)
	}

	if len(polys1) != len(polys2) {
		t.Error("rlp decode PolyMap fail, invalid len")
	}

	for k, v := range polys1 {
		v2, ok := polys2[k]
		if !ok {
			t.Error("rlp decode PolyMap fail, invalid key")
		}

		if v.s.Cmp(v2.s) != 0 {
			t.Error("rlp decode PolyMap fail, invalid value")
		}

		if len(v.poly) != len(v2.poly) {
			t.Error("rlp decode PolyMap fail, invalid poly len")
		}

		for i := 0; i < len(v.poly); i++ {
			if v.poly[i].Cmp(&v2.poly[i]) != 0 {
				t.Error("rlp decode PolyMap fail, invalid poly value")
			}
		}
	}
}

func BenchmarkRandomBeacon_Stop(b *testing.B) {
	var (
		epocher epochLeader.Epocher
		key keystore.Key
		rb RandomBeacon
		wg sync.WaitGroup
	)

	var err error

	key.PrivateKey2, err = crypto.GenerateKey()
	if err != nil {
		b.Error("generate sec256 fail, ", err)
	}

	posconfig.Cfg().MinerKey = &key

	selfPrivate.D = posconfig.Cfg().GetMinerBn256SK()
	selfPrivate.G1 = posconfig.Cfg().GetMinerBn256PK()

	commityPrivate = selfPrivate


	rb.getRBProposerGroupF = tmpGetRBProposerGroup
	rb.getCji = tmpGetCji

	for i := 0; i < b.N; i++ {
		fmt.Println("benchmark loop once")
		rb.Init(&epocher)

		wg.Add(1)
		go callRBLoop(&rb, &wg)
		time.Sleep(time.Second*3)
		rb.Stop()
		wg.Wait()
	}
}