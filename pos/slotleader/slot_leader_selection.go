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
	"github.com/wanchain/go-wanchain/core/state"

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

var (
	EpochBaseTime = uint64(0)
)

const (
	// EpochGenesisTime is the pos start time such as: 2018-12-12 00:00:00 == 1544544000
	EpochGenesisTime = uint64(1544544000)

	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 10

	// SlotCount is slot count in an epoch
	SlotCount = 10

	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 6

	StageTwoProofCount = 2

	// SlotStage1 is 40% of slot count
	SlotStage1 = uint64(SlotCount * 4 / 10)
	// SlotStage2 is 80% of slot count
	SlotStage2       = uint64(SlotCount * 8 / 10)
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

	slotLeaderSelectionStage3 = iota + 1 //2

	slotLeaderSelectionStageFinished = iota + 1 //2

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
	epochLeadersArray []string            // len(pki)=65 hex.EncodeToString
	epochLeadersMap   map[string][]uint64 // key: pki value: []uint64 the indexs of this pki. hex.EncodeToString
	key               *keystore.Key
	stateDb           *state.StateDB
	epochInstance     interface{}

	slotLeadersPtrArray    [SlotCount]*ecdsa.PublicKey
	epochLeadersPtrArray   [EpochLeaderCount]*ecdsa.PublicKey
	validEpochLeadersIndex [EpochLeaderCount]bool // true: can be used to slot leader false: can not be used to slot leader

	stageOneMi       [EpochLeaderCount]*ecdsa.PublicKey
	stageTwoAlphaPKi [EpochLeaderCount][EpochLeaderCount]*ecdsa.PublicKey
	stageTwoProof    [EpochLeaderCount][StageTwoProofCount]*big.Int //[0]: e; [1]:Z
	slotCreated      bool

	testOrNot bool
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
func (s *SlotLeaderSelection) Loop(stateDb *state.StateDB, rc *rpc.Client, key *keystore.Key, epochInstance interface{}, epochID uint64, slotID uint64) {
	functrace.Enter("SlotLeaderSelection Loop")
	s.rc = rc
	s.key = key
	s.stateDb = stateDb
	s.epochInstance = epochInstance

	//epochID, slotID, err := GetEpochSlotID()
	log.Debug("Now epchoID and slotID:", "epochID", posdb.Uint64ToString(epochID), "slotID", posdb.Uint64ToString(slotID))

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
		//s.generateSlotLeadsGroup(epochID)

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

		s.buildEpochLeaderGroup(epochID)

		// err := s.startStage2Work()
		// if err != nil {
		// 	s.log(err.Error())
		// } else {
		// 	s.setWorkStage(epochID, slotLeaderSelectionStage3)
		// }

	case slotLeaderSelectionStage3:
		log.Debug("Enter slotLeaderSelectionStage3")

		//If New epoch start
		s.workingEpochID, err = s.getWorkingEpochID()
		if epochID > s.workingEpochID {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
		}

		if slotID < SlotStage2 {
			break
		}

		err := s.generateSecurityMsg(epochID, s.key.PrivateKey)
		if err != nil {
			s.log(err.Error())
		} else {
			s.setWorkStage(epochID, slotLeaderSelectionStageFinished)
		}
	case slotLeaderSelectionStageFinished:
		log.Debug("Enter slotLeaderSelectionStageFinished")

		//If New epoch start
		s.workingEpochID, err = s.getWorkingEpochID()
		if epochID > s.workingEpochID {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
		}
	default:
	}
	functrace.Exit()
}

// startStage1Work start the stage 1 work and send tx
func (s *SlotLeaderSelection) startStage1Work() error {
	functrace.Enter("")
	selfPublicKey, _ := s.getLocalPublicKey()

	selfPublicKeyIndex, inEpochLeaders := s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey))]
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
	selfPublicKeyIndex, inEpochLeaders = s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey))]
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

