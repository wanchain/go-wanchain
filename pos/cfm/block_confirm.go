package cfm

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"time"
)

const (
	// security block difference between main block chain and side block chain
	SecBlkDiff = 50
	//SecBlkDiff = 3
	MaxUint64  = uint64(^(uint64(0)))
	SecPowBlks = 12
)

var (
	ErrNullBlk = errors.New("can not read block")
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
	log.Info("InitCFM success")
}

func GetCFM() *CFM {
	return c
}

func (c *CFM) GetMaxStableBlkNumber() uint64 {
	// In pow phase
	if posconfig.FirstEpochId == 0 {
		return c.getPowMaxStableBlkNumber(c.getCurrentBlkNumber())
	}
	// In pos phase
	timeNow := uint64(time.Now().Unix())
	// stopNumber is the min block number, startNumber is max bock number
	blkStatusArr, stopNumber, startNumber, err := c.scanAllBlockStatus(timeNow)

	maxStableBlkNumber := c.getMaxStableBlkNumber(blkStatusArr, stopNumber, startNumber, err)

	log.Debug("GetMaxStableBlkNumber",
		"maxStableBlkNumber", maxStableBlkNumber,
		"Pow2PosUpgradeBlockNumber", posconfig.Pow2PosUpgradeBlockNumber,
		"FirstEpochId", posconfig.FirstEpochId)

	// get max stable on block of pos phase
	if maxStableBlkNumber >= posconfig.Pow2PosUpgradeBlockNumber {
		return maxStableBlkNumber
	}
	return posconfig.Pow2PosUpgradeBlockNumber
}

func (c *CFM) getCurrentBlkNumber() uint64 {
	curBlk := c.bc.CurrentBlock()
	if curBlk == nil {
		log.SyslogErr("confirm block", "scanAllBlockStatus get currentBlock", ErrNullBlk.Error())
		return 0
	}
	return curBlk.NumberU64()
}

func (c *CFM) getPowMaxStableBlkNumber(curBlkNumber uint64) uint64 {
	if curBlkNumber < uint64(SecPowBlks) {
		return 0
	}
	return uint64(curBlkNumber - uint64(SecPowBlks))
}

func (c *CFM) getMaxStableBlkNumber(blkStatusArr []*BlkStatus, stopNumber uint64, startNumber uint64, err error) uint64 {
	if startNumber < stopNumber {
		return 0
	}

	if err != nil {
		return stopNumber
	}

	if len(blkStatusArr) == 0 {
		return 0
	}
	if uint64(len(blkStatusArr)) != (startNumber - stopNumber) {
		return stopNumber
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

func (c *CFM) scanAllBlockStatus(timeNow uint64) (blkStatus []*BlkStatus, stop uint64, start uint64, err error) {
	blkStatusArr := make([]*BlkStatus, 0)
	curBlk := c.bc.CurrentBlock()
	if curBlk == nil {
		log.SyslogErr("confirm block", "scanAllBlockStatus get currentBlock", ErrNullBlk.Error())
		return blkStatusArr, 0, 0, ErrNullBlk
	}

	startNumber := curBlk.NumberU64()
	var stopNumber uint64
	if startNumber > uint64(posconfig.K) {
		stopNumber = startNumber - uint64(posconfig.K)
	} else {
		stopNumber = 0
	}

	sbs := SuffixBlkStatic{0, 0}
	var inWhiteList = false
	hash := curBlk.Hash()
	for i := startNumber; i > stopNumber && i < MaxUint64; i-- {
		blk := c.bc.GetBlock(hash, i)
		if blk == nil {
			log.SyslogErr("confirm block", "scanAllBlockStatus", ErrNullBlk.Error(), "block number", i)
			return blkStatusArr, stopNumber, startNumber, ErrNullBlk
		}

		inWhiteList = c.isInWhiteList(blk.Coinbase())

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
		diffBlk := int64(2*sbs.SuffixBlockTrusted + sbs.SuffixBlockNonTrusted - slotsCount)
		var status = false
		if diffBlk > SecBlkDiff {
			status = true
		}

		log.Debug("scanAllBlockStatus",
			"Number", blk.NumberU64(),
			"hash", blk.Hash(),
			"ParentHash", blk.ParentHash(),
			"Coinbase", blk.Coinbase(),
			"wl", inWhiteList,
			"now", timeNow,
			"blokTime", blk.Time().Uint64(),
			"slotCounts", slotsCount,
			"Sx", sbs.SuffixBlockTrusted,
			"NHx", sbs.SuffixBlockNonTrusted,
			"diffBlk", diffBlk)

		blkStatusArr = append(blkStatusArr, &BlkStatus{blk.NumberU64(), status})
		hash = blk.ParentHash()
	}
	return blkStatusArr, stopNumber, startNumber, nil
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
	return uint64((stopTime-startTime)/slotTime + 1)
}
