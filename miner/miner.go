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

// Package miner implements Ethereum block creation and mining.
package miner

import (
	"fmt"
	"sync/atomic"

	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/randombeacon"

	"time"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/eth/downloader"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/event"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/rpc"
)

// Backend wraps all methods required for mining.
type Backend interface {
	AccountManager() *accounts.Manager
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	ChainDb() ethdb.Database
	Etherbase() (common.Address, error)
}

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	mux *event.TypeMux

	worker *worker

	coinbase common.Address
	mining   int32
	eth      Backend
	engine   consensus.Engine

	canStart    int32 // can start indicates whether we can start the mining operation
	shouldStart int32 // should start indicates whether we should start after sync
	timerStop   chan interface{}
}

func New(eth Backend, config *params.ChainConfig, mux *event.TypeMux, engine consensus.Engine) *Miner {
	miner := &Miner{
		eth:       eth,
		mux:       mux,
		engine:    engine,
		worker:    newWorker(config, engine, common.Address{}, eth, mux),
		canStart:  1,
		timerStop: make(chan interface{}),
	}
	miner.Register(NewCpuAgent(eth.BlockChain(), engine))
	//posInit(eth, nil)
	go miner.update()
	return miner
}

func PosInit(s Backend, key *keystore.Key) *epochLeader.Epocher {
	log.Info("BackendTimerLoop is running!!!!!!")

	// config
	if key != nil {
		pos.Cfg().MinerKey = key
	}

	epocher := epochLeader.NewEpocher(s.BlockChain())

	randombeacon.GetRandonBeaconInst().Init(epocher)

	eerr := epocher.SelectLeadersLoop(0)
	eerr1 := epocher.SelectLeadersLoop(1)

	slotleader.GetSlotLeaderSelection().Init(s.BlockChain(), nil, key, epocher)

	fmt.Println("posInit: ", eerr, eerr1)
	return epocher
}

// BackendTimerLoop is pos main time loop
func (self *Miner) BackendTimerLoop(s Backend) {
	log.Info("BackendTimerLoop is running!!!!!!")
	// get wallet
	eb, errb := s.Etherbase()
	if errb != nil {
		panic(errb)
	}
	wallet, errf := s.AccountManager().Find(accounts.Account{Address: eb})
	if wallet == nil || errf != nil {
		panic(errf)
	}
	type getKey interface {
		GetUnlockedKey(address common.Address) (*keystore.Key, error)
	}
	key, err := wallet.(getKey).GetUnlockedKey(eb)
	if key == nil || err != nil {
		panic(err)
	}
	log.Debug("Get unlocked key success address:" + eb.Hex())
	epocher := PosInit(s, key)
	// get rpcClient
	url := pos.Cfg().NodeCfg.IPCEndpoint()
	rc, err := rpc.Dial(url)
	if err != nil {
		fmt.Println("err:", err)
		panic(err)
	}

	// get epocher
	//epocher := epochLeader.NewEpocher(s.BlockChain())

	//epochTimer := time.NewTicker(20 * time.Second)
	//slotTimer := time.NewTicker(6 * time.Second)
	for {
		// wait until block1
		h := s.BlockChain().GetHeaderByNumber(1)
		//fmt.Println(h)
		if nil == h {
			time.Sleep(pos.SlotTime * time.Second)
			continue
		} else {
			pos.EpochBaseTime = h.Time.Uint64()
			cur := uint64(time.Now().Unix())
			if  cur < pos.EpochBaseTime+pos.SlotTime {
				time.Sleep(time.Duration((pos.EpochBaseTime+pos.SlotTime - cur))*time.Second)
			}
	}


		slotleader.CalEpochSlotID()
		epochid, slotid := slotleader.GetEpochSlotID()
		fmt.Println("epochid, slotid", epochid, slotid)
		//if epochid >= 2 && posdb.GetEpochBlock(epochid) == 0 {
		//	time.Sleep(pos.SlotTime * time.Second)
		//	continue
		//}
		self.worker.chainSlotTimer <- struct{}{}
		stateDb, err2 := s.BlockChain().StateAt(s.BlockChain().CurrentBlock().Root())
		if err2 != nil {
			fmt.Println(err2)
			time.Sleep(pos.SlotTime * time.Second)
			continue
		}

		//epochid, slotid, err := slotleader.GetEpochSlotID()
		fmt.Println("epochid, slotid: ", epochid, slotid)
		//if err != nil {
		//	fmt.Println("haven't block 1 base")
		//	continue
		//}

		// only the begin of epocher
		if slotid == 0 {
			fmt.Println("epocher begin")

			//epocher.SelectLeadersLoop(epochid)
			//
			//epl := epocher.GetEpochLeaders(epochid)
			//for idx, item := range epl {
			//	fmt.Println("epoleader idx=" + strconv.Itoa(idx) + "  data=" + common.ToHex(item))
			//}
			//rbl := epocher.GetRBProposerGroup(epochid)
			//for idx, item := range rbl {
			//	fmt.Println("rb leader idx=" + strconv.Itoa(idx) + "  data=" + common.ToHex(item.Marshal()))
			//}
			//fmt.Println(rbl)
		}

		// every slot
		fmt.Println("Every slot run:")
		//epocher.SelectLeaders(rb, Nr, Ne, stateDbEpoch, epochid)
		//Add for slot leader selection
		slotleader.GetSlotLeaderSelection().Loop(rc, key, epocher, epochid, slotid)
		//epocher.SelectLeaders()
		go randombeacon.GetRandonBeaconInst().Loop(stateDb, epocher, rc, epochid, slotid)
		cur := uint64(time.Now().Unix())
		sleepTime := pos.SlotTime - (cur - pos.EpochBaseTime - (epochid*pos.SlotCount+slotid)*pos.SlotTime)
		fmt.Println("timeloop sleep: ", sleepTime)
		select {
		case <-self.timerStop:
			return
		case <-time.After(time.Duration(time.Second * time.Duration(sleepTime))):
			continue
		}
	}
	return
}

