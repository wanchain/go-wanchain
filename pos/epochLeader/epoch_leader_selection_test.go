package epochLeader

import (
	"testing"
	"fmt"
	"github.com/wanchain/go-wanchain/pos/slotleader"
)

func TestGetGetEpochLeaders(t *testing.T) {
	epochID, slotID, err := slotleader.GetEpochSlotID()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("epochID:", epochID, " slotID:", slotID)
	}
}
