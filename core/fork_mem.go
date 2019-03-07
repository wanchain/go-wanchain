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
	"sort"
)

type chainType uint
const(
	BLOCKCHAIN = iota //0
	HEADERCHAIN //1
)

type ForkMemBlockChain struct {
	ctype 			chainType
	kBufferedChains  map[string][]common.Hash
	kBufferedBlks	 map[common.Hash]*types.Block
	curMaxBlkNum	 int64
	lock sync.RWMutex
}




func NewForkMemBlockChain(ctype chainType) *ForkMemBlockChain{

	f := &ForkMemBlockChain{}
	f.ctype = ctype
	f.kBufferedChains = make(map[string][]common.Hash)
	f.kBufferedBlks = make(map[common.Hash]*types.Block)
	f.curMaxBlkNum = 0
	return f
}

type BlockSorter [] *types.Block

//Len()
func (s BlockSorter) Len() int {
	return len(s)
}

func (s BlockSorter) Less(i, j int) bool {
	return s[i].NumberU64() < s[j].NumberU64()
}

//Swap()
func (s BlockSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}


func (f *ForkMemBlockChain) calEpochSlotIDFromTime(timeUnix uint64) (epochId uint64, slotId uint64) {
	if pos.EpochBaseTime == 0 {
		return
	}

	epochTimespan := uint64(pos.SlotTime * pos.SlotCount)
	epochId = uint64((timeUnix - pos.EpochBaseTime) / epochTimespan)
	slotId = uint64((timeUnix - pos.EpochBaseTime) / pos.SlotTime % pos.SlotCount)
	return
}

func (f *ForkMemBlockChain) GetBlockEpochIdAndSlotId(header *types.Header) (blkEpochId uint64, blkSlotId uint64, err error) {
	blkTime := header.Time.Uint64()

	blkTd := header.Difficulty.Uint64()

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


func (f *ForkMemBlockChain) Maxvalid(workBlk *types.Block) (types.Blocks,error){
	f.lock.Lock()
	defer f.lock.Unlock()

	var chainBlks types.Blocks
	var midSidBlk *types.Block

	if workBlk == nil {
		return nil,errors.New("can not get current block in working chain")
	}

	workBlkNum := int64(workBlk.NumberU64())
	//if work block is in the highest one or higher than buffer,use work blk,work chain will not change
	if workBlkNum >= f.curMaxBlkNum {
		return nil,nil
	}

	maxNumKey := big.NewInt(int64(f.curMaxBlkNum)).Text(16)
	hashs := f.kBufferedChains[maxNumKey]

	minSid := ^uint64(0)
	midSidBlk = f.kBufferedBlks[hashs[0]]
	epidOld := uint64(0)
	//if there are more
	if len(hashs) > 1 {
		//same block height
		for _, hs := range hashs {

			blk := f.kBufferedBlks[hs]
			epidNew, sid, err := f.GetBlockEpochIdAndSlotId(blk.Header())
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
	}
		// reduce new chain
	for ; midSidBlk != nil && int64(midSidBlk.NumberU64()) != workBlkNum; midSidBlk = f.kBufferedBlks[midSidBlk.ParentHash()] {
		chainBlks = append(chainBlks, midSidBlk)
	}

		//find common prefix
	if midSidBlk != nil && midSidBlk.Hash() != workBlk.Hash() {
		for {
				chainBlks = append(chainBlks, midSidBlk)
				if (workBlk!=nil && workBlk.NumberU64()==1)||(workBlk!=nil&&workBlk.Hash()==midSidBlk.Hash()&&workBlk.NumberU64()==midSidBlk.NumberU64()) {
					break
				}

				midSidBlk = f.kBufferedBlks[midSidBlk.ParentHash()]
				if midSidBlk == nil {
					return nil, errors.New("can not find common prefix")
				}

				workBlk = f.kBufferedBlks[workBlk.ParentHash()]
			 	if workBlk == nil {
					return nil, errors.New("can not find common prefix")
				}

		}
	}

	sort.Sort(BlockSorter(chainBlks))

	return chainBlks, nil
}


func (f *ForkMemBlockChain) PushHeaders(headerChain []*types.Header) error{

	if f.ctype != HEADERCHAIN {
		return errors.New("error chain type which require HEADERCHAIN")
	}

	for _,header := range headerChain {

		blk := types.NewBlockWithHeader(header)
		err := f.push(blk)
		if err != nil {
			return err
		}

	}

	return nil
}

func (f *ForkMemBlockChain) PushBlocks(blockChain types.Blocks) error{

	if f.ctype != BLOCKCHAIN {
		return errors.New("error chain type which require BLOCKCHAIN")
	}

	for _,blk := range blockChain {
		err := f.push(blk)
		if err != nil {
			return err
		}
	}

	return nil
}


func (f *ForkMemBlockChain) push(block *types.Block) error{
	f.lock.Lock()
	defer f.lock.Unlock()

	newbn := int64(block.NumberU64())

	if f.curMaxBlkNum == 0 {
		f.curMaxBlkNum = newbn
	} else {
		//input need to be continous block
		if f.curMaxBlkNum + 1 == newbn {
			f.curMaxBlkNum = newbn
		} else if newbn > f.curMaxBlkNum +1 {
			//if block number is bigger 1 than current max block ,return future block
			return consensus.ErrFutureBlock
		}
	}

	num := block.Number().Text(16)

	f.kBufferedChains[num] = append(f.kBufferedChains[num],block.Hash())
	f.kBufferedBlks[block.Hash()] = block


	return nil
}

func (f *ForkMemBlockChain) PopBack() {
	f.lock.Lock()
	defer f.lock.Unlock()

	//need to store k data
	if len(f.kBufferedChains) > int(pos.Cfg().K) {

		blkNumBeforeK := f.curMaxBlkNum - int64(2*pos.Cfg().K)

		if blkNumBeforeK < 0 {
			return
		}

		bnText := big.NewInt(int64(blkNumBeforeK))

		blkHashs := f.kBufferedChains[bnText.Text(16)]

		for _,bh := range blkHashs {
			delete(f.kBufferedBlks,bh)
		}

		delete(f.kBufferedChains,bnText.Text(16))
	}

	return
}