// update keeps track of the downloader events. Please be aware that this is a one shot type of update loop.
// It's entered once and as soon as `Done` or `Failed` has been broadcasted the events are unregistered and
// the loop is exited. This to prevent a major security vuln where external parties can DOS you with blocks
// and halt your mining operation for as long as the DOS continues.
func (self *Miner) update() {
	events := self.mux.Subscribe(downloader.StartEvent{}, downloader.DoneEvent{}, downloader.FailedEvent{})
out:
	for ev := range events.Chan() {
		switch ev.Data.(type) {
		case downloader.StartEvent:
			atomic.StoreInt32(&self.canStart, 0)
			if self.Mining() {
				self.Stop()
				atomic.StoreInt32(&self.shouldStart, 1)
				log.Info("Mining aborted due to sync")
			}
		case downloader.DoneEvent, downloader.FailedEvent:
			shouldStart := atomic.LoadInt32(&self.shouldStart) == 1

			atomic.StoreInt32(&self.canStart, 1)
			atomic.StoreInt32(&self.shouldStart, 0)
			if shouldStart {
				self.Start(self.coinbase)
			}
			// unsubscribe. we're only interested in this event once
			events.Unsubscribe()
			// stop immediately and ignore all further pending events
			break out
		}
	}
}

func (self *Miner) Start(coinbase common.Address) {
	atomic.StoreInt32(&self.shouldStart, 1)
	self.worker.setEtherbase(coinbase)
	self.coinbase = coinbase

	if atomic.LoadInt32(&self.canStart) == 0 {
		log.Info("Network syncing, will start miner afterwards")
		return
	}
	atomic.StoreInt32(&self.mining, 1)

	log.Info("Starting mining operation")
	self.worker.start()
	self.worker.commitNewWork()
	if self.worker.config.Pluto != nil {
		go self.BackendTimerLoop(self.eth)
	}
}

func (self *Miner) Stop() {
	self.worker.stop()
	atomic.StoreInt32(&self.mining, 0)
	atomic.StoreInt32(&self.shouldStart, 0)
	if self.worker.config.Pluto != nil {
		self.timerStop <- nil
	}
}

func (self *Miner) Register(agent Agent) {
	if self.Mining() {
		agent.Start()
	}
	self.worker.register(agent)
}

func (self *Miner) Unregister(agent Agent) {
	self.worker.unregister(agent)
}

func (self *Miner) Mining() bool {
	return atomic.LoadInt32(&self.mining) > 0
}

func (self *Miner) HashRate() (tot int64) {
	if pow, ok := self.engine.(consensus.PoW); ok {
		tot += int64(pow.Hashrate())
	}
	// do we care this might race? is it worth we're rewriting some
	// aspects of the worker/locking up agents so we can get an accurate
	// hashrate?
	for agent := range self.worker.agents {
		if _, ok := agent.(*CpuAgent); !ok {
			tot += agent.GetHashRate()
		}
	}
	return
}

func (self *Miner) SetExtra(extra []byte) error {
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("Extra exceeds max length. %d > %v", len(extra), params.MaximumExtraDataSize)
	}
	self.worker.setExtra(extra)
	return nil
}

// Pending returns the currently pending block and associated state.
func (self *Miner) Pending() (*types.Block, *state.StateDB) {
	return self.worker.pending()
}

// PendingBlock returns the currently pending block.
//
// Note, to access both the pending block and the pending state
// simultaneously, please use Pending(), as the pending state can
// change between multiple method calls
func (self *Miner) PendingBlock() *types.Block {
	return self.worker.pendingBlock()
}

func (self *Miner) SetEtherbase(addr common.Address) {
	self.coinbase = addr
	self.worker.setEtherbase(addr)
}