//GetAlpha get alpha of epochID
func (s *SlotLeaderSelection) GetAlpha(epochID uint64, selfIndex uint64) (*big.Int, error) {
	if s.testOrNot {
		ret := big.NewInt(123)
		return ret, nil
	}
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
	if s.key == nil {
		return nil, errors.New("getLocalPublicKey error, do not found unlock address")
	}
	return &s.key.PrivateKey.PublicKey, nil
}

func (s *SlotLeaderSelection) GetLocalPublicKey() (*ecdsa.PublicKey, error) {
	return s.getLocalPublicKey()
}

func (s *SlotLeaderSelection) getLocalPrivateKey() (*ecdsa.PrivateKey, error) {
	return s.key.PrivateKey, nil
}

// GetEpochSlotID get current epochID and slotID in this epoch by local time
// returns epochID, slotID, error
func GetEpochSlotID() (uint64, uint64, error) {
	if EpochBaseTime == 0 {
		return 0, 0, nil
	}

	epochTimespan := uint64(SlotTime * SlotCount)
	timeUnix := uint64(time.Now().Unix())

	if EpochGenesisTime > timeUnix {
		return 0, 0, errors.New("Epoch genesis time is not arrive")
	}

	epochID := uint64((timeUnix - EpochBaseTime) / epochTimespan)

	epochIndex := uint64((timeUnix - EpochBaseTime) / epochTimespan)

	epochStartTime := epochIndex*epochTimespan + EpochBaseTime

	timeInEpoch := timeUnix - epochStartTime

	slotID := uint64(timeInEpoch / SlotTime)

	return epochID, slotID, nil
}

//getEpochLeaders get epochLeaders of epochID in StateDB
func (s *SlotLeaderSelection) getEpochLeaders(epochID uint64) [][]byte {
	//test := false
	test := false
	if test {
		//test: generate test publicKey
		epochLeaderAllBytes, err := posdb.GetDb().Get(epochID, EpochLeaders)
		if err != nil {
			return nil
		}
		piecesCount := len(epochLeaderAllBytes) / LengthPublicKeyBytes
		ret := make([][]byte, 0)
		var pubKeyByte []byte
		for i := 0; i < piecesCount; i++ {
			if i < piecesCount-1 {
				pubKeyByte = epochLeaderAllBytes[i*LengthPublicKeyBytes : (i+1)*LengthPublicKeyBytes]
			} else {
				pubKeyByte = epochLeaderAllBytes[i*LengthPublicKeyBytes:]
			}
			ret = append(ret, pubKeyByte)
		}
		return ret
	} else {
		type epoch interface {
			GetEpochLeaders(epochID uint64) [][]byte
		}

		epochLeaders := s.epochInstance.(epoch).GetEpochLeaders(epochID)

		if epochLeaders != nil {
			log.Debug(fmt.Sprintf("getEpochLeaders called return len(epochLeaders):%d", len(epochLeaders)))
		}
		return epochLeaders
	}
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
	s.testOrNot = false
	s.slotCreated = false
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
		}
		for k := 0; k < StageTwoProofCount; k++ {
			s.stageTwoProof[i][k] = nil
		}
	}

	for i := 0; i < SlotCount; i++ {
		s.slotLeadersPtrArray[i] = nil

	}
}

