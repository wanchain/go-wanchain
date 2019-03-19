package util

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/pos/posconfig"
)

func CalEpochSlotID(time uint64) (epochId, slotId uint64) {
	if posconfig.EpochBaseTime == 0 {
		return
	}
	//timeUnix := uint64(time.Now().Unix())
	timeUnix := time
	epochTimespan := uint64(posconfig.SlotTime * posconfig.SlotCount)
	epochId = uint64((timeUnix - posconfig.EpochBaseTime) / epochTimespan)
	slotId = uint64((timeUnix - posconfig.EpochBaseTime) / posconfig.SlotTime % posconfig.SlotCount)
	fmt.Println("CalEpochSlotID:", epochId, slotId)
	return epochId, slotId
}

//PkEqual only can use in same curve. return whether the two points equal
func PkEqual(pk1, pk2 *ecdsa.PublicKey) bool {
	if pk1 == nil || pk2 == nil {
		return false
	}

	if hex.EncodeToString(pk1.X.Bytes()) == hex.EncodeToString(pk2.X.Bytes()) &&
		hex.EncodeToString(pk1.Y.Bytes()) == hex.EncodeToString(pk2.Y.Bytes()) {
		return true
	}
	return false
}
