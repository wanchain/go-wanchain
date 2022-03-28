package incentive

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"github.com/ethereum/go-ethereum/pos/util"
	"github.com/ethereum/go-ethereum/pos/util/convert"
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

func getSlotLeaderActivity(chain consensus.ChainHeaderReader, epochID uint64, slotCount int, headerInput *types.Header) ([]common.Address, []int, float64, int) {
	if chain == nil {
		log.SyslogErr("getSlotLeaderActivity chain reader is empty.")
		return []common.Address{}, []int{}, float64(0), 0
	}

	if headerInput == nil {
		log.SyslogErr("getSlotLeaderActivity headerInput is nil.")
		return []common.Address{}, []int{}, float64(0), 0
	}

	header := headerInput
	currentNumber := header.Number.Uint64()
	log.Info("getSlotLeaderActivity", "currentNumber", currentNumber, "epochID", epochID)

	ctrlCount := 0
	if currentNumber == 0 {
		return []common.Address{}, []int{}, float64(0), 0
	}
	miners := make(map[common.Address]int)
	for i := currentNumber - 1; (i >= util.FirstPosBlockNumber()) && (i != 0); i-- {
		header = chain.GetHeaderByHash(header.ParentHash)
		if header == nil {
			log.Error("getSlotLeaderActivity header == nil")
			break
		}

		epID := getEpochIDFromDifficulty(header.Difficulty)
		log.Debug("getSlotLeaderActivity find", "header", header.Hash, "Number", header.Number, "epochID", epID)

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
			log.Info("getSlotLeaderActivity finish", "header", header.Hash, "Number", header.Number, "epochID", epID)
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