func (s *SlotLeaderSelection) dumpData() {

	fmt.Printf("~~~~~~~~~~~dumpData begin~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
	fmt.Printf("\t\t\ts.slotCreated = %v\n\n", s.slotCreated)
	fmt.Printf("\t\t\ts.epochLeadersArray= %v\n\n", s.epochLeadersArray)
	fmt.Printf("\t\t\ts.epochLeadersMap= %v\n\n", s.epochLeadersMap)
	fmt.Printf("\t\t\ts.epochLeadersPtrArray= %v\n\n", s.epochLeadersPtrArray)
	fmt.Printf("\t\t\ts.validEpochLeadersIndex= %v\n\n", s.validEpochLeadersIndex)

	fmt.Printf("\t\t\ts.stageOneMi= %v\n\n", s.stageOneMi)
	fmt.Printf("\t\t\ts.stageTwoAlphaPKi= %v\n\n", s.stageTwoAlphaPKi)
	fmt.Printf("\t\t\ts.stageTwoProof= %v\n\n", s.stageTwoProof)
	fmt.Printf("\t\t\ts.slotLeadersPtrArray= %v\n\n", s.slotLeadersPtrArray)
	for index, value := range s.epochLeadersPtrArray {
		fmt.Printf("\tindex := %d, %v\t\n", index, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	fmt.Printf("~~~~~~~~~~~dumpData end~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
}

func (s *SlotLeaderSelection) buildEpochLeaderGroup(epochID uint64) error {
	functrace.Enter()
	s.clearData()
	// build Array and map

	for index, value := range s.getEpochLeaders(epochID) {
		//fmt.Printf("\n:::::buildEpochLeaderGroup\n")
		//fmt.Printf("\n:::::value is %v\n",value)
		//fmt.Printf("\n:::::hex.EncodeToString(value) is %v\n",hex.EncodeToString(value))
		s.epochLeadersArray = append(s.epochLeadersArray, hex.EncodeToString(value))
		s.epochLeadersMap[hex.EncodeToString(value)] = append(s.epochLeadersMap[hex.EncodeToString(value)], uint64(index))
		//fmt.Printf("\n:::::s.epochLeadersPtrArray[%d], crypto.ToECDSAPub(value) is %v\n",index,crypto.ToECDSAPub(value))
		s.epochLeadersPtrArray[index] = crypto.ToECDSAPub(value)
		//fmt.Printf("\n:::::s.epochLeadersPtrArray[%d], hex.EncodeToString is %v\n",index,
		//	hex.EncodeToString(crypto.FromECDSAPub(s.epochLeadersPtrArray[index])))
	}
	functrace.Exit()
	return nil
}

func (s *SlotLeaderSelection) GetSlotLeaders(epochID uint64) (slotLeaders []*ecdsa.PublicKey, err error) {
	if !s.slotCreated {
		//return nil, errors.New("slot leaders group not ready")
		log.Debug("slot leaders group not ready use a fake one")
		fakeSlotLeaders := make([]*ecdsa.PublicKey, 0)
		for i := 0; i < SlotCount; i++ {
			fakeSlotLeaders = append(fakeSlotLeaders, s.epochLeadersPtrArray[i%10])
		}
		return fakeSlotLeaders, nil
	}

	if len(s.slotLeadersPtrArray) != SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	return s.slotLeadersPtrArray[:], nil
}

func (s *SlotLeaderSelection) GetSlotLeader(epochID uint64, slotID uint64) (slotLeader *ecdsa.PublicKey, err error) {
	if !s.slotCreated {
		//return nil, errors.New("slot leaders group not ready")
		log.Debug("slot leaders group not ready use a fake one")
		return s.epochLeadersPtrArray[slotID%10], nil
	}
	if len(s.slotLeadersPtrArray) != SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	if slotID >= SlotCount {
		return nil, errors.New("slot id index out of range")
	}
	return s.slotLeadersPtrArray[slotID], nil
}

// from random proposer
func (s *SlotLeaderSelection) getRandom(epochID uint64) (ret *big.Int, err error) {
	ret = big.NewInt(0).SetUint64(uint64(13456789092))
	return ret, nil

	//randomByte,err := posdb.GetDb().Get(epochID, vm.RANDOMBEACON_DB_KEY)
	//if err != nil {
	//	return nil, err
	//}
	//ret = big.NewInt(0).SetBytes(randomByte)
	//return ret, nil
}

// from random proposer
func (s *SlotLeaderSelection) getSMAPieces(epochID uint64) (ret []*ecdsa.PublicKey, err error) {
	// 1. get SMA[pre]
	piecesPtr := make([]*ecdsa.PublicKey, 0)
	if epochID == uint64(0) {
		//genesis SMA
		smaGenesis := []string{"0491bcdcc06d2ba82a90877228268e4f7f1235ebbfc696a19ffb85590f30df9d3fa50ec030759a89b8bdab3aa3a844283a18c6567cd5451aab5c1df866b81a0c1c",
			"0491bcdcc06d2ba82a90877228268e4f7f1235ebbfc696a19ffb85590f30df9d3fa50ec030759a89b8bdab3aa3a844283a18c6567cd5451aab5c1df866b81a0c1c",
			"0466eea24039a99afba154d4c64ee7a8cf120a54d979ed55f793ce6cc2fc3635a5b79c4de545cb1ed493521a1ce6da95770a71d207121b01d26df88f6ee86d0b22",
			"04d8e72df643e3af9bd3e2f92eb5a8d5b24ac9833c1ccb5607f13f9b33649bf10c8e9a671e1c886e453333684f6ac01b008d1629f93a85a4f216a13b5f1862a8c1",
			"04a9f7bf968fee493b51b045827488a35731f2d7af063b51d31a42ab43b51468e36678dbbcf77410345d0540cbe65e348af242251b0f2ab10580ea5bca91839305",
			"04461b6877fc12569520a994505fbd60d8c58ef3b7fd3984efc0a2494bbc8f68f4364da4013495f9b29128553b874d6442cbac51fc51cfed39851867737ab4a02e",
			"04ab3825a803d2bf43335c5cd211de170e06c8a359aafbb49136e1b8a61e7a7f42a8b9cc30c0ad8f501e76a2750d676444f578b432d2d35ff24e858049e3eea14c",
			"049e4c0f311b5e3593d9ee09720abd62c23002a7b40da13d2483c85936d560f384ff7e85bcf0332ec53f2be84273e341dfdd4ad63f4f1e219ea7ab2b28b0dd8621",
			"0400994bbd33ad79912478e26bd92331468da9b69d9ac2ef1c80f4597acf463552d341508b1adb40e4b69d76361bd80230f9fdee2414db8bbdfdcb83e23e1d6ff7",
			"0444f09890a83ba77cbee7d432be95c850780362bc6f1e8f0198dabf1164a67a3d084e10ae3e1caf29f2638945c73867e456edddfc5c6fd5cb0e7e7c7558be003d",
			"04b274ff6d60c8d2a752887dd79ddd42d11bd363eee3e98ae3781872a2716cd8fdf7c24e8a910fe16633862a0e256b43552eac03f5fe2971d76d0a66bb43927af8"}
		for index, _ := range s.epochLeadersPtrArray {
			smaGenesisByte, err := hex.DecodeString(smaGenesis[index])
			if err != nil {
				return nil, err
			}
			piecesPtr = append(piecesPtr, crypto.ToECDSAPub(smaGenesisByte))
		}
		return piecesPtr, nil

	} else {
		// pieces: alpha[1]*G, alpha[2]*G, .....
		pieces, err := posdb.GetDb().Get(epochID, SecurityMsg)
		fmt.Printf("getSMAPieces: get from db, pieces is = %v\n", pieces)
		fmt.Printf("getSMAPieces: get from db, hex.encodingToString(pieces) is = %v\n", hex.EncodeToString(pieces))
		if err != nil {
			return nil, err
		}
		piecesCount := len(pieces) / LengthPublicKeyBytes
		var pubKeyByte []byte
		for i := 0; i < piecesCount; i++ {
			if i < piecesCount-1 {
				pubKeyByte = pieces[i*LengthPublicKeyBytes : (i+1)*LengthPublicKeyBytes]
			} else {
				pubKeyByte = pieces[i*LengthPublicKeyBytes:]
			}
			fmt.Printf("getSMAPieces: one hex.EncodeToString(piece) is = %v\n\n", hex.EncodeToString(pubKeyByte))
			piecesPtr = append(piecesPtr, crypto.ToECDSAPub(pubKeyByte))
		}
		return piecesPtr, nil
	}
}
func (s *SlotLeaderSelection) generateSlotLeadsGroup(epochID uint64) error {
	err := s.buildEpochLeaderGroup(epochID)
	if err != nil {
		return errors.New("build epoch leader group error!")
	}

	piecesPtr, err := s.getSMAPieces(epochID)
	if err != nil {
		return errors.New("get securiy message error!")
	}
	// 2. get random
	random, err := s.getRandom(epochID)
	if err != nil {
		return errors.New("get random message error!")
	}

	// 5. return slot leaders pointers.
	slotLeadersPtr := make([]*ecdsa.PublicKey, 0)
	fmt.Printf("len(piecesPtr)=%v\n", len(piecesPtr))
	fmt.Printf("len(epochLeadersPtrArray)=%v\n", len(s.epochLeadersPtrArray))
	fmt.Printf("len(random.Bytes)=%v\n", len(random.Bytes()))
	fmt.Printf("SlotCount= %d\n", SlotCount)
	//fmt.Printf("===========================before GenerateSlotLeaderSeq\n")
	//s.dumpData()
	epochLeadersPtrArray := s.epochLeadersPtrArray
	for i := 0; i < 10; i++ {
		ret := crypto.S256().IsOnCurve(s.epochLeadersPtrArray[i].X, s.epochLeadersPtrArray[i].Y)
		if !ret {
			log.Error("not")
		}
		ret = crypto.S256().IsOnCurve(epochLeadersPtrArray[i].X, epochLeadersPtrArray[i].Y)
		if !ret {
			log.Error("not")
		}
	}

	//slotLeadersPtr, _, err = uleaderselection.GenerateSlotLeaderSeq(piecesPtr[:], epochLeadersPtrArray[:], random.Bytes(), SlotCount)
	slotLeadersPtr, _, err = uleaderselection.GenerateSlotLeaderSeq(s.epochLeadersPtrArray[:], epochLeadersPtrArray[:], random.Bytes(), SlotCount)

	//fmt.Printf("===========================after GenerateSlotLeaderSeq\n")
	//s.dumpData()

	if err != nil {
		return err
	}
	// 6. insert slot address to local DB
	for index, val := range slotLeadersPtr {
		_, err = posdb.GetDb().PutWithIndex(uint64(epochID+1), uint64(index), SlotLeader, crypto.FromECDSAPub(val))
		//s.epochLeadersPtrArray[index] = val
		s.slotLeadersPtrArray[index] = val
		if err != nil {
			return err
		}
	}
	s.slotCreated = true
	return nil
}

func (s *SlotLeaderSelection) inEpochLeadersOrNot(pkIndex uint64, pkBytes []byte) bool {
	return (pkIndex < uint64(len(s.epochLeadersArray))) && (hex.EncodeToString(pkBytes) == s.epochLeadersArray[pkIndex])
}
func (s *SlotLeaderSelection) InEpochLeadersOrNotByPk(pkBytes []byte) bool {
	_, ok := s.epochLeadersMap[hex.EncodeToString(pkBytes)]
	return ok
}
func (s *SlotLeaderSelection) getStateDb() (stateDb *state.StateDB, err error) {
	if s.stateDb == nil {
		return nil, errors.New("Do not have stateDb instance now")
	}
	return s.stateDb, nil
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

	indexs, exist := s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(selfPk))]
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
	epochIDBuf := strconv.FormatUint(epochID, 10)
	epochIDBufDec, err := hex.DecodeString(epochIDBuf)
	if err != nil {
		return nil, nil, err
	}
	keyBuf.Write(epochIDBufDec)

	selfIndexStr := strconv.FormatUint(selfIndex, 10)
	selfIndexBufDec, err := hex.DecodeString(selfIndexStr)
	if err != nil {
		return nil, nil, err
	}
	keyBuf.Write(selfIndexBufDec)

	keyBuf.Write([]byte("slotLeaderStag2"))
	keyHash := crypto.Keccak256Hash(keyBuf.Bytes())

	data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if data == nil {
		return nil, nil, errors.New("can not find from statedb:" + fmt.Sprintf("addr:%s, key:%s, epochID:%d, selfIndex:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), epochID, selfIndex))
	}
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

		alphaPkis, proofs, err := s.getStage2TxAlphaPki(epochID, uint64(i))
		if err != nil {
			continue
		}

		if (len(alphaPkis) != EpochLeaderCount) || (len(proofs) != StageTwoProofCount) {
			s.validEpochLeadersIndex[i] = false
		} else {

			for j := 0; j < EpochLeaderCount; j++ {
				//s.stageTwoAlphaPKi[i][j] = crypto.ToECDSAPub([]byte(alphaPkis[j]))
				alphaPkiDecodeBytes, err := hex.DecodeString(alphaPkis[j])
				if err != nil {
					return err
				}
				s.stageTwoAlphaPKi[i][j] = crypto.ToECDSAPub(alphaPkiDecodeBytes)
			}

			for j := 0; j < StageTwoProofCount; j++ {
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
	_, err = posdb.GetDb().Put(uint64(epochID+1), SecurityMsg, smasBytes.Bytes())
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
	publicKeys = s.epochLeadersPtrArray[:]
	_, ArrayPiece, proof, err := uleaderselection.GenerateArrayPiece(publicKeys, alpha)
	functrace.Exit()
	return ArrayPiece, proof, err
}
func (s *SlotLeaderSelection) buildStage2TxPayload(epochID uint64, selfIndex uint64) (string, error) {
	var epochIDStr, selfIndexStr, selfPKHexStr, payLoadStr string
	var alphaPkiHexStr, proofHexStr []string

	alphaPkiHexStr = make([]string, 0)
	proofHexStr = make([]string, 0)

	epochIDStr = strconv.FormatUint(epochID, 10)
	epochIDHexStr := hex.EncodeToString([]byte(epochIDStr))

	selfIndexStr = strconv.FormatUint(selfIndex, 10)
	selfIndexHexStr := hex.EncodeToString([]byte(selfIndexStr))

	var selfPk *ecdsa.PublicKey
	var err error
	if s.testOrNot {
		selfPk = s.epochLeadersPtrArray[selfIndex]
	} else {
		selfPk, err = s.getLocalPublicKey()
		if err != nil {
			return "", err
		}
	}

	selfPKHexStr = hex.EncodeToString(crypto.FromECDSAPub(selfPk))

	alphaPki, proof, err := s.buildArrayPiece(epochID, selfIndex)
	if err != nil {
		return "", err
	}

	for _, value := range alphaPki {
		alphaPkiHexStr = append(alphaPkiHexStr, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	alphaPkiHexStrAll := strings.Join(alphaPkiHexStr, "-")

	for _, valueProof := range proof {
		proofHexStr = append(proofHexStr, hex.EncodeToString(valueProof.Bytes()))
	}
	proofHexStrAll := strings.Join(proofHexStr, "-")

	payLoadStr = strings.Join([]string{epochIDHexStr, selfIndexHexStr, selfPKHexStr, alphaPkiHexStrAll, proofHexStrAll}, "+")
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

	if hex.EncodeToString(epID) == hex.EncodeToString(big.NewInt(0).SetUint64(epochID).Bytes()) &&
		hex.EncodeToString(idxID) == hex.EncodeToString(big.NewInt(0).SetUint64(index).Bytes()) &&
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
	arg["from"] = s.key.Address
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
	arg["from"] = s.key.Address
	arg["to"] = &to
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(4710000))

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
