package randombeacon

import (
	"testing"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"strings"
	"github.com/wanchain/go-wanchain/common"
)

var(
	rbscDefinition = `[{"constant":false,"inputs":[{"name":"Info","type":"string"}],"name":"dkg","outputs":[{"name":"Info","type":"string"}],"payable":false,"type":"function","stateMutability":"nonpayable"}]`
	rbscAbi, errRbscInit = abi.JSON(strings.NewReader(rbscDefinition))
)

func TestLoop(t * testing.T)  {
	//str := "hello"
	//ret, err := rbscAbi.Pack("dkg", &str, big.NewInt(11))
	//if err != nil {
	//	t.Error("abi pack fail, err:%s", err.Error())
	//	return
	//}
	//
	//println("dkg abi packed payload, payload:%s", common.Bytes2Hex(ret))
	//
	////type ST struct {
	////	Info string
	////	Time *big.Int
	////}
	////
	////var st ST
	//
	//err = rbscAbi.Unpack(&st, "dkg", ret[4:])
	//if err != nil {
	//	t.Error("abi unpack, err:%s", err.Error())
	//} else {
	//	println("abi unpack, str:%s", st.Info)
	//}



	str := "hello"
	ret, err := rbscAbi.Pack("dkg", &str)
	if err != nil {
		t.Error("abi pack fail, err:%s", err.Error())
		return
	}

	println("dkg abi packed payload, payload:", common.Bytes2Hex(ret))

	var str2 = ""
	err = rbscAbi.Unpack(&str2, "dkg", ret[4:])
	if err != nil {
		t.Error("abi unpack, err:", err.Error())
	} else {
		println("abi unpack, str:", str2)
	}
}
