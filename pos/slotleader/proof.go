package slotleader

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"

	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/uleaderselection"
)

//ProofMes = [PK, Gt, skGt] []*PublicKey
//Proof = [e,z] []*big.Int
func (s *SlotLeaderSelection) GetSlotLeaderProof(PrivateKey *ecdsa.PrivateKey, epochID uint64, slotID uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {
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
		//3. RB PRE

		rbPtr, err = s.getRandom(0)
		if err != nil {
			log.Error(err.Error())
			return nil, nil, err
		}
	} else {
		//3. RB PRE

		rbPtr, err = s.getRandom(epochID)
		if err != nil {
			log.Error(err.Error())
			return nil, nil, err
		}
	}



	rbBytes := rbPtr.Bytes()
	//4. CR PRE
	//crsPtr, err := s.getCRs(epochID)
	//if err != nil {
	//	log.Error(err.Error())
	//	return nil, nil, err
	//}

	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof2(PrivateKey, smaPiecesPtr, epochLeadersPtrPre, rbBytes[:],slotID,epochID)

	return profMeg, proof, err
}

func (s *SlotLeaderSelection) VerifySlotProof(epochID uint64, Proof []*big.Int, ProofMeg []*ecdsa.PublicKey) bool {

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
