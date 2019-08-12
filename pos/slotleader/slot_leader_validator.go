package slotleader

import (
	"crypto/ecdsa"
	"errors"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"math/big"
)

func (s *SLS) ValidateBody(block *types.Block) error {

	extraSeal := 65
	header := block.Header()
	blkTd := block.Difficulty().Uint64()
	epochID := (blkTd >> 32)
	slotID := ((blkTd & 0xffffffff) >> 8)

	//if epochID == 0 {
	//	return nil
	//}

	//extraType := header.Extra[0]
	//start := 1
	//if extraType == 'g' {
	//	start = 33
	//}

	//prootStartTime := time.Now().Nanosecond()

	proof, proofMeg, err := s.GetInfoFromHeadExtra(epochID, header.Extra[:len(header.Extra)-extraSeal])

	if err != nil {
		log.Error("Can not GetInfoFromHeadExtra, verify failed", "error", err.Error())
		return errors.New("Can not GetInfoFromHeadExtra, verify failed")
	}

	if !s.VerifySlotProof(block, epochID, slotID, proof, proofMeg) {
		log.Error("VerifyPackedSlotProof failed", "number", block.NumberU64(), "epochID", epochID, "slotID", slotID)
		return errors.New("VerifyPackedSlotProof failed")
	}

	//prootEndTime := time.Now().Nanosecond()

	//log.Error("proof verify ecplapsed time","beginTime",prootStartTime,"endTime",prootEndTime,"elipse time",prootEndTime - prootStartTime  )

	return nil
}

func (s *SLS) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}


func (s *SLS) GetAllSlotLeaders(epochID uint64) (slotLeader []*ecdsa.PublicKey) {
	if epochID == 0 {
		return nil
	}

	slotLeadersPtrArray := make([]*ecdsa.PublicKey,0)
	// read from local db
	for i := 0; i < posconfig.SlotCount; i++ {
		pkByte, err := posdb.GetDb().GetWithIndex(epochID, uint64(i), SlotLeader)
		if err != nil {
			return nil
		}
		slotLeadersPtrArray = append(slotLeadersPtrArray,crypto.ToECDSAPub(pkByte))
	}

	return slotLeadersPtrArray
}