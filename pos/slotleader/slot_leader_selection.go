package slotleader

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util/convert"
	"math/big"
	"time"

	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/slotleader/slottools"

	"github.com/wanchain/go-wanchain/rpc"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/uleaderselection"
)

//CompressedPubKeyLen means a compressed public key byte len.
const lengthPublicKeyBytes = 65
const (
	StageTwoProofCount = 2
	EpochLeaders       = "epochLeaders"
	SecurityMsg        = "securityMsg"
	CR                 = "cr"
	SlotLeader         = "slotLeader"
)
const (
	slotLeaderSelectionInit = iota + 1 //1
	//Ready to start slot leader selection stage1
	slotLeaderSelectionStage1 = iota + 1 //2
	//Slot leader selection stage1 finish
	slotLeaderSelectionStage2        = iota + 1 //3
	slotLeaderSelectionStage3        = iota + 1 //4
	slotLeaderSelectionStageFinished = iota + 1 //5
)

var (
	errorRetry = 3
)
var (
	curEpochId = uint64(0)
	curSlotId  = uint64(0)
)

//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	workingEpochID         uint64
	workStage              int
	rc                     *rpc.Client
	epochLeadersArray      []string            // len(pki)=65 hex.EncodeToString
	epochLeadersMap        map[string][]uint64 // key: pki value: []uint64 the indexes of this pki. hex.EncodeToString
	key                    *keystore.Key
	stateDb                *state.StateDB
	epochInstance          interface{}
	slotLeadersPtrArray    [posconfig.SlotCount]*ecdsa.PublicKey
	slotLeadersIndex       [posconfig.SlotCount]uint64
	epochLeadersPtrArray   [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	validEpochLeadersIndex [posconfig.EpochLeaderCount]bool // true: can be used to slot leader false: can not be used to slot leader
	stageOneMi             [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoAlphaPKi       [posconfig.EpochLeaderCount][posconfig.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoProof          [posconfig.EpochLeaderCount][StageTwoProofCount]*big.Int //[0]: e; [1]:Z
	slotCreateStatus       map[uint64]bool
	blockChain             *core.BlockChain

	epochLeadersPtrArrayGenesis [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	stageOneMiGenesis           [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoAlphaPKiGenesis     [posconfig.EpochLeaderCount][posconfig.EpochLeaderCount]*ecdsa.PublicKey
	stageTwoProofGenesis        [posconfig.EpochLeaderCount][StageTwoProofCount]*big.Int //[0]: e; [1]:Z
	randomGenesis               *big.Int
	smaGenesis                  [posconfig.EpochLeaderCount]*ecdsa.PublicKey
	sendTransactionFn           SendTxFn
}

var slotLeaderSelection *SlotLeaderSelection

// Pack is use to pack info for slot proof
type Pack struct {
	Proof    [][]byte
	ProofMeg [][]byte
}

func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}
func (s *SlotLeaderSelection) GetLocalPublicKey() (*ecdsa.PublicKey, error) {
	return s.getLocalPublicKey()
}

func GetEpochSlotID() (uint64, uint64) {
	return curEpochId, curSlotId
}
func CalEpochSlotID() {
	if posconfig.EpochBaseTime == 0 {
		return
	}
	timeUnix := uint64(time.Now().Unix())
	epochTimeSpan := uint64(posconfig.SlotTime * posconfig.SlotCount)
	curEpochId = uint64((timeUnix - posconfig.EpochBaseTime) / epochTimeSpan)
	curSlotId = uint64((timeUnix - posconfig.EpochBaseTime) / posconfig.SlotTime % posconfig.SlotCount)
	fmt.Println("CalEpochSlotID:", curEpochId, curSlotId)
}
func (s *SlotLeaderSelection) GetEpochLeadersPK(epochID uint64) []*ecdsa.PublicKey {
	return s.getEpochLeadersPK(epochID)
}

func (s *SlotLeaderSelection) GetSlotCreateStatusByEpochID(epochID uint64) bool {
	_, ok := s.slotCreateStatus[epochID]
	return ok
}

func (s *SlotLeaderSelection) GetSlotLeader(epochID uint64, slotID uint64) (slotLeader *ecdsa.PublicKey, err error) {
	if epochID == 0 {
		b, err := hex.DecodeString(posconfig.GenesisPK)
		if err != nil {
			return nil, slottools.ErrInvalidGenesisPk
		}
		return crypto.ToECDSAPub(b), nil
	}
	_, ok := s.slotCreateStatus[epochID]
	if !ok {
		return nil, slottools.ErrSlotLeaderGroupNotReady
	}
	if len(s.slotLeadersPtrArray) != posconfig.SlotCount {
		return nil, slottools.ErrSlotLeaderGroupNotReady
	}
	if slotID >= posconfig.SlotCount {
		return nil, slottools.ErrSlotIDOutOfRange
	}
	return s.slotLeadersPtrArray[slotID], nil
}

// GetSma uses to get SMA information of the epoch.
func (s *SlotLeaderSelection) GetSma(epochID uint64) (ret []*ecdsa.PublicKey, isGenesis bool, err error) {
	return s.getSMAPieces(epochID)
}

func (s *SlotLeaderSelection) GetStage2TxAlphaPki(epochID uint64, selfIndex uint64) (alphaPkis []*ecdsa.PublicKey, proofs []*big.Int, err error) {
	stateDb, err := s.getCurrentStateDb()
	if err != nil {
		return nil, nil, err
	}

	slotLeaderPrecompileAddr := vm.GetSlotLeaderSCAddress()

	keyHash := vm.GetSlotLeaderStage2KeyHash(convert.Uint64ToBytes(epochID), convert.Uint64ToBytes(selfIndex))

	data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if data == nil {
		log.Debug(fmt.Sprintf("try to get stateDB addr:%s, key:%s", slotLeaderPrecompileAddr.Hex(), keyHash.Hex()))
		return nil, nil, slottools.ErrNoTx2TransInDB
	}

	epID, slfIndex, _, alphaPki, proof, err := slottools.RlpUnpackStage2DataForTx(data, vm.GetSlotLeaderScAbiString())
	if err != nil {
		return nil, nil, err
	}

	if epID != epochID || slfIndex != selfIndex {
		return nil, nil, slottools.ErrRlpUnpackErr
	}

	return alphaPki, proof, nil
}

func (s *SlotLeaderSelection) GetSlotLeaderStage2TxIndexes(epochID uint64) (indexesSentTran []bool, err error) {
	var ret [posconfig.EpochLeaderCount]bool
	stateDb, err := s.getCurrentStateDb()
	if err != nil {
		return ret[:], err
	}

	slotLeaderPrecompileAddr := vm.GetSlotLeaderSCAddress()

	keyHash := vm.GetSlotLeaderStage2IndexesKeyHash(convert.Uint64ToBytes(epochID))

	log.Debug(fmt.Sprintf("GetSlotLeaderStage2TxIndexes:try to get stateDB addr:%s, key:%s", slotLeaderPrecompileAddr.Hex(), keyHash.Hex()))

	data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)

	if data == nil {
		return ret[:], slottools.ErrNoTx2TransInDB
	}

	err = rlp.DecodeBytes(data, &ret)
	if err != nil {
		return ret[:], slottools.ErrNoTx2TransInDB
	}
	return ret[:], nil
}

// GetStg1StateDbInfo can get data from StateDB the pk and mi are in 65 bytes length un compress format
func (s *SlotLeaderSelection) GetStg1StateDbInfo(epochID uint64, index uint64) (mi []byte, err error) {
	stateDb, err := s.getCurrentStateDb()
	if err != nil {
		return nil, err
	}

	slotLeaderPrecompileAddr := vm.GetSlotLeaderSCAddress()
	keyHash := vm.GetSlotLeaderStage1KeyHash(convert.Uint64ToBytes(epochID), convert.Uint64ToBytes(index))

	// Read and Verify
	readBuf := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if readBuf == nil {
		return nil, slottools.ErrNoTx1TransInDB
	}

	epID, idxID, miPoint, err := slottools.RlpUnpackStage1DataForTx(readBuf, vm.GetSlotLeaderScAbiString())
	if err != nil {
		return nil, slottools.ErrRlpUnpackErr
	}
	mi = crypto.FromECDSAPub(miPoint)
	//pk and mi is 65 bytes length

	if epID == epochID &&
		idxID == index &&
		err == nil {
		return
	}

	return nil, slottools.ErrVerifyStg1Data
}

func init() {
	slotLeaderSelection = &SlotLeaderSelection{}
	slotLeaderSelection.epochLeadersMap = make(map[string][]uint64)
	slotLeaderSelection.epochLeadersArray = make([]string, 0)
	slotLeaderSelection.slotCreateStatus = make(map[uint64]bool)
	slottools.SetSlotLeaderInst(slotLeaderSelection)
	s := slotLeaderSelection
	s.randomGenesis = big.NewInt(1)
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

	for i := 0; i < posconfig.EpochLeaderCount; i++ {

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

		for j := 0; j < posconfig.EpochLeaderCount; j++ {
			// AlphaIPki stage2Genesis
			alphaIPkj := new(ecdsa.PublicKey)
			alphaIPkj.Curve = crypto.S256()
			alphaIPkj.X, alphaIPkj.Y = crypto.S256().ScalarMult(s.epochLeadersPtrArrayGenesis[j].X, s.epochLeadersPtrArrayGenesis[j].Y, alphas[i].Bytes())

			s.stageTwoAlphaPKiGenesis[i][j] = alphaIPkj
		}

	}

	epochLeadersPreHexStr := make([]string, 0)
	for _, value := range s.epochLeadersPtrArrayGenesis {
		epochLeadersPreHexStr = append(epochLeadersPreHexStr, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	log.Debug("slot_leader_selection:init", "genesis epoch leaders", epochLeadersPreHexStr)

	smaPiecesHexStr := make([]string, 0)
	for _, value := range s.smaGenesis {
		smaPiecesHexStr = append(smaPiecesHexStr, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	log.Debug("slot_leader_selection:init", "genesis sma pieces", smaPiecesHexStr)

}

func (s *SlotLeaderSelection) getAlpha(epochID uint64, selfIndex uint64) (*big.Int, error) {
	if posconfig.SelfTestMode {
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

func (s *SlotLeaderSelection) getLocalPublicKey() (*ecdsa.PublicKey, error) {
	if s.key == nil || s.key.PrivateKey == nil {
		return nil, slottools.ErrInvalidLocalPublicKey
	}
	return &s.key.PrivateKey.PublicKey, nil
}

func (s *SlotLeaderSelection) getLocalPrivateKey() (*ecdsa.PrivateKey, error) {
	return s.key.PrivateKey, nil
}

func (s *SlotLeaderSelection) getEpochLeaders(epochID uint64) [][]byte {
	//test := false
	if posconfig.SelfTestMode {
		//test: generate test publicKey
		epochLeaderAllBytes, err := posdb.GetDb().Get(epochID, EpochLeaders)
		if err != nil {
			return nil
		}
		piecesCount := len(epochLeaderAllBytes) / lengthPublicKeyBytes
		ret := make([][]byte, 0)
		var pubKeyByte []byte
		for i := 0; i < piecesCount; i++ {
			if i < piecesCount-1 {
				pubKeyByte = epochLeaderAllBytes[i*lengthPublicKeyBytes : (i+1)*lengthPublicKeyBytes]
			} else {
				pubKeyByte = epochLeaderAllBytes[i*lengthPublicKeyBytes:]
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
		return s.getEpoch0LeadersPK(), slottools.ErrInvalidPreEpochLeaders
	}

	return pks, nil
}

func (s *SlotLeaderSelection) getEpoch0LeadersPK() []*ecdsa.PublicKey {
	pks := make([]*ecdsa.PublicKey, posconfig.EpochLeaderCount)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		pkBuf, err := hex.DecodeString(posconfig.GenesisPK)
		if err != nil {
			panic("posconfig.GenesisPK is Error")
		}
		pks[i] = crypto.ToECDSAPub(pkBuf)
	}
	return pks
}

// isLocalPkInPreEpochLeaders check if local pk is in pre epoch leader.
// If get pre epoch leader length is 0, return true,err to use epoch 0 info
func (s *SlotLeaderSelection) isLocalPkInPreEpochLeaders(epochID uint64) (canBeContinue bool, err error) {

	localPk, err := s.getLocalPublicKey()
	if err != nil {
		log.Error("SlotLeaderSelection.IsLocalPkInPreEpochLeaders getLocalPublicKey error", "error", err)
		panic("SlotLeaderSelection.IsLocalPkInPreEpochLeaders getLocalPublicKey error")
	}

	if epochID == 0 {
		for _, value := range s.epochLeadersPtrArrayGenesis {
			if util.PkEqual(localPk, value) {
				return true, nil
			}
		}
		return false, nil
	}

	prePks, err := s.getPreEpochLeadersPK(epochID)
	if err != nil {
		return true, slottools.ErrInvalidPreEpochLeaders
	}

	for i := 0; i < len(prePks); i++ {
		if util.PkEqual(localPk, prePks[i]) {
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
		}
		log.Error("getWorkStage error: " + err.Error())
		panic("getWorkStage error")
	}
	workStageUint64 := convert.BytesToUint64(ret)
	return int(workStageUint64)
}

//saveWorkStage save the work stage of epochID in levelDB
func (s *SlotLeaderSelection) setWorkStage(epochID uint64, workStage int) error {
	workStageBig := big.NewInt(int64(workStage))
	_, err := posdb.GetDb().Put(epochID, "slotLeaderWorkStage", workStageBig.Bytes())
	return err
}

func (s *SlotLeaderSelection) clearData() {
	s.epochLeadersArray = make([]string, 0)
	s.epochLeadersMap = make(map[string][]uint64)

	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		s.epochLeadersPtrArray[i] = nil
		s.validEpochLeadersIndex[i] = true

		s.stageOneMi[i] = nil

		for j := 0; j < posconfig.EpochLeaderCount; j++ {
			s.stageTwoAlphaPKi[i][j] = nil
		}
		for k := 0; k < StageTwoProofCount; k++ {
			s.stageTwoProof[i][k] = nil
		}
	}

	for i := 0; i < posconfig.SlotCount; i++ {
		s.slotLeadersPtrArray[i] = nil
	}

	for i := 0; i < posconfig.SlotCount; i++ {
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
	log.Debug("\n")
	currentEpochID := s.getWorkingEpochID()
	log.Debug("dumpPreEpochLeaders", "currentEpochID", curEpochId)
	if currentEpochID == 0 {
		return
	}

	preEpochLeaders := s.getEpochLeaders(currentEpochID - 1)
	for i := 0; i < len(preEpochLeaders); i++ {
		log.Debug("dumpPreEpochLeaders", "index", i, "preEpochLeader", hex.EncodeToString(preEpochLeaders[i]))
	}

	log.Debug("\n")
}
func (s *SlotLeaderSelection) dumpCurrentEpochLeaders() {
	log.Debug("\n")
	currentEpochID := s.getWorkingEpochID()
	log.Debug("dumpCurrentEpochLeaders", "currentEpochID", currentEpochID)
	if currentEpochID == 0 {
		return
	}

	for index, value := range s.epochLeadersPtrArray {
		log.Debug("dumpCurrentEpochLeaders", "index", index, "curEpochLeader", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

}

func (s *SlotLeaderSelection) dumpSlotLeaders() {
	log.Debug("\n")
	currentEpochID := s.getWorkingEpochID()
	log.Debug("dumpSlotLeaders", "currentEpochID", currentEpochID)
	if currentEpochID == 0 {
		return
	}

	for index, value := range s.slotLeadersPtrArray {
		log.Debug("dumpSlotLeaders", "index", s.slotLeadersIndex[index], "curSlotLeader", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

}

func (s *SlotLeaderSelection) dumpLocalPublicKey() {
	log.Debug("\n")
	localPublicKey, _ := s.getLocalPublicKey()
	log.Debug("dumpLocalPublicKey", "current Local publickey", hex.EncodeToString(crypto.FromECDSAPub(localPublicKey)))

}

func (s *SlotLeaderSelection) dumpLocalPublicKeyIndex() {
	log.Debug("\n")
	localPublicKey, _ := s.getLocalPublicKey()
	localPublicKeyByte := crypto.FromECDSAPub(localPublicKey)
	log.Debug("current Local publickey", "indexs in current epochLeaders", s.epochLeadersMap[hex.EncodeToString(localPublicKeyByte)])

}

func (s *SlotLeaderSelection) buildEpochLeaderGroup(epochID uint64) {
	functrace.Enter()
	// build Array and map
	data := s.getEpochLeaders(epochID)
	if data == nil {
		log.Error("SlotLeaderSelection", "buildEpochLeaderGroup", "no epoch leaders", "epochID", epochID)
		panic("No epoch leaders")
		return
	}
	for index, value := range data {
		s.epochLeadersArray = append(s.epochLeadersArray, hex.EncodeToString(value))
		s.epochLeadersMap[hex.EncodeToString(value)] = append(s.epochLeadersMap[hex.EncodeToString(value)], uint64(index))
		s.epochLeadersPtrArray[index] = crypto.ToECDSAPub(value)
	}
	functrace.Exit()
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

		piecesCount := len(pieces) / lengthPublicKeyBytes
		var pubKeyByte []byte
		for i := 0; i < piecesCount; i++ {
			if i < piecesCount-1 {
				pubKeyByte = pieces[i*lengthPublicKeyBytes : (i+1)*lengthPublicKeyBytes]
			} else {
				pubKeyByte = pieces[i*lengthPublicKeyBytes:]
			}
			piecesPtr = append(piecesPtr, crypto.ToECDSAPub(pubKeyByte))
		}
		return piecesPtr, false, nil
	}
}

func (s *SlotLeaderSelection) generateSlotLeadsGroup(epochID uint64) error {
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
		return slottools.ErrInvalidRandom
	}
	log.Debug("generateSlotLeadsGroup", "Random got", hex.EncodeToString(random.Bytes()))

	// return slot leaders pointers.
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

	if len(epochLeadersPtrArray) != posconfig.EpochLeaderCount {
		return fmt.Errorf("fail to get epochLeader:%d", epochIDGet)
	}

	for i := 0; i < len(piecesPtr); i++ {
		ret := crypto.S256().IsOnCurve(piecesPtr[i].X, piecesPtr[i].Y)
		if !ret {
			return slottools.ErrNotOnCurve
		}
	}
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		ret := crypto.S256().IsOnCurve(epochLeadersPtrArray[i].X, epochLeadersPtrArray[i].Y)
		if !ret {
			return slottools.ErrNotOnCurve
		}
	}
	slotLeadersPtr, _, slotLeadersIndex, err := uleaderselection.GenerateSlotLeaderSeqAndIndex(piecesPtr[:], epochLeadersPtrArray[:], random.Bytes(), posconfig.SlotCount, epochID)
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
	log.Info("generateSlotLeadsGroup success")

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
	log.Debug("isLocalPkInCurrentEpochLeaders", "local public key:", hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey)))
	log.Debug("isLocalPkInCurrentEpochLeaders", "s.epochLeadersMap:", s.epochLeadersMap)
	return false
}

// create alpha1*pki,alpha1*PKi,alphaN*PKi,...
// used to create security message.
func (s *SlotLeaderSelection) buildSecurityPieces(epochID uint64) (pieces []*ecdsa.PublicKey, err error) {

	selfPk, err := s.getLocalPublicKey()
	if err != nil {
		return nil, err
	}

	indexes, exist := s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(selfPk))]
	if exist == false {
		log.Warn(fmt.Sprintf("%v not in epoch leaders", hex.EncodeToString(crypto.FromECDSAPub(selfPk))))
		return nil, nil
	}

	selfPkReceivePiecesMap := make(map[uint64][]*ecdsa.PublicKey, 0)
	for _, selfIndex := range indexes {
		for i := 0; i < len(s.epochLeadersArray); i++ {
			if (s.stageTwoAlphaPKi[i][selfIndex] != nil) && (s.validEpochLeadersIndex[i]) {
				selfPkReceivePiecesMap[selfIndex] = append(selfPkReceivePiecesMap[selfIndex], s.stageTwoAlphaPKi[i][selfIndex])
			}
		}
	}
	piece := make([]*ecdsa.PublicKey, 0)
	for _, value := range selfPkReceivePiecesMap {
		piece = value
		break
	}
	// the value in selfPk Received Pieces Map should be same,so we can return the first one.
	return piece, nil
}

func (s *SlotLeaderSelection) collectStagesData(epochID uint64) (err error) {
	indexesSentTran, err := s.GetSlotLeaderStage2TxIndexes(epochID)
	log.Debug("collectStagesData", "indexesSentTran", indexesSentTran)
	if err != nil {
		return slottools.ErrCollectTxData
	}
	for i := 0; i < posconfig.EpochLeaderCount; i++ {

		if !indexesSentTran[i] {
			s.validEpochLeadersIndex[i] = false
			continue
		}

		alphaPki, proof, err := s.GetStage2TxAlphaPki(epochID, uint64(i))
		if err != nil {
			log.Warn("GetStage2TxAlphaPki", "error", err.Error(), "index", i)
			s.validEpochLeadersIndex[i] = false
			continue
		}

		if (len(alphaPki) != posconfig.EpochLeaderCount) || (len(proof) != StageTwoProofCount) {
			log.Warn("GetStage2TxAlphaPki", "error", "len(alphaPkis) or len(proofs) is wrong.", "index", i)
			s.validEpochLeadersIndex[i] = false
		} else {
			for j := 0; j < posconfig.EpochLeaderCount; j++ {
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
		return slottools.ErrPkNotInCurrentEpochLeadersGroup
	}
	// collect data
	err := s.collectStagesData(epochID)
	if err != nil {
		return slottools.ErrCollectTxData
	}

	// build security self pieces. alpha1*pki, alpha2*pk2, alpha3*pk3....
	ArrayPiece, err := s.buildSecurityPieces(epochID)
	if err != nil {
		log.Warn("generateSecurityMsg:buildSecurityPieces", "error", err.Error())
		return err
	}

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
	return nil
}

// used for stage2 payload
// stage2 tx payload 1(alpha * Pk1, alpha * Pk2, ..., alpha * Pkn)
// stage2 tx payload 2 proof pai[i]
// []*ecdsa : payload1 []*big.Int payload2

func (s *SlotLeaderSelection) buildArrayPiece(epochID uint64, selfIndex uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {

	// get alpha
	alpha, err := s.getAlpha(epochID, selfIndex)
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
	if posconfig.SelfTestMode {
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
			posdb.GetDb().Put(0, "slotLeaderCurrentSlotID", convert.Uint64ToBytes(0))
			return 0
		}
	}
	retUint64 := convert.BytesToUint64(ret)
	return retUint64
}

func (s *SlotLeaderSelection) setWorkingEpochID(workingEpochID uint64) error {
	_, err := posdb.GetDb().Put(0, "slotLeaderCurrentSlotID", convert.Uint64ToBytes(workingEpochID))
	return err
}
