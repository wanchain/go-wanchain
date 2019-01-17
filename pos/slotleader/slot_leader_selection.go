package slotleader

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos"

	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/postools/slottools"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/rpc"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/uleaderselection"
)

//CompressedPubKeyLen means a compressed public key byte len.
const CompressedPubKeyLen = 33
const LengthPublicKeyBytes = 65
const LengthCR = 32

const (
	StageTwoProofCount   = 2
	EpochLeaders         = "epochLeaders"
	SecurityMsg          = "securityMsg"
	CR                   = "cr"
	RandFromProposer     = "randFromProposer"
	RandomSeqs           = "randomSeqs"
	SlotLeader           = "slotLeader"
	slotLeaderTxMinCount = 2
)

const (
	slotLeaderSelectionInit = iota + 1 //1
	//Ready to start slot leader selection stage1
	slotLeaderSelectionStage1 = iota + 1 //2

	//Slot leader selection stage1 finish
	slotLeaderSelectionStage2 = iota + 1 //3

	slotLeaderSelectionStage3 = iota + 1 //4

	slotLeaderSelectionStageFinished = iota + 1 //5

)

var (
	wanCscPrecompileAddr = common.BytesToAddress(big.NewInt(210).Bytes())
	ErrEpochID           = errors.New("EpochID is not valid")
	errorRetry           = 3
	ErrorCount           = uint64(0)
	WarnCount            = uint64(0)
)

