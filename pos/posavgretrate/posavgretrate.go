package pos

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/rlp"
)

func (a PosApi) GetEpochStakerInfoAll(epochID uint64) ([]ApiStakerInfo, error) {
	targetBlkNum := epochLeader.GetEpocher().GetTargetBlkNumber(epochID)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return nil, errors.New("epocher instance do not exist")
	}
	//block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	block := epocherInst.GetBlkChain().GetHeaderByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root)
	if err != nil {
		return nil, err
	}
	ess := make([]ApiStakerInfo, 0)
	stateDb.ForEachStorageByteArray(vm.StakersInfoAddr, func(key common.Hash, value []byte) bool {
		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(value, &staker)
		if err != nil {
			log.SyslogErr(err.Error())
			return true
		}

		infors, pb, err := epochLeader.CalEpochProbabilityStaker(&staker, epochID)
		if err != nil || pb == nil {
			// this validator has no enough
			return true
		}

		es := ApiStakerInfo{}
		es.Infors = make([]ApiClientProbability, len(infors))
		for i := 0; i < len(infors); i++ {
			if i == 0 {
				es.Infors[i].Addr = infors[i].ValidatorAddr
			} else {
				es.Infors[i].Addr = infors[i].WalletAddr
			}
			es.Infors[i].Probability = (*math.HexOrDecimal256)(infors[i].Probability)
		}
		es.TotalProbability = (*math.HexOrDecimal256)(pb)
		es.FeeRate = staker.FeeRate
		es.Addr = staker.Address
		ess = append(ess, es)
		return true
	})
	return ess, nil
}
