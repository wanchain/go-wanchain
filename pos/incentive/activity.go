package incentive

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"
)

var whiteList map[common.Address]int

func activityInit() {
	whiteList = make(map[common.Address]int, 0)
	for _, value := range posconfig.WhiteList {
		b := hexutil.MustDecode(value)
		address := crypto.PubkeyToAddress(*(crypto.ToECDSAPub(b)))
		whiteList[address] = 1
	}
}

func isInWhiteList(coinBase common.Address) bool {
	if _, ok := whiteList[coinBase]; ok {
		return true
	}
	return false
}

func checkEpochLeaders(epochLeaders [][]byte) bool {
	if epochLeaders == nil || len(epochLeaders) == 0 {
		return false
	}

	for i := 0; i < len(epochLeaders); i++ {
		pk := crypto.ToECDSAPub(epochLeaders[i])
		if pk == nil {
			return false
		}
	}
	return true
}

func getEpochLeaderActivity(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int) {
	if stateDb == nil {
		log.SyslogErr("getEpochLeaderActivity with an empty stateDb")
		return []common.Address{}, []int{}
	}

	epochLeaders := util.GetEpocherInst().GetEpochLeaders(epochID)
	if !checkEpochLeaders(epochLeaders) {
		log.SyslogErr("incentive activity GetEpochLeaders error", "epochID", epochID)
		return []common.Address{}, []int{}
	}

	// Only the first 24 person have incentive, other 26 do not have incentive.
	wlInfo := vm.GetEpochWLInfo(stateDb, epochID)

	lenRaw := uint64(len(epochLeaders))
	wlLen := wlInfo.WlCount.Uint64()
	if lenRaw > wlLen {
		epochLeaders = epochLeaders[0 : lenRaw-wlLen]
	}

	addrs := make([]common.Address, len(epochLeaders))
	activity := make([]int, len(addrs))
	for i := 0; i < len(addrs); i++ {
		addrs[i] = crypto.PubkeyToAddress(*crypto.ToECDSAPub(epochLeaders[i]))
		activity[i] = 0
	}

	for i := 0; i < len(addrs); i++ {
		epochIDBuf := convert.Uint64ToBytes(epochID)
		selfIndexBuf := convert.Uint64ToBytes(uint64(i))
		keyHash := vm.GetSlotLeaderStage2KeyHash(epochIDBuf, selfIndexBuf)

		data := stateDb.GetStateByteArray(vm.GetSlotLeaderSCAddress(), keyHash)
		if data == nil {
			continue
		}

		epID, slfIndex, selfPk, _, _, err := vm.RlpUnpackStage2DataForTx(data)
		if err != nil {
			continue
		}

		if epID != epochID || uint64(i) != slfIndex {
			continue
		}
		//TODO: CHECK
		addr := crypto.PubkeyToAddress(*selfPk)
		if addr.Hex() == addrs[i].Hex() {
			activity[i] = 1
		}
	}

	return addrs, activity
}

func getRnpAddrFromLeader(leaders []vm.Leader) []common.Address {
	if leaders == nil || len(leaders) == 0 {
		return nil
	}

	addrs := make([]common.Address, len(leaders))
	for i := 0; i < len(leaders); i++ {
		if leaders[i].SecAddr.Hex() == "0x0000000000000000000000000000000000000000" {
			return nil
		}
		addrs[i] = leaders[i].SecAddr
	}

	return addrs
}

func getRandomProposerActivity(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int) {
	if stateDb == nil {
		log.SyslogErr("getRandomProposerActivity with an empty stateDb")
		return []common.Address{}, []int{}
	}

	if getRandomProposerAddress == nil {
		log.SyslogErr("incentive activity getRandomProposerAddress == nil", "epochID", epochID)
		return []common.Address{}, []int{}
	}

	leaders := getRandomProposerAddress(epochID)
	addrs := getRnpAddrFromLeader(leaders)
	if addrs == nil {
		log.SyslogErr("incentive activity getRandomProposerAddress error", "epochID", epochID)
		return []common.Address{}, []int{}
	}

	activity := make([]int, len(addrs))
	for i := 0; i < len(addrs); i++ {
		if vm.IsRBActive(stateDb, epochID, uint32(i)) {
			activity[i] = 1
		} else {
			activity[i] = 0
		}
	}
	return addrs, activity
}

func getSlotLeaderActivity(chain consensus.ChainReader, epochID uint64, slotCount int) ([]common.Address, []int, float64, int) {
	if chain == nil {
		log.SyslogErr("getSlotLeaderActivity chain reader is empty.")
		return []common.Address{}, []int{}, float64(0), 0
	}
	ctrlCount := 0
	currentNumber := chain.CurrentHeader().Number.Uint64()
	if currentNumber == 0 {
		return []common.Address{}, []int{}, float64(0), 0
	}
	miners := make(map[common.Address]int)
	for i := currentNumber - 1; (i >= util.FirstPosBlockNumber()) && (i != 0); i-- {
		header := chain.GetHeaderByNumber(i)
		if header == nil {
			continue
		}

		epID := getEpochIDFromDifficulty(header.Difficulty)
		if epID == epochID {
			if isInWhiteList(header.Coinbase) {
				ctrlCount++
				continue
			}

			cnt, ok := miners[header.Coinbase]
			if ok {
				cnt++
				miners[header.Coinbase] = cnt
			} else {
				miners[header.Coinbase] = 1
			}
		}

		if epID < epochID {
			break
		}
	}

	addrs := make([]common.Address, 0)
	blocks := make([]int, 0)

	for k, v := range miners {
		addrs = append(addrs, k)
		blocks = append(blocks, v)
	}

	epochBlockCnt := sumIntArray(blocks)
	epochBlockCnt += ctrlCount
	if epochBlockCnt > slotCount {
		epochBlockCnt = slotCount
	}
	activePercent := float64(epochBlockCnt) / float64(slotCount)
	return addrs, blocks, activePercent, ctrlCount
}
