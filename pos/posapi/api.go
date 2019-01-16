package posapi

import (
	"encoding/hex"
	"fmt"

	"context"
	"errors"
	"math/big"

	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/internal/ethapi"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/postools"
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

func (a PosApi) GetSlotLeadersByEpochID(epochID uint64) map[uint64]string {
	infoMap := make(map[uint64]string, 0)
	for i := uint64(0); i < pos.SlotCount; i++ {
		buf, err := posdb.GetDb().GetWithIndex(epochID, i, slotleader.SlotLeader)
		if err != nil {
			infoMap[i] = fmt.Sprintf("epochID:%d, index:%d, error:%s \n", err.Error())
		} else {
			infoMap[i] = hex.EncodeToString(buf)
		}
	}

	return infoMap
}

func (a PosApi) GetEpochLeadersByEpochID(epochID uint64) (map[int]string, error) {
	infoMap := make(map[int]string, 0)

	type epoch interface {
		GetEpochLeaders(epochID uint64) [][]byte
	}

	selector := posdb.GetEpocherInst()

	if selector == nil {
		return nil, errors.New("GetEpocherInst error")
	}

	epochLeaders := selector.(epoch).GetEpochLeaders(epochID)

	for i := 0; i < len(epochLeaders); i++ {
		infoMap[i] = hex.EncodeToString(epochLeaders[i])
	}

	return infoMap, nil
}

func (a PosApi) GetLocalPK() ([]byte, error) {
	pk, err := slotleader.GetSlotLeaderSelection().GetLocalPublicKey()
	if err != nil {
		return nil, err
	}

	return crypto.FromECDSAPub(pk), nil
}

func (a PosApi) GetBootNodePK() ([]byte, error) {
	return hex.DecodeString(pos.GenesisPK)
}

func (a PosApi) GetSlotScCallTimesByEpochID(epochID uint64) uint64 {
	return vm.GetSlotScCallTimes(epochID)
}

func (a PosApi) GetSmaByEpochID(epochID uint64) string {
	pks, err := slotleader.GetSlotLeaderSelection().GetSma(epochID)
	info := ""
	if err != nil {
		info = "" + err.Error() + "\n"
	}

	info += fmt.Sprintf("sma count:%d \n", len(pks))

	for i := 0; i < len(pks); i++ {
		info += fmt.Sprintf("epochID:%d, index:%d, SMA:%s \n", epochID, i, hex.EncodeToString(crypto.FromECDSAPub(pks[i])))
	}

	return info
}

func (a PosApi) GetRandomProposersByEpochID(epochID uint64) string {
	info := ""

	leaders := posdb.GetRBProposerGroup(epochID)
	info += fmt.Sprintf("random proposer count:%d \n", len(leaders))

	for i := 0; i < len(leaders); i++ {
		info += fmt.Sprintf("epochID:%d, index:%d, random proposer:%s \n", epochID, i, hex.EncodeToString(leaders[i].Marshal()))
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
