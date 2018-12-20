package pos

import (
	"fmt"
	"time"
	//"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/node"
	"github.com/wanchain/gw_forpull/pos/slotleader"
)
func BackendTimerLoop(stack *node.Node) {
	time.Sleep(10*time.Second)
	rc, errA := stack.Attach();
	if errA != nil {
		fmt.Println("err:", errA)
		panic(errA)
	}
	for {
		select {
			case <- time.After(10*time.Second):
				fmt.Println("time")
				// acm := stack.AccountManager()
				//acm.Find(accountxxx)

				//Add for slot leader selection
				slotleader.GetSlotLeaderSelection().Loop(rc)


		}
	}
	return
}
