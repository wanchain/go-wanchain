package pos

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/pos/epochLeader"
	"github.com/ethereum/go-ethereum/pos/incentive"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"github.com/ethereum/go-ethereum/pos/posdb"
	"github.com/ethereum/go-ethereum/pos/util"
)

type PosAvgRet struct {
	avgdb *posdb.Db
}

var posavgret *PosAvgRet
var Testinjected = false // TODO: remove

func NewPosAveRet() *PosAvgRet {

	if posavgret == nil {
		db := posdb.NewDb(posconfig.AvgRetDB)
		posavgret = &PosAvgRet{avgdb: db}
	}

	util.SetPosAvgInst(posavgret)
	return posavgret
}

func (p *PosAvgRet) GetPosAverageReturnRate(epochID uint64) (uint64, error) {

	// val, err := p.avgdb.GetWithIndex(epochID, 0, "")

	// if err == nil && val != nil {
	// 	p2 := binary.BigEndian.Uint64(val)
	// 	if p2 != 0 {
	// 		return p2, nil
	// 	}
	// }

	reward := incentive.YearReward(epochID)

	amount, err := p.GetAllStakeAndReturn(epochID)
	if err != nil {
		return 0, err
	}

	if amount.Int64() == 0 {
		return 0, nil
	}

	a := reward.Mul(reward, big.NewInt(posconfig.RETURN_DIVIDE))

	p2 := a.Div(a, amount).Uint64()
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, p2)

	// p.avgdb.PutWithIndex(epochID, 0, "", buf)

	return p2, nil

}

func (p *PosAvgRet) GetAllStakeAndReturn(epochID uint64) (*big.Int, error) {

	targetBlkNum := util.GetEpochBlock(epochID)
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

	totalAmount := stateDb.GetBalance(vm.WanCscPrecompileAddr)

	return totalAmount, nil

}

func (p *PosAvgRet) GetAllIncentive(epochID uint64) (*big.Int, error) {
	return incentive.GetEpochIncentive(epochID)
}
func (p *PosAvgRet) GetYearReward(epochID uint64) *big.Int {
	return incentive.YearReward(epochID)
}
