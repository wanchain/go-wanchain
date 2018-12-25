package slotleader

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	Rand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/accounts/keystore"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"

	"github.com/btcsuite/btcd/btcec"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/uleaderselection"
)

//CompressedPubKeyLen means a compressed public key byte len.
const CompressedPubKeyLen = 33
const LengthPublicKeyBytes = 65

const (
	// EpochGenesisTime is the pos start time such as: 2018-12-12 00:00:00 == 1544544000
	EpochGenesisTime = uint64(1544544000)

	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 10

	// SlotCount is slot count in an epoch
	SlotCount = 180

	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 1

	// SlotStage1 is 40% of slot count
	SlotStage1 = uint64(SlotCount * 0.4)
	// SlotStage2 is 80% of slot count
	SlotStage2       = int(SlotCount * 0.8)
	EpochLeaders     = "epochLeaders"
	SecurityMsg      = "securityMsg"
	RandFromProposer = "randFromProposer"
	RandomSeqs       = "randomSeqs"
	SlotLeader       = "slotLeader"
)

const (
	//Ready to start slot leader selection stage1
	slotLeaderSelectionStage1 = iota + 1 //1

	//Slot leader selection stage1 finish
	slotLeaderSelectionStage2 = iota + 1 //2
)

var (
	wanCscPrecompileAddr = common.BytesToAddress(big.NewInt(210).Bytes())
	ErrEpochID           = errors.New("EpochID is not valid")
)

//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	workingEpochID    uint64
	workStage         int
	rc                *rpc.Client
	epochLeadersArray []string            // len(pki)=65
	epochLeadersMap   map[string][]uint64 // key: pki value: []uint64 the indexs of this pki
	key               *keystore.Key

	slotLeadersPtrArray    [SlotCount]*ecdsa.PublicKey
	epochLeadersPtrArray   [EpochLeaderCount]*ecdsa.PublicKey
	validEpochLeadersIndex [EpochLeaderCount]bool // true: can be used to slot leader false: can not be used to slot leader

	stageOneMi       [EpochLeaderCount]*ecdsa.PublicKey
	stageTwoAlphaPKi [EpochLeaderCount][EpochLeaderCount]*ecdsa.PublicKey
	stageTwoProof    [EpochLeaderCount][EpochLeaderCount]*big.Int
}

var slotLeaderSelection *SlotLeaderSelection

func init() {
	slotLeaderSelection = &SlotLeaderSelection{}
	slotLeaderSelection.epochLeadersMap = make(map[string][]uint64)
	slotLeaderSelection.epochLeadersArray = make([]string, 0)
}

//GetSlotLeaderSelection get the SlotLeaderSelection's object
func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}

//--------------Workflow functions-------------------------------------------------------------

//Loop check work every 10 second. Called by backend loop
//It's all slotLeaderSelection's main workflow loop
//It's not loop at all, it is loop called by backend
func (s *SlotLeaderSelection) Loop(rc *rpc.Client, key *keystore.Key) {
	functrace.Enter("SlotLeaderSelection Loop")
	s.rc = rc
	s.key = key

	epochID, slotID, err := GetEpochSlotID()
	s.log("Now epchoID:" + posdb.Uint64ToString(epochID) + " slotID:" + posdb.Uint64ToString(slotID))

	workStage, err := s.getWorkStage(epochID)

	if err != nil {
		if err.Error() == "leveldb: not found" {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
			workStage = slotLeaderSelectionStage1
		} else {
			s.log("getWorkStage error: " + err.Error())
		}
	}

	switch workStage {
	case slotLeaderSelectionStage1:
		log.Debug("Enter slotLeaderSelectionStage1")
		s.generateSlotLeadsGroup(epochID)
		s.buildEpochLeaderGroup(epochID)
		s.setWorkingEpochID(epochID)
		err := s.startStage1Work()
		if err != nil {
			s.log(err.Error())
		} else {
			s.setWorkStage(epochID, slotLeaderSelectionStage2)
		}

	case slotLeaderSelectionStage2:
		log.Debug("Enter slotLeaderSelectionStage2")

		//If New epoch start
		s.workingEpochID, err = s.getWorkingEpochID()
		if epochID > s.workingEpochID {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
		}

		if slotID < SlotStage1 {
			break
		}

		err := s.startStage2Work()
		if err != nil {
			s.log(err.Error())
		}

	default:
	}
	functrace.Exit()
}

