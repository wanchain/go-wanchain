package posapi

import (
	"encoding/hex"
	"fmt"

	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/postools"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/rpc"
)

type PosApi struct {
	chain consensus.ChainReader
}

func APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "pos",
		Version:   "1.0",
		Service:   &PosApi{chain: chain},
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

func (a PosApi) GetSlotLeadersByEpochID(epochID uint64) string {
	info := ""
	for i := uint64(0); i < pos.SlotCount; i++ {
		buf, err := posdb.GetDb().GetWithIndex(epochID, i, slotleader.SlotLeader)
		if err != nil {
			info += fmt.Sprintf("epochID:%d, index:%d, error:%s \n", err.Error())
		}
		info += fmt.Sprintf("epochID:%d, index:%d, pk:%s \n", epochID, i, hex.EncodeToString(buf))
	}

	return info
}

func (a PosApi) GetEpochLeadersByEpochID(epochID uint64) string {
	info := ""

	type epoch interface {
		GetEpochLeaders(epochID uint64) [][]byte
	}

	selector := posdb.GetEpocherInst()

	if selector == nil {
		return "GetEpocherInst error"
	}

	epochLeaders := selector.(epoch).GetEpochLeaders(epochID)
	info += fmt.Sprintf("epoch leader count:%d \n", len(epochLeaders))

	for i := 0; i < len(epochLeaders); i++ {
		info += fmt.Sprintf("epochID:%d, index:%d, pk:%s \n", epochID, i, hex.EncodeToString(epochLeaders[i]))
	}

	return info
}

func (a PosApi) GetSmaByEpochID(epochID uint64) string {
	pks, err := slotleader.GetSlotLeaderSelection().GetSma(epochID)
	info := "" + err.Error() + "\n"
	info += fmt.Sprintf("sma count:%d \n", len(pks))

	for i := 0; i < len(pks); i++ {
		info += fmt.Sprintf("epochID:%d, index:%d, SMA:%s \n", epochID, i, hex.EncodeToString(crypto.FromECDSAPub(pks[i])))
	}

	return info
}

func (a PosApi) GetRandomProposersByEpochID(epochID uint64) string {
	info := ""

	type epoch interface {
		GetRBProposerGroup(epochID uint64) []bn256.G1
	}

	selector := posdb.GetEpocherInst()

	if selector == nil {
		return "GetEpocherInst error"
	}

	leaders := selector.(epoch).GetRBProposerGroup(epochID)
	info += fmt.Sprintf("random proposer count:%d \n", len(leaders))

	for i := 0; i < len(leaders); i++ {
		info += fmt.Sprintf("epochID:%d, index:%d, random proposer:%s \n", epochID, i, hex.EncodeToString(leaders[i].Marshal()))
	}

	return info
}
