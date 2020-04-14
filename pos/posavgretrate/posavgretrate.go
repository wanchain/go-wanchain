package pos

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/incentive"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
)

type PosAvgRet struct {

}

var posavgret *PosAvgRet

func NewPosAveRet() *PosAvgRet {

	if posavgret == nil {
		posavgret = &PosAvgRet{}
	}

	util.SetPosAvgInst(posavgret)

	return posavgret
}

func (p *PosAvgRet) GetOneEpochAvgReturn(epochID uint64) (uint64, error) {

	targetBlkNum := epochLeader.GetEpocher().GetTargetBlkNumber(epochID)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return 0, errors.New("epocher instance do not exist")
	}

	//block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	block := epocherInst.GetBlkChain().GetHeaderByNumber(targetBlkNum)
	if block == nil {
		return 0, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root)
	if err != nil {
		return uint64(0), err
	}

	stakerSet := make(map[common.Address]*big.Int)
	stakeTotal := big.NewInt(0)
	stateDb.ForEachStorageByteArray(vm.StakersInfoAddr, func(key common.Hash, value []byte) bool {

		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(value, &staker)
		if err != nil {
			log.SyslogErr(err.Error())
			return true
		}

		if staker.LockEpochs == posconfig.TARGETS_LOCKED_EPOCH {
			stakerSet[staker.Address] = staker.Amount
			stakeTotal = stakeTotal.Add(stakeTotal,staker.Amount)
		}


		return true

	})


	c, err := incentive.GetEpochPayDetail(epochID)
	if err != nil {
		return 0, nil
	}

	incentiveTotal := big.NewInt(0)
	for i := 0; i < len(c); i++ {
		if len(c[i]) == 0 {
			continue
		}
		if _,ok := stakerSet[c[i][0].ValidatorAddr]; ok {
			incentiveTotal = incentiveTotal.Add(incentiveTotal,c[i][0].Incentive)
		}
	}

	ret := incentiveTotal.Mul(incentiveTotal,big.NewInt(posconfig.RETURN_DIVIDE)).Div(incentiveTotal,stakeTotal).Uint64()

	return ret,nil

}