// startStage1Work start the stage 1 work and send tx
func (s *SlotLeaderSelection) startStage1Work() error {
	functrace.Enter("")
	selfPublicKey, _ := s.getLocalPublicKey()

	selfPublicKeyIndex, inEpochLeaders := s.epochLeadersMap[string(crypto.FromECDSAPub(selfPublicKey))]
	if inEpochLeaders {
		log.Debug(fmt.Sprintf("Local node in epoch leaders times: %d", len(selfPublicKeyIndex)))

		for i := 0; i < len(selfPublicKeyIndex); i++ {
			workingEpochID, err := s.getWorkingEpochID()
			if err != nil {
				return err
			}
			data, err := s.GenerateCommitment(selfPublicKey, workingEpochID, selfPublicKeyIndex[i])
			if err != nil {
				return err
			}

			err = s.sendStage1Tx(data)
			if err != nil {
				s.log(err.Error())
				return err
			}
		}
	} else {
		log.Debug("Local node is not in epoch leaders")
	}

	functrace.Exit()
	return nil
}

// startStage2Work start the stage 2 work and send tx
func (s *SlotLeaderSelection) startStage2Work() error {

	functrace.Enter("startStage2Work")
	s.getWorkingEpochID()
	selfPublicKey, _ := s.getLocalPublicKey()
	selfPublicKeyIndex := make([]uint64, 0)
	var inEpochLeaders bool
	selfPublicKeyIndex, inEpochLeaders = s.epochLeadersMap[string(crypto.FromECDSAPub(selfPublicKey))]
	if inEpochLeaders {
		for i := 0; i < len(selfPublicKeyIndex); i++ {
			workingEpochID, err := s.getWorkingEpochID()
			if err != nil {
				return err
			}
			data, err := s.buildStage2TxPayload(workingEpochID, uint64(selfPublicKeyIndex[i]))
			if err != nil {
				return err
			}

			err = s.sendStage2Tx(data)
			if err != nil {
				return err
			}
		}
	}
	functrace.Exit()
	return nil
}

//GenerateCommitment generate a commitment and send it by tx message
//Returns the commitment buffer []byte which is publicKey and alpha * publicKey
//payload should be send with tx.
func (s *SlotLeaderSelection) GenerateCommitment(publicKey *ecdsa.PublicKey,
	epochID uint64, selfIndexInEpochLeader uint64) ([]byte, error) {
	functrace.Enter()
	if publicKey == nil || publicKey.X == nil || publicKey.Y == nil {
		return nil, errors.New("Invalid input parameters")
	}

	if !crypto.S256().IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, errors.New("Public key point is not on S256 curve")
	}

	alpha, err := uleaderselection.RandFieldElement(Rand.Reader)
	if err != nil {
		return nil, err
	}
	fmt.Println("alpha:", hex.EncodeToString(alpha.Bytes()))

	commitment, err := uleaderselection.GenerateCommitment(publicKey, alpha)
	if err != nil {
		return nil, err
	}

	pk := btcec.PublicKey(*commitment[0])
	mi := btcec.PublicKey(*commitment[1])

	pkCompress := pk.SerializeCompressed()
	miCompress := mi.SerializeCompressed()
	epochIDBuf := big.NewInt(0).SetUint64(epochID).Bytes()
	selfIndexBuf := posdb.Uint64ToBytes(selfIndexInEpochLeader)

	log.Debug("epochIDBuf(hex): " + hex.EncodeToString(epochIDBuf))
	log.Debug("selfIndexBuf: " + hex.EncodeToString(selfIndexBuf))
	log.Debug("pkCompress: " + hex.EncodeToString(pkCompress))
	log.Debug("miCompress: " + hex.EncodeToString(miCompress))

	buffer, err := s.RlpPackCompressedPK(epochIDBuf, selfIndexBuf, pkCompress, miCompress)

	posdb.GetDb().PutWithIndex(epochID, selfIndexInEpochLeader, "alpha", alpha.Bytes())

	log.Debug(fmt.Sprintf("put alpha epochID:%d, selfIndex:%d, alpha:%s", epochID, selfIndexInEpochLeader, alpha.String()))

	functrace.Exit()
	return buffer, err
}

