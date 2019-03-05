package incentive

import (
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/postools"
	"github.com/wanchain/go-wanchain/pos/postools/slottools"
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

func TestGetEpochLeaderActivity(t *testing.T) {
	generateTestAddrs()
	generateTestStaker()

	epochID := uint64(0)
	for i := 0; i < len(epAddrs); i++ {
		epochIDBuf := postools.Uint64ToBytes(epochID)
		selfIndexBuf := postools.Uint64ToBytes(uint64(i))
		keyHash := vm.GetSlotLeaderStage2KeyHash(epochIDBuf, selfIndexBuf)

		buf, err := slottools.RlpPackStage2DataForTx(epochID, uint64(i), epPks[i], epPks, []*big.Int{big.NewInt(100), big.NewInt(100), big.NewInt(100)}, vm.GetSlotLeaderScAbiString())
		if err != nil {
			t.FailNow()
		}
		statedb.SetStateByteArray(vm.GetSlotLeaderSCAddress(), keyHash, buf)
	}

	addrs, activity := getEpochLeaderActivity(statedb, epochID, epAddrs)
	for i := 0; i < len(addrs); i++ {
		if addrs[i].Hex() != epAddrs[i].Hex() {
			t.FailNow()
		}
		if activity[i] != 1 {
			t.FailNow()
		}
	}
}
