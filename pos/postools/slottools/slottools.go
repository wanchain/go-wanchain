package slottools

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
)

func GetStage1FunctionID(abiString string) ([4]byte, error) {
	var slotStage1ID [4]byte

	abi, err := GetAbi(abiString)
	if err != nil {
		return slotStage1ID, err
	}

	copy(slotStage1ID[:], abi.Methods["slotLeaderStage1MiSave"].Id())

	return slotStage1ID, nil
}

func GetStage2FunctionID(abiString string) ([4]byte, error) {
	var slotStage2ID [4]byte

	abi, err := GetAbi(abiString)
	if err != nil {
		return slotStage2ID, err
	}

	copy(slotStage2ID[:], abi.Methods["slotLeaderStage2InfoSave"].Id())

	return slotStage2ID, nil
}

func GetAbi(abiString string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(abiString))
}

func UnpackStage1Data(input []byte, abiString string) ([]byte, error) {
	return input[4:], nil
}

func UnpackStage2Data(input []byte, abiString string) ([]byte, error) {
	abi, err := GetAbi(abiString)
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
	id, err := GetStage1FunctionID(abiString)
	outBuf := make([]byte, len(id)+len(input))
	copy(outBuf[:4], id[:])
	copy(outBuf[4:], input[:])
	return outBuf, err
}

// PackStage1Data can pack stage1 data into abi []byte for tx payload
func PackStage2Data(input string, abiString string) ([]byte, error) {

	inputBytes, err := rlp.EncodeToBytes(input)

	if err != nil {
		return nil, err
	}
	abi, err := GetAbi(abiString)
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

type stage1Data struct {
	EpochID    uint64
	SelfIndex  uint64
	MiCompress []byte
}

// RlpPackStage1DataForTx
func RlpPackStage1DataForTx(epochID uint64, selfIndex uint64, mi *ecdsa.PublicKey, abiString string) ([]byte, error) {
	pkBuf, err := PkCompress(mi)
	if err != nil {
		return nil, err
	}
	data := &stage1Data{
		EpochID:    epochID,
		SelfIndex:  selfIndex,
		MiCompress: pkBuf,
	}

	buf, err := rlp.EncodeToBytes(data)
	if err != nil {
		return nil, err
	}

	return PackStage1Data(buf, abiString)
}

// RlpUnpackStage1DataForTx
func RlpUnpackStage1DataForTx(input []byte, abiString string) (epochID uint64, selfIndex uint64, mi *ecdsa.PublicKey, err error) {
	var data *stage1Data

	buf, _ := UnpackStage1Data(input, abiString)

	err = rlp.DecodeBytes(buf, &data)
	if err != nil {
		return
	}

	epochID = data.EpochID
	selfIndex = data.SelfIndex
	mi, err = PkUncompress(data.MiCompress)
	return
}

// RlpGetStage1IDFromTx
func RlpGetStage1IDFromTx(input []byte, abiString string) (epochIDBuf []byte, selfIndexBuf []byte, err error) {
	var data *stage1Data

	buf, _ := UnpackStage1Data(input, abiString)

	err = rlp.DecodeBytes(buf, &data)
	if err != nil {
		return
	}
	epochIDBuf = postools.Uint64ToBytes(data.EpochID)
	selfIndexBuf = postools.Uint64ToBytes(data.SelfIndex)
	return
}

// PkCompress
func PkCompress(pk *ecdsa.PublicKey) ([]byte, error) {
	if !crypto.S256().IsOnCurve(pk.X, pk.Y) {
		return nil, errors.New("Pk point is not on S256 curve")
	}
	pkBtc := btcec.PublicKey(*pk)
	return pkBtc.SerializeCompressed(), nil
}

// PkUncompress
func PkUncompress(buf []byte) (*ecdsa.PublicKey, error) {
	key, err := btcec.ParsePubKey(buf, btcec.S256())
	if err != nil {
		return nil, err
	}

	privK, _ := crypto.GenerateKey()
	pk := &privK.PublicKey
	pk.X = key.X
	pk.Y = key.Y
	return pk, nil
}

type stage2Data struct {
	EpochID   uint64
	SelfIndex uint64
	SelfPk    []byte
	AlphaPki  [][]byte
	Proof     []*big.Int
}

func RlpPackStage2DataForTx(epochID uint64, selfIndex uint64, selfPK *ecdsa.PublicKey, alphaPki []*ecdsa.PublicKey, proof []*big.Int, abiString string) ([]byte, error) {
	pk, err := PkCompress(selfPK)
	if err != nil {
		return nil, err
	}

	pks := make([][]byte, len(alphaPki))
	for i := 0; i < len(alphaPki); i++ {
		pks[i], err = PkCompress(alphaPki[i])
		if err != nil {
			return nil, err
		}
	}

	data := &stage2Data{
		EpochID:   epochID,
		SelfIndex: selfIndex,
		SelfPk:    pk,
		AlphaPki:  pks,
		Proof:     proof,
	}

	buf, err := rlp.EncodeToBytes(data)
	if err != nil {
		return nil, err
	}

	id, err := GetStage2FunctionID(abiString)
	if err != nil {
		return nil, err
	}

	outBuf := make([]byte, len(id)+len(buf))
	copy(outBuf[:4], id[:])
	copy(outBuf[4:], buf[:])

	return outBuf, nil
}

func RlpUnpackStage2DataForTx(input []byte, abiString string) (epochID uint64, selfIndex uint64, selfPK *ecdsa.PublicKey, alphaPki []*ecdsa.PublicKey, proof []*big.Int, err error) {
	inputBuf := input[4:]

	var data stage2Data
	err = rlp.DecodeBytes(inputBuf, &data)
	if err != nil {
		return
	}

	epochID = data.EpochID
	selfIndex = data.SelfIndex
	selfPK, err = PkUncompress(data.SelfPk)
	if err != nil {
		return
	}

	alphaPki = make([]*ecdsa.PublicKey, len(data.AlphaPki))
	for i := 0; i < len(data.AlphaPki); i++ {
		alphaPki[i], err = PkUncompress(data.AlphaPki[i])
		if err != nil {
			return
		}
	}

	proof = data.Proof
	return
}

func RlpGetStage2IDFromTx(input []byte, abiString string) (epochIDBuf []byte, selfIndexBuf []byte, err error) {
	inputBuf := input[4:]

	var data stage2Data
	err = rlp.DecodeBytes(inputBuf, &data)
	if err != nil {
		return
	}

	epochIDBuf = postools.Uint64ToBytes(data.EpochID)
	selfIndexBuf = postools.Uint64ToBytes(data.SelfIndex)
	return
}
