package cfm

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"time"
)

const (
	// security block difference between main block chain and side block chain
	SecBlkDiff = 50
	//SecBlkDiff = 3
	MaxUint64 = uint64(^(uint64(0)))
)

type CFM struct {
	bc        *core.BlockChain
	whiteList map[common.Address]int
}

type SuffixBlkStatic struct {
	SuffixBlockTrusted    uint64
	SuffixBlockNonTrusted uint64
}

type BlkStatus struct {
	BlkNumber uint64
	Stable    bool
}

var c *CFM

func InitCFM(bc *core.BlockChain) {
	c = &CFM{}
	c.bc = bc
	c.whiteList = make(map[common.Address]int, 0)
	for _, value := range posconfig.WhiteList {

		b := hexutil.MustDecode(value)
		address := crypto.PubkeyToAddress(*(crypto.ToECDSAPub(b)))
		c.whiteList[address] = 1
	}
}

func GetCFM() *CFM {
	return c
}

func (c *CFM) GetMaxStableBlkNumber() uint64 {

	timeNow := uint64(time.Now().Unix())
	blkStatusArr := c.scanAllBlockStatus(timeNow)
	return c.getMaxStableBlkNumber(blkStatusArr)
}

func (c *CFM) getMaxStableBlkNumber(blkStatusArr []*BlkStatus) uint64 {

	if len(blkStatusArr) == 0 {
		return 0
	}
	// false
	var blkNumber uint64
	if blkStatus := blkStatusArr[len(blkStatusArr)-1]; !blkStatus.Stable {
		if blkStatus.BlkNumber > 0 {
			return blkStatus.BlkNumber - 1
		} else {
			return 0
		}
	}
	// true,true,true,...,false
	for i := uint64(len(blkStatusArr) - 2); i >= 0 && i < MaxUint64; i-- {
		blkStatus := blkStatusArr[i]
		blkNumber = blkStatus.BlkNumber
		if !blkStatus.Stable {
			break
		}
	}
	if blkNumber > 0 {
		return blkNumber - 1
	} else {
		return 0
	}
}

func (c *CFM) scanAllBlockStatus(timeNow uint64) []*BlkStatus {
	blkStatusArr := make([]*BlkStatus, 0)
	curBlk := c.bc.CurrentBlock()
	if curBlk == nil {
		return blkStatusArr
	}

	startNumber := curBlk.NumberU64()
	stopNumber := startNumber - posconfig.K
	if stopNumber <= 0 {
		stopNumber = 0
	}

	sbs := SuffixBlkStatic{0, 0}
	inWhiteList := c.isInWhiteList(curBlk.Coinbase())
	parentHash := curBlk.ParentHash()
	blkStatusArr = append(blkStatusArr, &BlkStatus{curBlk.NumberU64(), false})
	for i := startNumber - 1; i >= stopNumber && i < MaxUint64; i-- {
		blk := c.bc.GetBlock(parentHash, i)
		if blk == nil {
			return blkStatusArr
		}

		if inWhiteList {
			sbs.SuffixBlockTrusted = sbs.SuffixBlockTrusted + 1
		} else {
			sbs.SuffixBlockNonTrusted = sbs.SuffixBlockNonTrusted + 1
		}

		slotsCount := c.getSlotsCount(blk.Time().Uint64(), timeNow, posconfig.SlotTime)
		//X				= Sx + NHX + Empty
		//Empty			= X - Sx - NHX
		//Sx - Empty 	= Sx - (X-Sx-NHX) = Sx -X + Sx +NHX = 2Sx+NHX-X
		//diffBlk := 2*slotsCount + sbs.SuffixBlockNonTrusted - sbs.SuffixBlockTrusted
		diffBlk := 2*sbs.SuffixBlockTrusted + sbs.SuffixBlockNonTrusted - slotsCount
		var status = false
		if diffBlk > SecBlkDiff {
			status = true
		}
		blkStatusArr = append(blkStatusArr, &BlkStatus{blk.NumberU64(), status})
		inWhiteList = c.isInWhiteList(blk.Coinbase())
		parentHash = blk.ParentHash()
	}
	return blkStatusArr
}

func (c *CFM) IsInWhiteList(coinBase common.Address) bool {
	return c.isInWhiteList(coinBase)
}

func (c *CFM) isInWhiteList(coinBase common.Address) bool {
	if _, ok := c.whiteList[coinBase]; ok {
		return true
	} else {
		return false
	}
}

func (c *CFM) getSlotsCount(startTime, stopTime uint64, slotTime uint64) uint64 {
	if stopTime <= startTime {
		return 0
	}

	return uint64((stopTime - startTime) / slotTime)
}
