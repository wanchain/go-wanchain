package epochLeader

import (
	"testing"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"fmt"
)


func TestGetEpochLeaders(t *testing.T) {

	epochID, slotID, err := slotleader.GetEpochSlotID()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("epochID:", epochID, " slotID:", slotID)
	}

	//GetEpochLeaders()
}

func TestGetRBProposerGroup(t *testing.T) {

	epochID, slotID, err := slotleader.GetEpochSlotID()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("epochID:", epochID, " slotID:", slotID)
	}

	//GetEpochLeaders()
}