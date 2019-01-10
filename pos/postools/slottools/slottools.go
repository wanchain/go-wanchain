package slottools

import (
	"encoding/hex"
	"errors"
	"github.com/wanchain/go-wanchain/log"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
)

func GetStage1FunctionID(abiString string) ([4]byte, error) {
	var slotStage1ID [4]byte

	abi, err := GetStage1Abi(abiString)
	if err != nil {
		return slotStage1ID, err
	}

	copy(slotStage1ID[:], abi.Methods["slotLeaderStage1MiSave"].Id())

	return slotStage1ID, nil
}

func GetStage1Abi(abiString string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(abiString))
}

func UnpackStage1Data(input []byte, abiString string) ([]byte, error) {
	abi, err := GetStage1Abi(abiString)
	if err != nil {
		return nil, err
	}
	var data string
	err = abi.Unpack(&data, "slotLeaderStage1MiSave", input[4:])
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(data)
}

func UnpackStage2Data(input []byte, abiString string) ([]byte, error) {
	abi, err := GetStage1Abi(abiString)
	if err != nil {
		return nil, err
	}
	var data string
	err = abi.Unpack(&data, "slotLeaderStage2InfoSave", input)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(data)
}

// RlpPackCompressedPK pack infomations into rlp []byte
func RlpPackCompressedPK(epochIDBuf []byte, selfIndexBuf []byte, pkCompress []byte, miCompress []byte) ([]byte, error) {
	return rlp.EncodeToBytes([][]byte{epochIDBuf, selfIndexBuf, pkCompress, miCompress})
}

// RlpUnpackCompressedPK can unpack from packed data get 4 params
func RlpUnpackWithCompressedPK(buf []byte) (epochIDBuf []byte, selfIndexBuf []byte, pkCompress []byte, miCompress []byte, err error) {
	var output [][]byte
	err = rlp.DecodeBytes(buf, &output)
	epochIDBuf = output[0]
	selfIndexBuf = output[1]
	pkCompress = output[2]
	miCompress = output[3]
	return
}

// RlpUnpackCompressedPK can unpack from packed data get 4 params and uncompress the pk
func RlpUnpackAndWithUncompressPK(buf []byte) (epochIDBuf []byte, selfIndexBuf []byte, pkUncompress []byte, miUncompress []byte, err error) {
	var output [][]byte
	err = rlp.DecodeBytes(buf, &output)
	epochIDBuf = output[0]
	selfIndexBuf = output[1]
	pk, err := btcec.ParsePubKey(output[2], btcec.S256())
	pkUncompress = pk.SerializeUncompressed()
	mi, err := btcec.ParsePubKey(output[3], btcec.S256())
	miUncompress = mi.SerializeUncompressed()
	return
}

func RlpUnpackStage2Data(buf []byte) (epochIDBuf string, selfIndexBuf string, pk string, alphaPki []string, proof []string, err error) {
	if buf == nil {
		return "", "", "", nil, nil, errors.New("RlpUnpackStage2Data Input buf is nil")
	}

	var strAll string
	var strResult []string
	err = rlp.DecodeBytes(buf, &strAll)
	strResult = strings.Split(strAll, "+")

	epochIDBuf = strResult[0]
	selfIndexBuf = strResult[1]
	pk = strResult[2]

	alphaPki = strings.Split(strResult[3], "-")
	proof = strings.Split(strResult[4], "-")

	return
}

// PackStage1Data can pack stage1 data into abi []byte for tx payload
func PackStage1Data(input []byte, abiString string) ([]byte, error) {
	abi, err := GetStage1Abi(abiString)
	if err != nil {
		return nil, err
	}
	data := hex.EncodeToString(input)
	return abi.Pack("slotLeaderStage1MiSave", data)
}

// PackStage1Data can pack stage1 data into abi []byte for tx payload
func PackStage2Data(input string, abiString string) ([]byte, error) {

	inputBytes, err := rlp.EncodeToBytes(input)

	if err != nil {
		return nil, err
	}
	abi, err := GetStage1Abi(abiString)
	if err != nil {
		return nil, err
	}
	data := hex.EncodeToString(inputBytes)
	return abi.Pack("slotLeaderStage2InfoSave", data)
}

// GetFuncIDFromPayload get function id from payload data
func GetFuncIDFromPayload(payload []byte) ([4]byte, error) {
	var methodID [4]byte
	if len(payload) < 4 {
		return methodID, errors.New("input is too short")
	}

	copy(methodID[:], payload[:4])

	return methodID, nil
}

// InEpochLeadersOrNotByPk can verify the tx sender
func InEpochLeadersOrNotByPk(epochID uint64, pkBytes []byte) bool {
	ok := false
	epochLeaders := posdb.GetEpocherInst().GetEpochLeaders(epochID)
	if len(epochLeaders) != pos.EpochLeaderCount {
		log.Warn("epoch leader is not ready use epoch 0 at InEpochLeadersOrNotByPk", "epochID", epochID)
		epochLeaders = posdb.GetEpocherInst().GetEpochLeaders(0)
	}

	for i := 0; i < len(epochLeaders); i++ {
		if hex.EncodeToString(pkBytes) == hex.EncodeToString(epochLeaders[i]) {
			ok = true
			break
		}
	}
	return ok
}

// InPreEpochLeadersOrNotByPk can verify the tx sender in pre epochLeaders
//func InPreEpochLeadersOrNotByPk(epochID uint64, pkBytes []byte) bool {
//	if epochID == 0 {
//		return true
//	}
//
//	epochLeaders := posdb.GetEpocherInst().GetEpochLeaders(epochID - 1)
//	if len(epochLeaders) != pos.EpochLeaderCount {
//		epochLeaders = posdb.GetEpocherInst().GetEpochLeaders(0)
//	}
//
//	for i := 0; i < len(epochLeaders); i++ {
//		if hex.EncodeToString(pkBytes) == hex.EncodeToString(epochLeaders[i]) {
//			return true
//		}
//	}
//	return false
//}
