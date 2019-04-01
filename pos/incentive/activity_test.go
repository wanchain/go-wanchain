package incentive

import (
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"
)

type TestChainReader struct {
}

func (t *TestChainReader) CurrentHeader() *types.Header {
	return &types.Header{Number: big.NewInt(int64(100))}
}

func (t *TestChainReader) GetHeaderByNumber(number uint64) *types.Header {
	return &types.Header{Number: big.NewInt(int64(100)), Difficulty: big.NewInt(0), Coinbase: slAddrs[int(number)%len(slAddrs)]}
}

func (t *TestChainReader) Config() *params.ChainConfig                             { return nil }
func (t *TestChainReader) GetHeader(hash common.Hash, number uint64) *types.Header { return nil }
func (t *TestChainReader) GetHeaderByHash(hash common.Hash) *types.Header          { return nil }
func (t *TestChainReader) GetBlock(hash common.Hash, number uint64) *types.Block   { return nil }

func TestGetSlotLeaderActivity(t *testing.T) {
	generateTestAddrs()
	generateTestStaker()

	chain := &TestChainReader{}
	addrs, blks, activity := getSlotLeaderActivity(chain, 0, 100)
	fmt.Println(addrs, blks, activity)

	if activity != 0.99 {
		fmt.Println("activity(0.99):", activity)
		t.FailNow()
	}

	for i := 0; i < len(addrs); i++ {
		if !addressInclude(addrs[i], slAddrs) {
			t.FailNow()
		}
	}

	blkCmp := []int{5, 5, 5, 5, 5, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	sort.Ints(blkCmp)
	sort.Ints(blks)

	for i := 0; i < len(addrs); i++ {
		if blks[i] != blkCmp[i] {
			t.FailNow()
		}
	}
}

type TestSelectLead struct{}

func (t *TestSelectLead) SelectLeadersLoop(epochID uint64) error { return nil }
func (t *TestSelectLead) GetEpochLeaders(epochID uint64) [][]byte {
	buf := make([][]byte, len(epPks))
	for i := 0; i < len(epPks); i++ {
		buf[i] = crypto.FromECDSAPub(epPks[i])
	}
	return buf
}
func (t *TestSelectLead) GetProposerBn256PK(epochID uint64, idx uint64, addr common.Address) []byte {
	return nil
}

func TestGetEpochLeaderAddressAndActivity(t *testing.T) {
	generateTestAddrs()
	generateTestStaker()

	epochID := uint64(0)
	util.SetEpocherInst(&TestSelectLead{})

	for i := 0; i < len(epAddrs); i++ {
		epochIDBuf := convert.Uint64ToBytes(epochID)
		selfIndexBuf := convert.Uint64ToBytes(uint64(i))
		keyHash := vm.GetSlotLeaderStage2KeyHash(epochIDBuf, selfIndexBuf)

		buf, err := vm.RlpPackStage2DataForTx(epochID, uint64(i), epPks[i], epPks, []*big.Int{big.NewInt(100), big.NewInt(100), big.NewInt(100)}, vm.GetSlotLeaderScAbiString())
		if err != nil {
			t.FailNow()
		}
		statedb.SetStateByteArray(vm.GetSlotLeaderSCAddress(), keyHash, buf)
	}

	addrs, activity := getEpochLeaderActivity(statedb, epochID)

	for i := 0; i < len(addrs); i++ {
		if addrs[i].Hex() != epAddrs[i].Hex() {
			t.FailNow()
		}
		if activity[i] != 1 {
			t.FailNow()
		}
	}
}

func testGetRBAddress(epochID uint64) []vm.Leader {
	leaders := make([]vm.Leader, len(rpAddrs))
	for i := 0; i < len(rpAddrs); i++ {
		leaders[i].SecAddr = rpAddrs[i]
	}
	return leaders
}

func testSimulateData(epochID uint64, index uint32) {
	sig := []byte{13, 7, 16, 93}
	hash := vm.GetRBKeyHash(sig, epochID, index)
	randomBeaconPrecompileAddr := common.BytesToAddress(big.NewInt(610).Bytes())
	statedb.SetStateByteArray(randomBeaconPrecompileAddr, *hash, []byte{1, 2, 3})

	hash = vm.GetRBKeyHash([]byte{101}, epochID, index)
	statedb.SetStateByteArray(randomBeaconPrecompileAddr, *hash, []byte{1, 2, 3})
}

func TestGetRandomProposerActivity(t *testing.T) {
	generateTestAddrs()
	generateTestStaker()
	setRBAddressInterface(testGetRBAddress)

	epochID := 0

	addrs, activity := getRandomProposerActivity(statedb, uint64(epochID))

	for i := 0; i < len(addrs); i++ {
		if addrs[i].Hex() != rpAddrs[i].Hex() {
			t.FailNow()
		}
		if activity[i] != 0 {
			t.FailNow()
		}
	}

	for i := 0; i < len(addrs); i++ {
		testSimulateData(uint64(epochID), uint32(i))
	}

	addrs, activity = getRandomProposerActivity(statedb, uint64(epochID))

	for i := 0; i < len(addrs); i++ {
		if addrs[i].Hex() != rpAddrs[i].Hex() {
			t.FailNow()
		}
		if activity[i] == 0 {
			t.FailNow()
		}
	}
}
