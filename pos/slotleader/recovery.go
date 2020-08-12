package slotleader

// GetRecoveryEpochID used to get the recovery default epochID
func GetRecoveryEpochID(epochID uint64) uint64 {
	seekBackCount := uint64(10) // use 10 epoch before state

	// get the last epochID in blockchain
	epochLast := GetSlotLeaderSelection().getLastEpochIDFromChain()

	// first get
	if epochLast < epochID {
		return epochLast - seekBackCount
	}

	// seek recovery failed.
	if epochID < epochLast-10*seekBackCount {
		return 0
	}

	// get more times or in same epoch
	return epochID - seekBackCount
}
