package slotleader

import (
	"crypto/ecdsa"
	Rand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/uleaderselection"
	"github.com/wanchain/go-wanchain/rpc"
)

var (
	errInvalidCommitParameter = errors.New("invalid input parameters")
)

func (s *SLS) GenerateDefaultSlotLeaders() error {
	//epochLeader := s.GetEpochDefaultLeadersPK(0)
	epochLeader := s.epochLeadersPtrArrayGenesis
	slotLeadersPtr, _, _, err := uleaderselection.GenerateSlotLeaderSeqAndIndex(s.smaGenesis[:],
		epochLeader[:], s.randomGenesis.Bytes(), posconfig.SlotCount, 0)
	if err != nil {
		log.SyslogAlert("generateSlotLeadsGroup", "epochid", 0, "error", err.Error())
		return err
	}

	for index, val := range slotLeadersPtr {
		s.defaultSlotLeadersPtrArray[index] = val
	}
	return nil
}

// Init use to initial slotleader module and input some params.
func (s *SLS) Init(blockChain *core.BlockChain, rc *rpc.Client, key *keystore.Key) {
	s.blockChain = blockChain
	s.rc = rc
	s.key = key
	if blockChain != nil {
		log.Info("SLS init success")
	}

	s.sendTransactionFn = util.SendPosTx
	s.initSma()
	s.GenerateDefaultSlotLeaders()
}

func (s *SLS) initSma() {

	s.randomGenesis = posconfig.GetRandomGenesis()

	epochDefaultLeaders := s.GetEpochDefaultLeadersPK(0)

	for index, value := range epochDefaultLeaders {
		s.epochLeadersPtrArrayGenesis[index] = value
	}

	alphas := make([]*big.Int, 0)
	for _, value := range epochDefaultLeaders {
		tempInt := new(big.Int).SetInt64(0)
		tempInt.SetBytes(crypto.Keccak256(crypto.FromECDSAPub(value)))
		alphas = append(alphas, tempInt)
	}

	for i := 0; i < posconfig.EpochLeaderCount; i++ {

		// AlphaPK  stage1Genesis
		mi0 := new(ecdsa.PublicKey)
		mi0.Curve = crypto.S256()
		mi0.X, mi0.Y = crypto.S256().ScalarMult(s.epochLeadersPtrArrayGenesis[i].X, s.epochLeadersPtrArrayGenesis[i].Y,
			alphas[i].Bytes())
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
			// AlphaIPki stage2Genesis, used to verify genesis proof
			alphaIPkj := new(ecdsa.PublicKey)
			alphaIPkj.Curve = crypto.S256()
			alphaIPkj.X, alphaIPkj.Y = crypto.S256().ScalarMult(s.epochLeadersPtrArrayGenesis[j].X,
				s.epochLeadersPtrArrayGenesis[j].Y, alphas[i].Bytes())

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

	log.Debug("initSma:init", "genesis sma pieces", smaPiecesHexStr)
	log.SyslogInfo("SLS initSma success")
}

//Loop check work every Slot time. Called by backend loop.
//It's all slotLeaderSelection's main workflow loop.
//It does not loop at all, it is loop called by the backend.
func (s *SLS) Loop(rc *rpc.Client, key *keystore.Key, epochID uint64, slotID uint64) {
	s.rc = rc
	s.key = key

	log.Info("Now epchoID and slotID:", "epochID", convert.Uint64ToString(epochID), "slotID",
		convert.Uint64ToString(slotID))
	log.Info("Last on chain epchoID and slotID:", "epochID", s.getLastEpochIDFromChain(), "slotID",
		s.getLastSlotIDFromChain())

	//Check if epoch is new
	s.checkNewEpochStart(epochID)
	workStage := s.getWorkStage(epochID)

	//If the gwan restart, try to recover the epoch leader and slot leader
	if workStage != slotLeaderSelectionInit && workStage != slotLeaderSelectionStageFinished {
		if !s.isEpochLeaderMapReady() {
			s.doInit(epochID)
			if !s.isEpochLeaderMapReady() {
				s.setWorkStage(epochID, slotLeaderSelectionStageFinished)
			}
		}
	}

	switch workStage {
	case slotLeaderSelectionInit:
		s.doInit(epochID)
		if !s.isLocalPkInCurrentEpochLeaders() {
			s.setWorkStage(epochID, slotLeaderSelectionStageFinished)
		} else {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
		}

	case slotLeaderSelectionStage1:
		if slotID < (posconfig.Sma1Start + 1) {
			break
		}

		if slotID > (posconfig.Sma1End - 1) {
			s.setWorkStage(epochID, slotLeaderSelectionStage3)
			log.Warn("Passed the moment of slotLeaderSelectionStage1 wait for next epoch", "epochID",
				epochID, "slotID", slotID)
			break
		}

		err := s.startStage1Work()
		if err != nil {
			log.SyslogErr(err.Error())
			s.setWorkStage(epochID, slotLeaderSelectionStage3)
		} else {
			s.setWorkStage(epochID, slotLeaderSelectionStage2)
		}
	case slotLeaderSelectionStage2:
		if slotID < (posconfig.Sma2Start + 1) {
			break
		}

		if slotID > (posconfig.Sma2End - 1) {
			s.setWorkStage(epochID, slotLeaderSelectionStage3)
			break
		}

		go doStage2Work()
		s.setWorkStage(epochID, slotLeaderSelectionStage3)
	case slotLeaderSelectionStage3:
		if slotID < posconfig.Sma3Start {
			break
		}

		err := s.generateSecurityMsg(epochID, s.key.PrivateKey)
		if err != nil {
			log.Warn(err.Error())
		} else {
			log.Info("generateSecurityMsg SMA success!")
		}

		if err != nil && errorRetry > 0 {
			errorRetry--
			break
		}

		s.setWorkStage(epochID, slotLeaderSelectionStageFinished)
		errorRetry = 3
	case slotLeaderSelectionStageFinished:

	default:
	}
}

// doInit is used for init in each epoch
func (s *SLS) doInit(epochID uint64) {
	s.clearData()
	s.buildEpochLeaderGroup(epochID)
	s.setWorkingEpochID(epochID)
}

func (s *SLS) startStage1Work() error {
	selfPublicKey, err := s.getLocalPublicKey()
	if err != nil {
		return err
	}

	selfPublicKeyIndex, inEpochLeaders := s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey))]
	if inEpochLeaders {
		log.Debug(fmt.Sprintf("Local node in epoch leaders times: %d", len(selfPublicKeyIndex)))

		workingEpochID := s.getWorkingEpochID()

		for i := 0; i < len(selfPublicKeyIndex); i++ {
			data, err := s.generateCommitment(selfPublicKey, workingEpochID, selfPublicKeyIndex[i])
			if err != nil {
				log.Error("generateCommitment error", "error", err.Error())
				continue
			}
			err = s.sendSlotTx(data, s.sendTransactionFn)
			if err != nil {
				log.Error("sendSlotTx error", "error", err.Error())
				continue
			}
		}
	} else {
		log.Debug("Local node is not in epoch leaders")
	}
	return nil
}

