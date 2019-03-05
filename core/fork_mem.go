package core

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/pos"
	"fmt"
	"errors"
	"sync"
	"math/big"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/log"
)

type ForkMem struct {
	kBufferedChains  map[string][]common.Hash
	kBufferedBlks	 map[common.Hash]*types.Block
	curMaxBlkNum	 uint64
	lock sync.RWMutex
}

func NewForkMem() *ForkMem{

	f := &ForkMem{}
	f.kBufferedChains = make(map[string][]common.Hash)
	f.kBufferedBlks = make(map[common.Hash]*types.Block)
	f.curMaxBlkNum = 0
	return f
}


func (f *ForkMem) calEpochSlotIDFromTime(timeUnix uint64) (epochId uint64, slotId uint64) {
	if pos.EpochBaseTime == 0 {
		return
	}

	epochTimespan := uint64(pos.SlotTime * pos.SlotCount)
	epochId = uint64((timeUnix - pos.EpochBaseTime) / epochTimespan)
	slotId = uint64((timeUnix - pos.EpochBaseTime) / pos.SlotTime % pos.SlotCount)
	return
}

func (f *ForkMem) GetBlockEpochIdAndSlotId(block *types.Block) (blkEpochId uint64, blkSlotId uint64, err error) {
	blkTime := block.Time().Uint64()

	blkTd := block.Difficulty().Uint64()

	blkEpochId = (blkTd >> 32)
	blkSlotId = ((blkTd & 0xffffffff) >> 8)

	calEpochId, calSlotId := f.calEpochSlotIDFromTime(blkTime)
	//calEpochId,calSlotId := uint64(blkTime),uint64(blkTime)

	if calEpochId != blkEpochId {
		fmt.Println(calEpochId, blkEpochId, calSlotId, blkSlotId)
		return 0, 0, errors.New("epochid and slotid is not match with blk time")
	}

	return
}


func (f *ForkMem) Maxvalid(bc *BlockChain) (types.Blocks,error){

	var chainBlks types.Blocks
	var midSidBlk *types.Block

	if bc == nil {
		return nil,errors.New("working block chain is nil")
	}

	workBlk := bc.CurrentBlock()
	if workBlk == nil {
		return nil,errors.New("can not get current block in working chain")
	}

	workBlkNum := workBlk.NumberU64()
	//if work block is in the highest one or higher than buffer,use work blk,work chain will not change
	if workBlkNum >= f.curMaxBlkNum {
		return nil,nil
	}

	maxNumKey := big.NewInt(int64(f.curMaxBlkNum)).Text(16)
	hashs := f.kBufferedChains[maxNumKey]

	minSid := ^uint64(0)
	midSidBlk = nil
	epidOld := uint64(0)
	//same block height
	for _,hs := range hashs {

		blk := f.kBufferedBlks[hs]
		epidNew,sid,err := f.GetBlockEpochIdAndSlotId(blk)
		if err != nil {
			continue
		}

		if epidOld == 0 {
			epidOld = epidNew
		}

		if sid < minSid {
			minSid = sid
			midSidBlk = blk
		}
	}

	// reduce new chain
	for ; midSidBlk != nil && midSidBlk.NumberU64() != workBlkNum; midSidBlk = f.kBufferedBlks[midSidBlk.ParentHash()] {
		chainBlks = append(chainBlks, midSidBlk)
	}

	//find common prefix
	for {

		if workBlk.Hash() == midSidBlk.Hash() && workBlk.NumberU64()==midSidBlk.NumberU64() {
			break
		}

		midSidBlk = f.kBufferedBlks[midSidBlk.ParentHash()]
		if midSidBlk == nil {
			return nil,errors.New("can not find common prefix")
		}

		chainBlks = append(chainBlks, midSidBlk)
		workBlk = bc.GetBlock(workBlk.ParentHash(), workBlk.NumberU64()-1)
	}

	return chainBlks,nil
}


func (f *ForkMem) Push(blockChain types.Blocks) error{

	for _,blk := range blockChain {

		if len(f.kBufferedBlks) > 0 {
			parent := f.kBufferedBlks[blk.ParentHash()]

			if parent == nil {
				log.Debug("Unknown parent of propagated block", "number", blk.Number(), "hash", blk.Hash(), "parent", blk.ParentHash())
				return errors.New("not find parent hash in buffer")
			}
		}

		err := f.push(blk)
		if err != nil {
			return err
		}
	}

	return nil
}


func (f *ForkMem) push(block *types.Block) error{
	f.lock.Lock()
	defer f.lock.Unlock()

	newbn := block.NumberU64()

	if f.curMaxBlkNum == 0 {
		f.curMaxBlkNum = newbn
	} else {
		//input need to be continous block
		if f.curMaxBlkNum + 1 == newbn {
			f.curMaxBlkNum = newbn
		} else if f.curMaxBlkNum > newbn+1 {
			//if block number is bigger 1 than current max block ,return future block
			return consensus.ErrFutureBlock
		} else if newbn < f.curMaxBlkNum-pos.Stage1K {
			//if the block number is older k than current max block,return old block error
			return consensus.ErrOldblockNumber
		}
	}

	num := block.Number().Text(16)

	f.kBufferedChains[num] = append(f.kBufferedChains[num],block.Hash())
	f.kBufferedBlks[block.Hash()] = block


	return nil
}

func (f *ForkMem) PopBack() {

	//need to store k data
	if len(f.kBufferedChains) > int(pos.Cfg().K) {

		blkNumBeforeK := f.curMaxBlkNum - uint64(pos.Cfg().K)

		bnText := big.NewInt(int64(blkNumBeforeK))

		blkHashs := f.kBufferedChains[bnText.Text(16)]

		for _,bh := range blkHashs {
			delete(f.kBufferedBlks,bh)
		}

		delete(f.kBufferedChains,bnText.Text(16))
	}

	return
}



func (f *ForkMem) GetBlock([]common.Hash) *types.Block{
	return nil
}



//f.initKsecMap()
//
//blkNum := block.Number().Uint64()
//startKBlkNum := uint64((blkNum/pos.Stage1K)*pos.Stage1K)
//endAlignBlkNum := uint64((blkNum/pos.Stage1K + 1)*pos.Stage1K)
//
//
////push current block into map
//idx := int(blkNum - startKBlkNum)
//if f.kSecureChain[idx] == nil {
//f.kSecureChain[idx] =  make([]common.Hash,0)
//}
//
//f.kSecureChain[idx] = append(f.kSecureChain[idx],block.Hash())
//f.kAllBlksMap[block.Hash()] = block
//
//blkKeyArray := make([]int,0)
//blksChain := make(types.Blocks,0)
//if blkNum==endAlignBlkNum-1 || len(f.kSecureChain)==int(pos.Stage1K){
//for key,_:= range f.kSecureChain {
//blkKeyArray = append(blkKeyArray,key)
//}
//
//sort.Ints(blkKeyArray)
//
//k := blkKeyArray[0]
//for (uint64(k) + startKBlkNum) < endAlignBlkNum {
//hashs := f.kSecureChain[k]
//l := len(hashs)
//if l==0 {
//return
//} else {
//blksChain = append(blksChain,f.selectBlk(hashs))
//}
//k += 1
//}
//
//} else {
//return
//}

