package pos

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/incentive"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
)

type PosAvgRet struct {
	avgdb *posdb.Db

}

var posavgret *PosAvgRet
var Testinjected = false

func NewPosAveRet() *PosAvgRet {

	if posavgret == nil {
		db :=  posdb.NewDb(posconfig.AvgRetDB)
		posavgret = &PosAvgRet{avgdb:db}
	}

	util.SetPosAvgInst(posavgret)

	return posavgret
}



func (p *PosAvgRet) GetOneEpochAvgReturnFor90LockEpoch(epochID uint64) (uint64, error) {

	val,err :=p.avgdb.GetWithIndex(epochID,0,"")
	if err == nil && val != nil{
		return binary.BigEndian.Uint64(val),nil
	}

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

		fmt.Println(staker.LockEpochs)

		//if staker.LockEpochs == posconfig.TARGETS_LOCKED_EPOCH {
		if staker.LockEpochs <= posconfig.TARGETS_LOCKED_EPOCH {
			stakerSet[staker.Address] = staker.Amount
			stakeTotal = stakeTotal.Add(stakeTotal,staker.Amount)
		}


		return true

	})

///////////////////////////////test code/////////////////////////////

	validator := []string{	"0xf7a2681f8Cf9661B6877de86034166422cd8C308",
							"0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8",
							"0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e",
						 }

	stakeTotal = big.NewInt(0)
	for i:=0;i<len(validator);i++ {
		addr := common.HexToAddress(validator[i])
		stakerSet[addr] = big.NewInt(10)

		stk,_ := big.NewInt(0).SetString("10000000000000000000",10)

		stakeTotal =  big.NewInt(0).Add(stakeTotal,stk)
	}

////////////////////////////////////////////////////////////////////////


	if stakeTotal.Cmp(big.NewInt(0)) == 0 {
		return 0, errors.New("not get staker")
	}

	c, err := incentive.GetEpochPayDetail(epochID)
	if err != nil {
		return 0, err
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

	incentiveTotal = big.NewInt(0).Mul(incentiveTotal,big.NewInt(posconfig.RETURN_DIVIDE))
	ret :=  big.NewInt(0).Div(incentiveTotal,stakeTotal).Uint64()



	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, ret)
	p.avgdb.PutWithIndex(epochID,0,"",buf)


	return ret,nil

}



func (p *PosAvgRet) GetAllStakeAndReturn(epochID uint64) (*big.Int, error) {

	targetBlkNum := epochLeader.GetEpocher().GetTargetBlkNumber(epochID)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return nil,errors.New("epocher instance do not exist")
	}

	//block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	block := epocherInst.GetBlkChain().GetHeaderByNumber(targetBlkNum)
	if block == nil {
		return nil,errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root)
	if err != nil {
		return nil,err
	}

	totalAmount := stateDb.GetBalance(vm.WanCscPrecompileAddr)



	return totalAmount,nil

}