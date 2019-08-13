package slotleader

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/uleaderselection"
	"github.com/wanchain/go-wanchain/pos/util/convert"

	"github.com/btcsuite/btcd/btcec"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
)

func TestSlotLeaderSelectionGetInstance(t *testing.T) {
	posdb.GetDb().DbInit("test")
	SlsInit()
	slot := GetSlotLeaderSelection()
	if slot == nil {
		t.Fail()
	}
}

func TestPublicKeyCompress(t *testing.T) {
	privKey, _ := crypto.GenerateKey()

	fmt.Println("Is on curve: ", crypto.S256().IsOnCurve(privKey.X, privKey.Y))

	fmt.Println("public key:", hex.EncodeToString(crypto.FromECDSAPub(&privKey.PublicKey)))

	pk := btcec.PublicKey(privKey.PublicKey)

	fmt.Println("public key uncompress:", hex.EncodeToString(pk.SerializeUncompressed()), "len: ", len(pk.SerializeUncompressed()))

	fmt.Println("public key compress:", hex.EncodeToString(pk.SerializeCompressed()), "len: ", len(pk.SerializeCompressed()))

	keyCompress := pk.SerializeCompressed()

	key, _ := btcec.ParsePubKey(keyCompress, btcec.S256())

	pKey := ecdsa.PublicKey(*key)

	fmt.Println("public key:", hex.EncodeToString(crypto.FromECDSAPub(&pKey)))
}

func TestRlpEncodeAndDecode(t *testing.T) {

	privKey, _ := crypto.GenerateKey()
	pk := btcec.PublicKey(privKey.PublicKey)
	keyCompress := pk.SerializeCompressed()

	var test = [][]byte{
		new(big.Int).SetInt64(1).Bytes(),
		keyCompress,
		keyCompress,
	}

	fmt.Println("before encode:", test)

	buf, err := rlp.EncodeToBytes(test)

	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("encode buf: ", hex.EncodeToString(buf))

	fmt.Println("encode len: ", len(buf))

	var output [][]byte
	rlp.DecodeBytes(buf, &output)

	fmt.Println("after decode:", output)
}

func TestAbiPack2(t *testing.T) {

}

// TestByteToString is test for bytes compare with string() convert
func TestByteToString(t *testing.T) {
	testBytes := make([]byte, 0)
	for i := 0; i < 255; i++ {
		testBytes = append(testBytes, byte(i))
	}
	fmt.Println("bytes: ", testBytes)
	fmt.Println("string: ", string(testBytes))
	fmt.Println("string len:", len(string(testBytes)))

	testBytes2 := make([]byte, 0)
	for i := 0; i < 255; i++ {
		testBytes2 = append(testBytes2, byte(i))
	}

	if string(testBytes) != string(testBytes2) {
		t.Fail()
	}
}

func TestNumToString(t *testing.T) {
	value, err := hex.DecodeString("0")
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(value)
}

func TestCompare(t *testing.T) {
	epID := []byte{84}
	epochID := uint64(84)
	idxID := []byte{1}
	index := uint64(1)

	fmt.Println(hex.EncodeToString(epID))
	fmt.Println(hex.EncodeToString(idxID))
	fmt.Println(hex.EncodeToString(convert.Uint64ToBytes(epochID)))
	fmt.Println(hex.EncodeToString(convert.Uint64ToBytes(index)))

	if hex.EncodeToString(epID) == hex.EncodeToString(convert.Uint64ToBytes(epochID)) &&
		hex.EncodeToString(idxID) == hex.EncodeToString(convert.Uint64ToBytes(index)) {
		return
	}

	t.Fail()
}

func TestProof(t *testing.T) {

	type Test struct {
		Proof    [][]byte
		ProofMeg [][]byte
	}

	key1, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()

	a := &Test{Proof: [][]byte{big.NewInt(999).Bytes(), big.NewInt(111).Bytes()}, ProofMeg: [][]byte{crypto.FromECDSAPub(&key1.PublicKey), crypto.FromECDSAPub(&key2.PublicKey)}}

	fmt.Println(a)

	buf, err := rlp.EncodeToBytes(a)
	if err != nil {
		t.Fail()
	}

	fmt.Println(hex.EncodeToString(buf))

	var b Test

	err = rlp.DecodeBytes(buf, &b)
	if err != nil {
		t.Fail()
	}

	fmt.Println(b)

}

