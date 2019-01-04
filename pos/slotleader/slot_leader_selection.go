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
			fakeSlotLeaders = append(fakeSlotLeaders, s.epochLeadersPtrArray[i%10])
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
		return s.epochLeadersPtrArray[slotID%10], nil
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
		log.Error("vm.GetR return nil, use a default value")
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
		smaGenesis := []string{"0491bcdcc06d2ba82a90877228268e4f7f1235ebbfc696a19ffb85590f30df9d3fa50ec030759a89b8bdab3aa3a844283a18c6567cd5451aab5c1df866b81a0c1c",
			"04461b6877fc12569520a994505fbd60d8c58ef3b7fd3984efc0a2494bbc8f68f4364da4013495f9b29128553b874d6442cbac51fc51cfed39851867737ab4a02e",
			"0466eea24039a99afba154d4c64ee7a8cf120a54d979ed55f793ce6cc2fc3635a5b79c4de545cb1ed493521a1ce6da95770a71d207121b01d26df88f6ee86d0b22",
			"04d8e72df643e3af9bd3e2f92eb5a8d5b24ac9833c1ccb5607f13f9b33649bf10c8e9a671e1c886e453333684f6ac01b008d1629f93a85a4f216a13b5f1862a8c1",
			"04a9f7bf968fee493b51b045827488a35731f2d7af063b51d31a42ab43b51468e36678dbbcf77410345d0540cbe65e348af242251b0f2ab10580ea5bca91839305",
			"04461b6877fc12569520a994505fbd60d8c58ef3b7fd3984efc0a2494bbc8f68f4364da4013495f9b29128553b874d6442cbac51fc51cfed39851867737ab4a02e",
			"04ab3825a803d2bf43335c5cd211de170e06c8a359aafbb49136e1b8a61e7a7f42a8b9cc30c0ad8f501e76a2750d676444f578b432d2d35ff24e858049e3eea14c",
			"049e4c0f311b5e3593d9ee09720abd62c23002a7b40da13d2483c85936d560f384ff7e85bcf0332ec53f2be84273e341dfdd4ad63f4f1e219ea7ab2b28b0dd8621",
			"0400994bbd33ad79912478e26bd92331468da9b69d9ac2ef1c80f4597acf463552d341508b1adb40e4b69d76361bd80230f9fdee2414db8bbdfdcb83e23e1d6ff7",
			"0444f09890a83ba77cbee7d432be95c850780362bc6f1e8f0198dabf1164a67a3d084e10ae3e1caf29f2638945c73867e456edddfc5c6fd5cb0e7e7c7558be003d",
			"04b274ff6d60c8d2a752887dd79ddd42d11bd363eee3e98ae3781872a2716cd8fdf7c24e8a910fe16633862a0e256b43552eac03f5fe2971d76d0a66bb43927af8"}
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
			sumpub.X, sumpub.Y = crypto.S256().Add(sumpub.X, sumpub.Y, SMA[i].X, SMA[i].Y)
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
						temp.X, temp.Y = crypto.S256().Add(temp.X, temp.Y, SMA[j].X, SMA[j].Y)
					} else if que == 0 {
						temp.X = new(big.Int).Set(SMA[j].X)
						temp.Y = new(big.Int).Set(SMA[j].Y)
						que = 1
					}

				}
			}
			if temp.X == nil { //if bi-1,x are all 0, set cr[i] = hash(RB)
				rb, err := s.getRandom(epochID)
				if err != nil {
					return nil, err
				}
				cri := new(big.Int).SetBytes(crypto.Keccak256(rb.Bytes()))
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

		log.Debug("*********************getCRs Put CR **********************", "epochIDPut", 1)
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
	fmt.Printf("*******GenerateSlotLeaderSeqAndIndex rb:%s\n", hex.EncodeToString(random.Bytes()))

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

	log.Debug("*********************generateSlotLeadsGroup Put CR **********************", "epochIDPut", epochID+1)

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
	if epochID == 0 || epochID == 1 {
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

	if epochID == 0 || epochID == 1 {
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
