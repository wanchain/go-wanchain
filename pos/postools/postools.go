package postools

import (
	"fmt"
	"github.com/wanchain/go-wanchain/pos"
	"time"
)

func CalEpochSlotID(time uint64) (epochId, slotId uint64){
	if pos.EpochBaseTime == 0 {
		return
	}
	//timeUnix := uint64(time.Now().Unix())
	timeUnix := time
	epochTimespan := uint64(pos.SlotTime * pos.SlotCount)
	epochId = uint64((timeUnix - pos.EpochBaseTime) / epochTimespan)
	slotId = uint64((timeUnix - pos.EpochBaseTime) / pos.SlotTime % pos.SlotCount)
	fmt.Println("CalEpochSlotID:", epochId, slotId)
}