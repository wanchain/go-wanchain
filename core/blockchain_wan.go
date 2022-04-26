// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package core implements the Ethereum consensus protocol.
package core

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"github.com/ethereum/go-ethereum/pos/posdb"
	"github.com/ethereum/go-ethereum/pos/util"
	posUtil "github.com/ethereum/go-ethereum/pos/util"
)

import (
	"encoding/binary"
	"math/big"
	"time"
)

func (bc *BlockChain) SetSlotValidator(validator Validator) {
	bc.slotValidator = validator
}

// Validator returns the current validator.
func (bc *BlockChain) SlotValidator() Validator {
	return bc.slotValidator
}

// if current block number +1 is >= pos first block
func (bc *BlockChain) IsInPosStage() bool {
	currentBlockNumber := bc.CurrentBlock().Number()
	currentBlockNumber = currentBlockNumber.Add(currentBlockNumber, big.NewInt(1))
	return bc.chainConfig.IsPosBlockNumber(currentBlockNumber)
}

func (bc *BlockChain) GetFirstPosBlockNumber() uint64 {
	return bc.Config().PosFirstBlock.Uint64()
}

func (bc *BlockChain) ChainRestartStatus() (bool, *types.Block) {

	//it is chain restarting phase if chain is restarted and current slot not more 1 epoch than start slot
	diff := bc.checkCQStartSlot - bc.stopSlot
	if diff > posconfig.SlotSecurityParam-1 &&
		bc.checkCQStartSlot > 0 &&
		bc.stopSlot > 0 {
		return true, bc.checkCQBlk
	}

	return false, nil
}

func (bc *BlockChain) SetChainRestartSuccess() {

	log.Info("")

	bc.checkCQBlk = nil
	bc.checkCQStartSlot = 0
	bc.stopSlot = 0
	bc.restartSucess = true
}

func (bc *BlockChain) SetRestartBlock(block *types.Block, preBlock *types.Block, useLocalTime bool) {

	if useLocalTime {

		epid, slid := posUtil.CalEpochSlotID(uint64(time.Now().Unix()))
		//record the restarting slot point
		bc.checkCQStartSlot = epid*posconfig.SlotCount + slid

		lastepid, lastlslid := posUtil.CalEpSlbyTd(block.Difficulty().Uint64())
		bc.stopSlot = lastepid*posconfig.SlotCount + lastlslid

		bc.restartSucess = false

	} else if block != nil && preBlock != nil {

		bc.checkCQBlk = block
		epid, slid := posUtil.CalEpSlbyTd(block.Difficulty().Uint64())
		//record the restarting slot point
		bc.checkCQStartSlot = epid*posconfig.SlotCount + slid

		stopepid, stoplslid := posUtil.CalEpSlbyTd(preBlock.Difficulty().Uint64())
		bc.stopSlot = stopepid*posconfig.SlotCount + stoplslid

		res, _ := bc.ChainRestartStatus()
		if res {
			bc.restartSucess = false
		}
	}

}

func (bc *BlockChain) checkRestarting(chain types.Blocks) ([]uint, error) {
	idxs := make([]uint, 0)
	for i, block := range chain {

		if block.NumberU64() <= posconfig.Pow2PosUpgradeBlockNumber+2 {
			continue
		}
		//it is chain restarting phase if chain is restarted and current slot not more 1 epoch than start slot
		epid, slid := posUtil.CalEpSlbyTd(block.Difficulty().Uint64())
		curSlots := epid*posconfig.SlotCount + slid

		var preBlock *types.Block
		if i == 0 {
			preBlock = bc.GetBlockByHash(block.ParentHash())
		} else {
			preBlock = chain[i-1]
		}

		if preBlock == nil {
			return nil, errors.New("can not find parent block in check restart")
		}

		preepid, preslid := posUtil.CalEpSlbyTd(preBlock.Difficulty().Uint64())
		preSlots := preepid*posconfig.SlotCount + preslid

		diff := curSlots - preSlots

		//log.Info("the slot diff","diff",diff)
		if diff > posconfig.SlotSecurityParam-1 {
			idxs = append(idxs, uint(i))
			//fmt.Println("restart point=",i)
		}
	}

	return idxs, nil

}
func (bc *BlockChain) updateReOrg(epochId uint64, slotid uint64, length uint64) {

	reOrgDb := posdb.GetDbByName(posconfig.ReorgLocalDB)
	if reOrgDb == nil {
		reOrgDb = posdb.NewDb(posconfig.ReorgLocalDB)
	}

	numberBytes, _ := reOrgDb.Get(epochId, "reorgNumber")

	num := uint64(0)
	if numberBytes != nil {
		num = binary.BigEndian.Uint64(numberBytes) + 1
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, num)

	reOrgDb.Put(epochId, "reorgNumber", b)

	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, length)
	reOrgDb.Put(epochId, "reorgLength", b)
}

func (bc *BlockChain) SwitchClientEngine() error {
	for _, agent := range bc.agents {
		agent.SwitchEngine(bc.posEngine)
	}

	bc.engine = bc.posEngine

	return nil
}

func PeekChainHeight(db ethdb.Database) uint64 {
	head := GetHeadBlockHash(db)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		log.Warn("Empty database, resetting chain")
		return uint64(0)
	}

	return GetBlockNumber(db, head)
}

