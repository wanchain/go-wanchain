package slotleader

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"math/big"
	"testing"
)


// TestLoop use to test main loop
func TestLoop(t *testing.T) {
	posdb.GetDb().DbInit("test")
	GetSlotLeaderSelection().Loop(nil, nil, nil, nil)
	GetSlotLeaderSelection().Loop(nil, nil, nil, nil)

	GetSlotLeaderSelection().setCurrentWorkStage(slotLeaderSelectionStage1)

	GetSlotLeaderSelection().Loop(nil, nil, nil, nil)
	GetSlotLeaderSelection().Loop(nil, nil, nil, nil)

	GetSlotLeaderSelection().setWorkingEpochID(1)

	GetSlotLeaderSelection().Loop(nil, nil, nil, nil)

}

func TestGetEpochSlotID(t *testing.T) {
	epochID, slotID, err := GetEpochSlotID()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("epochID:", epochID, " slotID:", slotID)
	}
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
	payload, err := slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
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

	_, err = slot.GenerateCommitment(nil, epochID, 0)
	if err == nil {
		t.Fail()
	}

	// _, err = slot.GenerateCommitment(&privKey.PublicKey, 0)
	// if err == nil {
	// 	t.Fail()
	// }

	privKey.PublicKey.X = nil
	privKey.PublicKey.Y = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey.PublicKey.Curve = nil
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
	if err == nil {
		t.Fail()
	}

	privKey, err = crypto.GenerateKey()
	privKey2, _ := crypto.GenerateKey()

	privKey.X = privKey2.X
	_, err = slot.GenerateCommitment(&privKey.PublicKey, epochID, 0)
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

