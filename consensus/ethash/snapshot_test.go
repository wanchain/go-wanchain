package ethash

import (
	"container/list"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"testing"
)

func TestInitial(t *testing.T) {
	s := make([]string, 0)
	s = append(s, "appendix")
	fmt.Println(s)
	fmt.Println("jsust jkdsk")
	var rsw = list.New()
	if rsw == nil {
		fmt.Println("-----------------")
	}
	a := common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b")
	rsw.PushFront(a)
	for e := rsw.Front(); e != nil; e = e.Next() {
		if _, ok := e.Value.(common.Address); ok {
			wSigner := e.Value.(common.Address)
			fmt.Println(wSigner.String())
		}
	}

	var (
		aStr []string
	)

	aStr = append(aStr, "something")
}
