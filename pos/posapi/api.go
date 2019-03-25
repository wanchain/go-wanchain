package posapi

import (
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/rlp"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/incentive"

	"context"
	"errors"
	"math/big"

	"encoding/binary"

	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/internal/ethapi"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/rpc"
)

type PosApi struct {
	chain   consensus.ChainReader
	backend ethapi.Backend
}

func APIs(chain consensus.ChainReader, backend ethapi.Backend) []rpc.API {
	return []rpc.API{{
		Namespace: "pos",
		Version:   "1.0",
		Service:   &PosApi{chain, backend},
		Public:    true,
	}}
}

func (a PosApi) Version() string {
	return "1.0"
}

func (a PosApi) GetSlotLeadersByEpochID(epochID uint64) map[string]string {
	infoMap := make(map[string]string, 0)
	for i := uint64(0); i < posconfig.SlotCount; i++ {
		buf, err := posdb.GetDb().GetWithIndex(epochID, i, slotleader.SlotLeader)
		if err != nil {
			infoMap[fmt.Sprintf("%06d", i)] = fmt.Sprintf("epochID:%d, index:%d, error:%s \n", epochID, i, err.Error())
		} else {
			infoMap[fmt.Sprintf("%06d", i)] = hex.EncodeToString(buf)
		}
	}

	return infoMap
}

func (a PosApi) GetEpochLeadersByEpochID(epochID uint64) (map[string]string, error) {
	infoMap := make(map[string]string, 0)

	type epoch interface {
		GetEpochLeaders(epochID uint64) [][]byte
	}

//	selector := posdb.GetEpocherInst()
	selector := epochLeader.GetEpocher()

	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}

	epochLeaders := selector.GetEpochLeaders(epochID)
	//epochLeaders := selector.(epoch).GetEpochLeaders(epochID)

	for i := 0; i < len(epochLeaders); i++ {
		infoMap[fmt.Sprintf("%06d", i)] = hex.EncodeToString(epochLeaders[i])
	}

	return infoMap, nil
}

func (a PosApi) GetLocalPK() (string, error) {
	pk, err := slotleader.GetSlotLeaderSelection().GetLocalPublicKey()
	if err != nil {
		return "nil", err
	}

	return hex.EncodeToString(crypto.FromECDSAPub(pk)), nil
}

func (a PosApi) GetBootNodePK() string {
	return posconfig.GenesisPK
}

func (a PosApi) GetSlotScCallTimesByEpochID(epochID uint64) uint64 {
	return vm.GetSlotScCallTimes(epochID)
}

func (a PosApi) GetSmaByEpochID(epochID uint64) (map[string]string, error) {
	pks, _, err := slotleader.GetSlotLeaderSelection().GetSma(epochID)
	if err != nil {
		return nil, err
	}

	info := make(map[string]string, len(pks))

	for i := 0; i < len(pks); i++ {
		info[fmt.Sprintf("%06d", i)] = hex.EncodeToString(crypto.FromECDSAPub(pks[i]))
	}

	return info, nil
}

func (a PosApi) GetRandomProposersByEpochID(epochID uint64) map[string]string {
	leaders := posdb.GetRBProposerGroup(epochID)
	info := make(map[string]string, 0)
	for i := 0; i < len(leaders); i++ {
		info[fmt.Sprintf("%06d", i)] = hex.EncodeToString(leaders[i].Marshal())
	}
	return info
}

func (a PosApi) GetSlotCreateStatusByEpochID(epochID uint64) bool {
	return slotleader.GetSlotLeaderSelection().GetSlotCreateStatusByEpochID(epochID)
}

func (a PosApi) Random(epochId uint64, blockNr int64) (*big.Int, error) {
	state, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(blockNr))
	if err != nil {
		return nil, err
	}

	r := vm.GetStateR(state, epochId)
	if r == nil {
		return nil, errors.New("no random number exists")
	}

	return r, nil
}

func (a PosApi) GetReorg(epochID uint64) ([]uint64, error) {
	reOrgDb := posdb.GetDbByName("forkdb")
	if reOrgDb == nil {
		return nil, errors.New("not find db")
	}

	var forkNum, reOrgNum, reOrgLen uint64

	forkNum = 0
	reOrgNum = 0

	forkBytes, err := reOrgDb.Get(epochID, "forkNumber")
	if err == nil && forkBytes != nil {
		forkNum = binary.BigEndian.Uint64(forkBytes)
	}

	reorBytes, err := reOrgDb.Get(epochID, "reorgNumber")
	if err == nil && reorBytes != nil {
		reOrgNum = binary.BigEndian.Uint64(reorBytes)
	}

	lenBytes, err := reOrgDb.Get(epochID, "reorgLength")
	if err == nil && reorBytes != nil {
		reOrgLen = binary.BigEndian.Uint64(lenBytes)
	}

	return []uint64{forkNum, reOrgNum, reOrgLen}, nil
}

func (a PosApi) GetSijCount(epochId uint64, blockNr int64) (int, error) {
	state, _, err := a.backend.StateAndHeaderByNumber(context.Background(), rpc.BlockNumber(blockNr))
	if err != nil {
		return 0, err
	}
	j := 0
	for i := 0; i < posconfig.RandomProperCount; i++ {
		sigData, err := vm.GetSig(state, epochId, uint32(i))
		if err != nil {
			return 0, err
		}
		if sigData != nil {
			j++
		}
	}
	return j, nil
}

type StakerInfo struct {
	Addr      		 common.Address
	Infors           []vm.ClientProbability
	FeeRate          uint64
	TotalProbability *big.Int
}


