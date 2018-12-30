package posconfig

var (
	EpochBaseTime = uint64(0)

	// EpochGenesisTime is the pos start time such as: 2018-12-12 00:00:00 == 1544544000
	EpochGenesisTime = uint64(1544544000)
)

const (
	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 10

	// RandomProperCount is count of pk in random leader group which is select by stake
	RandomProperCount = 10

	// SlotCount is slot count in an epoch
	SlotCount = 30

	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 6

	// SelfTestMode config whether it is in a simlate tese mode
	SelfTestMode = false
)
