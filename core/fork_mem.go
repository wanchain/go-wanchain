package core

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/pos"
	"fmt"
	"errors"
	"github.com/wanchain/go-wanchain/ethdb"
)

type ForkMem struct {
	kBufferedChains  *ethdb.MemDatabase
	kBufferedBlks	 *ethdb.MemDatabase
}

func NewForkMem() *ForkMem{
	f := &ForkMem{}
	kBufferedChains,err:= ethdb.NewMemDatabase()
	if err != nil {
		return nil
	}
	f.kBufferedChains = kBufferedChains

	kBufferedBlks,err := ethdb.NewMemDatabase()

	if err != nil {
		return nil
	}

	f.kBufferedBlks = kBufferedBlks

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


func (f *ForkMem) selectBlk([]common.Hash) *types.Block{
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

