package util

import (
	"fmt"
	"testing"
)

func TestGetEpochSlotID(t *testing.T) {
	epochID, slotID := GetEpochSlotID()
	fmt.Println("epochID:", epochID, " slotID:", slotID)
}
