package slotleader

import (
	"bytes"
	"crypto/ecdsa"
	Rand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/rlp"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos"

	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/postools/slottools"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/rpc"

	"github.com/btcsuite/btcd/btcec"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/uleaderselection"
)

//CompressedPubKeyLen means a compressed public key byte len.
const CompressedPubKeyLen = 33
const LengthPublicKeyBytes = 65
const LengthCR = 32

const (
	StageTwoProofCount = 2

	// SlotStage1 is 40% of slot count
	SlotStage1 = uint64(pos.SlotCount * 4 / 10)
	// SlotStage2 is 80% of slot count
	SlotStage2       = uint64(pos.SlotCount * 8 / 10)
	EpochLeaders     = "epochLeaders"
	SecurityMsg      = "securityMsg"
	CR               = "cr"
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

	slotLeadersPtrArray    [pos.SlotCount]*ecdsa.PublicKey
	slotLeadersIndex       [pos.SlotCount]uint64
	epochLeadersPtrArray   [pos.EpochLeaderCount]*ecdsa.PublicKey
	validEpochLeadersIndex [pos.EpochLeaderCount]bool // true: can be used to slot leader false: can not be used to slot leader

	stageOneMi       [pos.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoAlphaPKi [pos.EpochLeaderCount][pos.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoProof    [pos.EpochLeaderCount][StageTwoProofCount]*big.Int //[0]: e; [1]:Z
	slotCreated      bool
	blockChain       *core.BlockChain
}

// Pack is use to pack info for slot proof
type Pack struct {
	Proof    [][]byte
	ProofMeg [][]byte
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
func (s *SlotLeaderSelection) Init(blockChain *core.BlockChain, rc *rpc.Client, key *keystore.Key, epochInstance interface{}) {
	s.blockChain = blockChain
	s.rc = rc
	s.key = key
	s.epochInstance = epochInstance
	if blockChain != nil {
		log.Info("SlotLeaderSelecton init success")
	}
}

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
			log.Error("getWorkStage error: " + err.Error())
		}
	}

	switch workStage {
	case slotLeaderSelectionStage1:
		log.Debug("Enter slotLeaderSelectionStage1")
		err := s.generateSlotLeadsGroup(epochID)
		if err != nil {
			log.Debug(err.Error())
		}
		//s.buildEpochLeaderGroup(epochID)

		s.setWorkingEpochID(epochID)
		err = s.startStage1Work()
		if err != nil {
			log.Error(err.Error())
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

		// if slotID < SlotStage1 {
		// 	break
		// }

		//s.buildEpochLeaderGroup(epochID)

		err := s.startStage2Work()
		if err != nil {
			log.Error(err.Error())
		} else {
			s.setWorkStage(epochID, slotLeaderSelectionStage3)
		}

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
			log.Error(err.Error())
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
				log.Error(err.Error())
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
	epochIDBuf := posdb.Uint64ToBytes(epochID)
	selfIndexBuf := posdb.Uint64ToBytes(selfIndexInEpochLeader)

	log.Debug("epochIDBuf(hex): " + hex.EncodeToString(epochIDBuf))
	log.Debug("selfIndexBuf: " + hex.EncodeToString(selfIndexBuf))
	log.Debug("pkCompress: " + hex.EncodeToString(pkCompress))
	log.Debug("miCompress: " + hex.EncodeToString(miCompress))

	buffer, err := slottools.RlpPackCompressedPK(epochIDBuf, selfIndexBuf, pkCompress, miCompress)

	posdb.GetDb().PutWithIndex(epochID, selfIndexInEpochLeader, "alpha", alpha.Bytes())

	log.Debug(fmt.Sprintf("----Put alpha epochID:%d, selfIndex:%d, alpha:%s, mi:%s, pk:%s", epochID, selfIndexInEpochLeader, alpha.String(), hex.EncodeToString(crypto.FromECDSAPub(commitment[1])), hex.EncodeToString(crypto.FromECDSAPub(commitment[0]))))

	functrace.Exit()
	return buffer, err
}

//GetAlpha get alpha of epochID
func (s *SlotLeaderSelection) GetAlpha(epochID uint64, selfIndex uint64) (*big.Int, error) {
	if pos.SelfTestMode {
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

var (
	curEpochId = uint64(0)
	curSlotId  = uint64(0)
)

func GetEpochSlotID() (uint64, uint64, error) {
	return curEpochId, curSlotId, nil
}
func CalEpochSlotID() {
	if pos.EpochBaseTime == 0 {
		return
	}
	timeUnix := uint64(time.Now().Unix())
	epochTimespan := uint64(pos.SlotTime * pos.SlotCount)
	curEpochId = uint64((timeUnix - pos.EpochBaseTime) / epochTimespan)
	curSlotId = uint64((timeUnix - pos.EpochBaseTime) / pos.SlotTime % pos.SlotCount)
	fmt.Println("CalEpochSlotID:", curEpochId, curSlotId)
}

//getEpochLeaders get epochLeaders of epochID in StateDB
func (s *SlotLeaderSelection) getEpochLeaders(epochID uint64) [][]byte {
	//test := false
	//test := true
	if pos.SelfTestMode {
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

		selector := posdb.GetEpocherInst()

		if selector == nil {
			return nil
		}

		epochLeaders := selector.(epoch).GetEpochLeaders(epochID)
		//for i := 0; i<len(epochLeaders); i++  {
		//	log.Debug(fmt.Sprintf("%s\n", hex.EncodeToString(epochLeaders[i])))
		//}

		if epochLeaders != nil {
			log.Debug(fmt.Sprintf("getEpochLeaders called return len(epochLeaders):%d", len(epochLeaders)))
		}
		return epochLeaders
	}
}

func (s *SlotLeaderSelection) getEpochLeadersPK(epochID uint64) []*ecdsa.PublicKey {
	bufs := s.getEpochLeaders(epochID)

	pks := make([]*ecdsa.PublicKey, len(bufs))
	for i := 0; i < len(bufs); i++ {
		pks[i] = crypto.ToECDSAPub(bufs[i])
	}

	return pks
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
	s.slotCreated = false
	// clear Array
	s.epochLeadersArray = make([]string, 0)
	// clear map
	s.epochLeadersMap = make(map[string][]uint64)

	for i := 0; i < pos.EpochLeaderCount; i++ {
		s.epochLeadersPtrArray[i] = nil
		s.validEpochLeadersIndex[i] = true

		s.stageOneMi[i] = nil

		for j := 0; j < pos.EpochLeaderCount; j++ {
			s.stageTwoAlphaPKi[i][j] = nil
		}
		for k := 0; k < StageTwoProofCount; k++ {
			s.stageTwoProof[i][k] = nil
		}
	}

	for i := 0; i < pos.SlotCount; i++ {
		s.slotLeadersPtrArray[i] = nil

	}

	for i := 0; i < pos.SlotCount; i++ {
		s.slotLeadersIndex[i] = 0

	}
}

func (s *SlotLeaderSelection) dumpData() {

	fmt.Printf("~~~~~~~~~~~dumpData begin~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
	// fmt.Printf("\t\t\ts.slotCreated = %v\n\n", s.slotCreated)
	// fmt.Printf("\t\t\ts.epochLeadersArray= %v\n\n", s.epochLeadersArray)
	// fmt.Printf("\t\t\ts.epochLeadersMap= %v\n\n", s.epochLeadersMap)
	// fmt.Printf("\t\t\ts.epochLeadersPtrArray= %v\n\n", s.epochLeadersPtrArray)
	// fmt.Printf("\t\t\ts.validEpochLeadersIndex= %v\n\n", s.validEpochLeadersIndex)

	// fmt.Printf("\t\t\ts.stageOneMi= %v\n\n", s.stageOneMi)
	// fmt.Printf("\t\t\ts.stageTwoAlphaPKi= %v\n\n", s.stageTwoAlphaPKi)
	// fmt.Printf("\t\t\ts.stageTwoProof= %v\n\n", s.stageTwoProof)
	// fmt.Printf("\t\t\ts.slotLeadersPtrArray= %v\n\n", s.slotLeadersPtrArray)
	// fmt.Printf("\t\t\ts.epochLeadersPtrArray= %v\n\n", s.epochLeadersPtrArray)
	// for index, value := range s.epochLeadersPtrArray {
	// 	fmt.Printf("\tepochLeadersPtrArray index := %d, %v\t\n", index, crypto.FromECDSAPub(value))
	// }
	for index, value := range s.epochLeadersPtrArray {
		fmt.Printf("\tepochLeadersPtrArrayHex index := %d, %v\t\n", index, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	for index, value := range s.slotLeadersPtrArray {
		fmt.Printf("\tslotLeadersPtrArrayHex index := %d, %v\t\n", index, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	// for index, value := range s.slotLeadersPtrArray {
	// 	fmt.Printf("\tslotLeadersPtrArray index := %d, %v\t\n", index, crypto.FromECDSAPub(value))
	// }
	fmt.Printf("~~~~~~~~~~~dumpData end~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n")
}

func (s *SlotLeaderSelection) buildEpochLeaderGroup(epochID uint64) error {
	functrace.Enter()
	s.clearData()
	// build Array and map
	data := s.getEpochLeaders(epochID)
	fmt.Printf("Data from jqg: %v", data)
	//for index, value := range s.getEpochLeaders(epochID) {
	for index, value := range data {
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
		for i := 0; i < pos.SlotCount; i++ {
			fakeSlotLeaders = append(fakeSlotLeaders, s.epochLeadersPtrArray[i%pos.EpochLeaderCount])
		}
		return fakeSlotLeaders, nil
	}

	if len(s.slotLeadersPtrArray) != pos.SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	return s.slotLeadersPtrArray[:], nil
}

func (s *SlotLeaderSelection) GetSlotLeader(epochID uint64, slotID uint64) (slotLeader *ecdsa.PublicKey, err error) {
	if !s.slotCreated {
		//return nil, errors.New("slot leaders group not ready")
		log.Debug("slot leaders group not ready use a fake one")
		return s.epochLeadersPtrArray[slotID%pos.EpochLeaderCount], nil
	}
	if len(s.slotLeadersPtrArray) != pos.SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	if slotID >= pos.SlotCount {
		return nil, errors.New("slot id index out of range")
	}
	return s.slotLeadersPtrArray[slotID], nil
}

func (s *SlotLeaderSelection) updateStateDB() {
	stateDb, err := s.blockChain.StateAt(s.blockChain.CurrentBlock().Root())
	if err != nil {
		log.Error("Update stateDb error in SlotLeaderSelection.updateStateDB", "error", err.Error())
	}
	s.stateDb = stateDb
}

// from random proposer
func (s *SlotLeaderSelection) getRandom(epochID uint64) (ret *big.Int, err error) {
	s.updateStateDB()
	rb := vm.GetR(s.stateDb, epochID)
	if rb == nil {
		log.Error("vm.GetR return nil, use a default value", "epochID", epochID)
		rb = big.NewInt(1)
	}
	return rb, nil
}

// from random proposer
func (s *SlotLeaderSelection) getSMAPieces(epochID uint64) (ret []*ecdsa.PublicKey, err error) {
	// 1. get SMA[pre]
	piecesPtr := make([]*ecdsa.PublicKey, 0)
	if epochID == uint64(0) {
		//genesis SMA
		smaGenesis := []string{
			"04a056cf2226d66fb83d38f815be0fe4d8e9f283f26fb9628277ed6cc1d439043ae7769afa90ba26b10f5812d64d19cdb4fda3d811a3290cf320d4246e3b6f305c",
			"04a664e950a415fb25b84b9764af2cdd2b6d148165b1f2c94062ccb52dc8bba9300291a41fac357890391ff93a6246d7d611b761dd6960520774600319f1f8267c",
			"043d7e056e4812b03bb05e871b850141a7b3da44fdc273f17dd30da7ea897b2584bd575a752ed72baffb46b0eb298b6d2bfc701d033194e4390ae4fc8ea8f692bf",
			"04f36c0eef900dcc9b090d21cbf9bcdf8ea7cf49afe2027623dedbfa5e3ede7ba17e8d461957b17dab08c8bde2f086637ed2995d221450abcdede7b615ef50cf28",
			"0426d9f3a35cb0cedeaa9a1ebe9d60bea9bfbd573f13bb1e3c06fdc12da37806b8cdcdc1f990904d3d326a471c3a60593edcc8fe69a0752b72b7ba39a1e174afc6",
			"043b2a3d1bef96a5be59d72b09f9be35375b6509316b08e2045b9b1c63f1256445ab663d0f2f1190552b0841f417cb6fe7e0d18b520828ffcb96f3299c0d930eba",
			"049b87c21421225b4b02614a0526bddaa2dd594b2ececc9bfb54192e730b5b2f0b5d925a5c0c1c3864f64eee9ba9d2e267da020a8bce5eb74834fb15211386a6d1",
			"0451d120465201d3a40561aab1f1500be0fafb3a1d62e94726fc5d4f6a9ae2fa799cb392e9df47e1320cbb1dfabcd6c8f854f156f8b11b694aba4e474911ff1b46",
			"04f12e295120e402a40f5078fdc05af6cf128a58486a0b4e9ae5f4683d02347e07aae117a24f7dc34d582eb60aee27e7952838f86fa48bea7298fff8b056850b5d",
			"04a25b4d84b693d06ffdf061cfc192543029fe77852881fa3a4e99d30e22f453cf7df41de277ffa7a6c38f29db5032686b6a200dadb5ed3f278598a524ddc47e59",
			"040c9036f3452c9dd55dd442765108b3207c426043dc68c586494c4fe5c90a88ff9cee59b44d8e58430504804a1a057b04a73eb38329da40ac4a1fa00e6d58b457",
			"04e1a251aa2a9a93ead901bfbbac5ed84b29110c4aebaf4b885d664d19522fe5840c2174cf39090117edd49f6f71bffb27c8eaa254a1847a79bbc1b2f2bb7a8b20",
			"04a98d62ee4d2ba44258771e74af6c09464dfdefb0173c9086bce75b0a0a940c37e7f58459c85592a3914b0fec8264fb113f077553ef13b0b83d8a3df94905cd66",
			"04811e0d2c60f34068784e35f48291f7ea7a9675fa7a4ae41059fbb1ef896b6526fb6f4a3cb012287f712d772fe096543139140f1fb71e8eabc26a10b120cecbec",
			"0496fe2183e47600dd2e97469895220ce4fc4c8a75e277d2313b326bf614f11ce9243320384e31f195c3d4857ad225721c2d1b48705c0a69cf1e490bc891fbeb33",
			"049c9e935c5959f7d76cae25264b1f7c48b74d120277aa33817deefa803b042ca4e8b9c61d690e2455e5da4e208bc96e096f7359d6c30958b654c49c4f3182ce8b",
			"0464060fb9fae9bd4204c408a7f3f7692d9193e96272d061acac096070c3ab49047e29e7ef1d3ac4265a674476e67317ad1acf2479764dedb5d5f56246e6d8ab16",
			"049f643280179a56e8341a485c3e179e6d3de49ad2037860351d2fe00b471b05d85f170f7cff86bf30f5cfc28e7ecf299ceca1f48d9718525d01d9ddaac7013c7a",
			"048a5a01d85b21862018abb696014fe24c46b4e2590db8d9cad9039b8e3e5722d85d0161096095d85f6e963df9468008556f427f780151299ded472be31a066c5d",
			"0466f403cf282afdffb79eb511becb3eb97d074893db2b1594b223aea45498874f7d774c204bac9bffa9c7a4fc996286581d3e107a50a81683aafca5866c04569a",
			"04a43975dcf19b0b3c3272f42276acde2f29ed5b0b19190bbabfd3bbdb70f1e25bcd64844d3e705356618cee089ee2076096cf74883619756b3fb062236b30bc25",
			"04a743bd8d9e6f9cd8c508fd8431b91a68def1f7a43eaf70830bc0b11e85e94b050e2323fec831ae8fc553347c9846bdaa5960226f3512863a39b7f0fc899f1e0c",
			"044c4b1c4336aa9099e51076c06fafb1d4c7295f57db211b77f7394648f52f122d7fb7aa1a37a4f76b50058b3ae53aed0334eda1264a8fab6feb90e0b4abd6d4b2",
			"049d6ce2c99b1792eaf967108fa4a61bc3a76c1f2e46be9c68f787e73115d1c0376060320eae2e1c8a630caf9902a0aca9c684fd36ed47adb8bc0fe0ef8b742b22",
			"04cbd0a79ee7c7ac96c5cba0291a88280a537668d4660182eba2e42ff0fd540be7aea10b91f630b5e4525c35c30c6b2eda32418a615a8d7f69f401671f03cbac19",
			"04c89bb261c61ec63c9bb2452e4fe0cec56e2000aec7e15942c1bad5a82883a035b48ed3454fcc5089774024cfe7123cd5589a255cbb73e79a1f8bb5467b2c6187",
			"041fb21377e0548f9af27be15b93e42b6b52cbdcf0f632dfa7a99b2684369078f75038210ccc3806e408fe28d6052cc533fcb7f3b32417bf28544cf10b5b99f38d",
			"0443412bcf3d3c450beeec37e16d1414199409a14ebfee2fd7eda04c5d0d06a2e7978d2ae88f8e315e20b8742becb4148ef02a47f7da74ac0354318d9cd8a2b0f4",
			"0423f7dae58ecf9ca842cda76f117e3b4d33b10de2d061ce593944b19a68abefd1f70879d7001007a2044c5bf71e42a72410dc8b110bf673a93a7d6bcee1bbe97e",
			"0416aae0c557ad27edc39c6f971acd571898d7c1406f236ba77ae1539e1a85d274f2d58e16e3b9a5749241e05cd1495a43a860cae9b7f6742b1862ae453eae4f98",
			"04e1727da482cf4b097e6dd580fce2a1bbb20ce6c93acbb74cab9125863b0370cfd0b61968a0fa65431e9f7ee0cc1f4037316597ca64057e5eabb69ae1cc84502e",
			"04ccdbc3a2ca85200ea64f2b35336a19021225ce71c4f343f5a8bae0a062a0a64254d70e3de677731220fe4a37a036d950cf8b34be7d81bc4571d2ad8892e17152",
			"0461b43ebe265d6fc878e72854ea502880ad747377166449d52aa049037da97456869b992582f3719ae9294540f6dc47c20fa9d3b07ad4228dc85368b4b59ea7c5",
			"04001e435b141280e3bd622d93ef7d33b0874961a40e2a02cefd6684738d7e41079fc2246b5d36dee47ddfabf93a98f68bed6e15f0e63dbd2482a94285939704a7",
			"04543df25a6017b1abb76e6b48c89e95f84d9876f56adf5e7ee972b943ae16d022c24fc5b49f77d6a931872b01ccdd503df2363a51c72f4f9bbef33e2c9a63e1be",
			"049f4e6311e64c4bf9de2eb35fc59ae98ccfa8017874482cc80f0472c861f7553808228302d8d3ffc7b38f4091f6a603e4ae2cbe9cf7908e45cd8600f0924cc300",
			"04599e85462df724d8a2071abd3bae3b9068267fef35bd6972424540d420f9859c5bbfed61fdbdb5fdc948c9a9f3716e809c85b1f524b6c350b0ef434bdc3e724f",
			"04fe8047a63fa39769a6e071585fc24717ad34af5482ca0ffd747ed3f77cfeea5fd8776dfa2c33a7afa36bd18968629c274b385c42bb3bda2be314a348b43d0a61",
			"0419dc38105f1bff9349faa9e7da3d99b005f35339c2fa57dbd742da922ad39c46ad6469c5d705b9756c5f6d153d614fc1a298edb67d31f59086b997b2a32e58db",
			"04e0b1464591ece7df575d1c28ecafe8407edfbc01402a15ae244afc20a65aee722755a5f8fd0ef2870840c3524d6172645735e4632dcdb857c47c9dd2aefec175",
			"04eb6bc8ad4859798e67ca46f6c184f39416004ea5482bb87034832ba711a0c69272f4b38137a1750abfcf68f7bb6f3c9de4758dbdc8573fdccdec65e47332cbe4",
			"04e51ac6d9920bd2304043b6baa69446646995ce03d437ea381fbaa88a0de188ea713b3cff30070fddb708b5215cccd64f286d89424e623d470acc776c2052e540",
			"04d6a114d264f103e92fd9f525414aaaa19d753266cce8419b55f35cbd62ca12b1318aba0d2cf06a68ab077088c615f37a3875f196f2d800a40202ab187def5e68",
			"04ab64cac5b0587171ab7359165f754bf8ad2da721e794a992c9b4fc387f9feae664a869a5dce91a485ac10447dd918ae6d4c7f5c9bc9989f9b94169497c93a426",
			"04b7d1bf28f960d55ee88ac92ee2d894c6763da4372aff518ae91206a358879d0baaf1302921042138119912fda04b138e1a834f94a8142ea65e9032e9909fc57b",
			"0409acf48f99ed374ea68abf6f02c89f4024f7d5c6e9f00be823a51cef517d6ce52f72374937d2f6c13e46562fa8c5560918909c82649aee979a6f4c885e5ee4ba",
			"046f150ab5d57a9c07a120e3228f6a1301287c5eb41fc2b40b9df621c30c8a40c7933c761d4cbf1886778019a226cd648ea89b0451d1715907ac4229d6a3089b58",
			"04fd182056522a1dd0bf20b1b9f9757506bad0bc14d42f840310d581510cfa3ab3962204afa0077b632d7f57ad3d0504582ceb36ea6a23e496585a5bfe5ac8efc0",
			"04b2ddf91110eaacba9cf7395be1b29913dcbbb9ba743cbe0695ca4632bf6337ea9f178a20308398c32a6f90dea9f7e0607870d243714bcfd031ed86ebda4a993f",
			"04135c7c6145c1250cf2ad7805763bc5ee279f30fb92fa666f72962ece86d17c5aca2832e3b9c5cad7e104e2d8b78b7edb9420a33650a371d268ea68c0e529bd08",
			"046755f3282b21fa20957a537653f8709d4ed706bced16cfb5d523224eedd3ad459087d973895239241569950d106067ea3fe7d06e8078eccc4107bb04d71c955d",
			"0453ac0dab37aa88b0ebe9ba3dc7a3e6a49fdd01b3eea30749d0bb8b0be7395c7d8fa20bd0faf4b53c0af4c0a8a0dbb25a1fd842dbea741d7480462a5a7a4a64a1",
			"04110c767024c47e2aad49351be876cd6c787694cb871db216d12af2d37d80caad57bb6b7fa2946d3f724a215433bec677d51166b06891a21e5636ac5c882374d9",
			"04eb7892f3ee24d48b94e84190e788773acf9f1167914b7bd4bb69e034c9027bd30c0dd97e2a4502f7aa289b712758bf383b1363985cc310dca6459baefd75b447",
			"04bfb970927bac1b14d001b28ad7294ec3617a11772a814875da7c93d8cf123fd8f04d8a9098ecd88964377313e2ddbbdb0b31de72bc2398115ed30319dda44d8b",
			"04203ef3cf9d8cb96e514bbf25eae660aa4fa26bd72bfffdac74bb932c72870e5733245fd18acbcb034bbc9ce5b8e68ea7b8583f278b358233f29ca0822543bc9f",
			"0403699439685adefdb4f380985935b33488aaa0d15e8984d538cea46a767b710b5d527978765fc76813abf0bf8a34298b3462f559e638cedbf268b0e9a9a65c94",
			"0402cfaceb6b6f1d9fac0a03472fe3afaf943b2d4b6effab64964faf0104748ad2487cb5f87cd4d579b0b7a16a18556bfdcb3fd6c4f9f01abbbec77d0520f237ee",
			"0403c37d042712dbe319d8a1660e715b72219d8e64a82dd5ac26498f0e5a9b5dcb9cc7cd64061ffd3732a240495bff35cfdbaf4e777475ec3e63ead3ffa03b26c1",
			"04e943bc39542e0163959546c45854095f696fe39ef32d1877c5b6168579f94b16a720dfd169593807a8503a1d26b6e667a3959e96b8b8d710c70677d7918808b1",
			"0491625fc1c0796489e047b7bb6b303bf55fbde14346cec426c3a677f39692f92b770c7868c5c44a706bb495d79ce3a62637d220c855f214f96db05eecec709a10",
			"049a1151cc3f107e0004c1b286d8d0e5d6b45e928e71277b0c7daee8ea4abfb90ea26f7e09af478ba35fd70d3aa9b6dc3e99875a24798db868c5c095984aebfa45",
			"040284eaacbf37af367c6f88a9cfd8779714bace25befccc452dd0af3b29878df146e62bf9dfd0585b636774529b576f6885445841206965e5d5d0e87a84f9afd0",
			"049230f1c7f7d8034bd13db5f6ffa188ba5d4022ad0f8293c4418a7a06dd30b3571f715d3d2062212e06ee68357f86177f18474e6e43c9113163f79cee50fd2db6"}
		//genesis SMA
		for i := 0; i < pos.EpochLeaderCount; i++ {
			sma0Bytes, err := hex.DecodeString(smaGenesis[i])
			if err != nil {
				return nil, err
			}
			piecesPtr = append(piecesPtr, crypto.ToECDSAPub(sma0Bytes))
		}
		return piecesPtr, nil

	} else {
		// pieces: alpha[1]*G, alpha[2]*G, .....
		pieces, err := posdb.GetDb().Get(epochID, SecurityMsg)
		if err != nil {
			log.Error("getSMAPieces error", "epochID", epochID, "SecurityMsg", SecurityMsg)
		}
		fmt.Printf("getSMAPieces: get from db, epochID:%d, key:%s, pieces is = %v\n", epochID, SecurityMsg, pieces)
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
			fmt.Printf("epchoID=%d,getSMAPieces: one hex.EncodeToString(piece) is = %v\n\n", epochID, hex.EncodeToString(pubKeyByte))
			piecesPtr = append(piecesPtr, crypto.ToECDSAPub(pubKeyByte))
		}
		return piecesPtr, nil
	}
}
func (s *SlotLeaderSelection) getCRs(epochID uint64) (ret []*big.Int, err error) {
	// 1. get SMA[pre]
	if epochID == uint64(0) {
		SMA, err := s.getSMAPieces(epochID)
		if err != nil {
			return nil, err
		}
		//calculate the cr sequence
		cr := make([]*big.Int, 0)
		na := len(SMA)
		//cr[0] = hash(alpha1*G+alpha2*G+...+alphan*G)
		sumpub := new(ecdsa.PublicKey)
		sumpub.Curve = crypto.S256()
		sumpub.X = new(big.Int).Set(SMA[0].X)
		sumpub.Y = new(big.Int).Set(SMA[0].Y)
		for i := 1; i < na; i++ {
			sumpub.X, sumpub.Y = uleaderselection.Wadd(sumpub.X, sumpub.Y, SMA[i].X, SMA[i].Y)
		}
		cr0 := new(big.Int).SetBytes(crypto.Keccak256(crypto.FromECDSAPub(sumpub)))
		cr = append(cr, cr0)
		//cr[i] = hash(bi-1,0*alpha1*G+bi-1,1*alpha2*G+...+bi-1,n-1*alphan*G)
		//cr[i-1]=bi-1,0||bi-1,1||...||bi-1,n-1
		for i := 1; i < pos.SlotCount; i++ {
			temp := new(ecdsa.PublicKey)
			temp.Curve = crypto.S256()
			que := 0
			for j := 0; j < na; j++ {
				if cr[i-1].Bit(j) == 1 {
					if que == 1 {
						temp.X, temp.Y = uleaderselection.Wadd(temp.X, temp.Y, SMA[j].X, SMA[j].Y)
					} else if que == 0 {
						temp.X = new(big.Int).Set(SMA[j].X)
						temp.Y = new(big.Int).Set(SMA[j].Y)
						que = 1
					}

				}
			}
			if temp.X == nil { //if bi-1,x are all 0, set cr[i] = hash(RB)
				//rb, err := s.getRandom(epochID)
				// if err != nil {
				// 	return nil, err
				// }
				cri := cr0
				cr = append(cr, cri)
			} else {
				cri := new(big.Int).SetBytes(crypto.Keccak256(crypto.FromECDSAPub(temp)))
				cr = append(cr, cri)
			}
		}

		// insert CRS to local DB
		crBuf, err := rlp.EncodeToBytes(cr)
		if err != nil {
			log.Error("getCRs rlp.EncodeToBytes error", "error", err.Error())
			return nil, err
		}

		log.Debug("*********************getCRs Put CR **********************", "epochIDPut", 0)
		_, err = posdb.GetDb().Put(uint64(0), CR, crBuf)
		if err != nil {
			return nil, err
		}
		return cr, nil
	} else {
		crsBytes, err := posdb.GetDb().Get(epochID, CR)
		fmt.Printf("getSMAPieces: get from db, hex.encodingToString(pieces) is = %v\n", hex.EncodeToString(crsBytes))
		if err != nil {
			return nil, err
		}
		var crsPtr []*big.Int
		err = rlp.DecodeBytes(crsBytes, &crsPtr)
		if err != nil {
			log.Error("getCRs rlp.DecodeBytes error", "error", err.Error())
			return nil, err
		}
		return crsPtr, nil
	}
}
func (s *SlotLeaderSelection) generateSlotLeadsGroup(epochID uint64) error {
	functrace.Enter()
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
	fmt.Printf("\nRandom got is %v\n", hex.EncodeToString(random.Bytes()))
	// 5. return slot leaders pointers.
	slotLeadersPtr := make([]*ecdsa.PublicKey, 0)
	fmt.Printf("len(piecesPtr)=%v\n", len(piecesPtr))
	fmt.Printf("len(epochLeadersPtrArray)=%v\n", len(s.epochLeadersPtrArray))
	fmt.Printf("len(random.Bytes)=%v\n", len(random.Bytes()))
	fmt.Printf("SlotCount= %d\n", pos.SlotCount)
	//fmt.Printf("===========================before GenerateSlotLeaderSeq\n")
	//s.dumpData()
	//epochLeadersPtrArray := s.epochLeadersPtrArray
	if epochID == 0 {
		log.Info("Epoch 0 do not have pre epoch leaders")
		return nil
	}
	epochLeadersPtrArray := s.getEpochLeadersPK(epochID - 1)
	if len(epochLeadersPtrArray) != pos.EpochLeaderCount {
		return errors.New(fmt.Sprintf("fail to get epochLeader:%d", epochID-1))
	}
	for i := 0; i < len(piecesPtr); i++ {
		ret := crypto.S256().IsOnCurve(piecesPtr[i].X, piecesPtr[i].Y)
		if !ret {
			log.Error("not")
		}
		ret = crypto.S256().IsOnCurve(epochLeadersPtrArray[i].X, epochLeadersPtrArray[i].Y)
		if !ret {
			log.Error("not")
		}
	}

	//slotLeadersPtr, crs, err := uleaderselection.GenerateSlotLeaderSeq(piecesPtr[:], epochLeadersPtrArray[:], random.Bytes(), pos.SlotCount)
	slotLeadersPtr, crs, slotLeadersIndex, err := uleaderselection.GenerateSlotLeaderSeqAndIndex(piecesPtr[:], epochLeadersPtrArray[:], random.Bytes(), pos.SlotCount)
	fmt.Printf("*******GenerateSlotLeaderSeqAndIndex rb:%s, epochID:%d\n", hex.EncodeToString(random.Bytes()), epochID)

	fmt.Printf("===========================after GenerateSlotLeaderSeq\n")
	//s.dumpData()

	if err != nil {
		return err
	}

	log.Debug(fmt.Sprintf("===================================\n\n\n\n GenerateSlotLeaderSeq success \n\n\n\n=============================="))

	// 6. insert slot address to local DB
	for index, val := range slotLeadersPtr {
		_, err = posdb.GetDb().PutWithIndex(uint64(epochID), uint64(index), SlotLeader, crypto.FromECDSAPub(val))
		//s.epochLeadersPtrArray[index] = val
		s.slotLeadersPtrArray[index] = val
		if err != nil {
			return err
		}
	}
	// insert CRS to local DB
	crBuf, err := rlp.EncodeToBytes(crs)
	if err != nil {
		log.Error("generateSlotLeadsGroup rlp.EncodeToBytes error", "error", err.Error())
		return err
	}

	_, err = posdb.GetDb().Put(uint64(epochID), CR, crBuf)
	if err != nil {
		return err
	}

	log.Debug("*********************generateSlotLeadsGroup Put CR **********************", "epochIDPut", epochID)
	for i := 0; i < len(crs); i++ {
		log.Debug("cr value", "epochID", epochID, "crIndex", i, "crValue", hex.EncodeToString(crs[i].Bytes()))
	}

	for index, value := range slotLeadersIndex {

		s.slotLeadersIndex[index] = value
	}

	s.slotCreated = true
	s.dumpData()
	return nil
}

func (s *SlotLeaderSelection) inEpochLeadersOrNot(pkIndex uint64, pkBytes []byte) bool {
	return (pkIndex < uint64(len(s.epochLeadersArray))) && (hex.EncodeToString(pkBytes) == s.epochLeadersArray[pkIndex])
}

func (s *SlotLeaderSelection) getStateDb() (stateDb *state.StateDB, err error) {
	s.updateStateDB()
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
		log.Debug(fmt.Sprintf("VerifyDleqProof: len(pk):%d, len(alphaPK):%d, len(stageTwoProof):%d", len(s.epochLeadersPtrArray[:]), len(s.stageTwoAlphaPKi[index][:]), len(s.stageTwoProof[index][:])))
		log.Debug(fmt.Sprintln("VerifyDleqProof: pk:", s.epochLeadersPtrArray[:], " , alphaPK:", s.stageTwoAlphaPKi[index][:], ", stageTwoProof:", s.stageTwoProof[index][:]))
		// verify proof[index]
		if s.epochLeadersPtrArray[0] == nil {
			return false, errors.New("Epoch leaders are not ready")
		}
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
		log.Debug(fmt.Sprintf("%v not in epoch leaders", hex.EncodeToString(crypto.FromECDSAPub(selfPk))))
		return nil, nil
	}

	selfPkRecievedPicesMap := make(map[uint64][]*ecdsa.PublicKey, 0)
	for _, selfIndex := range indexs {
		for i := 0; i < len(s.epochLeadersArray); i++ {
			if s.stageTwoAlphaPKi[i][selfIndex] != nil {
				selfPkRecievedPicesMap[selfIndex] = append(selfPkRecievedPicesMap[selfIndex], s.stageTwoAlphaPKi[i][selfIndex])
			}
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
	epochIDBufDec := posdb.Uint64ToBytes(epochID)
	if err != nil {
		return nil, nil, err
	}
	keyBuf.Write(epochIDBufDec)

	selfIndexBufDec := posdb.Uint64ToBytes(selfIndex)
	if err != nil {
		return nil, nil, err
	}
	keyBuf.Write(selfIndexBufDec)

	keyBuf.Write([]byte("slotLeaderStag2"))
	keyHash := crypto.Keccak256Hash(keyBuf.Bytes())

	log.Debug(fmt.Sprintf("try to get stateDB addr:%s, key:%s", slotLeaderPrecompileAddr.Hex(), keyHash.Hex()))

	data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if data == nil {
		log.Debug("can not find from statedb:" + fmt.Sprintf("addr:%s, key:%s, epochID:%d, selfIndex:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), epochID, selfIndex))
		return nil, nil, errors.New("can not find from statedb:" + fmt.Sprintf("addr:%s, key:%s, epochID:%d, selfIndex:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), epochID, selfIndex))
	}

	_, _, _, alphaPki, proof, err := slottools.RlpUnpackStage2Data(data)
	if err != nil {
		return nil, nil, err
	}
	return alphaPki, proof, nil
}

func (s *SlotLeaderSelection) collectStagesData(epochID uint64) (err error) {
	for i := 0; i < pos.EpochLeaderCount; i++ {
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

		if (len(alphaPkis) != pos.EpochLeaderCount) || (len(proofs) != StageTwoProofCount) {
			s.validEpochLeadersIndex[i] = false
		} else {

			for j := 0; j < pos.EpochLeaderCount; j++ {
				//s.stageTwoAlphaPKi[i][j] = crypto.ToECDSAPub([]byte(alphaPkis[j]))
				alphaPkiDecodeBytes, err := hex.DecodeString(alphaPkis[j])
				if err != nil {
					return err
				}
				s.stageTwoAlphaPKi[i][j] = crypto.ToECDSAPub(alphaPkiDecodeBytes)
			}

			for j := 0; j < StageTwoProofCount; j++ {
				//proof, err := strconv.ParseInt(proofs[j], 10, 64)
				//if err != nil {
				//	return err
				//}
				var err bool
				s.stageTwoProof[i][j], err = big.NewInt(0).SetString(proofs[j], 16)
				if !err {
					return errors.New("proofs error")
				}
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
	for i := 0; i < pos.EpochLeaderCount; i++ {
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
	ArrayPieceClean := make([]*ecdsa.PublicKey, 0)
	for i := 0; i < len(ArrayPiece); i++ {
		if ArrayPiece[i] != nil {
			ArrayPieceClean = append(ArrayPieceClean, ArrayPiece[i])
		}
	}
	ArrayPiece = ArrayPieceClean
	fmt.Println("len(ArrayPiece):", len(ArrayPiece))
	smasPtr, err = uleaderselection.GenerateSMA(PrivateKey, ArrayPiece)
	if err != nil {
		return err
	}
	for _, value := range smasPtr {
		smasBytes.Write(crypto.FromECDSAPub(value))
		fmt.Printf("\n&&&&&&&&&&&&&epochID+1 = %d set security message is %v\n", epochID+1, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	_, err = posdb.GetDb().Put(uint64(epochID+1), SecurityMsg, smasBytes.Bytes())
	if err != nil {
		return err
	}

	log.Debug(fmt.Sprintf("----Generate SMA Success-----epochID:%d, key:%s, bytes:%s", epochID+1, SecurityMsg, hex.EncodeToString(smasBytes.Bytes())))

	return nil
}

//ProofMes = [PK, Gt, skGt] []*PublicKey
//Proof = [e,z] []*big.Int
func (s *SlotLeaderSelection) GetSlotLeaderProof(PrivateKey *ecdsa.PrivateKey, epochID uint64, slotID uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {
	//1. SMA PRE
	smaPiecesPtr, err := s.getSMAPieces(epochID)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, err
	}

	//2. epochLeader PRE
	var epochLeadersPtrPre []*ecdsa.PublicKey
	if epochID == 0 {
		epochLeadersPtrPre = make([]*ecdsa.PublicKey, pos.EpochLeaderCount)
		for i := 0; i < pos.EpochLeaderCount; i++ {
			buf, err := hex.DecodeString(pos.GenesisPK)
			if err != nil {
				log.Error("hex.DecodeString(pos.GenesisPK) error!")
				continue
			}
			epochLeadersPtrPre[i] = crypto.ToECDSAPub(buf)
		}
	} else {
		epochLeadersPtrPre = s.getEpochLeadersPK(epochID - 1)
	}

	//3. RB PRE
	var rbPtr *big.Int

	rbPtr, err = s.getRandom(epochID)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, err
	}

	rbBytes := rbPtr.Bytes()
	//4. CR PRE
	crsPtr, err := s.getCRs(epochID)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, err
	}

	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof(PrivateKey, smaPiecesPtr, epochLeadersPtrPre, rbBytes[:], crsPtr[:], int(slotID))

	return profMeg, proof, err
}

func (s *SlotLeaderSelection) VerifySlotProof(epochID uint64, Proof []*big.Int, ProofMeg []*ecdsa.PublicKey) bool {

	var epochLeadersPtrPre []*ecdsa.PublicKey

	if epochID == 0 {
		epochLeadersPtrPre = make([]*ecdsa.PublicKey, pos.EpochLeaderCount)
		for i := 0; i < pos.EpochLeaderCount; i++ {
			buf, err := hex.DecodeString(pos.GenesisPK)
			if err != nil {
				log.Error("hex.DecodeString(pos.GenesisPK) error!")
				continue
			}
			epochLeadersPtrPre[i] = crypto.ToECDSAPub(buf)
		}
	} else {
		epochLeadersPtrPre = s.getEpochLeadersPK(epochID - 1)
	}

	//3. RB PRE
	rbPtr, err := s.getRandom(epochID)
	if err != nil {
		log.Error(err.Error())
		return false
	}

	rbBytes := rbPtr.Bytes()

	return uleaderselection.VerifySlotLeaderProof(Proof[:], ProofMeg[:], epochLeadersPtrPre[:], rbBytes[:])
}

// PackSlotProof can make a pack info for header seal
func (s *SlotLeaderSelection) PackSlotProof(epochID uint64, slotID uint64, privKey *ecdsa.PrivateKey) ([]byte, error) {
	proofMeg, proof, err := s.GetSlotLeaderProof(privKey, epochID, slotID)
	if err != nil {
		return nil, err
	}

	objToPack := &Pack{Proof: posdb.BigIntArrayToByteArray(proof), ProofMeg: posdb.PkArrayToByteArray(proofMeg)}

	buf, err := rlp.EncodeToBytes(objToPack)

	return buf, err
}

func (s *SlotLeaderSelection) GetInfoFromHeadExtra(epochID uint64, input []byte) ([]*big.Int, []*ecdsa.PublicKey, error) {
	var info Pack
	err := rlp.DecodeBytes(input, &info)
	if err != nil {
		log.Error("GetInfoFromHeadExtra rlp.DecodeBytes failed", "epochID", epochID, "input", hex.EncodeToString(input))
		return nil, nil, err
	}

	return posdb.ByteArrayToBigIntArray(info.Proof), posdb.ByteArrayToPkArray(info.ProofMeg), nil
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
	var selfPKHexStr, payLoadStr string
	var alphaPkiHexStr, proofHexStr []string

	alphaPkiHexStr = make([]string, 0)
	proofHexStr = make([]string, 0)

	epochIDHexStr := posdb.Uint64ToString(epochID)

	selfIndexHexStr := posdb.Uint64ToString(selfIndex)

	var selfPk *ecdsa.PublicKey
	var err error
	if pos.SelfTestMode {
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
	epID, idxID, pk, mi, err := slottools.RlpUnpackAndWithUncompressPK(readBuf)
	if err != nil {
		return nil, nil, errors.New("getStg1StateDbInfo: RlpUnpackAndWithUncompressPK error")
	}

	if hex.EncodeToString(epID) == hex.EncodeToString(posdb.Uint64ToBytes(epochID)) &&
		hex.EncodeToString(idxID) == hex.EncodeToString(posdb.Uint64ToBytes(index)) &&
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

	//Set payload infomation--------------
	payload, err := slottools.PackStage1Data(data, vm.GetSlotLeaderScAbiString())
	if err != nil {
		log.Debug("PackStage1Data err:" + err.Error())
		return err
	}

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(1500000))
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)
	log.Debug("Write data of payload", "length", len(payload))

	_, err = pos.SendTx(s.rc, arg)
	return err
}
func (s *SlotLeaderSelection) sendStage2Tx(data string) error {
	//test
	fmt.Println("Ready send tx:", data)

	if s.rc == nil {
		return errors.New("rc is not ready")
	}

	//Set payload infomation--------------
	payload, err := slottools.PackStage2Data(data, vm.GetSlotLeaderScAbiString())
	if err != nil {
		return err
	}

	arg := map[string]interface{}{}
	arg["from"] = s.key.Address
	arg["to"] = vm.GetSlotLeaderSCAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(1500000))
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)
	log.Debug("Write data of payload", "length", len(payload))

	_, err = pos.SendTx(s.rc, arg)
	return err
}
