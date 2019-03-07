package slotleader

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/pos"

	"github.com/btcsuite/btcd/btcec"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
)

// TestLoop use to test main loop
func TestLoop(t *testing.T) {
	pos.SelfTestMode = true
	posdb.GetDb().DbInit("test")
	GetSlotLeaderSelection().Loop(nil, nil, nil, 0, 0)
	GetSlotLeaderSelection().Loop(nil, nil, nil, 0, 0)

	GetSlotLeaderSelection().setCurrentWorkStage(slotLeaderSelectionStage1)

	GetSlotLeaderSelection().Loop(nil, nil, nil, 0, 0)
	GetSlotLeaderSelection().Loop(nil, nil, nil, 0, 0)

	GetSlotLeaderSelection().setWorkingEpochID(1)

	GetSlotLeaderSelection().Loop(nil, nil, nil, 0, 0)

}

func TestGetEpochSlotID(t *testing.T) {
	epochID, slotID := GetEpochSlotID()
	fmt.Println("epochID:", epochID, " slotID:", slotID)
}

func TestSlotLeaderSelectionGetInstance(t *testing.T) {
	posdb.GetDb().DbInit("test")
	slot := GetSlotLeaderSelection()
	if slot == nil {
		t.Fail()
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

	alpha, err := slot.GetAlpha(epochID, 0)
	if alpha == nil || err != nil {
		t.Fail()
	}

	var output [][]byte
	rlp.DecodeBytes(payload, &output)

	if hex.EncodeToString(pkCompress.SerializeCompressed()) != hex.EncodeToString(output[2]) {
		t.Fail()
	}

	fmt.Println("epochID: ", hex.EncodeToString(output[0]))
	fmt.Println("selfIndex: ", hex.EncodeToString(output[1]))

	fmt.Println("payload 0: ", hex.EncodeToString(output[2]))
	fmt.Println("payload 1: ", hex.EncodeToString(output[3]))
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
	fmt.Println(hex.EncodeToString(posdb.Uint64ToBytes(epochID)))
	fmt.Println(hex.EncodeToString(posdb.Uint64ToBytes(index)))

	if hex.EncodeToString(epID) == hex.EncodeToString(posdb.Uint64ToBytes(epochID)) &&
		hex.EncodeToString(idxID) == hex.EncodeToString(posdb.Uint64ToBytes(index)) {
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
	var sendtrans [pos.EpochLeaderCount]bool
	for index, _ := range sendtrans {
		sendtrans[index] = false
	}
	fmt.Println(sendtrans)

	sendtrans[0] = true
	sendtrans[pos.EpochLeaderCount-1] = true

	bytes, err := rlp.EncodeToBytes(sendtrans)
	if err != nil {
		t.Error(err.Error())
	}

	db := posdb.NewDb("testArraySave")
	db.Put(uint64(0), "TestArraySave", bytes)

	var sendtransGet [pos.EpochLeaderCount]bool
	bytesGet, err := db.Get(uint64(0), "TestArraySave")
	if err != nil {
		t.Error(err.Error())
	}
	err = rlp.DecodeBytes(bytesGet, &sendtransGet)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(sendtransGet)
}

// func TestWholeFlow(t *testing.T) {
// 	pos.SelfTestMode = true

// 	fmt.Println("\n=============whole flow==============================================")
// 	// 1. build N PK for epoch leader and insert into DB
// 	// 1.1 input genesis epoch leader group
// 	PrivateKeys := make([]*ecdsa.PrivateKey, 0)
// 	for i := 0; i < pos.EpochLeaderCount; i++ {
// 		privateksample, err := crypto.GenerateKey()
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		PrivateKeys = append(PrivateKeys, privateksample)
// 	}

// 	var epochLeadersBuffer bytes.Buffer
// 	for _, value := range PrivateKeys {
// 		epochLeadersBuffer.Write(crypto.FromECDSAPub(&value.PublicKey))
// 	}
// 	// insert EpochLeader groups in local db, and the epoch ID = 0(means epoch 0 used this group, this group is genesis group)
// 	posdb.GetDb().Put(uint64(0), EpochLeaders, epochLeadersBuffer.Bytes())

// 	bytesGeted, err := posdb.GetDb().Get(uint64(0), EpochLeaders)
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		t.Fail()
// 	}

// 	fmt.Println("Created by test  epochLeadersBuffer %%%%%%%%%%%%%%%")
// 	fmt.Printf("%v\n\n", epochLeadersBuffer.Bytes())

// 	fmt.Println("Read from local DB epochLeadersBuffer%%%%%%%%%%%%%%%")
// 	fmt.Printf("%v\n\n", bytesGeted)

// 	// 1.2 input genesis Security Message set security msg equals epochLeaders.
// 	posdb.GetDb().Put(uint64(0), SecurityMsg, epochLeadersBuffer.Bytes())

// 	// 1.3 get random by call GetRandom.

// 	var selfPublicKey *ecdsa.PublicKey
// 	var selfPrivateKey *ecdsa.PrivateKey
// 	selfPublicKey = &(PrivateKeys[0].PublicKey)
// 	selfPrivateKey = PrivateKeys[0]

// 	s := GetSlotLeaderSelection()

// 	s.dumpData()

// 	var epochID uint64
// 	epochID = uint64(0)
// 	err = s.generateSlotLeadsGroup(epochID)
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		t.Fail()
// 	}

// 	s.dumpData()

// 	fmt.Printf("Local private len = %d key: %v\n", len(crypto.FromECDSA(selfPrivateKey)), hex.EncodeToString(crypto.FromECDSA(selfPrivateKey)))
// 	fmt.Printf("Local public len =%d key: %v\n", len(crypto.FromECDSAPub(selfPublicKey)), hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey)))

// 	// scan the generated
// 	for index, value := range s.epochLeadersPtrArray {
// 		fmt.Printf("\tindex := %d, %v\t\n", index, hex.EncodeToString(crypto.FromECDSAPub(value)))
// 	}

// 	fmt.Println("\t===================Generated slot leaders========================================")
// 	// scan the generated
// 	for index, value := range s.slotLeadersPtrArray {
// 		fmt.Printf("\tslotindex := %d, indexInEpoch=%d, %v\t\n",
// 			index,
// 			s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(value))][0],
// 			hex.EncodeToString(crypto.FromECDSAPub(value)))
// 	}
// 	// 2. read slot index from db
// 	fmt.Println("\t===================Read slot leaders from local db========================================")
// 	for i := 0; i < pos.SlotCount; i++ {
// 		oneSlotBytes, err := posdb.GetDb().GetWithIndex(uint64(epochID+1), uint64(i), SlotLeader)
// 		if err != nil {
// 			fmt.Println(err.Error())
// 			t.Fail()
// 		}

// 		fmt.Printf("\tEpochID:=%d slotID:=%d,slotLeader:=%v\n",
// 			epochID+1,
// 			i,
// 			hex.EncodeToString(oneSlotBytes))
// 	}

// 	// 7. simulate all epochleders and send tx2
// 	for index, _ := range s.epochLeadersPtrArray {
// 		// encode

// 		data, err := s.buildStage2TxPayload(epochID, uint64(index))
// 		fmt.Printf("\ndata from buildStag32TxPayload is %v\n", data)

// 		if err != nil {
// 			fmt.Println(err.Error())
// 			t.Fail()
// 		}
// 		payload, err := slottools.PackStage2Data(data, vm.GetSlotLeaderScAbiString())
// 		if err != nil {
// 			t.Fail()
// 		}

// 		fmt.Printf("\nAfter PackStage2Data %v\n", payload)
// 		// decode

// 		unpackedData, err := slottools.UnpackStage2Data(payload[4:], vm.GetSlotLeaderScAbiString())
// 		fmt.Printf("\n unpackedData= %v\n", (unpackedData))

// 		epochIDBuf, selfIndexBuf, pki, alphaPki, proof, err := slottools.RlpUnpackStage2Data(unpackedData)

// 		epochIDBufDec, err := hex.DecodeString(epochIDBuf)
// 		epochID, err := strconv.ParseInt(string(epochIDBufDec), 10, 64)

// 		fmt.Printf("\n epochIDBufDec= %v\n", (epochID))

// 		selfIndexBufDec, err := hex.DecodeString(selfIndexBuf)
// 		selfIndex, err := strconv.ParseInt(string(selfIndexBufDec), 10, 64)
// 		fmt.Printf("\n selfIndexBufDec= %v\n", (selfIndex))

// 		pkiDec, err := hex.DecodeString(pki)
// 		fmt.Printf("\n pkiDec= %v\n", (pkiDec))

// 		for _, value := range alphaPki {
// 			alphaPkiDec, err := hex.DecodeString(value)
// 			if err != nil {
// 				fmt.Println(err.Error())
// 			}
// 			fmt.Printf("\n alphaPki= %v\n", (alphaPkiDec))
// 		}

// 		for _, value := range proof {
// 			proofDec, err := hex.DecodeString(value)
// 			if err != nil {
// 				fmt.Println(err.Error())
// 			}
// 			fmt.Printf("\n proof= %v\n", (proofDec))
// 		}

// 		break
// 	}

// 	// 8. collect all trans

// 	// 9. generate SMA
// 	// 10. insert SMA into DB
// 	//testBytes := make([]byte, 0)
// 	//for i := 0; i < 255; i++ {
// 	//	testBytes = append(testBytes, byte(i))
// 	//}
// 	//fmt.Println("bytes: ", testBytes)
// 	//fmt.Println("string: ", string(testBytes))
// 	//fmt.Println("string len:", len(string(testBytes)))
// 	//
// 	//testBytes2 := make([]byte, 0)
// 	//for i := 0; i < 255; i++ {
// 	//	testBytes2 = append(testBytes2, byte(i))
// 	//}
// 	//
// 	//if string(testBytes) != string(testBytes2) {
// 	//	t.Fail()
// 	//}
// }
