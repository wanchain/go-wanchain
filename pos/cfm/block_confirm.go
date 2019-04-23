package cfm

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	"time"
)

const (
	// security block difference between main block chain and side block chain
	SecBlkDiff = 50
)

type CFM struct {
	bc        *core.BlockChain
	whiteList map[common.Address]int
}

type SuffixBlkStatic struct {
	SuffixBlockTrusted    uint64
	SuffixBlockNonTrusted uint64
}

var c *CFM

func InitCFM(bc *core.BlockChain) {
	c = &CFM{}
	c.bc = bc
	c.whiteList = make(map[common.Address]int, 0)
	for _, value := range posconfig.WhiteList {

		b := hexutil.MustDecode(value)
		address := crypto.PubkeyToAddress(*crypto.ToECDSAPub(b))
		c.whiteList[address] = 1
	}
}

func GetCFM() *CFM {
	return c
}

func (c *CFM) IsBlkCfm(blkNumber uint64) bool {
	if !(c.isBlkOnLocalChain(blkNumber)) {
		return false
	}
	timeNow := uint64(time.Now().Unix())
	return c.scanAndCheck(blkNumber, timeNow)
}

func (c *CFM) scanAndCheck(blkNumber, timeNow uint64) bool {
	curBlk := c.bc.CurrentBlock()
	if curBlk == nil {
		return false
	}

	if curBlk.NumberU64() <= blkNumber {
		return false
	}

	startNumber := curBlk.NumberU64()
	stopNumber := startNumber - posconfig.K
	if stopNumber < 0 {
		stopNumber = 0
	}

	sbs := SuffixBlkStatic{0, 0}
	inWhiteList := c.isInWhiteList(curBlk.Coinbase())
	parentHash := curBlk.ParentHash()
	for i := startNumber - 1; i >= stopNumber; i-- {
		blk := c.bc.GetBlock(parentHash, i)
		if blk == nil {
			return false
		}

		if inWhiteList {
			sbs.SuffixBlockTrusted = sbs.SuffixBlockTrusted + 1
		} else {
			sbs.SuffixBlockNonTrusted = sbs.SuffixBlockNonTrusted + 1
		}

		if i <= blkNumber {
			slotsCount := c.getSlotsCount(blk.Time().Uint64(), timeNow)
			//X				= Sx + NHX + Empty
			//Empty			= X - Sx - NHX
			//Sx - Empty 	= Sx - (X-Sx-NHX) = Sx -X + Sx +NHX = 2Sx+NHX-X
			diffBlk := 2*slotsCount + sbs.SuffixBlockNonTrusted - sbs.SuffixBlockTrusted
			if diffBlk < SecBlkDiff {
				return false
			}
		}
		inWhiteList = c.isInWhiteList(blk.Coinbase())
		parentHash = blk.ParentHash()
	}
	return true
}

func (c *CFM) isInWhiteList(coinBase common.Address) bool {
	if _, ok := c.whiteList[coinBase]; ok {
		return true
	} else {
		return false
	}
}

func (c *CFM) isBlkOnLocalChain(blkNumber uint64) bool {
	block := c.bc.GetBlockByNumber(blkNumber)
	if block == nil {
		return false
	}
	return true
}

func (c *CFM) getSlotsCount(startTime, stopTime uint64) uint64 {
	if stopTime <= startTime {
		return 0
	}

	stopEpochID, stopSlotID := util.CalEpochSlotID(stopTime)
	if stopEpochID == 0 && stopSlotID == 0 {
		return 0
	}
	startEpochID, startSlotID := util.CalEpochSlotID(startTime)

	return (stopEpochID-startEpochID)*posconfig.SlotCount + (stopSlotID - startSlotID)
}
