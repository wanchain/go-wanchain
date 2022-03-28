package slotleader

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"github.com/ethereum/go-ethereum/pos/util"
)

// GetRecoveryEpochID used to get the recovery default epochID
func GetRecoveryEpochID(epochID uint64) uint64 {
	preEpochBlock := util.GetEpochBlock(epochID - 1)
	log.Info("GetRecoveryEpochID GetEpochBlock", "epochID", epochID-1, "ret", preEpochBlock, "realEpoch", GetSlotLeaderSelection().getEpochIDFromBlockNumber(preEpochBlock))

	prePreEpochBlock := util.GetEpochBlock(epochID - 2)
	log.Info("GetRecoveryEpochID GetEpochBlock", "epochID", epochID-2, "ret", prePreEpochBlock, "realEpoch", GetSlotLeaderSelection().getEpochIDFromBlockNumber(prePreEpochBlock))

	t1 := GetSlotLeaderSelection().getBlockTime(preEpochBlock)
	t2 := GetSlotLeaderSelection().getBlockTime(prePreEpochBlock)

	// In normal case t1 should >= t2, if not, we should get new t1 and t2 from block chain, not memory cache.
	if t1 < t2 {
		util.RemoveEpochBlockCache(preEpochBlock)
		util.RemoveEpochBlockCache(prePreEpochBlock)

		preEpochBlock = util.GetEpochBlock(epochID - 1)
		log.Info("GetRecoveryEpochID GetEpochBlock", "epochID", epochID-1, "ret", preEpochBlock, "realEpoch", GetSlotLeaderSelection().getEpochIDFromBlockNumber(preEpochBlock))

		prePreEpochBlock = util.GetEpochBlock(epochID - 2)
		log.Info("GetRecoveryEpochID GetEpochBlock", "epochID", epochID-2, "ret", prePreEpochBlock, "realEpoch", GetSlotLeaderSelection().getEpochIDFromBlockNumber(prePreEpochBlock))

		t1 = GetSlotLeaderSelection().getBlockTime(preEpochBlock)
		t2 = GetSlotLeaderSelection().getBlockTime(prePreEpochBlock)
	}

	var epochGet uint64

	if t1-t2 <= posconfig.SlotCount*posconfig.SlotTime {
		if preEpochBlock <= posconfig.Pow2PosUpgradeBlockNumber {
			return posconfig.FirstEpochId
		}
		// get the last epochID in blockchain
		epochGet = GetSlotLeaderSelection().getEpochIDFromBlockNumber(preEpochBlock)
	} else {
		if prePreEpochBlock <= posconfig.Pow2PosUpgradeBlockNumber {
			return posconfig.FirstEpochId
		}
		// get the last epochID in blockchain
		epochGet = GetSlotLeaderSelection().getEpochIDFromBlockNumber(prePreEpochBlock)
	}

	log.Info("GetRecoveryEpochID", "inputEpochId=", epochID, "outputEpochId=", epochGet-posconfig.SeekBackCount,
		"preEpochBlock", preEpochBlock, "prePreEpochBlock", prePreEpochBlock, "t1", t1, "t2", t2, "sub", t1-t2, "epochGet", epochGet)
	// get more times or in same epoch
	return epochGet - posconfig.SeekBackCount
}
