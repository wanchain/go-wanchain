package slotleader

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"

	"github.com/wanchain/go-wanchain/pos/util"

	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/uleaderselection"
	"github.com/wanchain/go-wanchain/pos/util/convert"
	"github.com/wanchain/go-wanchain/rlp"
)

const (
	LenProof    = 2
	LenProofMeg = 3
)

//ProofMes 	= [PK, Gt, skGt] 	[]*PublicKey
//Proof 	= [e,z] 			[]*big.Int
func (s *SLS) VerifySlotProof(block *types.Block, epochID uint64, slotID uint64, Proof []*big.Int, ProofMeg []*ecdsa.PublicKey) bool {
	if epochID <= posconfig.FirstEpochId+2 {
		return s.verifySlotProofByGenesis(epochID, slotID, Proof, ProofMeg)
	}

	var epochLeadersPtrPre []*ecdsa.PublicKey
	var isDefault bool

	epochLeadersPtrPre, isDefault = s.GetPreEpochLeadersPK(epochID)

	if isDefault {
		log.Debug("VerifySlotProof", "isDefault", isDefault, "epochID", epochID)
		return s.verifySlotProofByGenesis(epochID, slotID, Proof, ProofMeg)
	}

	rbPtr, err := s.getRandom(block, epochID)
	if err != nil {
		log.SyslogErr(err.Error())
		return false
	}

	rbBytes := rbPtr.Bytes()
	// stage two info from trans
	validEpochLeadersIndex, stageTwoAlphaPKi, err := s.getStageTwoFromTrans(epochID)
	if err != nil {
		log.SyslogErr(err.Error())
		// no stage2 trans on the block chain.
		return s.verifySlotProofByGenesis(epochID, slotID, Proof, ProofMeg)
	}

	var hasValidTx bool
	hasValidTx = false
	log.Debug("VerifySlotProof:VerifyDleqProof", "validEpochLeadersIndex", validEpochLeadersIndex)
	for _, valid := range validEpochLeadersIndex {

		if valid {
			hasValidTx = true
			break
		}
	}
	if !hasValidTx {
		return s.verifySlotProofByGenesis(epochID, slotID, Proof, ProofMeg)
	}

	var publicKey *ecdsa.PublicKey
	publicKey = ProofMeg[0]

	publicKeyIndexes := make([]int, 0)
	for index, value := range epochLeadersPtrPre {
		if uleaderselection.PublicKeyEqual(publicKey, value) {
			publicKeyIndexes = append(publicKeyIndexes, index)
		}
	}

	// Verify skGt
	var skGtValid bool
	skGtValid = false
	for _, index := range publicKeyIndexes {

		smaPieces := make([]*ecdsa.PublicKey, 0)
		for i := 0; i < posconfig.EpochLeaderCount; i++ {
			if validEpochLeadersIndex[i] {
				smaPieces = append(smaPieces, stageTwoAlphaPKi[i][index])
			}
		}

		if len(smaPieces) == 0 {
			log.SyslogErr("len(smaPieces) == 0 in proof.go")
			return false
		}

		log.Debug("VerifySlotLeaderProofskGT aphaiPki", "index", index, "epochID", epochID, "slotID", slotID)
		log.Debug("VerifySlotLeaderProofskGT", "epochID", epochID, "slotID", slotID, "slotLeaderRb", rbBytes[:])

		smaPiecesHexStr := make([]string, 0)
		for _, value := range smaPieces {
			smaPiecesHexStr = append(smaPiecesHexStr, hex.EncodeToString(crypto.FromECDSAPub(value)))
		}
		log.Debug("VerifySlotLeaderProof", "epochID", epochID, "slotID", slotID, "smaPiecesHexStr", smaPiecesHexStr)

		// get skGT from trans
		skGt := s.getSkGtFromTrans(epochLeadersPtrPre, epochID, slotID, rbBytes[:], smaPieces[:])

		if uleaderselection.PublicKeyEqual(skGt, ProofMeg[2]) {
			skGtValid = true
			break
		}
	}

	if !skGtValid {
		log.Warn("VerifySlotLeaderProof Fail skGt is not valid", "epochID", epochID, "slotID", slotID)
		return false
	}
	log.Debug("VerifySlotLeaderProof skGt is verified successfully.", "epochID", epochID, "slotID", slotID)

	// verify slot leader proof
	return uleaderselection.VerifySlotLeaderProof(Proof[:], ProofMeg[:], epochLeadersPtrPre[:], rbBytes[:])
}

func (s *SLS) PackSlotProof(epochID uint64, slotID uint64, prvKey *ecdsa.PrivateKey) ([]byte, error) {
	proofMeg, proof, err := s.getSlotLeaderProof(prvKey, epochID, slotID)
	if err != nil {
		return nil, err
	}

	objToPack := &Pack{Proof: convert.BigIntArrayToByteArray(proof), ProofMeg: convert.PkArrayToByteArray(proofMeg)}

	buf, err := rlp.EncodeToBytes(objToPack)

	return buf, err
}

