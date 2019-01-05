package pos

//import (
//	"fmt"
//	"time"
//	"github.com/wanchain/go-wanchain/accounts"
//	"github.com/wanchain/go-wanchain/node"
//	"github.com/wanchain/go-wanchain/rpc"
//	"github.com/wanchain/go-wanchain/eth"
//	"github.com/wanchain/go-wanchain/pos/slotleader"
//)
//func BackendTimerLoop(b eth.ApiBackend) {
//
//
//				//Add for slot leader selection
//				slotleader.GetSlotLeaderSelection().Loop(rc)
//
//
//}

var (
	lastBlockEpoch  =  make(map[uint64] uint64)

)

func UpdateEpochBlock(epochID uint64, blockNumber uint64) {
	lastBlockEpoch[epochID] = blockNumber
}

func GetEpochBlock(epochID uint64) uint64 {
	if epochID < 2 {
		return uint64(0)
	}
	targetEpoch := epochID - 2
	return lastBlockEpoch[targetEpoch]
}