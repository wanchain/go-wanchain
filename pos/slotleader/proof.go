package slotleader

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos"

	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/uleaderselection"
)

//ProofMes = [PK, Gt, skGt] []*PublicKey
//Proof = [e,z] []*big.Int
func (s *SlotLeaderSelection) GetSlotLeaderProofByGenesis(PrivateKey *ecdsa.PrivateKey, epochID uint64, slotID uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {
	//1. SMA PRE
	smaPiecesPtr := s.smaGenesis
	epochLeadersPtrPre := s.epochLeadersPtrArrayGenesis
	rbBytes := s.randomGenesis.Bytes()

	log.Debug("GetSlotLeaderProofByGenesis", "epochID", epochID, "slotID", slotID)
	log.Debug("GetSlotLeaderProofByGenesis", "epochID", epochID, "slotID", slotID, "slotLeaderRb", hex.EncodeToString(rbBytes[:]))
	for index, value := range epochLeadersPtrPre {
		log.Debug("GetSlotLeaderProofByGenesis", "epochID", epochID, "slotID",
			slotID, "Genesis epochLeaderPtrPre", index, "Genesis epochLeaderPtrPre value", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

	for index, value := range smaPiecesPtr {
		log.Debug("GetSlotLeaderProofByGenesis", "epochID", epochID, "slotID",
			slotID, "Genesis sma pieces index", index, "Genesis sma pieces value", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}
	log.Debug("GetSlotLeaderProofByGenesis", "epochID", epochID, "slotID", slotID, "rb", hex.EncodeToString(rbBytes))

	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof2(PrivateKey, smaPiecesPtr[:], epochLeadersPtrPre[:], rbBytes[:], slotID, epochID)
	return profMeg, proof, err
}

func (s *SlotLeaderSelection) GetSlotLeaderProof(PrivateKey *ecdsa.PrivateKey, epochID uint64, slotID uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {
	_, err := s.getPreEpochLeadersPK(epochID)
	if epochID == uint64(0) || err != nil {
		return s.GetSlotLeaderProofByGenesis(PrivateKey, epochID, slotID)
	}

	//1. SMA PRE
	smaPiecesPtr, err := s.getSMAPieces(epochID)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, err
	}

	var rbPtr *big.Int
	//2. epochLeader PRE
	epochLeadersPtrPre, err := s.getPreEpochLeadersPK(epochID)
	if err != nil {
		log.Warn(err.Error())
	} else {
		//3. RB PRE

		rbPtr, err = s.getRandom(epochID)
		if err != nil {
			log.Error(err.Error())
			return nil, nil, err
		}
	}

	rbBytes := rbPtr.Bytes()

	log.Debug("GetSlotLeaderProof", "epochID", epochID, "slotID", slotID)

	log.Debug("GetSlotLeaderProof", "epochID", epochID, "slotID", slotID, "slotLeaderRb", hex.EncodeToString(rbBytes))

	for index, value := range epochLeadersPtrPre {
		log.Debug("GetSlotLeaderProof", "epochID", epochID, "slotID",
			slotID, "epochLeaderPtrPre", index, "epochLeaderPtrPre value", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

	for index, value := range smaPiecesPtr {
		log.Debug("GetSlotLeaderProof", "epochID", epochID, "slotID",
			slotID, "sma pieces index", index, "sma pieces value", hex.EncodeToString(crypto.FromECDSAPub(value)))
	}

	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof2(PrivateKey, smaPiecesPtr, epochLeadersPtrPre, rbBytes[:], slotID, epochID)

	return profMeg, proof, err
}

func (s *SlotLeaderSelection) VerifySlotProofByGenesis(epochID uint64, slotID uint64, Proof []*big.Int, ProofMeg []*ecdsa.PublicKey) bool {

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
		for i := 0; i < pos.EpochLeaderCount; i++ {
			smaPieces = append(smaPieces, s.stageTwoAlphaPKiGenesis[i][index])
		}

		if len(smaPieces) == 0 {
			return false
		}
		smaLen := new(big.Int).SetInt64(int64(len(smaPieces)))
		log.Debug("VerifySlotProofByGenesis", "epochID", epochID, "slotID", slotID, "slotLeaderRb", hex.EncodeToString(s.randomGenesis.Bytes()))
		log.Debug("VerifySlotProofByGenesis aphaiPki", "index", index, "epochID", epochID, "slotID", slotID)
		for i := 0; i < len(smaPieces); i++ {
			log.Debug("VerifySlotProofByGenesis", "piece index", i, "piece", hex.EncodeToString(crypto.FromECDSAPub(smaPieces[i])))
		}

		var buffer bytes.Buffer
		buffer.Write(s.randomGenesis.Bytes())
		buffer.Write(posdb.Uint64ToBytes(epochID))
		buffer.Write(posdb.Uint64ToBytes(slotID))
		temp := buffer.Bytes()

		skGt := new(ecdsa.PublicKey)
		skGt.Curve = crypto.S256()

		for i := 0; i < pos.EpochLeaderCount; i++ {
			tempHash := crypto.Keccak256(temp)
			tempBig := new(big.Int).SetBytes(tempHash)
			cstemp := new(big.Int).Mod(tempBig, smaLen)

			log.Debug("VerifySlotProofByGenesis", "skGtPiece index", i, "alphaiPki index", cstemp.Int64())

			if i == 0 {
				skGt.X = new(big.Int).Set(smaPieces[cstemp.Int64()].X)
				skGt.Y = new(big.Int).Set(smaPieces[cstemp.Int64()].Y)
			} else {
				skGt.X, skGt.Y = uleaderselection.Wadd(skGt.X, skGt.Y, smaPieces[cstemp.Int64()].X, smaPieces[cstemp.Int64()].Y)
			}
			temp = tempHash
		}

		if uleaderselection.PublicKeyEqual(skGt, ProofMeg[2]) {
			skGtValid = true
			break
		}
	}
	if !skGtValid {
		log.Warn("VerifySlotProofByGenesis Fail skGt is not valid", "epochID", epochID, "slotID", slotID)
		return false
	}
	return uleaderselection.VerifySlotLeaderProof(Proof[:], ProofMeg[:], s.epochLeadersPtrArrayGenesis[:], s.randomGenesis.Bytes()[:])
}

func (s *SlotLeaderSelection) VerifySlotProof(epochID uint64, slotID uint64, Proof []*big.Int, ProofMeg []*ecdsa.PublicKey) bool {
	_, errGenesis := s.getPreEpochLeadersPK(epochID)
	if epochID == 0 || errGenesis != nil {
		return s.VerifySlotProofByGenesis(epochID, slotID, Proof, ProofMeg)
	}

	var epochLeadersPtrPre []*ecdsa.PublicKey

	epochLeadersPtrPre, err := s.getPreEpochLeadersPK(epochID)
	if err != nil {
		log.Warn(err.Error())
	}

	//3. RB PRE
	rbPtr, err := s.getRandom(epochID)
	if err != nil {
		log.Error(err.Error())
		return false
	}

	rbBytes := rbPtr.Bytes()

	// 4. get preEpoch Tx1 data and Tx2 data
	var validEpochLeadersIndex [pos.EpochLeaderCount]bool // true: can be used to slot leader false: can not be used to slot leader
	var stageOneMi [pos.EpochLeaderCount]*ecdsa.PublicKey
	var stageTwoAlphaPKi [pos.EpochLeaderCount][pos.EpochLeaderCount]*ecdsa.PublicKey
	var stageTwoProof [pos.EpochLeaderCount][StageTwoProofCount]*big.Int //[0]: e; [1]:Z

	for i := 0; i < pos.EpochLeaderCount; i++ {
		validEpochLeadersIndex[i] = true
	}

	for i := 0; i < pos.EpochLeaderCount; i++ {
		_, mi, _ := s.getStg1StateDbInfo(epochID-1, uint64(i))
		if len(mi) == 0 {
			validEpochLeadersIndex[i] = false
			continue
		} else {
			stageOneMi[i] = crypto.ToECDSAPub(mi)
		}

		alphaPkis, proofs, err := s.getStage2TxAlphaPki(epochID-1, uint64(i))
		if err != nil {
			validEpochLeadersIndex[i] = false
			continue
		}
		if (len(alphaPkis) != pos.EpochLeaderCount) || (len(proofs) != StageTwoProofCount) {
			validEpochLeadersIndex[i] = false
			continue
		} else {

			for j := 0; j < pos.EpochLeaderCount; j++ {
				alphaPkiDecodeBytes, err := hex.DecodeString(alphaPkis[j])
				if err != nil {
					continue
				}
				stageTwoAlphaPKi[i][j] = crypto.ToECDSAPub(alphaPkiDecodeBytes)
			}

			for j := 0; j < StageTwoProofCount; j++ {
				var err bool
				stageTwoProof[i][j], err = big.NewInt(0).SetString(proofs[j], 16)
				if !err {
					continue
				}
			}
		}

	}

	// 5. verify tx1 data and tx2 data
	for i := 0; i < pos.EpochLeaderCount; i++ {
		if !validEpochLeadersIndex[i] {
			continue
		}
		if !uleaderselection.PublicKeyEqual(stageOneMi[i], stageTwoAlphaPKi[i][i]) {
			validEpochLeadersIndex[i] = false
			continue
		} else {

			if !uleaderselection.VerifyDleqProof(epochLeadersPtrPre[:], stageTwoAlphaPKi[i][:], stageTwoProof[i][:]) {
				validEpochLeadersIndex[i] = false
				continue
			}
		}
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
		for i := 0; i < pos.EpochLeaderCount; i++ {
			if validEpochLeadersIndex[i] {
				smaPieces = append(smaPieces, stageTwoAlphaPKi[i][index])
			}
		}

		if len(smaPieces) == 0 {
			log.Error("len(smaPieces) == 0 in proof.go")
			return false
		}

		smaLen := new(big.Int).SetInt64(int64(len(smaPieces)))

		log.Debug("VerifySlotLeaderProof aphaiPki", "index", index, "epochID", epochID, "slotID", slotID)
		log.Debug("VerifySlotLeaderProof", "epochID", epochID, "slotID", slotID, "slotLeaderRb", rbBytes[:])
		for i := 0; i < len(smaPieces); i++ {
			log.Debug("VerifySlotLeaderProof", "piece index", i, "piece", hex.EncodeToString(crypto.FromECDSAPub(smaPieces[i])))
		}

		var buffer bytes.Buffer
		buffer.Write(rbBytes[:])
		buffer.Write(posdb.Uint64ToBytes(epochID))
		buffer.Write(posdb.Uint64ToBytes(slotID))
		temp := buffer.Bytes()

		skGt := new(ecdsa.PublicKey)
		skGt.Curve = crypto.S256()

		for i := 0; i < len(epochLeadersPtrPre); i++ {
			tempHash := crypto.Keccak256(temp)
			tempBig := new(big.Int).SetBytes(tempHash)
			cstemp := new(big.Int).Mod(tempBig, smaLen)

			log.Debug("VerifySlotLeaderProof", "skGtPiece index", i, "alphaiPki index", cstemp.Int64())

			if i == 0 {
				skGt.X = new(big.Int).Set(smaPieces[cstemp.Int64()].X)
				skGt.Y = new(big.Int).Set(smaPieces[cstemp.Int64()].Y)
			} else {
				skGt.X, skGt.Y = uleaderselection.Wadd(skGt.X, skGt.Y, smaPieces[cstemp.Int64()].X, smaPieces[cstemp.Int64()].Y)
			}
			temp = tempHash
		}

		if uleaderselection.PublicKeyEqual(skGt, ProofMeg[2]) {
			skGtValid = true
			break
		}
	}
	if !skGtValid {
		log.Warn("VerifySlotLeaderProof Fail skGt is not valid", "epochID", epochID, "slotID", slotID)
		return false
	}
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