func TestCRSave(t *testing.T) {

	fmt.Printf("hello world\n\n\n")

	info := ""

	i := 1
	info += fmt.Sprintf("hello world %d \n\n\n", i)

	fmt.Print(info)

	cr := make([]*big.Int, 100)
	for i := 0; i < 100; i++ {
		key, _ := crypto.GenerateKey()
		fmt.Println(hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey)))
		cr[i] = key.D
	}

	buf, err := rlp.EncodeToBytes(cr)
	if err != nil {
		t.Fail()
	}

	fmt.Println("buf len:", len(buf))

	var crOut []*big.Int
	err = rlp.DecodeBytes(buf, &crOut)
	if err != nil {
		t.Fail()
	}

	for i := 0; i < 100; i++ {
		if cr[i].String() != crOut[i].String() {
			t.Fail()
		}
	}

}

func TestArraySave(t *testing.T) {

	fmt.Printf("TestArraySave\n\n\n")
	var sendtrans [posconfig.EpochLeaderCount]bool
	for index := range sendtrans {
		sendtrans[index] = false
	}
	fmt.Println(sendtrans)

	sendtrans[0] = true
	sendtrans[posconfig.EpochLeaderCount-1] = true

	bytes, err := rlp.EncodeToBytes(sendtrans)
	if err != nil {
		t.Error(err.Error())
	}

	db := posdb.NewDb("testArraySave")
	db.Put(uint64(0), "TestArraySave", bytes)

	var sendtransGet [posconfig.EpochLeaderCount]bool
	bytesGet, err := db.Get(uint64(0), "TestArraySave")
	if err != nil {
		t.Error(err.Error())
	}
	err = rlp.DecodeBytes(bytesGet, &sendtransGet)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(sendtransGet)
	RmDB("testArraySave")
	os.RemoveAll("sl_leader_test")
}

func TestGetSMAPieces(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()

	// getSMAPieces
	pks, isGenesis, err := s.getSMAPieces(0)
	if err != nil {
		t.Error(err.Error())
	}
	if !isGenesis {
		t.Fail()
	}

	if len(pks) != posconfig.EpochLeaderCount {
		t.Fail()
	}

	// GetSma
	pks, isGenesis, err = s.GetSma(0)
	if err != nil {
		t.Error(err.Error())
	}
	if !isGenesis {
		t.Fail()
	}

	if len(pks) != posconfig.EpochLeaderCount {
		t.Fail()
	}
}

func TestDump(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	epochID := s.getWorkingEpochID()
	s.setWorkingEpochID(2)

	epochLeaderAllBytes := make([]byte, 65*posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKeyByes := crypto.FromECDSAPub(&prvKey.PublicKey)
		copy(epochLeaderAllBytes[i*65:], pubKeyByes[:])
	}
	posdb.GetDb().Put(2, EpochLeaders, epochLeaderAllBytes[:])
	posdb.GetDb().Put(1, EpochLeaders, epochLeaderAllBytes[:])

	posconfig.SelfTestMode = true
	go s.dumpData()

	s.setWorkingEpochID(epochID)
	posconfig.SelfTestMode = false
}

func TestClearData(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	epochID := s.getWorkingEpochID()
	s.setWorkingEpochID(2)

	epochLeaderAllBytes := make([]byte, 65*posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKeyByes := crypto.FromECDSAPub(&prvKey.PublicKey)
		copy(epochLeaderAllBytes[i*65:], pubKeyByes[:])
	}
	posdb.GetDb().Put(2, EpochLeaders, epochLeaderAllBytes[:])
	posdb.GetDb().Put(1, EpochLeaders, epochLeaderAllBytes[:])

	posconfig.SelfTestMode = true
	s.buildEpochLeaderGroup(1)

	go s.dumpData()
	s.clearData()
	go s.dumpData()

	s.setWorkingEpochID(epochID)
	posconfig.SelfTestMode = false
}