//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	workingEpochID         uint64
	workStage              int
	rc                     *rpc.Client
	epochLeadersArray      []string            // len(pki)=65 hex.EncodeToString
	epochLeadersMap        map[string][]uint64 // key: pki value: []uint64 the indexs of this pki. hex.EncodeToString
	key                    *keystore.Key
	stateDb                *state.StateDB
	epochInstance          interface{}
	slotLeadersPtrArray    [pos.SlotCount]*ecdsa.PublicKey
	slotLeadersIndex       [pos.SlotCount]uint64
	epochLeadersPtrArray   [pos.EpochLeaderCount]*ecdsa.PublicKey
	validEpochLeadersIndex [pos.EpochLeaderCount]bool // true: can be used to slot leader false: can not be used to slot leader
	stageOneMi             [pos.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoAlphaPKi       [pos.EpochLeaderCount][pos.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoProof          [pos.EpochLeaderCount][StageTwoProofCount]*big.Int //[0]: e; [1]:Z
	slotCreateStatus       map[uint64]bool
	blockChain             *core.BlockChain

	epochLeadersPtrArrayGenesis [pos.EpochLeaderCount]*ecdsa.PublicKey
	stageOneMiGenesis           [pos.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoAlphaPKiGenesis     [pos.EpochLeaderCount][pos.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoProofGenesis        [pos.EpochLeaderCount][StageTwoProofCount]*big.Int //[0]: e; [1]:Z
	randomGenesis               *big.Int
	smaGenesis                  [pos.EpochLeaderCount]*ecdsa.PublicKey
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
	slotLeaderSelection.slotCreateStatus = make(map[uint64]bool)

	s := slotLeaderSelection
	//genesis random
	s.randomGenesis = big.NewInt(1)
	//genesis epoch leaders
	epoch0Leaders := s.getEpoch0LeadersPK()
	for index, value := range epoch0Leaders {
		s.epochLeadersPtrArrayGenesis[index] = value
	}

	alphas := make([]*big.Int, 0)
	for _, value := range epoch0Leaders {
		tempInt := new(big.Int).SetInt64(0)
		tempInt.SetBytes(crypto.Keccak256(crypto.FromECDSAPub(value)))
		alphas = append(alphas, tempInt)
	}

	for i := 0; i < pos.EpochLeaderCount; i++ {

		// AlphaPK  stage1Genesis
		mi0 := new(ecdsa.PublicKey)
		mi0.Curve = crypto.S256()
		mi0.X, mi0.Y = crypto.S256().ScalarMult(s.epochLeadersPtrArrayGenesis[i].X, s.epochLeadersPtrArrayGenesis[i].Y, alphas[i].Bytes())
		s.stageOneMiGenesis[i] = mi0

		// G
		BasePoint := new(ecdsa.PublicKey)
		BasePoint.Curve = crypto.S256()
		BasePoint.X, BasePoint.Y = crypto.S256().ScalarBaseMult(big.NewInt(1).Bytes())

		// alphaG SMAGenesis
		smaPiece := new(ecdsa.PublicKey)
		smaPiece.Curve = crypto.S256()
		smaPiece.X, smaPiece.Y = crypto.S256().ScalarMult(BasePoint.X, BasePoint.Y, alphas[i].Bytes())
		s.smaGenesis[i] = smaPiece

		for j := 0; j < pos.EpochLeaderCount; j++ {
			// AlphaIPki stage2Genesis
			alphaIPkj := new(ecdsa.PublicKey)
			alphaIPkj.Curve = crypto.S256()
			alphaIPkj.X, alphaIPkj.Y = crypto.S256().ScalarMult(s.epochLeadersPtrArrayGenesis[j].X, s.epochLeadersPtrArrayGenesis[j].Y, alphas[i].Bytes())

			s.stageTwoAlphaPKiGenesis[i][j] = alphaIPkj
		}

	}
}

//GetSlotLeaderSelection get the SlotLeaderSelection's object
func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}

//---------------Information get/set functions--------------------------------------------

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

func GetEpochSlotID() (uint64, uint64) {
	return curEpochId, curSlotId
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

func (s *SlotLeaderSelection) getPreEpochLeadersPK(epochID uint64) ([]*ecdsa.PublicKey, error) {
	if epochID == 0 {
		return s.getEpoch0LeadersPK(), nil
	}

	pks := s.getEpochLeadersPK(epochID - 1)
	if len(pks) == 0 {
		log.Warn("Can not found pre epoch leaders return epoch 0", "epochIDPre", epochID-1)
		return s.getEpoch0LeadersPK(), errors.New("Can not found pre epoch leaders return epoch 0")
	}

	return pks, nil
}

func (s *SlotLeaderSelection) getEpoch0LeadersPK() []*ecdsa.PublicKey {
	pks := make([]*ecdsa.PublicKey, pos.EpochLeaderCount)
	for i := 0; i < pos.EpochLeaderCount; i++ {
		pkBuf, err := hex.DecodeString(pos.GenesisPK)
		if err != nil {
			panic("pos.GenesisPK is Error")
		}
		pks[i] = crypto.ToECDSAPub(pkBuf)
	}
	return pks
}

// isLocalPkInPreEpochLeaders check if local pk is in pre generate epochleader.
// If get pre epochleader length is 0, return true,err to use epoch 0 info
func (s *SlotLeaderSelection) isLocalPkInPreEpochLeaders(epochID uint64) (canBeContinue bool, err error) {

	localPk, err := s.getLocalPublicKey()
	if err != nil {
		log.Error("SlotLeaderSelection.IsLocalPkInPreEpochLeaders getLocalPublicKey error", "error", err)
		panic("SlotLeaderSelection.IsLocalPkInPreEpochLeaders getLocalPublicKey error")
	}

	if epochID == 0 {
		for _, value := range s.epochLeadersPtrArrayGenesis {
			if posdb.PkEqual(localPk, value) {
				return true, nil
			}
		}
		return false, nil
	}

	prePks, err := s.getPreEpochLeadersPK(epochID)
	if err != nil {
		return true, errors.New("can not get pre EpochLeaders PK")
	}

	for i := 0; i < len(prePks); i++ {
		if posdb.PkEqual(localPk, prePks[i]) {
			return true, nil
		}
	}
	return false, nil
}

//getWorkStage get work stage of epochID from levelDB
func (s *SlotLeaderSelection) getWorkStage(epochID uint64) int {
	ret, err := posdb.GetDb().Get(epochID, "slotLeaderWorkStage")
	if err != nil {
		if err.Error() == "leveldb: not found" {
			s.setWorkStage(epochID, slotLeaderSelectionInit)
			return slotLeaderSelectionInit
		} else {
			log.Error("getWorkStage error: " + err.Error())
			panic("getWorkStage error")
		}
	}
	workStageUint64 := posdb.BytesToUint64(ret)
	return int(workStageUint64)
}

//saveWorkStage save the work stage of epochID in levelDB
func (s *SlotLeaderSelection) setWorkStage(epochID uint64, workStage int) error {
	workStageBig := big.NewInt(int64(workStage))
	_, err := posdb.GetDb().Put(epochID, "slotLeaderWorkStage", workStageBig.Bytes())
	return err
}

func (s *SlotLeaderSelection) clearData() {
	// s.slotCreateStatus = make(map[uint64]bool)
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

	s.dumpPreEpochLeaders()
	s.dumpCurrentEpochLeaders()
	s.dumpSlotLeaders()
	s.dumpLocalPublicKey()
	s.dumpLocalPublicKeyIndex()
}

func (s *SlotLeaderSelection) dumpPreEpochLeaders() {
	log.Info("\n")
	currentEpochID := s.getWorkingEpochID()
	log.Info("dumpPreEpochLeaders", "currentEpochID", curEpochId)
	if currentEpochID == 0 {
		return
	}

	preEpochLeaders := s.getEpochLeaders(currentEpochID - 1)
	for i := 0; i < len(preEpochLeaders); i++ {
		log.Info("dumpPreEpochLeaders", "index", i, "preEpochLeader", hex.EncodeToString(preEpochLeaders[i]))
	}

	log.Info("\n")
}
func (s *SlotLeaderSelection) dumpCurrentEpochLeaders() {
	log.Info("\n")
	currentEpochID := s.getWorkingEpochID()
	log.Info("dumpCurrentEpochLeaders", "currentEpochID", currentEpochID)
	if currentEpochID == 0 {
		return
	}

	for index, value := range s.epochLeadersPtrArray {
		log.Info("dumpCurrentEpochLeaders", "index", index, "curEpochLeader", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

}

func (s *SlotLeaderSelection) dumpSlotLeaders() {
	log.Info("\n")
	currentEpochID := s.getWorkingEpochID()
	log.Info("dumpSlotLeaders", "currentEpochID", currentEpochID)
	if currentEpochID == 0 {
		return
	}

	for index, value := range s.slotLeadersPtrArray {
		log.Info("dumpSlotLeaders", "index", s.slotLeadersIndex[index], "curSlotLeader", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

}

func (s *SlotLeaderSelection) dumpLocalPublicKey() {
	log.Info("\n")
	localPublicKey, _ := s.getLocalPublicKey()
	log.Info("dumpLocalPublicKey", "current Local publickey", hex.EncodeToString(crypto.FromECDSAPub(localPublicKey)))

}

func (s *SlotLeaderSelection) dumpLocalPublicKeyIndex() {
	log.Info("\n")
	localPublicKey, _ := s.getLocalPublicKey()
	localPublicKeyByte := crypto.FromECDSAPub(localPublicKey)
	log.Info("current Local publickey", "indexs in current epochLeaders", s.epochLeadersMap[hex.EncodeToString(localPublicKeyByte)])

}

func (s *SlotLeaderSelection) buildEpochLeaderGroup(epochID uint64) {
	functrace.Enter()
	// build Array and map
	data := s.getEpochLeaders(epochID)
	for index, value := range data {
		s.epochLeadersArray = append(s.epochLeadersArray, hex.EncodeToString(value))
		s.epochLeadersMap[hex.EncodeToString(value)] = append(s.epochLeadersMap[hex.EncodeToString(value)], uint64(index))
		s.epochLeadersPtrArray[index] = crypto.ToECDSAPub(value)
	}
	functrace.Exit()
}

func (s *SlotLeaderSelection) GetSlotCreateStatusByEpochID(epochID uint64) bool {
	_, ok := s.slotCreateStatus[epochID]
	return ok
}

func (s *SlotLeaderSelection) GetSlotLeaders(epochID uint64) (slotLeaders []*ecdsa.PublicKey, err error) {

	_, ok := s.slotCreateStatus[epochID]
	if !ok {
		return nil, errors.New("slot leaders group not ready")
	}

	if len(s.slotLeadersPtrArray) != pos.SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	return s.slotLeadersPtrArray[:], nil
}

func (s *SlotLeaderSelection) GetSlotLeader(epochID uint64, slotID uint64) (slotLeader *ecdsa.PublicKey, err error) {
	_, ok := s.slotCreateStatus[epochID]
	if !ok {
		return nil, errors.New("slot leaders group not ready")
	}
	if len(s.slotLeadersPtrArray) != pos.SlotCount {
		return nil, errors.New("slot leaders group data is not integrated")
	}
	if slotID >= pos.SlotCount {
		return nil, errors.New("slot id index out of range")
	}
	return s.slotLeadersPtrArray[slotID], nil
}

// from random proposer
func (s *SlotLeaderSelection) getRandom(epochID uint64) (ret *big.Int, err error) {
	stateDb, err := s.getCurrentStateDb()
	if err != nil {
		log.Error("SlotLeaderSelection.getRandom getStateDb return error, use a default value", "epochID", epochID)
		rb := big.NewInt(1)
		return rb, nil
	}

	rb := vm.GetR(stateDb, epochID)
	if rb == nil {
		log.Error("vm.GetR return nil, use a default value", "epochID", epochID)
		rb = big.NewInt(1)
	}
	return rb, nil
}

// getSMAPieces can get the SMA info generate in pre epoch.
// It had been +1 when save into db, so do not -1 in get.
func (s *SlotLeaderSelection) getSMAPieces(epochID uint64) (ret []*ecdsa.PublicKey, isGenesis bool, err error) {
	// 1. get SMA[pre]
	piecesPtr := make([]*ecdsa.PublicKey, 0)
	if epochID == uint64(0) {
		return s.smaGenesis[:], true, nil
	} else {
		// pieces: alpha[1]*G, alpha[2]*G, .....
		pieces, err := posdb.GetDb().Get(epochID, SecurityMsg)
		if err != nil {
			log.Warn("getSMAPieces error use epoch 0 SMA", "epochID", epochID, "SecurityMsg", SecurityMsg)
			return s.smaGenesis[:], true, nil
		}

		piecesCount := len(pieces) / LengthPublicKeyBytes
		var pubKeyByte []byte
		for i := 0; i < piecesCount; i++ {
			if i < piecesCount-1 {
				pubKeyByte = pieces[i*LengthPublicKeyBytes : (i+1)*LengthPublicKeyBytes]
			} else {
				pubKeyByte = pieces[i*LengthPublicKeyBytes:]
			}
			piecesPtr = append(piecesPtr, crypto.ToECDSAPub(pubKeyByte))
		}
		return piecesPtr, false, nil
	}
}

func (s *SlotLeaderSelection) GetSma(epochID uint64) (ret []*ecdsa.PublicKey, isGenesis bool, err error) {
	return s.getSMAPieces(epochID)
}

func (s *SlotLeaderSelection) generateSlotLeadsGroup(epochID uint64) error {
	functrace.Enter()

	epochIDGet := epochID

	// get pre sma
	piecesPtr, isGenesis, _ := s.getSMAPieces(epochIDGet)

	canBeContinue, err := s.isLocalPkInPreEpochLeaders(epochID)
	if !canBeContinue {
		log.Warn("Local node is not in pre epoch leaders at generateSlotLeadsGroup", "epochID", epochID)
		return nil
	}

	if (err != nil && epochID > 1) || isGenesis {
		log.Warn("Can not find pre epoch SMA or not in Pre epoch leaders, use epoch 0.", "curEpochID", epochID, "preEpochID", epochID-1)
		epochIDGet = 0

	}

	// get random
	random, err := s.getRandom(epochIDGet)
	if err != nil {
		return errors.New("get random message error")
	}
	log.Info("generateSlotLeadsGroup", "Random got", hex.EncodeToString(random.Bytes()))

	// 5. return slot leaders pointers.
	slotLeadersPtr := make([]*ecdsa.PublicKey, 0)
	var epochLeadersPtrArray []*ecdsa.PublicKey
	if epochIDGet == 0 {
		epochLeadersPtrArray = s.getEpoch0LeadersPK()
	} else {
		epochLeadersPtrArray, err = s.getPreEpochLeadersPK(epochIDGet)
		if err != nil {
			log.Warn(err.Error())
		}
	}

	if len(epochLeadersPtrArray) != pos.EpochLeaderCount {
		return fmt.Errorf("fail to get epochLeader:%d", epochIDGet)
	}

	for i := 0; i < len(piecesPtr); i++ {
		ret := crypto.S256().IsOnCurve(piecesPtr[i].X, piecesPtr[i].Y)
		if !ret {
			return errors.New("piecesPtr is not on curve")
		}
	}
	for i := 0; i < pos.EpochLeaderCount; i++ {
		ret := crypto.S256().IsOnCurve(epochLeadersPtrArray[i].X, epochLeadersPtrArray[i].Y)
		if !ret {
			return errors.New("epochLeaders pk is not on curve")
		}
	}
	log.Info("Before GenerateSlotLeaderSeq")
	s.dumpData()

	slotLeadersPtr, _, slotLeadersIndex, err := uleaderselection.GenerateSlotLeaderSeqAndIndex(piecesPtr[:], epochLeadersPtrArray[:], random.Bytes(), pos.SlotCount, epochID)

	if err != nil {
		log.Error("generateSlotLeadsGroup", "error", err.Error())
		return err
	}

	// insert slot address to local DB
	for index, val := range slotLeadersPtr {
		_, err = posdb.GetDb().PutWithIndex(uint64(epochID), uint64(index), SlotLeader, crypto.FromECDSAPub(val))
		if err != nil {
			log.Error("generateSlotLeadsGroup:PutWithIndex", "error", err.Error())
			return err
		}
	}

	for index, val := range slotLeadersPtr {
		s.slotLeadersPtrArray[index] = val
	}

	for index, value := range slotLeadersIndex {
		s.slotLeadersIndex[index] = value
	}

	s.slotCreateStatus[epochID] = true
	log.Info("GenerateSlotLeaderSeq success")

	s.dumpData()
	return nil
}

func (s *SlotLeaderSelection) inEpochLeadersOrNot(pkIndex uint64, pkBytes []byte) bool {
	return (pkIndex < uint64(len(s.epochLeadersArray))) && (hex.EncodeToString(pkBytes) == s.epochLeadersArray[pkIndex])
}

func (s *SlotLeaderSelection) isLocalPkInCurrentEpochLeaders() bool {
	selfPublicKey, _ := s.getLocalPublicKey()
	var inEpochLeaders bool
	_, inEpochLeaders = s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey))]
	if inEpochLeaders {
		return true
	}
	return false
}

func (s *SlotLeaderSelection) verifySecurityPiece(index uint64) (valid bool, err error) {
	// MI == AlphaiPki
	if !uleaderselection.PublicKeyEqual(s.stageOneMi[index], s.stageTwoAlphaPKi[index][index]) {
		return false, errors.New("stageOneMi is not equal sageTwoAlphaPki")
	} else {
		log.Debug(fmt.Sprintln("VerifyDleqProof: pk:", s.epochLeadersPtrArray[:], " , alphaPK:", s.stageTwoAlphaPKi[index][:], ", stageTwoProof:", s.stageTwoProof[index][:]))
		// verify proof[index]

		for _, value := range s.epochLeadersPtrArray {
			log.Debug("verifySecurityPiece:VerifyDleqProof", "index", index, "epochLeader", hex.EncodeToString(crypto.FromECDSAPub(value)))
		}

		for _, valueAplaPk := range s.stageTwoAlphaPKi[index] {
			log.Debug("verifySecurityPiece:VerifyDleqProof", "index", index, "alphaPK", hex.EncodeToString(crypto.FromECDSAPub(valueAplaPk)))
		}

		for _, valueStage2Proof := range s.stageTwoProof[index] {
			log.Debug("verifySecurityPiece:VerifyDleqProof", "index", index, "stg2Proof", hex.EncodeToString(valueStage2Proof.Bytes()))
		}

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
		log.Warn(fmt.Sprintf("%v not in epoch leaders", hex.EncodeToString(crypto.FromECDSAPub(selfPk))))
		return nil, nil
	}

	selfPkRecievedPicesMap := make(map[uint64][]*ecdsa.PublicKey, 0)
	for _, selfIndex := range indexs {
		for i := 0; i < len(s.epochLeadersArray); i++ {
			if (s.stageTwoAlphaPKi[i][selfIndex] != nil) && (s.validEpochLeadersIndex[i]) {
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

func (s *SlotLeaderSelection) getStage2TxAlphaPki(epochID uint64, selfIndex uint64) (alphaPkis []*ecdsa.PublicKey, proofs []*big.Int, err error) {
	stateDb, err := s.getCurrentStateDb()
	if err != nil {
		return nil, nil, err
	}

	slotLeaderPrecompileAddr := vm.GetSlotLeaderSCAddress()

	keyHash := vm.GetSlotLeaderStage2KeyHash(posdb.Uint64ToBytes(epochID), posdb.Uint64ToBytes(selfIndex))

	log.Debug(fmt.Sprintf("try to get stateDB addr:%s, key:%s", slotLeaderPrecompileAddr.Hex(), keyHash.Hex()))

	data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if data == nil {
		return nil, nil, errors.New("getStage2TxAlphaPki can not find from statedb:" + fmt.Sprintf("addr:%s, key:%s, epochID:%d, selfIndex:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), epochID, selfIndex))
	}

	epID, slfIndex, _, alphaPki, proof, err := slottools.RlpUnpackStage2DataForTx(data, vm.GetSlotLeaderScAbiString())
	if err != nil {
		return nil, nil, err
	}

	if epID != epochID || slfIndex != selfIndex {
		return nil, nil, errors.New("Verify failed, epID != epochID || slfIndex != selfIndex in getStage2TxAlphaPki")
	}

	return alphaPki, proof, nil
}

func (s *SlotLeaderSelection) collectStagesData(epochID uint64) (err error) {
	for i := 0; i < pos.EpochLeaderCount; i++ {
		mi, err := s.getStg1StateDbInfo(epochID, uint64(i))
		if err != nil {
			log.Warn("getStg1StateDbInfo", "error", err.Error(), "index", i)
			s.validEpochLeadersIndex[i] = false
		} else {
			if len(mi) == 0 {
				log.Warn("getStg1StateDbInfo", "error", "len(mi)=0", "index", i)
				s.validEpochLeadersIndex[i] = false
			} else {
				s.stageOneMi[i] = crypto.ToECDSAPub(mi)
			}
		}

		if !s.validEpochLeadersIndex[i] {
			continue
		}

		alphaPki, proof, err := s.getStage2TxAlphaPki(epochID, uint64(i))
		if err != nil {
			log.Warn("getStage2TxAlphaPki", "error", err.Error(), "index", i)
			s.validEpochLeadersIndex[i] = false
			continue
		}

		if (len(alphaPki) != pos.EpochLeaderCount) || (len(proof) != StageTwoProofCount) {
			log.Warn("getStage2TxAlphaPki", "error", "len(alphaPkis) or len(proofs) is wrong.", "index", i)
			s.validEpochLeadersIndex[i] = false
		} else {
			for j := 0; j < pos.EpochLeaderCount; j++ {
				s.stageTwoAlphaPKi[i][j] = alphaPki[j]
			}

			for j := 0; j < StageTwoProofCount; j++ {
				s.stageTwoProof[i][j] = proof[j]
			}
		}
	}
	return nil
}

// create security message SMA and insert into localDB
func (s *SlotLeaderSelection) generateSecurityMsg(epochID uint64, PrivateKey *ecdsa.PrivateKey) error {
	if !s.isLocalPkInCurrentEpochLeaders() {
		log.Debug("generateSecurityMsg", "input public key", hex.EncodeToString(crypto.FromECDSAPub(&PrivateKey.PublicKey)))
		return errors.New("local public key is not in current Epoch leaders")
	}
	// collect data
	err := s.collectStagesData(epochID)
	if err != nil {
		return errors.New("collect stage data error!")
	}
	// verify security pieces
	for i := 0; i < pos.EpochLeaderCount; i++ {
		valid, errVSP := s.verifySecurityPiece(uint64(i))
		if !valid {
			log.Warn("generateSecurityMsg", "epochID", epochID, "index", i, "verifySecurityPiece error", errVSP.Error())
			s.validEpochLeadersIndex[i] = false
		}
	}

	// build security self pieces. alpha1*pki, alpha2*pk2, alpha3*pk3....
	ArrayPiece, err := s.buildSecurityPieces(epochID)
	if err != nil {
		log.Warn("generateSecurityMsg:buildSecurityPieces", "error", err.Error())
		return err
	}

	log.Info("generateSecurityMsg", "len(ArrayPiece)", len(ArrayPiece))

	smasPtr := make([]*ecdsa.PublicKey, 0)
	var smasBytes bytes.Buffer

	smasPtr, err = uleaderselection.GenerateSMA(PrivateKey, ArrayPiece)
	if err != nil {
		log.Warn("generateSecurityMsg:GenerateSMA", "error", err.Error())
		return err
	}
	for _, value := range smasPtr {
		smasBytes.Write(crypto.FromECDSAPub(value))
		log.Debug(fmt.Sprintf("epochID+1 = %d set security message is %v\n", epochID+1, hex.EncodeToString(crypto.FromECDSAPub(value))))
	}
	_, err = posdb.GetDb().Put(uint64(epochID+1), SecurityMsg, smasBytes.Bytes())
	if err != nil {
		log.Warn("generateSecurityMsg:Put", "error", err.Error())
		return err
	}

	log.Info("generateSecurityMsg", "Generate SMA Success.", "epochID+1", epochID+1, "len(SMA)", len(smasPtr))

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

func (s *SlotLeaderSelection) buildStage2TxPayload(epochID uint64, selfIndex uint64) ([]byte, error) {
	var selfPk *ecdsa.PublicKey
	var err error
	if pos.SelfTestMode {
		selfPk = s.epochLeadersPtrArray[selfIndex]
	} else {
		selfPk, err = s.getLocalPublicKey()
		if err != nil {
			return nil, err
		}
	}

	alphaPki, proof, err := s.buildArrayPiece(epochID, selfIndex)
	if err != nil {
		return nil, err
	}

	buf, err := slottools.RlpPackStage2DataForTx(epochID, selfIndex, selfPk, alphaPki, proof, vm.GetSlotLeaderScAbiString())

	return buf, err
}

func (s *SlotLeaderSelection) setCurrentWorkStage(workStage int) {
	currentEpochID := s.getWorkingEpochID()
	s.setWorkStage(currentEpochID, workStage)
}

func (s *SlotLeaderSelection) getWorkingEpochID() uint64 {
	ret, err := posdb.GetDb().Get(0, "slotLeaderCurrentSlotID")
	if err != nil {
		if err.Error() == "leveldb: not found" {
			posdb.GetDb().Put(0, "slotLeaderCurrentSlotID", postools.Uint64ToBytes(0))
			return 0
		}
	}
	retUint64 := postools.BytesToUint64(ret)
	return retUint64
}

func (s *SlotLeaderSelection) setWorkingEpochID(workingEpochID uint64) error {
	_, err := posdb.GetDb().Put(0, "slotLeaderCurrentSlotID", posdb.Uint64ToBytes(workingEpochID))
	return err
}

// getStg1StateDbInfo can get data from StateDB the pk and mi are in 65 bytes length uncompress format
func (s *SlotLeaderSelection) getStg1StateDbInfo(epochID uint64, index uint64) (mi []byte, err error) {
	stateDb, err := s.getCurrentStateDb()
	if err != nil {
		return nil, err
	}

	slotLeaderPrecompileAddr := vm.GetSlotLeaderSCAddress()
	keyHash := vm.GetSlotLeaderStage1KeyHash(posdb.Uint64ToBytes(epochID), posdb.Uint64ToBytes(index))

	// Read and Verify
	readBuf := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if readBuf == nil {
		return nil, errors.New("getStg1StateDbInfo: Found not data of key")
	}

	epID, idxID, miPoint, err := slottools.RlpUnpackStage1DataForTx(readBuf, vm.GetSlotLeaderScAbiString())
	if err != nil {
		return nil, errors.New("getStg1StateDbInfo: RlpUnpackStage1DataForTx error")
	}
	mi = crypto.FromECDSAPub(miPoint)
	//pk and mi is 65 bytes length

	if epID == epochID &&
		idxID == index &&
		err == nil {
		return
	}

	return nil, errors.New("Stg1 data get from StateDb verified failed")
}

func ErrorCountAdd() {
	ErrorCount++
}

func WarnCountAdd() {
	WarnCount++
}
