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
	"encoding/hex"
	"fmt"
	"time"
)

import (
	pos "github.com/ethereum/go-ethereum/pos/posavgretrate"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/pluto"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pos/cfm"
	"github.com/ethereum/go-ethereum/pos/epochLeader"
	"github.com/ethereum/go-ethereum/pos/incentive"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"github.com/ethereum/go-ethereum/pos/randombeacon"
	"github.com/ethereum/go-ethereum/pos/slotleader"
	"github.com/ethereum/go-ethereum/pos/util"
	"github.com/ethereum/go-ethereum/rpc"
)

func posWhiteList() {

}

func posPreInit(s Backend) {
	posconfig.Pow2PosUpgradeBlockNumber = s.BlockChain().Config().PosFirstBlock.Uint64()
	epochLeader.NewEpocher(s.BlockChain())
}

func PosInit(s Backend) *epochLeader.Epocher {
	log.Debug("PosInit is running")

	posconfig.Pow2PosUpgradeBlockNumber = s.BlockChain().Config().PosFirstBlock.Uint64()
	h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())
	if nil != h {
		epochId, _ := util.CalEpSlbyTd(h.Difficulty.Uint64())
		if epochId == 0 {
			panic("epochId ->posconfig.FirstEpochId = === 0 ")
		}
		posconfig.FirstEpochId = epochId
	}
	epochSelector := epochLeader.NewEpocher(s.BlockChain())
	//Set to epochID 0 to get a default leaders for epoch 0.
	err := epochSelector.SelectLeadersLoop(0)
	if err != nil {
		panic("PosInit failed.")
	}

	cfm.InitCFM(s.BlockChain())

	slotleader.SlsInit()
	sls := slotleader.GetSlotLeaderSelection()
	sls.Init(s.BlockChain(), nil, nil)

	incentive.Init(epochSelector.GetEpochProbability, epochSelector.SetEpochIncentive, epochSelector.GetRBProposerGroup)

	s.BlockChain().SetSlotValidator(sls)

	pos.NewPosAveRet()

	posconfig.ChainId = s.BlockChain().Config().ChainID.Uint64()

	return epochSelector
}

func posInitMiner(s Backend, key *keystore.Key) {
	log.Debug("posInitMiner is running")

	// config
	if key != nil {
		posconfig.Cfg().MinerKey = key
	}
	epochSelector := epochLeader.NewEpocher(s.BlockChain())
	randombeacon.GetRandonBeaconInst().Init(epochSelector)
}