// RlpPackCompressedPK pack infomations into rlp []byte
func (s *SlotLeaderSelection) RlpPackCompressedPK(epochIDBuf []byte, selfIndexBuf []byte, pkCompress []byte, miCompress []byte) ([]byte, error) {
	return rlp.EncodeToBytes([][]byte{epochIDBuf, selfIndexBuf, pkCompress, miCompress})
}

// RlpUnpackCompressedPK can unpack from packed data get 4 params
func (s *SlotLeaderSelection) RlpUnpackWithCompressedPK(buf []byte) (epochIDBuf []byte, selfIndexBuf []byte, pkCompress []byte, miCompress []byte, err error) {
	var output [][]byte
	err = rlp.DecodeBytes(buf, &output)
	epochIDBuf = output[0]
	selfIndexBuf = output[1]
	pkCompress = output[2]
	miCompress = output[3]
	return
}

// RlpUnpackCompressedPK can unpack from packed data get 4 params and uncompress the pk
func (s *SlotLeaderSelection) RlpUnpackAndWithUncompressPK(buf []byte) (epochIDBuf []byte, selfIndexBuf []byte, pkUncompress []byte, miUncompress []byte, err error) {
	var output [][]byte
	err = rlp.DecodeBytes(buf, &output)
	epochIDBuf = output[0]
	selfIndexBuf = output[1]
	pk, err := btcec.ParsePubKey(output[2], btcec.S256())
	pkUncompress = pk.SerializeUncompressed()
	mi, err := btcec.ParsePubKey(output[2], btcec.S256())
	miUncompress = mi.SerializeUncompressed()
	return
}

