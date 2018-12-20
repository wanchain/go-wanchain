package pos

import (
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos/slotleader"
)

func RBLoop() error {
	// get epoch id, slot id
	epochId, slotId, err := slotleader.GetEpochSlotID()
	if err != nil {
		return err
	}

	proposerId := GetRBProposerId()

	// check 4K 8K point
	if slotId == uint64(Cfg().K*(4+1)) {
		// do 4K
		_, err := vm.GetDKGData(epochId, proposerId)
		if err != nil {
			return err
		}

	} else if slotId == uint64(Cfg().K*(8+1)) {
		// do 8K

	}

	return nil
}

func GetRBProposerId() uint {
	// **************
	return 0
}