// backendTimerLoop is pos main time loop
func (self *Miner) backendTimerLoop(s Backend) {
	self.mu.Lock()
	defer self.mu.Unlock()

	log.Debug("backendTimerLoop is running")
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
	localPublicKey := hex.EncodeToString(crypto.FromECDSAPub(&key.PrivateKey.PublicKey))
	log.Debug("localPublicKey :" + localPublicKey)

	if pluto, ok := self.engine.(*pluto.Pluto); ok {
		pluto.Authorize(eb, wallet.SignData, key)
	}
	posInitMiner(s, key)
	// get rpcClient
	url := posconfig.Cfg().NodeCfg.IPCEndpoint()
	rc, err := rpc.Dial(url)
	if err != nil {
		fmt.Println("err:", err)
		panic(err)
	}

	var epochID, slotID uint64
	//curBlkNum := uint64(0)
	h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())

	if nil == h {
		stop := self.posStartInit(s, localPublicKey)
		log.Debug("backendTimerLoop", "-----------------stop------------------", stop)
		if stop {
			return
		}
	} else {
		epochID, slotID = util.CalEpSlbyTd(h.Difficulty.Uint64())
		posconfig.FirstEpochId = epochID
		if posconfig.FirstEpochId == 0 {
			panic("posconfig.FirstEpochId == 0")
		}
		log.Info("backendTimerLoop first pos block exist :", "FirstEpochId", posconfig.FirstEpochId)
	}

	for {
		cur := uint64(time.Now().Unix())
		sleepTime := posconfig.SlotTime - cur%posconfig.SlotTime
		//select {
		////case <-self.timerStop:
		////	randombeacon.GetRandonBeaconInst().Stop()
		////	return
		//case <-time.After(time.Duration(time.Second * time.Duration(sleepTime))):
		//}
		time.Sleep(time.Second * time.Duration(sleepTime))
		log.Debug("backendTimerLoop", "XXXXXXXXXXXXXXXXXXXXXMiningXXXXXXXXXXXXXXXXXX", self.Mining())
		if !self.Mining() {
			randombeacon.GetRandonBeaconInst().Stop()
			return
		}

		util.CalEpochSlotIDByNow()
		epochID, slotID = util.GetEpochSlotID()
		log.Debug("get current period", "epochid", epochID, "slotid", slotID)

		sls := slotleader.GetSlotLeaderSelection()
		sls.Loop(rc, key, epochID, slotID)

		prePks, isDefault := sls.GetPreEpochLeadersPK(epochID)
		log.Info("backendTimerLoop", "i", 0, "prePks", hex.EncodeToString(crypto.FromECDSAPub(prePks[0])), "isDefault", isDefault)

		targetEpochLeaderID := epochID
		if isDefault {
			targetEpochLeaderID = 0
			if epochID > posconfig.FirstEpochId+2 {
				log.Info("backendTimerLoop use default epoch leader.")
				if epochID >= posconfig.Cfg().MarsEpochId {
					log.Info("backendTimerLoop use Mars default epoch leader.", "epochId", epochID, "FirstEpochId", posconfig.FirstEpochId)
					epRecovery := epochID
					epRecovery = slotleader.GetRecoveryEpochID(epRecovery)
					prePks, isDefault = sls.GetPreEpochLeadersPK(epRecovery)
					if !isDefault {
						targetEpochLeaderID = epRecovery
					}
				}
			}
		}

		log.Info("IsLocalPkInEpochLeaders", "in", sls.IsLocalPkInEpochLeaders(prePks), "prePks", hex.EncodeToString(crypto.FromECDSAPub(prePks[0])))
		if sls.IsLocalPkInEpochLeaders(prePks) {
			leaderPub, err := sls.GetSlotLeader(targetEpochLeaderID, slotID)
			if err == nil {
				slotTime := (epochID*posconfig.SlotCount + slotID) * posconfig.SlotTime
				leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
				log.Info("leader ", "leader", leader)
				if leader == localPublicKey && len(self.worker.chainSlotTimer) < chainTimerSlotSize {
					self.worker.chainSlotTimer <- slotTime
				}
			}
		}

		// get state of k blocks ahead the last block
		stateDb, err := s.BlockChain().State()
		if err == nil {
			// random beacon loop
			randombeacon.GetRandonBeaconInst().Loop(stateDb, rc, epochID, slotID)
		} else {
			log.SyslogErr("Failed to get stateDb", "err", err)
		}

		memUse := float32(util.MemStat()) / 1024.0 / 1024.0 / 1024.0

		log.Info("Memory usage(GB)", "memory", memUse)

		//time.Sleep(time.Second)
	}
}

func (self *Miner) posStartInit(s Backend, localPublicKey string) (stop bool) {

	h0 := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64() - 1)
	if h0 == nil {
		panic("last ppow block can't find")
	}

	epochID, slotID := util.CalEpochSlotID(h0.Time)

	if slotID == posconfig.SlotCount-1 {
		epochID += 1
		slotID = 0
	} else {
		slotID += 1
	}

	leaderPub, _ := slotleader.GetSlotLeaderSelection().GetSlotLeader(0, slotID)
	leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
	log.Info("posStartInit leader ", "leader", leader, "self.Mining()", self.Mining())

	if leader == localPublicKey {
		cur := uint64(time.Now().Unix())
		//epochID, slotID := util.CalEpochSlotID(cur)

		slotTime := (epochID*posconfig.SlotCount + slotID) * posconfig.SlotTime
		if slotTime > cur {
			time.Sleep(time.Duration(time.Second * time.Duration(slotTime-cur)))
			//select {
			////case <-self.timerStop:
			////	return true
			//case <-time.After(time.Duration(time.Second * time.Duration(slotTime-cur))):
			//}
		}
		//todo check it later
		/*
			if !self.Mining() {
				return true
			}
		*/
		posconfig.FirstEpochId = epochID
		if posconfig.FirstEpochId == 0 {
			panic("epochId ->posconfig.FirstEpochId = === 0 ")
		}
		log.Info("backendTimerLoop :", "FirstEpochId", posconfig.FirstEpochId)

		self.worker.chainSlotTimer <- slotTime

	}

	for {
		log.Info("PosStartInit", "self.Mining()", self.Mining())
		h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())
		log.Info("PosStartInit", "GetHeaderByNumber header", h)

		if nil == h {
			//select {
			////case <-self.timerStop:
			////	return true
			//case <-time.After(time.Duration(time.Second)):
			//}
			time.Sleep(time.Duration(time.Second))
			if !self.Mining() {
				return true
			}

			log.Info("backendTimerLoop sleep,", "FirstEpochId", epochID)
		} else {
			epochID, slotID = util.CalEpSlbyTd(h.Difficulty.Uint64())
			posconfig.FirstEpochId = epochID
			if posconfig.FirstEpochId == 0 {
				panic("epochId ->posconfig.FirstEpochId = === 0 ")
			}
			log.Info("backendTimerLoop download the first pos block :", "FirstEpochId", posconfig.FirstEpochId)

			break
		}

	}
	return false
}