func (s *SlotLeaderSelection) RlpUnpackStage2Data(buf []byte) (epochIDBuf string, selfIndexBuf string, pk string, alphaPki []string, proof []string, err error) {
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

//GetAlpha get alpha of epochID
func (s *SlotLeaderSelection) GetAlpha(epochID uint64, selfIndex uint64) (*big.Int, error) {
	buf, err := posdb.GetDb().GetWithIndex(epochID, selfIndex, "alpha")
	if err != nil {
		return nil, err
	}

	var alpha = big.NewInt(0).SetBytes(buf)
	return alpha, nil
}

// GetStage1Abi can get a abi instance of slot leader selection stage1
func (s *SlotLeaderSelection) GetStage1Abi() (abi.ABI, error) {
	slotDefinition :=
		`[
			{
				"constant": false,
				"type": "function",
				"inputs": [
					{
						"name": "data",
						"type": "string"
					}
				],
				"name": "slotLeaderStage1MiSave",
				"outputs": [
					{
						"name": "data",
						"type": "string"
					}
				]
			},
			{
				"constant": false,
				"type": "function",
				"inputs": [
					{
						"name": "data",
						"type": "string"
					}
				],
				"name": "slotLeaderStage2InfoSave",
				"outputs": [
					{
						"name": "data",
						"type": "string"
					}
				]
			}
		]`
	return abi.JSON(strings.NewReader(slotDefinition))
}

// PackStage1Data can pack stage1 data into abi []byte for tx payload
func (s *SlotLeaderSelection) PackStage1Data(input []byte) ([]byte, error) {
	abi, err := s.GetStage1Abi()
	if err != nil {
		return nil, err
	}
	data := hex.EncodeToString(input)
	return abi.Pack("slotLeaderStage1MiSave", data)
}

// PackStage1Data can pack stage1 data into abi []byte for tx payload
func (s *SlotLeaderSelection) PackStage2Data(input string) ([]byte, error) {

	inputBytes, err := rlp.EncodeToBytes(input)

	if err != nil {
		return nil, err
	}
	abi, err := s.GetStage1Abi()
	if err != nil {
		return nil, err
	}
	data := hex.EncodeToString(inputBytes)
	return abi.Pack("slotLeaderStage2InfoSave", data)
}

func (s *SlotLeaderSelection) UnpackStage2Data(input []byte) ([]byte, error) {
	abi, err := s.GetStage1Abi()
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

// UnpackStage1Data use to unpack payload
// NOTE: Must Use payload[:] for input
func (s *SlotLeaderSelection) UnpackStage1Data(input []byte) ([]byte, error) {
	abi, err := s.GetStage1Abi()
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

// GetStage1FunctionID get function id of slot select stage1 from abi
func (s *SlotLeaderSelection) GetStage1FunctionID() ([4]byte, error) {
	var slotStage1ID [4]byte

	abi, err := s.GetStage1Abi()
	if err != nil {
		return slotStage1ID, err
	}

	copy(slotStage1ID[:], abi.Methods["slotLeaderStage1MiSave"].Id())

	return slotStage1ID, nil
}

// GetFuncIDFromPayload get function id from payload data
func (s *SlotLeaderSelection) GetFuncIDFromPayload(payload []byte) ([4]byte, error) {
	var methodID [4]byte
	if len(payload) < 4 {
		return methodID, errors.New("input is too short")
	}

	copy(methodID[:], payload[:4])

	return methodID, nil
}

//---------------Information get/set functions--------------------------------------------

//getLocalPublicKey get local public key from memory keystore
func (s *SlotLeaderSelection) getLocalPublicKey() (*ecdsa.PublicKey, error) {
	functrace.Enter()
	if s.key == nil {
		// test
		ret, err := posdb.GetDb().Get(0, "testSelfPK")
		pk := crypto.ToECDSAPub(ret)

		log.Warn("do not found unlock address use a test address:", hex.EncodeToString(crypto.FromECDSAPub(pk)))
		functrace.Exit()
		return pk, err
	}

	log.Debug("get public key success pk: " + hex.EncodeToString(crypto.FromECDSAPub(&s.key.PrivateKey.PublicKey)))

	functrace.Exit()
	return &s.key.PrivateKey.PublicKey, nil
}

func (s *SlotLeaderSelection) getLocalPrivateKey() (*ecdsa.PrivateKey, error) {
	return s.key.PrivateKey, nil
}

// GetEpochSlotID get current epochID and slotID in this epoch by local time
// returns epochID, slotID, error
func GetEpochSlotID() (uint64, uint64, error) {
	epochTimespan := uint64(SlotTime * SlotCount)
	timeUnix := uint64(time.Now().Unix())

	if EpochGenesisTime > timeUnix {
		return 0, 0, errors.New("Epoch genesis time is not arrive")
	}

	epochID := uint64((timeUnix - EpochGenesisTime) / epochTimespan)

	epochIndex := uint64((timeUnix - EpochGenesisTime) / epochTimespan)

	epochStartTime := epochIndex*epochTimespan + EpochGenesisTime

	timeInEpoch := timeUnix - epochStartTime

	slotID := uint64(timeInEpoch / SlotTime)

	return epochID, slotID, nil
}

//getEpochLeaders get epochLeaders of epochID in StateDB
func (s *SlotLeaderSelection) getEpochLeaders(epochID uint64) [][]byte {

	//test: generate test publicKey
	epochLeaders := make([][]byte, EpochLeaderCount)
	for i := 0; i < EpochLeaderCount; i++ {
		key, _ := crypto.GenerateKey()
		epochLeaders[i] = crypto.FromECDSAPub(&key.PublicKey)
	}
	selfPk, err := s.getLocalPublicKey()
	if err == nil {
		epochLeaders[EpochLeaderCount-1] = crypto.FromECDSAPub(selfPk)
	}

	return epochLeaders
}

//getWorkStage get work stage of epochID from levelDB
func (s *SlotLeaderSelection) getWorkStage(epochID uint64) (int, error) {
	ret, err := posdb.GetDb().Get(epochID, "slotLeaderWorkStage")
	if err != nil {
		return 0, err
	}
	workStageUint64 := posdb.BytesToUint64(ret)
	return int(workStageUint64), err
}

//saveWorkStage save the work stage of epochID in levelDB
func (s *SlotLeaderSelection) setWorkStage(epochID uint64, workStage int) error {
	workStageBig := big.NewInt(int64(workStage))
	_, err := posdb.GetDb().Put(epochID, "slotLeaderWorkStage", workStageBig.Bytes())
	return err
}
func (s *SlotLeaderSelection) clearData() {
	// clear Array
	s.epochLeadersArray = make([]string, 0)
	// clear map
	s.epochLeadersMap = make(map[string][]uint64)

	for i := 0; i < EpochLeaderCount; i++ {
		s.epochLeadersPtrArray[i] = nil
		s.validEpochLeadersIndex[i] = true

		s.stageOneMi[i] = nil

		for j := 0; j < EpochLeaderCount; j++ {
			s.stageTwoAlphaPKi[i][j] = nil
			s.stageTwoProof[i][j] = nil
		}
	}

	for i := 0; i < SlotCount; i++ {
		s.slotLeadersPtrArray[i] = nil

	}
}
func (s *SlotLeaderSelection) buildEpochLeaderGroup(epochID uint64) error {
	s.clearData()
	// build Array and map
	for index, value := range s.getEpochLeaders(epochID) {
		//s.epochLeadersArray[index] = string(value)
		s.epochLeadersArray = append(s.epochLeadersArray, string(value))
		s.epochLeadersMap[string(value)] = append(s.epochLeadersMap[string(value)], uint64(index))

		s.epochLeadersPtrArray[index] = crypto.ToECDSAPub(value)
	}
	functrace.Exit()
	return nil
}

func (s *SlotLeaderSelection) GetSlotLeaders(epochID uint64) (slotLeaders []*ecdsa.PublicKey, err error) {
	if len(s.slotLeadersPtrArray) != SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	return s.slotLeadersPtrArray[:], nil
}

func (s *SlotLeaderSelection) GetSlotLeader(epochID uint64, slotID uint64) (slotLeader *ecdsa.PublicKey, err error) {
	if len(s.slotLeadersPtrArray) != SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	if slotID >= SlotCount {
		return nil, errors.New("slot id index out of range")
	}
	return s.slotLeadersPtrArray[slotID], nil
}

func (s *SlotLeaderSelection) generateSlotLeadsGroup(epochID uint64) error {
	err := s.buildEpochLeaderGroup(epochID)
	if err != nil {
		return errors.New("build epoch leader group error!")
	}
	// 1. get SMA[pre]
	piecesPtr := make([]*ecdsa.PublicKey, 0)
	// pieces: alpha[1]*G, alpha[2]*G, .....
	pieces, err := posdb.GetDb().Get(epochID-1, SecurityMsg)
	if err != nil {
		return err
	}
	piecesCount := len(pieces) / LengthPublicKeyBytes
	var pubKeyByte []byte
	for i := 0; i < piecesCount; i++ {
		if i < piecesCount-2 {
			pubKeyByte = pieces[i*LengthPublicKeyBytes : (i+1)*LengthPublicKeyBytes]
		} else {
			pubKeyByte = pieces[i*LengthPublicKeyBytes:]
		}
		piecesPtr = append(piecesPtr, crypto.ToECDSAPub(pubKeyByte))
	}
	// 2. get random
	rb, err := posdb.GetDb().Get(epochID-1, RandFromProposer)
	if err != nil {
		return err
	}

	// 5. return slot leaders pointers.
	slotLeadersPtr := make([]*ecdsa.PublicKey, 0)
	slotLeadersPtr, _, err = uleaderselection.GenerateSlotLeaderSeq(piecesPtr, s.epochLeadersPtrArray[:], rb, SlotCount)
	if err != nil {
		return err
	}
	// 6. insert slot address to local DB
	for index, val := range slotLeadersPtr {
		_, err = posdb.GetDb().PutWithIndex(epochID, uint64(index), SlotLeader, crypto.FromECDSAPub(val))
		s.epochLeadersPtrArray[index] = val
		if err != nil {
			return err
		}
	}
	// should send signal to miners and tell miners can work?
	return nil
}

func (s *SlotLeaderSelection) inEpochLeadersOrNot(pkIndex uint64, pkBytes []byte) bool {
	return (pkIndex < uint64(len(s.epochLeadersArray))) && (string(pkBytes) == s.epochLeadersArray[pkIndex])
}
func (s *SlotLeaderSelection) InEpochLeadersOrNotByPk(pkBytes []byte) bool {
	_, ok := s.epochLeadersMap[string(pkBytes)]
	return ok
}
func (s *SlotLeaderSelection) getStateDb() (stateDb StateDB, err error) {
	return nil, nil
}

func (s *SlotLeaderSelection) verifySecurityPiece(index uint64) (valid bool, err error) {
	// MI == AlphaiPki
	if !uleaderselection.PublicKeyEqual(s.stageOneMi[index], s.stageTwoAlphaPKi[index][index]) {
		return false, nil
	} else {
		// verify proof[index]
		return uleaderselection.VerifyDleqProof(s.epochLeadersPtrArray[:], s.stageTwoAlphaPKi[index][:], s.stageTwoProof[index][:]), nil
	}
}

// create alpha1*pki,alpha1*PKi,alphaN*PKi,...
// used to create security message.
func (s *SlotLeaderSelection) buildSecurityPieces(epochID uint64) (pieces []*ecdsa.PublicKey, err error) {

	selfPk, err := s.getLocalPublicKey()
	if err != nil {
		return nil, err
	}

	indexs, exist := s.epochLeadersMap[string(crypto.FromECDSAPub(selfPk))]
	if exist == false {
		return nil, errors.New("not in epoch leaders")
	}

	selfPkRecievedPicesMap := make(map[uint64][]*ecdsa.PublicKey, 0)
	for _, selfIndex := range indexs {
		for i := 0; i < len(s.epochLeadersArray); i++ {
			selfPkRecievedPicesMap[selfIndex] = append(selfPkRecievedPicesMap[selfIndex], s.stageTwoAlphaPKi[i][selfIndex])
		}
	}
	piece := make([]*ecdsa.PublicKey, 0)
	for _, value := range selfPkRecievedPicesMap {
		piece = value
		break
	}
	// the value in selfPkRecievedPicesMap should be same,so we can return the first one.
	return piece, nil
}

func (s *SlotLeaderSelection) getStage2TxAlphaPki(epochID uint64, selfIndex uint64) (alphaPkis []string, proofs []string, err error) {

	stateDb, err := s.getStateDb()

	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())

	var keyBuf bytes.Buffer
	keyBuf.Write([]byte(strconv.FormatUint(epochID, 10)))
	keyBuf.Write([]byte(strconv.FormatUint(selfIndex, 10)))
	keyBuf.Write([]byte("slotLeaderStag2"))
	keyHash := crypto.Keccak256Hash(keyBuf.Bytes())

	data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	data1, err := s.UnpackStage2Data(data)
	//epochIDBuf,selfIndexBuf,pki,alphaPki,proof,err := s.RlpUnpackStage2Data(data1)
	_, _, _, alphaPki, proof, err := s.RlpUnpackStage2Data(data1)
	if err != nil {
		return nil, nil, err
	}
	return alphaPki, proof, nil
}

func (s *SlotLeaderSelection) collectStagesData(epochID uint64) (err error) {
	for i := 0; i < EpochLeaderCount; i++ {
		_, mi, _ := s.getStg1StateDbInfo(epochID, uint64(i))
		if len(mi) == 0 {
			s.validEpochLeadersIndex[i] = false
		} else {
			s.stageOneMi[i] = crypto.ToECDSAPub(mi)
		}

		alphaPkis, proofs, _ := s.getStage2TxAlphaPki(epochID, uint64(i))
		if (len(alphaPkis) != EpochLeaderCount) || (len(proofs) != EpochLeaderCount) {
			s.validEpochLeadersIndex[i] = false
		} else {

			for j := 0; j < EpochLeaderCount; j++ {
				s.stageTwoAlphaPKi[i][j] = crypto.ToECDSAPub([]byte(alphaPkis[j]))
			}

			for j := 0; j < EpochLeaderCount; j++ {
				proof, err := strconv.ParseInt(proofs[j], 10, 64)
				if err != nil {
					return err
				}
				s.stageTwoProof[i][j] = big.NewInt(0).SetInt64(proof)
			}
		}

	}
	return nil
}

// create security message SMA and insert into localDB
func (s *SlotLeaderSelection) generateSecurityMsg(epochID uint64, PrivateKey *ecdsa.PrivateKey) error {
	// collect data
	err := s.collectStagesData(epochID)
	if err != nil {
		return errors.New("collect stage data error!")
	}
	// verify security pieces
	for i := 0; i < EpochLeaderCount; i++ {
		valid, _ := s.verifySecurityPiece(uint64(i))
		if !valid {
			s.validEpochLeadersIndex[i] = false
		}
	}

	// build security self pieces. alpha1*pki, alpha2*pk2, alpha3*pk3....
	ArrayPiece, err := s.buildSecurityPieces(epochID)
	if err != nil {
		return err
	}
	smasPtr := make([]*ecdsa.PublicKey, 0)
	var smasBytes bytes.Buffer
	smasPtr, err = uleaderselection.GenerateSMA(PrivateKey, ArrayPiece)
	if err != nil {
		return err
	}
	for _, value := range smasPtr {
		smasBytes.Write(crypto.FromECDSAPub(value))
	}
	_, err = posdb.GetDb().Put(epochID, SecurityMsg, smasBytes.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// used for stage2 payload
// stage2 tx payload 1(alpha * Pk1, alpha * Pk2, ..., alpha * Pkn)
// stage2 tx payload 2 proof pai[i]
// []*ecdsa : payload1 []*big.Int payload2

func (s *SlotLeaderSelection) buildArrayPiece(epochID uint64, selfIndex uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {

	// get alpha
	alpha, err := s.GetAlpha(epochID, selfIndex)
	if err != nil {
		return nil, nil, err
	}

	publicKeys := make([]*ecdsa.PublicKey, 0)
	for _, value := range s.epochLeadersArray {
		publicKeys = append(publicKeys, crypto.ToECDSAPub([]byte(value)))
	}

	_, ArrayPiece, proof, err := uleaderselection.GenerateArrayPiece(publicKeys, alpha)
	functrace.Exit()
	return ArrayPiece, proof, err
}
func (s *SlotLeaderSelection) buildStage2TxPayload(epochID uint64, selfIndex uint64) (string, error) {
	var epochIDStr, selfIndexStr, selfPKStr, alphaPkiStr, proofStr, payLoadStr string

	epochIDStr = strconv.FormatUint(epochID, 10)
	selfIndexStr = strconv.FormatUint(selfIndex, 10)

	selfPk, err := s.getLocalPublicKey()
	if err != nil {
		return "", err
	}
	selfPKStr = string(crypto.FromECDSAPub(selfPk))

	alphaPki, proof, err := s.buildArrayPiece(epochID, selfIndex)
	if err != nil {
		return "", err
	}

	var alphaBuffer bytes.Buffer
	for index, value := range alphaPki {
		if index < len(alphaPki)-1 {
			alphaBuffer.Write(crypto.FromECDSAPub(value))
			alphaBuffer.Write([]byte("-"))
		} else {
			alphaBuffer.Write(crypto.FromECDSAPub(value))
		}

	}
	alphaPkiStr = string(alphaBuffer.Bytes())

	var proofBuffer bytes.Buffer
	for index, valueProof := range proof {
		if index < len(proof)-1 {
			proofBuffer.Write(valueProof.Bytes())
			proofBuffer.Write([]byte("-"))
		} else {
			proofBuffer.Write(valueProof.Bytes())
		}

	}
	proofStr = string(proofBuffer.Bytes())

	payLoadStr = strings.Join([]string{epochIDStr, selfIndexStr, selfPKStr, alphaPkiStr, proofStr}, "+")
	return payLoadStr, nil
}

func (s *SlotLeaderSelection) setCurrentWorkStage(workStage int) {
	currentEpochID, _ := s.getWorkingEpochID()
	s.setWorkStage(currentEpochID, workStage)
}

func (s *SlotLeaderSelection) log(info string) {
	log.Debug(info)
	fmt.Println(info)
}

func (s *SlotLeaderSelection) getWorkingEpochID() (uint64, error) {
	ret, err := posdb.GetDb().Get(0, "slotLeaderCurrentSlotID")
	retUint64 := posdb.BytesToUint64(ret)
	return retUint64, err
}

func (s *SlotLeaderSelection) setWorkingEpochID(workingEpochID uint64) error {
	_, err := posdb.GetDb().Put(0, "slotLeaderCurrentSlotID", posdb.Uint64ToBytes(workingEpochID))
	return err
}

// getStg1StateDbInfo can get data from StateDB the pk and mi are in 65 bytes length uncompress format
func (s *SlotLeaderSelection) getStg1StateDbInfo(epochID uint64, index uint64) (pk []byte, mi []byte, err error) {
	stateDb, err := s.getStateDb()
	if err != nil {
		return nil, nil, err
	}
	// address : sc slotLeaderPrecompileAddr
	// key:      hash(epochID,selfIndex,"slotLeaderStag2")
	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())

	var keyBuf bytes.Buffer
	keyBuf.Write(posdb.Uint64ToBytes(epochID))
	keyBuf.Write(posdb.Uint64ToBytes(index))
	keyBuf.Write([]byte("slotLeaderStag1"))
	keyHash := crypto.Keccak256Hash(keyBuf.Bytes())

	// Read and Verify
	readBuf := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if readBuf == nil {
		return nil, nil, errors.New("getStg1StateDbInfo: Found not data of key")
	}

	//pk and mi is 65 bytes length
	epID, idxID, pk, mi, err := s.RlpUnpackAndWithUncompressPK(readBuf)
	if err != nil {
		return nil, nil, errors.New("getStg1StateDbInfo: RlpUnpackAndWithUncompressPK error")
	}

	if hex.EncodeToString(epID) == posdb.Uint64ToString(epochID) &&
		hex.EncodeToString(idxID) == posdb.Uint64ToString(index) &&
		err == nil {
		return
	}

	return nil, nil, errors.New("Stg1 data get from StateDb verified failed")
}

//--------------Transacton create / send --------------------------------------------

func (s *SlotLeaderSelection) sendStage1Tx(data []byte) error {
	//test
	fmt.Println("Ready to send StageTx1 tx:", hex.EncodeToString(data))

	if s.rc == nil {
		return errors.New("rc is not ready")
	}

	ctx := context.Background()
	rc := s.rc

	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())

	var to = slotLeaderPrecompileAddr
	amount := new(big.Int).SetInt64(0)
	//amount.SetString("100", 10) // 100 tokens

	//type SendTxArgs struct {
	//	From     common.Address  `json:"from"`
	//	To       *common.Address `json:"to"`
	//	Gas      *hexutil.Big    `json:"gas"`
	//	GasPrice *hexutil.Big    `json:"gasPrice"`
	//	Value    *hexutil.Big    `json:"value"`
	//	Data     hexutil.Bytes   `json:"data"`
	//	Nonce    *hexutil.Uint64 `json:"nonce"`
	//}
	arg := map[string]interface{}{}
	arg["from"] = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	arg["to"] = &to
	arg["value"] = (*hexutil.Big)(amount)
	arg["txType"] = 1
	//Set payload infomation--------------

	payload, err := s.PackStage1Data(data)
	if err != nil {
		log.Debug("PackStage1Data err:" + err.Error())
		return err
	}

	log.Debug("ready to write data of payload: " + "0x" + hexutil.Encode(payload))

	arg["data"] = hexutil.Bytes(payload)

	log.Debug("finish to write data of payload")

	var txHash common.Hash
	callErr := rc.CallContext(ctx, &txHash, "eth_sendTransaction", arg)
	if nil != callErr {
		fmt.Println(callErr)
		log.Error("tx send failed")
		return errors.New("tx send failed")
	}
	fmt.Println(txHash)
	log.Debug("tx send success")
	return nil
}
func (s *SlotLeaderSelection) sendStage2Tx(data string) error {
	//test
	fmt.Println("Simulator send tx:", data)

	if s.rc == nil {
		return errors.New("rc is not ready")
	}

	ctx := context.Background()
	rc := s.rc
	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())

	var to = slotLeaderPrecompileAddr
	arg := map[string]interface{}{}
	arg["from"] = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	arg["to"] = &to
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(200000))

	arg["txType"] = 1
	//Set payload infomation--------------

	payload, err := s.PackStage2Data(data)
	if err != nil {
		return err
	}

	log.Debug("ready to write data of payload: " + "0x" + hexutil.Encode(payload))
	arg["data"] = hexutil.Bytes(payload)
	log.Debug("finish to write data of payload")
	var txHash common.Hash
	callErr := rc.CallContext(ctx, &txHash, "eth_sendTransaction", arg)
	if nil != callErr {
		fmt.Println(callErr)
		log.Error("tx send failed:" + callErr.Error())
		return errors.New("tx send failed")
	}
	fmt.Println(txHash)
	log.Debug("tx send success")
	return nil
}
