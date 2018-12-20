package pos

import (
	"fmt"
	"time"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/node"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/wanchain/go-wanchain/internal/ethapi"
	"github.com/wanchain/go-wanchain/pos/slotleader"
)
func BackendTimerLoop(b ethapi.Backend) {
	time.Sleep(10*time.Second)

	url := node.DefaultIPCEndpoint("gwan")
	rc, err := rpc.Dial(url)
	if err != nil {
		fmt.Println("err:", err)
		panic(err)
	}
	for {
		select {
			case <- time.After(10*time.Second):
				fmt.Println("time")
				acm := b.AccountManager()
				account := accounts.Account{}
				ks,err := acm.Find(account)
				fmt.Println(ks,err)

				//Add for slot leader selection
				slotleader.GetSlotLeaderSelection().Loop(rc)


		}
	}
	return
}