func (a PosApi) GetEpochStakerInfo(epochID uint64, addr common.Address) (StakerInfo, error) {
	skInfo := StakerInfo{}
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return skInfo, errors.New("epocher instance does not exist")
	}
	infors, feeRate, total, err := epocherInst.GetEpochProbability(epochID, addr)
	if err != nil {
		return skInfo, err
	}
	skInfo.TotalProbability = total
	skInfo.FeeRate = feeRate
	skInfo.Infors = infors
	skInfo.Addr = addr
	return skInfo, nil
}
type StakerJson struct {
	Address   common.Address
	PubSec256 string //stakeholder’s wan public key
	PubBn256  string //stakeholder’s bn256 public key

	Amount     *big.Int //staking wan value
	LockEpochs uint64   //lock time which is input by user. 0 means unexpired.
	From       common.Address

	StakingEpoch uint64 //the user’s staking time
	FeeRate      uint64
	Clients      []vm.ClientInfo
}
// this is the static snap of stekers by the block Number.
func (a PosApi) GetStakerInfo(targetBlkNum uint64) ([]StakerJson, error) {
	stakers := make([]StakerJson, 0)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return stakers, errors.New("epocher instance do not exist")
	}

	block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root())
	if err != nil {
		return stakers, err
	}
	stateDb.ForEachStorageByteArray(vm.StakersInfoAddr, func(key common.Hash, value []byte) bool {

		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(value, &staker)
		if err != nil {
			log.Error(err.Error())
			return true
		}
		stakeJson := StakerJson{}
		stakeJson.Address = staker.Address
		stakeJson.Amount = staker.Amount
		stakeJson.LockEpochs = staker.LockEpochs
		stakeJson.From = staker.From
		stakeJson.StakingEpoch = staker.StakingEpoch
		stakeJson.FeeRate = staker.FeeRate
		stakeJson.Clients = staker.Clients
		stakeJson.PubSec256 = hexutil.Encode(staker.PubSec256)
		stakeJson.PubBn256 = hexutil.Encode(staker.PubBn256)
		stakers = append(stakers, stakeJson)
		return true
	})
	return stakers, nil
}

func (a PosApi) GetEpochStakerInfoAll(epochID uint64) ([]StakerInfo, error) {
	targetBlkNum := epochLeader.GetEpocher().GetTargetBlkNumber(epochID)
	epocherInst := epochLeader.GetEpocher()
	if epocherInst == nil {
		return nil, errors.New("epocher instance do not exist")
	}
	block := epocherInst.GetBlkChain().GetBlockByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := epocherInst.GetBlkChain().StateAt(block.Root())
	if err != nil {
		return nil, err
	}
	ess := make([]StakerInfo,0)
	stateDb.ForEachStorageByteArray(vm.StakersInfoAddr, func(key common.Hash, value []byte) bool {
		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(value, &staker)
		if err != nil {
			log.Error(err.Error())
			return true
		}
		es := StakerInfo{}
		es.Infors = make([]vm.ClientProbability, 0)
		pb := epocherInst.CalProbability(epochID, staker.Amount, staker.LockEpochs, staker.StakingEpoch)
		for i := 0; i < len(staker.Clients); i++ {
			lockEpoch := staker.LockEpochs - (staker.Clients[i].StakingEpoch - staker.StakingEpoch)
			pc := epocherInst.CalProbability(epochID, staker.Clients[i].Amount, lockEpoch, staker.Clients[i].StakingEpoch)
			vc := vm.ClientProbability{}
			vc.Probability = pc
			vc.Addr = 	staker.Clients[i].Address
			es.Infors = append(es.Infors, vc)
			pb.Add(pb, pc)
		}
		es.TotalProbability = pb
		es.FeeRate = staker.FeeRate
		es.Addr = staker.Address
		ess = append(ess, es)
		return true
	})
	return ess, nil
}

func biToString(value *big.Int, err error) (string, error) {
	if err != nil {
		return "", nil
	}
	return value.String(), err
}
func (a PosApi) GetEpochIncentivePayDetail(epochID uint64) ([][]vm.ClientIncentive, error) {
	return incentive.GetEpochPayDetail(epochID)
}

func (a PosApi) GetTotalIncentive() (string, error) {
	return biToString(incentive.GetTotalIncentive())
}

func (a PosApi) GetEpochIncentive(epochID uint64) (string, error) {
	return biToString(incentive.GetEpochIncentive(epochID))
}

func (a PosApi) GetEpochRemain(epochID uint64) (string, error) {
	return biToString(incentive.GetEpochRemain(epochID))
}

func (a PosApi) GetTotalRemain() (string, error) {
	return biToString(incentive.GetTotalRemain())
}

func (a PosApi) GetIncentiveRunTimes() (string, error) {
	return biToString(incentive.GetRunTimes())
}

func (a PosApi) GetEpochGasPool(epochID uint64) (string, error) {
	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return "", err
	}
	return incentive.GetEpochGasPool(db, epochID).String(), nil
}

func (a PosApi) GetRBAddress(epochID uint64) []common.Address {
	return incentive.GetRBAddress(epochID)
}

func (a PosApi) GetIncentivePool(epochID uint64) ([]string, error) {
	s := slotleader.GetSlotLeaderSelection()
	db, err := s.GetCurrentStateDb()
	if err != nil {
		return nil, err
	}
	total, foundation, gasPool := incentive.GetIncentivePool(db, epochID)
	return []string{total.String(), foundation.String(), gasPool.String()}, nil
}