// Count blocks in front of specified block within 2k slots(exclude the specified block!!!).
// pos block number begin with 1, epoc and slot index begin from 0
//posconfig.SlotSecurityParam
func (bc *BlockChain) getBlocksCountIn2KSlots(blk *types.Block, secPara uint64) int {
	epochId, slotId := posUtil.CalEpochSlotID(blk.Time())
	endFlatSlotId := epochId*posconfig.SlotCount + slotId

	if endFlatSlotId == 0 {
		return 0
	}

	startFlatSlotId := uint64(0)
	if endFlatSlotId >= secPara {
		startFlatSlotId = endFlatSlotId - secPara
	}

	n := 0

	blkHeader := blk.Header()
	blkNumber := blk.NumberU64()
	for {

		blkNumber = blkHeader.Number.Uint64() - 1
		blkHeader = bc.GetHeaderByNumber(blkNumber)

		if nil == blkHeader {
			//never reached, because ppow blocks, safely remove this code?
			break
		}

		if blkHeader.Number.Cmp(bc.chainConfig.PosFirstBlock) < 0 {
			break
		}

		epochId, slotId = posUtil.CalEpochSlotID(blkHeader.Time)
		flatSlotId := epochId*posconfig.SlotCount + slotId
		if flatSlotId < startFlatSlotId || flatSlotId >= endFlatSlotId {
			break
		}
		n = n + 1

		if flatSlotId == uint64(0) {
			break
		}
	}

	return n
}

func (bc *BlockChain) ChainQualityHistory(epochid uint64, slotid uint64) (uint64, error) {
	curBlk := bc.CurrentBlock()
	startBlkNumber := curBlk.NumberU64() - posconfig.BlockSecurityParam

	for {

		bh := bc.GetHeaderByNumber(startBlkNumber)
		if bh != nil {
			blkEpid, blkSlid := posUtil.CalEpSlbyTd(bh.Difficulty.Uint64())
			if epochid <= blkEpid && blkSlid <= slotid {
				break
			}
		}

		startBlkNumber--
		if startBlkNumber < util.FirstPosBlockNumber() {
			return uint64(0), nil
		}
	}

	tblk := bc.GetBlockByNumber(startBlkNumber)
	blocksIn2K := bc.getBlocksCountIn2KSlots(tblk, posconfig.SlotSecurityParam)

	quality := blocksIn2K * 1000 / (posconfig.SlotSecurityParam)

	return uint64(quality), nil
}

func (bc *BlockChain) isWriteBlockSecure(block *types.Block) bool {

	epochId, slotId := posUtil.CalEpochSlotID(block.Time())

	endFlatSlotId := epochId*posconfig.SlotCount + slotId
	startId := endFlatSlotId - posconfig.SlotSecurityParam - 1

	if bc.cqCache.Len() > posconfig.BlockSecurityParam {

		if startId > bc.cqLastSlot {
			bc.cqCache.Purge()
		} else {
			k := bc.cqLastSlot - posconfig.SlotSecurityParam
			for ; k <= startId; k++ {
				bc.cqCache.Remove(k)
			}
		}

		blocksIn2K := bc.cqCache.Len()

		return blocksIn2K > posconfig.K
	}

	return false
}

func (bc *BlockChain) ChainQuality(epochid uint64, slotid uint64) (uint64, error) {

	blocksIn2K := 0

	curBlk := bc.CurrentBlock()

	blkEpid, blkSlid := posUtil.CalEpSlbyTd(curBlk.Difficulty().Uint64())
	blkSlots := blkEpid*posconfig.SlotCount + blkSlid

	expSlots := epochid*posconfig.SlotCount + slotid

	//get chainquality in history
	if expSlots < blkSlots-posconfig.BlockSecurityParam {
		return bc.ChainQualityHistory(epochid, slotid)
	}

	if expSlots >= (blkSlots+posconfig.SlotSecurityParam) || (epochid == posconfig.FirstEpochId && slotid == 0) {
		return 0, errors.New("wrong epoid or slotid")
	}

	//lastBlock := bc.epochGene.rbLeaderSelector.GetEpochLastBlkNumber(epochid)
	checkSlots := uint64(0)

	lastBlock := posUtil.GetEpochBlock(blkEpid)
	for i := lastBlock; i > 0; i-- {

		curBlkHeader := bc.GetHeaderByNumber(i)
		blkEpid, blkSlid = posUtil.CalEpSlbyTd(curBlkHeader.Difficulty.Uint64())
		checkSlots = blkEpid*posconfig.SlotCount + blkSlid

		if checkSlots <= expSlots {
			break
		}
	}

	//if the gap is empty block,then the quality is 0
	diff := expSlots - checkSlots
	if uint64(diff) >= posconfig.SlotSecurityParam {
		return uint64(0), nil
	} else {

		flatSlotId := epochid*posconfig.SlotCount + slotid

		cacheBeginId := bc.cqLastSlot - posconfig.SlotSecurityParam
		if flatSlotId <= bc.cqLastSlot && flatSlotId > cacheBeginId && bc.cqCache.Len() > posconfig.BlockSecurityParam {

			for ; flatSlotId > cacheBeginId; flatSlotId-- {
				blks, ok := bc.cqCache.Get(flatSlotId)
				if ok && blks != 0 {
					blocksIn2K++
				}
			}
		}

		if blocksIn2K == 0 {
			blocksIn2K = bc.getBlocksCountIn2KSlots(curBlk, posconfig.SlotSecurityParam-diff)
		}

		quality := blocksIn2K * 1000 / (posconfig.SlotSecurityParam)

		return uint64(quality), nil
	}

}

func (bc *BlockChain) biggerThanCriticalBlock(block *types.Block) bool {

	diff := int(posconfig.Cfg().SyncTargetBlokcNum - block.NumberU64())
	if diff > 2*posconfig.SlotSecurityParam {
		return false
	} else {
		return true
	}
}