func TestGetSlotLeader(t *testing.T) {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	t.Log("Current dir path ", dir)
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
	posdb.GetDb().DbInit(path.Join(dir, "sl_leader_test"))

	SlsInit()
	s := GetSlotLeaderSelection()

	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		s.defaultSlotLeadersPtrArray[i] = &prvKey.PublicKey
	}

	_, err := s.GetSlotLeader(0, 1)
	if err != nil {
		t.Fail()
	}

	////ErrSlotLeaderGroupNotReady
	_, err = s.GetSlotLeader(4, posconfig.SlotCount-1)
	if err != nil {
		t.Fail()
	}

	os.RemoveAll(path.Join(dir, "sl_leader_test"))

}

func TestGetLocalPublicKey(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	s.key = &keystore.Key{}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	s.key.PrivateKey = key
	// GetLocalPublicKey
	keyGot, err := s.GetLocalPublicKey()
	if err != nil {
		t.Fail()
	}

	if !uleaderselection.PublicKeyEqual(keyGot, &s.key.PrivateKey.PublicKey) {
		t.Fail()
	}
	// getLocalPublicKey
	keyGot1, err := s.getLocalPublicKey()
	if err != nil {
		t.Fail()
	}

	if !uleaderselection.PublicKeyEqual(keyGot1, &s.key.PrivateKey.PublicKey) {
		t.Fail()
	}
	//getLocalPrivateKey
	prvKeyGot, err := s.getLocalPrivateKey()
	if err != nil {
		t.Fail()
	}

	if prvKeyGot != key {
		t.Fail()
	}

}

func TestGetSlotCreateStatusByEpochID(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()

	if s.GetSlotCreateStatusByEpochID(0) {
		t.Fail()
	}
	s.slotCreateStatus[0] = true
	if !s.GetSlotCreateStatusByEpochID(0) {
		t.Fail()
	}
}

func TestGetSlotLeaderSelection(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()

	if s == nil {
		t.Fail()
	}
}

func TestGetEpochLeaders(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	posconfig.SelfTestMode = true

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
	posdb.GetDb().DbInit(path.Join(dir, "sl_leader_test"))

	epochLeaderAllBytes := make([]byte, 65*posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKeyByes := crypto.FromECDSAPub(&prvKey.PublicKey)
		copy(epochLeaderAllBytes[i*65:], pubKeyByes[:])
	}
	posdb.GetDb().Put(2, EpochLeaders, epochLeaderAllBytes[:])
	posdb.GetDb().Put(1, EpochLeaders, epochLeaderAllBytes[:])

	// getEpochLeaders
	epochLeadersBytes := s.getEpochLeaders(uint64(1))
	if len(epochLeadersBytes) != posconfig.EpochLeaderCount {
		t.Fail()
	}

	// GetEpochLeadersPK
	epochLeadersPks := s.GetEpochLeadersPK(uint64(2))
	if len(epochLeadersPks) != posconfig.EpochLeaderCount {
		t.Errorf("GetEpochLeadersPK error!")
		t.Fail()
	}

	// getEpochLeadersPK
	epochLeadersPks1 := s.getEpochLeadersPK(uint64(2))
	if len(epochLeadersPks1) != posconfig.EpochLeaderCount {
		t.Errorf("getEpochLeadersPK error!")
		t.Fail()
	}
	//getPreEpochLeadersPK
	epochLeadersPks2 := s.getEpochLeadersPK(uint64(2))
	if len(epochLeadersPks2) != posconfig.EpochLeaderCount {
		t.Errorf("getPreEpochLeadersPK error!")
		t.Fail()
	}
	posconfig.SelfTestMode = false
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
}

