package pos

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/incentive"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
	"strconv"
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

func (p *PosAvgRet) getStakeInfo(statedb *state.StateDB, addr common.Address) (*vm.StakerInfo, error) {

	key := vm.GetStakeInKeyHash(addr)
	stakerBytes, err := vm.GetInfo(statedb, vm.StakersInfoAddr, key)
	if stakerBytes == nil {
		return nil, errors.New("item doesn't exist")
	}
	var stakerInfo vm.StakerInfo
	err = rlp.DecodeBytes(stakerBytes, &stakerInfo)
	if err != nil {
		return nil, errors.New("parse staker info error")
	}
	return &stakerInfo, nil
}


//retTotal := uint64(0);
//for i:=uint64(0);i<posconfig.TARGETS_LOCKED_EPOCH;i++ {
//t1 := time.Now().Unix();
//ret,err := inst.GetOneEpochAvgReturnFor90LockEpoch(groupStartEpochId - i)
//if err!= nil {
//continue
//}
//
//t2 := time.Now().Unix();
//
//fmt.Println("get return time cost=" + convert.Uint64ToString(uint64(t2-t1)))
//
//retTotal += ret
//}
//
//p2 := uint64(retTotal/posconfig.TARGETS_LOCKED_EPOCH)

func (p *PosAvgRet) GetOneEpochAvgReturnFor90LockEpoch(epochID uint64) (uint64, error) {

	val,err :=p.avgdb.GetWithIndex(epochID,0,"")
	if err == nil && val != nil{
		return binary.BigEndian.Uint64(val),nil
	}

	retTotal := uint64(0);

	for i:=uint64(0);i<posconfig.TARGETS_LOCKED_EPOCH;i++ {

		fmt.Println("" + strconv.Itoa(int(i)))

		targetBlkNum := epochLeader.GetEpocher().GetTargetBlkNumber(epochID - i)
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
		selector := epochLeader.GetEpocher()
		if selector == nil {
			return 0, errors.New("GetEpocherInst error")
		}

		leaders := selector.GetEpochLeaders(epochID - i)
		addrs := make([]common.Address, len(leaders))
		for i := range leaders {
			pub := crypto.ToECDSAPub(leaders[i])
			if pub == nil {
				continue
			}

			addrs[i] = crypto.PubkeyToAddress(*pub)

		}

		for _, addr := range addrs {
			staker, err := p.getStakeInfo(stateDb, addr)
			if err != nil {
				continue
			}

			if staker.LockEpochs == posconfig.TARGETS_LOCKED_EPOCH {
				stakerSet[addr] = staker.Amount
			}
		}

		stakeTotal := big.NewInt(0)
		for _, val := range stakerSet {
			stakeTotal = stakeTotal.Add(stakeTotal, val)
		}

		if stakeTotal.Cmp(big.NewInt(0)) == 0 {
			return 0, errors.New("not get stake total")
		}

	///////////////////////////////test code/////////////////////////////
	/*
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
	*/
	////////////////////////////////////////////////////////////////////////

		c, err := incentive.GetEpochPayDetail(epochID - i)
		if err != nil {
			return 0, err
		}

		incentiveTotal := big.NewInt(0)
		for i := 0; i < len(c); i++ {
			if len(c[i]) == 0 {
				continue
			}

			if _, ok := stakerSet[c[i][0].ValidatorAddr]; ok {
				incentiveTotal = incentiveTotal.Add(incentiveTotal, c[i][0].Incentive)
			}
		}

		incentiveTotal = big.NewInt(0).Mul(incentiveTotal, big.NewInt(posconfig.RETURN_DIVIDE))

		retTotal += big.NewInt(0).Div(incentiveTotal, stakeTotal).Uint64()


	}

	p2 := uint64(retTotal/posconfig.TARGETS_LOCKED_EPOCH)

	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, p2)
	p.avgdb.PutWithIndex(epochID,0,"",buf)


	return p2,nil

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