package incentive

import (
	"math/big"
	"os"
	"testing"

	"github.com/wanchain/go-wanchain/core/vm"
)

func testInitDb() {
	os.RemoveAll("/tmp/pluto/gwan/incentive")
	initLocalDb("/tmp/pluto/gwan/incentive")
}

func TestInitLocalDB(t *testing.T) {
	testInitDb()
}

func TestGetEpochPayDetail(t *testing.T) {
	epochID := uint64(0)
	generateTestAddrs()
	testInitDb()

	payExample := [][]vm.ClientIncentive{
		{
			{Addr: epAddrs[0], Incentive: big.NewInt(100)},
			{Addr: epAddrs[1], Incentive: big.NewInt(200)},
		},
		{
			{Addr: epAddrs[3], Incentive: big.NewInt(300)},
			{Addr: epAddrs[4], Incentive: big.NewInt(400)},
			{Addr: epAddrs[5], Incentive: big.NewInt(500)},
		},
	}

	saveIncentiveHistory(epochID, nil)
	saveIncentiveHistory(epochID, payExample)
	pay, err := GetEpochPayDetail(epochID)
	if err != nil {
		t.FailNow()
	}

	for i := 0; i < len(pay); i++ {
		for m := 0; m < len(pay[i]); m++ {
			if pay[i][m].Addr.Hex() != payExample[i][m].Addr.Hex() || pay[i][m].Incentive.String() != payExample[i][m].Incentive.String() {
				t.FailNow()
			}
		}
	}

	saveIncentiveHistory(1, payExample)

	total, err := GetTotalIncentive()
	if total.Uint64() != 3000 || err != nil {
		t.FailNow()
	}

	total, err = GetEpochIncentive(1)
	if total.Uint64() != 1500 || err != nil {
		t.FailNow()
	}

	saveRemain(0, big.NewInt(100))
	saveRemain(1, big.NewInt(300))

	epRemain, err := GetEpochRemain(1)
	if err != nil || epRemain.Uint64() != 300 {
		t.FailNow()
	}
	epRemain, err = GetTotalRemain()
	if err != nil || epRemain.Uint64() != 400 {
		t.FailNow()
	}

	value, err := GetRunTimes()
	if err != nil || value.Uint64() != 2 {
		t.FailNow()
	}

}

func TestOtherApiSuccess(t *testing.T) {

	generateTestAddrs()
	generateTestStaker()

	chain := &TestChainReader{}

	ret := GetEpochGasPool(statedb, 0)
	if ret.String() != "0" {
		//t.FailNow()
	}

	rb := GetRBAddress(0)
	if rb != nil {
		//t.FailNow()
	}

	total, fdt, pool := GetIncentivePool(statedb, 0)
	if total.String() != "0" || fdt.String() != "0" || pool.String() != "0" {
		//t.FailNow()
	}

	// addr, act := GetEpochLeaderActivity(statedb, 0)
	// if len(addr) != 0 || len(act) != 0 {
	// 	//t.FailNow()
	// }

	addr, act := GetEpochRBLeaderActivity(statedb, 0)
	if len(addr) != 0 || len(act) != 0 {
		//t.FailNow()
	}

	addrs, cnt, actf, _ := GetSlotLeaderActivity(chain, 0)
	if len(addrs) != 0 || len(cnt) != 0 || actf != float64(0) {
		//t.FailNow()
	}
}

func TestOtherApiFail(t *testing.T) {
	ret := GetEpochGasPool(nil, 0)
	if ret.String() != "0" {
		t.FailNow()
	}

	tmp := getRandomProposerAddress
	getRandomProposerAddress = nil
	rb := GetRBAddress(0)
	if rb != nil {
		//t.FailNow()
	}
	getRandomProposerAddress = tmp

	total, fdt, pool := GetIncentivePool(nil, 0)
	if total.String() != "0" || fdt.String() != "0" || pool.String() != "0" {
		t.FailNow()
	}

	addr, act := GetEpochLeaderActivity(nil, 0)
	if len(addr) != 0 || len(act) != 0 {
		t.FailNow()
	}

	addr, act = GetEpochRBLeaderActivity(nil, 0)
	if len(addr) != 0 || len(act) != 0 {
		t.FailNow()
	}

	addrs, cnt, actf, ctrlCnt := GetSlotLeaderActivity(nil, 0)
	if len(addrs) != 0 || len(cnt) != 0 || actf != float64(0) || ctrlCnt != 0 {
		t.FailNow()
	}
}