func TestGetAlpha(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
	posdb.GetDb().DbInit(path.Join(dir, "sl_leader_test"))

	alpha := big.NewInt(0).SetUint64(uint64(^uint64(0)))
	posdb.GetDb().PutWithIndex(uint64(0), uint64(0), "alpha", alpha.Bytes())

	alphaGet, err := s.getAlpha(0, 0)
	if err != nil {
		t.Fail()
	}

	if alphaGet.Cmp(alpha) != 0 {
		t.Fail()
	}

	alpha = big.NewInt(0).SetUint64(0)
	posdb.GetDb().PutWithIndex(uint64(0), uint64(1), "alpha", alpha.Bytes())

	alphaGet, err = s.getAlpha(0, 1)
	if err != nil {
		t.Fail()
	}

	if alphaGet.Cmp(alpha) != 0 {
		t.Fail()
	}

	os.RemoveAll(path.Join(dir, "sl_leader_test"))
}

func TestIsLocalPKInPreEpochLeaders(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	s.key = &keystore.Key{}
	posconfig.SelfTestMode = true

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
	posdb.GetDb().DbInit(path.Join(dir, "sl_leader_test"))

	var prvKeyExist *ecdsa.PrivateKey
	epochLeaderAllBytes := make([]byte, 65*posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKeyByes := crypto.FromECDSAPub(&prvKey.PublicKey)
		copy(epochLeaderAllBytes[i*65:], pubKeyByes[:])

		if i == posconfig.EpochLeaderCount-1 {
			prvKeyExist = prvKey
		}

		// build epochLeadersMap
		s.epochLeadersMap[hex.EncodeToString(pubKeyByes)] = append(s.epochLeadersMap[hex.EncodeToString(pubKeyByes)],
			uint64(i))
	}

	posdb.GetDb().Put(4, EpochLeaders, epochLeaderAllBytes[:])
	posdb.GetDb().Put(3, EpochLeaders, epochLeaderAllBytes[:])

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	s.key.PrivateKey = key

	//isLocalPkInPreEpochLeaders
	pks, _ := s.GetPreEpochLeadersPK(4)
	inOrNot := s.IsLocalPkInEpochLeaders(pks)
	if inOrNot == true {
		t.Fail()
	}

	s.key.PrivateKey = prvKeyExist
	//isLocalPkInPreEpochLeaders
	pks, _ = s.GetPreEpochLeadersPK(4)
	inOrNot = s.IsLocalPkInEpochLeaders(pks)

	if inOrNot == false {
		t.Fail()
	}

	posconfig.SelfTestMode = false
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
}

func TestBuildEpochLeaderGroup(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	s.key = &keystore.Key{}

	posconfig.SelfTestMode = true

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
	posdb.GetDb().DbInit(path.Join(dir, "sl_leader_test"))

	epochLeaderAllBytes := make([]byte, 65*posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKeyByes := crypto.FromECDSAPub(&prvKey.PublicKey)
		copy(epochLeaderAllBytes[i*65:], pubKeyByes[:])
	}

	posdb.GetDb().Put(2, EpochLeaders, epochLeaderAllBytes[:])
	posdb.GetDb().Put(1, EpochLeaders, epochLeaderAllBytes[:])

	// buildEpochLeaderGroup
	s.buildEpochLeaderGroup(2)
	if len(s.epochLeadersArray) != posconfig.EpochLeaderCount {
		t.Errorf("len(s.epochLeadersArray) error!")
		t.Fail()
	}
	for _, value := range s.epochLeadersArray {
		if len(value) != 130 {
			t.Errorf("epochLeadersArray has not valid length")
			t.Fail()
		}
	}

	if len(s.epochLeadersMap) != posconfig.EpochLeaderCount {
		t.Errorf("len(s.epochLeadersMap) error!")
		t.Fail()
	}
	for _, value := range s.epochLeadersMap {
		if len(value) == 0 {
			t.Errorf("epochLeadersMap has no index")
			t.Fail()
		}
	}

	if len(s.epochLeadersPtrArray) != posconfig.EpochLeaderCount {
		t.Errorf("len(s.epochLeadersPtrArray) error!")
		t.Fail()
	}
	for _, value := range s.epochLeadersPtrArray {
		if value == nil {
			t.Errorf("epochLeadersPtrArray nil")
			t.Fail()
		}
	}

	posconfig.SelfTestMode = false
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
}

