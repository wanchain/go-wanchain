package slotleader

import (
	"errors"
	"math/big"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/log"
)

func (s *SLS) ValidateBody(block *types.Block) error {

	extraSeal := 65
	header := block.Header()
	blkTd := block.Difficulty().Uint64()
	epochID := (blkTd >> 32)
	slotID := ((blkTd & 0xffffffff) >> 8)

	if epochID == 0 {
		return nil
	}

	proof, proofMeg, err := s.GetInfoFromHeadExtra(epochID, header.Extra[:len(header.Extra)-extraSeal])

	if err != nil {
		log.Error("Can not GetInfoFromHeadExtra, verify failed", "error", err.Error())
		return errors.New("Can not GetInfoFromHeadExtra, verify failed")
	}

	if !s.VerifySlotProof(block, epochID, slotID, proof, proofMeg) {
		log.Error("VerifyPackedSlotProof failed", "number", block.NumberU64(), "epochID", epochID, "slotID", slotID)
		return errors.New("VerifyPackedSlotProof failed")
	}

	return nil
}

func (s *SLS) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}
