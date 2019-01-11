package epochLeader

import (
	"testing"
	"fmt"
	"github.com/wanchain/go-wanchain/pos/slotleader"
)

func TestGetGetEpochLeaders(t *testing.T) {
	epochID, slotID := slotleader.GetEpochSlotID()
	fmt.Println("epochID:", epochID, " slotID:", slotID)
}