func (s *SLS) GetInfoFromHeadExtra(epochID uint64, input []byte) ([]*big.Int, []*ecdsa.PublicKey, error) {
	var info Pack
	err := rlp.DecodeBytes(input, &info)
	if err != nil {
		log.SyslogErr("GetInfoFromHeadExtra rlp.DecodeBytes failed",
			"epochID", epochID,
			"input", hex.EncodeToString(input),
			"err", err.Error())
		return nil, nil, err
	}

	proof := convert.ByteArrayToBigIntArray(info.Proof)
	proofMeg := convert.ByteArrayToPkArray(info.ProofMeg)

	if len(proof) != LenProof {
		return nil, nil, uleaderselection.ErrInvalidProof
	}

	for _, proofItem := range proof {
		if proofItem == nil || proofItem.Cmp(uleaderselection.Big0) == 0 {
			log.SyslogErr("GetInfoFromHeadExtra failed proof is nil or proof item is Big0")
			return nil, nil, uleaderselection.ErrInvalidProof
		}
	}
	if len(proofMeg) != LenProofMeg {
		return nil, nil, uleaderselection.ErrInvalidProofMeg
	}
	for _, proofMegItem := range proofMeg {
		if proofMegItem == nil || !proofMegItem.IsOnCurve(proofMegItem.X, proofMegItem.Y) {
			log.SyslogErr("GetInfoFromHeadExtra failed proofMeg is nil or proofMeg item is not on curve")
			return nil, nil, uleaderselection.ErrInvalidProofMeg
		}
	}
	return proof, proofMeg, nil
}