func TestGetRandom(t *testing.T) {
	var (
		db, _      = ethdb.NewMemDatabase()
		stateDb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	)

	if stateDb == nil {
		t.Fail()
	}
	SlsInit()
	s := GetSlotLeaderSelection()
	vmcfg := vm.Config{}
	gspec := core.DefaultPPOWTestingGenesisBlock()
	gspec.MustCommit(db)
	chain, err := core.NewBlockChain(db, gspec.Config, nil, vmcfg, nil)
	s.blockChain = chain

	if err != nil {
		t.Fail()
	}

	ret, err := s.getRandom(nil, 0)
	fmt.Printf("ret of get randome 0x%v\n", hex.EncodeToString(ret.Bytes()))
	RmDB("epochGendb")
}

func TestBuildStage2TxPayload(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	s.key = &keystore.Key{}
	posconfig.SelfTestMode = true

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
	posdb.GetDb().DbInit(path.Join(dir, "sl_leader_test"))

	epochLeaderAllBytes := make([]byte, 65*posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		prvKey, _ := crypto.GenerateKey()
		pubKeyByes := crypto.FromECDSAPub(&prvKey.PublicKey)
		copy(epochLeaderAllBytes[i*65:], pubKeyByes[:])
	}

	posdb.GetDb().Put(2, EpochLeaders, epochLeaderAllBytes[:])
	posdb.GetDb().Put(1, EpochLeaders, epochLeaderAllBytes[:])

	s.buildEpochLeaderGroup(2)
	// test below functions
	// buildArrayPiece
	// RlpPackStage2DataForTx
	// buildStage2TxPayload
	stage2TxBytes, err := s.buildStage2TxPayload(2, 0)
	if err != nil {
		t.Fail()
	}

	fmt.Printf("bytes of stage2TxBytes is %v\n", hex.EncodeToString(stage2TxBytes))
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
}

func TestBuildSecurityPieces(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	s.key = &keystore.Key{}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	s.key.PrivateKey = key

	// getLocalPublicKey
	keyGot1, err := s.getLocalPublicKey()
	if err != nil {
		t.Fail()
	}

	if !uleaderselection.PublicKeyEqual(keyGot1, &s.key.PrivateKey.PublicKey) {
		t.Fail()
	}

	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		s.validEpochLeadersIndex[i] = true
	}
	s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))] = []uint64{0}
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		alpha := big.NewInt(123)
		alphaPk := new(ecdsa.PublicKey)
		alphaPk.Curve = crypto.S256()
		alphaPk.X, alphaPk.Y = crypto.S256().ScalarMult(key.PublicKey.X, key.PublicKey.Y, alpha.Bytes())

		s.stageTwoAlphaPKi[i][0] = alphaPk
		// only used to len(s.epochLeadersPtrArray)
		//s.epochLeadersArray[i] = hex.EncodeToString(crypto.FromECDSAPub(alphaPk))
		s.epochLeadersArray = append(s.epochLeadersArray, hex.EncodeToString(crypto.FromECDSAPub(alphaPk)))
	}

	pieces, err := s.buildSecurityPieces(0)
	if err != nil {
		t.Fail()
	}

	if len(pieces) != posconfig.EpochLeaderCount {
		t.Fail()
	}

	for _, pk := range pieces {
		if !pk.IsOnCurve(pk.X, pk.Y) {
			t.Fail()
		}
	}
}

