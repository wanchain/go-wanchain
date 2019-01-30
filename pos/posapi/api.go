package posapi

import (
	"encoding/hex"
	"fmt"

	"context"
	"errors"
	"math/big"

	"encoding/binary"

	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/internal/ethapi"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/postools"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
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
		Public:    false,
	}}
}

func (a PosApi) Version() string {
	return "1.0"
}

func (a PosApi) GetSlotErrorCount() string {
	return postools.Uint64ToString(slotleader.ErrorCount)
}

func (a PosApi) GetSlotWarnCount() string {
	return postools.Uint64ToString(slotleader.WarnCount)
}

func (a PosApi) GetSlotLeadersByEpochID(epochID uint64) map[string]string {
	infoMap := make(map[string]string, 0)
	for i := uint64(0); i < pos.SlotCount; i++ {
		buf, err := posdb.GetDb().GetWithIndex(epochID, i, slotleader.SlotLeader)
		if err != nil {
			infoMap[fmt.Sprintf("%06d", i)] = fmt.Sprintf("epochID:%d, index:%d, error:%s \n", err.Error())
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

	selector := posdb.GetEpocherInst()

	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}

	epochLeaders := selector.(epoch).GetEpochLeaders(epochID)

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
	return pos.GenesisPK
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
	for i := 0; i < pos.RandomProperCount; i++ {
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

func (a PosApi) GetEpochStakerInfo(epochID uint64,pubk string) ([]string, error) {
	epocherInst := epochLeader.GetEpocher()

	if epocherInst == nil {
		return nil,errors.New("epocher instance do not exist")
	}

	return epocherInst.GetEpochStakers(epochID,pubk)
}