func (s *SLS) getSlotLeaderProofByGenesis(PrivateKey *ecdsa.PrivateKey, epochID uint64,
	slotID uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {

	//1. SMA PRE
	smaPiecesPtr := s.smaGenesis
	epochLeadersPtrPre := s.epochLeadersPtrArrayGenesis
	rbBytes := s.randomGenesis.Bytes()

	log.Debug("getSlotLeaderProofByGenesis", "epochID", epochID, "slotID", slotID)
	log.Debug("getSlotLeaderProofByGenesis", "epochID", epochID, "slotID", slotID, "slotLeaderRb",
		hex.EncodeToString(rbBytes[:]))
	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof(PrivateKey, smaPiecesPtr[:],
		epochLeadersPtrPre[:], rbBytes[:], slotID, epochID)
	return profMeg, proof, err
}

func (s *SLS) getSlotLeaderProof(PrivateKey *ecdsa.PrivateKey, epochID uint64,
	slotID uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {
	if epochID <= posconfig.FirstEpochId+2 {
		return s.getSlotLeaderProofByGenesis(PrivateKey, 0, slotID)
	}
	epochLeadersPtrPre, isDefault := s.GetPreEpochLeadersPK(epochID)
	if isDefault {
		log.Warn("getSlotLeaderProof", "isDefault", isDefault)
		return s.getSlotLeaderProofByGenesis(PrivateKey, 0, slotID)
	}

	//SMA PRE
	smaPiecesPtr, isGenesis, _ := s.getSMAPieces(epochID)
	if isGenesis {
		return s.getSlotLeaderProofByGenesis(PrivateKey, 0, slotID)
	}

	//RB PRE
	var rbPtr *big.Int
	rbPtr, _ = s.getRandom(nil, epochID)
	rbBytes := rbPtr.Bytes()

	log.Debug("getSlotLeaderProof", "epochID", epochID, "slotID", slotID)
	log.Debug("getSlotLeaderProof", "epochID", epochID, "slotID", slotID, "slotLeaderRb", hex.EncodeToString(rbBytes))

	epochLeadersHexStr := make([]string, 0)
	for _, value := range epochLeadersPtrPre {
		epochLeadersHexStr = append(epochLeadersHexStr, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	log.Debug("getSlotLeaderProof", "epochID", epochID, "slotID", slotID, "epochLeadersHexStr", epochLeadersHexStr)

	smaPiecesHexStr := make([]string, 0)
	for _, value := range smaPiecesPtr {
		smaPiecesHexStr = append(smaPiecesHexStr, hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	log.Debug("getSlotLeaderProof", "epochID", epochID, "slotID", slotID, "smaPiecesHexStr", smaPiecesHexStr)

	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof(PrivateKey, smaPiecesPtr, epochLeadersPtrPre,
		rbBytes[:], slotID, epochID)

	return profMeg, proof, err
}

func (s *SLS) verifySlotProofByGenesis(epochID uint64, slotID uint64, Proof []*big.Int,
	ProofMeg []*ecdsa.PublicKey) bool {

	var publicKey *ecdsa.PublicKey
	publicKey = ProofMeg[0]

	publicKeyIndexes := make([]int, 0)
	for index, value := range s.epochLeadersPtrArrayGenesis {
		if uleaderselection.PublicKeyEqual(publicKey, value) {
			publicKeyIndexes = append(publicKeyIndexes, index)
		}
	}

	// Verify skGt
	var skGtValid bool
	skGtValid = false
	for _, index := range publicKeyIndexes {

		smaPieces := make([]*ecdsa.PublicKey, 0)
		for i := 0; i < posconfig.EpochLeaderCount; i++ {
			smaPieces = append(smaPieces, s.stageTwoAlphaPKiGenesis[i][index])
		}

		if len(smaPieces) == 0 {
			return false
		}

		log.Debug("verifySlotProofByGenesis", "epochID", epochID, "slotID", slotID, "slotLeaderRb",
			hex.EncodeToString(s.randomGenesis.Bytes()))
		log.Debug("verifySlotProofByGenesis aphaiPki", "index", index, "epochID", epochID, "slotID", slotID)
		eps := s.epochLeadersPtrArrayGenesis
		skGt := s.getSkGtFromTrans(eps[:], 0, slotID, s.randomGenesis.Bytes()[:],
			smaPieces[:])
		if uleaderselection.PublicKeyEqual(skGt, ProofMeg[2]) {
			skGtValid = true
			break
		}
	}
	if !skGtValid {
		log.Warn("verifySlotProofByGenesis Fail skGt is not valid", "epochID", epochID, "slotID", slotID)
		return false
	}
	log.Debug("verifySlotProofByGenesis skGt is verified successfully.", "epochID", epochID, "slotID", slotID)
	eps := s.epochLeadersPtrArrayGenesis
	return uleaderselection.VerifySlotLeaderProof(Proof[:], ProofMeg[:], eps[:],
		s.randomGenesis.Bytes()[:])
}

func (s *SLS) getSkGtFromTrans(epochLeadersPtrPre []*ecdsa.PublicKey, epochID uint64, slotID uint64, rbBytes []byte,
	smaPieces []*ecdsa.PublicKey) (skGtRet *ecdsa.PublicKey) {

	var buffer bytes.Buffer
	buffer.Write(rbBytes[:])
	buffer.Write(convert.Uint64ToBytes(epochID))
	buffer.Write(convert.Uint64ToBytes(slotID))
	temp := buffer.Bytes()

	smaLen := big.NewInt(0).SetUint64(uint64(len(smaPieces)))

	skGt := new(ecdsa.PublicKey)
	skGt.Curve = crypto.S256()

	for i := 0; i < len(epochLeadersPtrPre); i++ {
		tempHash := crypto.Keccak256(temp)
		tempBig := new(big.Int).SetBytes(tempHash)
		cstemp := new(big.Int).Mod(tempBig, smaLen)

		if i == 0 {
			skGt.X = new(big.Int).Set(smaPieces[cstemp.Int64()].X)
			skGt.Y = new(big.Int).Set(smaPieces[cstemp.Int64()].Y)
		} else {
			skGt.X, skGt.Y = uleaderselection.Wadd(skGt.X, skGt.Y, smaPieces[cstemp.Int64()].X,
				smaPieces[cstemp.Int64()].Y)
		}
		temp = tempHash
	}
	return skGt
}

func (s *SLS) getStageTwoFromTrans(epochID uint64) (validEpochLeadersIndex [posconfig.EpochLeaderCount]bool,
	stageTwoAlphaPKi [posconfig.EpochLeaderCount][posconfig.EpochLeaderCount]*ecdsa.PublicKey, err error) {

	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		validEpochLeadersIndex[i] = true
	}

	indexesSentTran, err := s.getSlotLeaderStage2TxIndexes(epochID - 1)
	log.Debug("VerifySlotProof", "indexesSentTran", indexesSentTran)
	if err != nil {
		log.SyslogErr("getStageTwoFromTrans", "indexesSentTran error", err.Error())
		return validEpochLeadersIndex, stageTwoAlphaPKi, err
	}

	hash := util.GetEpochBlockHash(epochID - 1)
	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		if !indexesSentTran[i] {
			validEpochLeadersIndex[i] = false
			continue
		}
		// TODO:
		bkey := make([]byte, 0)
		bkey = append(bkey, hash[:]...)
		bkey = append(bkey, big.NewInt(int64(i)).Bytes()...)
		ckey := crypto.Keccak256Hash(bkey)

		var alphaPki []*ecdsa.PublicKey
		alphaPkiCached, ok := APkiCache.Get(ckey)
		if !ok {
			var err error
			statedb, _ := s.getCurrentStateDb()
			alphaPki, _, err = vm.GetStage2TxAlphaPki(statedb, epochID-1, uint64(i))
			if err != nil {
				log.Debug("VerifySlotProof:GetStage2TxAlphaPki", "index", i, "error", err.Error())
				validEpochLeadersIndex[i] = false
				continue
			}
			APkiCache.Add(ckey, alphaPki)
		} else {
			alphaPki = alphaPkiCached.([]*ecdsa.PublicKey)
		}
		for j := 0; j < posconfig.EpochLeaderCount; j++ {
			stageTwoAlphaPKi[i][j] = alphaPki[j]
		}

	}
	return validEpochLeadersIndex, stageTwoAlphaPKi, nil
}