func doStage2Work() {
	s := GetSlotLeaderSelection()
	err := s.startStage2Work()
	if err != nil {
		log.Error(err.Error())
	}
}

func (s *SLS) startStage2Work() error {
	functrace.Enter("startStage2Work")
	s.getWorkingEpochID()
	selfPublicKey, err := s.getLocalPublicKey()
	if err != nil {
		return err
	}
	selfPublicKeyIndex := make([]uint64, 0)
	var inEpochLeaders bool
	selfPublicKeyIndex, inEpochLeaders = s.epochLeadersMap[hex.EncodeToString(crypto.FromECDSAPub(selfPublicKey))]
	if inEpochLeaders {
		for i := 0; i < len(selfPublicKeyIndex); i++ {
			workingEpochID := s.getWorkingEpochID()
			data, err := s.buildStage2TxPayload(workingEpochID, uint64(selfPublicKeyIndex[i]))
			if err != nil {
				log.Error("buildStage2TxPayload error", "error", err.Error())
				continue
			}
			err = s.sendSlotTx(data, s.sendTransactionFn)
			if err != nil {
				log.Error("sendSlotTx error", "error", err.Error())
				continue
			}
		}
	}
	functrace.Exit()
	return nil
}

//generateCommitment generate a commitment and send it by tx message
//Returns the commitment buffer []byte which is publicKey and alpha * publicKey
//payload should be send with tx.
func (s *SLS) generateCommitment(publicKey *ecdsa.PublicKey,
	epochID uint64, selfIndexInEpochLeader uint64) ([]byte, error) {
	if publicKey == nil || publicKey.X == nil || publicKey.Y == nil {
		return nil, errInvalidCommitParameter
	}

	if !crypto.S256().IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, vm.ErrNotOnCurve
	}

	alpha, err := uleaderselection.RandFieldElement(Rand.Reader)
	if err != nil {
		return nil, err
	}
	//fmt.Println("alpha:", hex.EncodeToString(alpha.Bytes()))

	commitment, err := uleaderselection.GenerateCommitment(publicKey, alpha)
	if err != nil {
		return nil, err
	}

	buffer, err := vm.RlpPackStage1DataForTx(epochID, selfIndexInEpochLeader, commitment[1],
		vm.GetSlotLeaderScAbiString())

	posdb.GetDb().PutWithIndex(epochID, selfIndexInEpochLeader, "alpha", alpha.Bytes())

	log.Debug(fmt.Sprintf("----Put alpha epochID:%d, selfIndex:%d, alpha:%s, mi:%s, pk:%s", epochID,
		selfIndexInEpochLeader, alpha.String(), hex.EncodeToString(crypto.FromECDSAPub(commitment[1])),
		hex.EncodeToString(crypto.FromECDSAPub(commitment[0]))))

	return buffer, err
}

func (s *SLS) checkNewEpochStart(epochID uint64) {
	//If New epoch start
	workingEpochID := s.getWorkingEpochID()
	if epochID > workingEpochID {
		s.setWorkStage(epochID, slotLeaderSelectionInit)
	}
}

func (s *SLS) getWorkingEpochID() uint64 {
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

func (s *SLS) setWorkingEpochID(workingEpochID uint64) error {
	_, err := posdb.GetDb().Put(0, "slotLeaderCurrentSlotID", convert.Uint64ToBytes(workingEpochID))
	return err
}

func (s *SLS) getWorkStage(epochID uint64) int {
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

func (s *SLS) setWorkStage(epochID uint64, workStage int) error {
	workStageBig := big.NewInt(int64(workStage))
	_, err := posdb.GetDb().Put(epochID, "slotLeaderWorkStage", workStageBig.Bytes())
	return err
}