func TestAbiPack(t *testing.T) {

	s := GetSlotLeaderSelection()

	data := []byte("Each winter, npm, Inc. circulates a survey of software developers and npm users to solicit your ")

	payload, err := s.PackStage1Data(data)
	if err != nil {
		t.Fail()
	}
	id1, _ := s.GetStage1FunctionID()
	id2, _ := s.GetFuncIDFromPayload(payload)
	if id1 != id2 {
		t.Fail()
	}

	unpack, err := s.UnpackStage1Data(payload)
	if err != nil {
		t.Fail()
	}

	if hex.EncodeToString(unpack) != hex.EncodeToString(data) {
		t.Fail()
	}
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


func TestCompare(t *testing.T) {
	epID := []byte{84}
	epochID := uint64(84)
	idxID := []byte{1}
	index := uint64(1)

	fmt.Println(hex.EncodeToString(epID))
	fmt.Println(hex.EncodeToString(idxID))
	fmt.Println(hex.EncodeToString(big.NewInt(0).SetUint64(epochID).Bytes()))
	fmt.Println(hex.EncodeToString(big.NewInt(0).SetUint64(index).Bytes()))

	if hex.EncodeToString(epID) == hex.EncodeToString(big.NewInt(0).SetUint64(epochID).Bytes()) &&
		hex.EncodeToString(idxID) == hex.EncodeToString(big.NewInt(0).SetUint64(index).Bytes()) {
		return
	}

	t.Fail()
}

func TestWholeFlow(t *testing.T) {
	fmt.Println("\n=============whole flow==============================================")
	// 1. build N PK for epoch leader and insert into DB
	// 1.1 input genesis epoch leader group
	PrivateKeys := make([]*ecdsa.PrivateKey, 0)
	for i := 0; i < EpochLeaderCount; i++ {
		privateksample, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		PrivateKeys = append(PrivateKeys, privateksample)
	}

	var epochLeadersBuffer bytes.Buffer
	for _,value := range PrivateKeys {
		epochLeadersBuffer.Write(crypto.FromECDSAPub(&value.PublicKey))
	}
	// insert EpochLeader groups in local db, and the epoch ID = 0(means epoch 0 used this group, this group is genesis group)
	posdb.GetDb().Put(uint64(0),EpochLeaders,epochLeadersBuffer.Bytes())

	bytesGeted,err := posdb.GetDb().Get(uint64(0),EpochLeaders)
	if err!=nil {
		fmt.Println(err.Error())
		t.Fail()
	}

	fmt.Println("Created by test  epochLeadersBuffer %%%%%%%%%%%%%%%")
	fmt.Printf("%v\n\n",epochLeadersBuffer.Bytes())

	fmt.Println("Read from local DB epochLeadersBuffer%%%%%%%%%%%%%%%")
	fmt.Printf("%v\n\n",bytesGeted)

	// 1.2 input genesis Security Message set security msg equals epochLeaders.
	posdb.GetDb().Put(uint64(0),SecurityMsg,epochLeadersBuffer.Bytes())

	// 1.3 get random by call GetRandom.

	var selfPublicKey	 	*ecdsa.PublicKey
	var selfPrivateKey 		*ecdsa.PrivateKey
	selfPublicKey = &(PrivateKeys[0].PublicKey)
	selfPrivateKey = PrivateKeys[0]

	s := GetSlotLeaderSelection()

	s.dumpData()

	var epochID uint64
	epochID = uint64(0)
	err = s.generateSlotLeadsGroup(epochID)
	if err != nil {
		fmt.Println(err.Error())
		t.Fail()
	}

	s.dumpData()

	fmt.Printf("Local private len = %d key: %v\n",len(crypto.FromECDSA(selfPrivateKey)),hex.EncodeToString(crypto.FromECDSA(selfPrivateKey)))
	fmt.Printf("Local public len =%d key: %v\n", len(crypto.FromECDSAPub(selfPublicKey)),hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey)))

	// scan the generated
	for index, value := range s.epochLeadersPtrArray{
		fmt.Printf("\tindex := %d, %v\t\n",index,hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

	fmt.Println("\t===================Generated slot leaders========================================")
	// scan the generated
	for index, value := range s.slotLeadersPtrArray{
		fmt.Printf("\tslotindex := %d, indexInEpoch=%d, %v\t\n",
			index,
			s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(value))][0],
			hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	// 2. read slot index from db
	//fmt.Println("\t===================Read slot leaders from local db========================================")
	//for i:=0; i<SlotCount; i++ {
	//	oneSlotBytes,err:=posdb.GetDb().GetWithIndex(uint64(epochID+1),uint64(i),SlotLeader)
	//	if err != nil {
	//		fmt.Println(err.Error())
	//		t.Fail()
	//	}
	//
	//	fmt.Printf("\tEpochID:=%d slotID:=%d,slotLeader:=%v\n",
	//		epochID+1,
	//		i,
	//		hex.EncodeToString(oneSlotBytes))
	//}

	// 6. simulate all epochleders and send tx1
	// tx1 transaction sent and tx1 value is inserted into local DB

	//for index,value := range s.epochLeadersPtrArray {
	//	//epochLeadersBuffer.Write(crypto.FromECDSAPub(&value.PublicKey))
	//	data, err := s.GenerateCommitment(value,epochID,uint64(index))
	//	if err!= nil {
	//		fmt.Println(err.Error())
	//		t.Fail()
	//	}
	//
	//	err = s.sendStage1Tx(data)
	//	if err!= nil {
	//		fmt.Println(err.Error())
	//		t.Fail()
	//	}
	//}

	// 7. simulate all epochleders and send tx2
	//for index,_:= range s.epochLeadersPtrArray {
	//
	//	data, err := s.buildStage2TxPayload(epochID, uint64(index))
	//	fmt.Printf("\ndata from buildStag32TxPayload is %v\n", data)
	//
	//	if err!= nil {
	//		fmt.Println(err.Error())
	//		t.Fail()
	//	}
	//
	//	//err = s.sendStage2Tx(data)
	//	//if err!= nil {
	//	//	fmt.Println(err.Error())
	//	//	t.Fail()
	//	//}
	//}

	// 8. collect all trans

	// 9. generate SMA
	// 10. insert SMA into DB
	//testBytes := make([]byte, 0)
	//for i := 0; i < 255; i++ {
	//	testBytes = append(testBytes, byte(i))
	//}
	//fmt.Println("bytes: ", testBytes)
	//fmt.Println("string: ", string(testBytes))
	//fmt.Println("string len:", len(string(testBytes)))
	//
	//testBytes2 := make([]byte, 0)
	//for i := 0; i < 255; i++ {
	//	testBytes2 = append(testBytes2, byte(i))
	//}
	//
	//if string(testBytes) != string(testBytes2) {
	//	t.Fail()
	//}
}