func TestGenerateSecurityMsg(t *testing.T) {
	// init
	var (
		db, _      = ethdb.NewMemDatabase()
		stateDb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	)

	if stateDb == nil {
		t.Fail()
	}
	SlsInit()
	s := GetSlotLeaderSelection()
	s.stateDbTest = stateDb
	// build block chain
	vmcfg := vm.Config{}
	//gspec := core.DefaultPPOWTestingGenesisBlock()
	gspec := core.DefaultGenesisBlock()
	gspec.MustCommit(db)
	_, err := core.NewBlockChain(db, gspec.Config, nil, vmcfg, nil)

	if err != nil {
		t.Fail()
	}

	// build local key
	s.key = &keystore.Key{}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}
	s.key.PrivateKey = key

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
	posdb.GetDb().DbInit(path.Join(dir, "sl_leader_test"))

	// build current epoch leaders s.epochLeadersMap
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		s.validEpochLeadersIndex[i] = true
	}
	s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(&key.PublicKey))] = []uint64{0}
	alpha := big.NewInt(123)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		alpha = alpha.Add(alpha, alpha)
		alphaPk := new(ecdsa.PublicKey)
		alphaPk.Curve = crypto.S256()
		alphaPk.X, alphaPk.Y = crypto.S256().ScalarMult(key.PublicKey.X, key.PublicKey.Y, alpha.Bytes())

		s.stageTwoAlphaPKi[i][0] = alphaPk
		// only used to len(s.epochLeadersPtrArray)
		//s.epochLeadersArray[i] = hex.EncodeToString(crypto.FromECDSAPub(alphaPk))
		s.epochLeadersArray = append(s.epochLeadersArray, hex.EncodeToString(crypto.FromECDSAPub(alphaPk)))
	}

	posconfig.SelfTestMode = true

	for i := 1; i < posconfig.EpochLeaderCount; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Fail()
		}

		s.epochLeadersPtrArray[i] = &key.PublicKey
	}

	s.epochLeadersPtrArray[0] = &key.PublicKey
	epochID := uint64(0)

	epochLeaderAllBytes := make([]byte, 65*posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		pubKeyByes := crypto.FromECDSAPub(s.epochLeadersPtrArray[i])
		copy(epochLeaderAllBytes[i*65:], pubKeyByes[:])
	}
	posdb.GetDb().Put(epochID, EpochLeaders, epochLeaderAllBytes[:])

	// build stg2 trans and input into state db
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		txPayLoadByte, err := s.buildStage2TxPayload(epochID, uint64(i))
		if err != nil {
			t.Fail()
		}

		epochIDBuf, selfIndexBuf, _ := vm.RlpGetStage2IDFromTx(txPayLoadByte)
		keyHash := vm.GetSlotLeaderStage2KeyHash(epochIDBuf, selfIndexBuf)
		stateDb.SetStateByteArray(vm.GetSlotLeaderSCAddress(),
			keyHash,
			txPayLoadByte)

	}

	indexKeyHash := vm.GetSlotLeaderStage2IndexesKeyHash(convert.Uint64ToBytes(epochID))
	var sendtrans [posconfig.EpochLeaderCount]bool
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		sendtrans[i] = true
	}

	indexBytes, _ := rlp.EncodeToBytes(sendtrans)
	stateDb.SetStateByteArray(vm.GetSlotLeaderSCAddress(),
		indexKeyHash,
		indexBytes)

	stateDb1, _ := s.getCurrentStateDb()
	indexBytesGot := stateDb1.GetStateByteArray(vm.GetSlotLeaderSCAddress(), indexKeyHash)
	if len(indexBytes) != len(indexBytesGot) {
		t.Fail()
	}
	// build security pieces
	//pieces,_:= s.buildSecurityPieces(epochID)
	// create SMA
	err = s.generateSecurityMsg(epochID, key)
	if err != nil {
		t.Logf("generate security message error. err:%v \n", err.Error())
		t.Fail()
	}

	// check SMA from local db
	smaPieces, _, _ := s.getSMAPieces(uint64(epochID + 1))
	for _, smaValue := range smaPieces {
		fmt.Println(hex.EncodeToString(crypto.FromECDSAPub(smaValue)))
	}
	if len(smaPieces) != posconfig.EpochLeaderCount {
		t.Fail()
	}
	// un init
	RmDB("epochGendb")
	os.RemoveAll(path.Join(dir, "sl_leader_test"))
}
