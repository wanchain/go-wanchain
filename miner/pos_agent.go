package miner

import (
	"encoding/hex"
	"fmt"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus/pluto"

	//"github.com/wanchain/go-wanchain/common/hexutil"
	"time"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/cfm"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/incentive"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/randombeacon"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rpc"
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
	//if posconfig.EpochBaseTime == 0 {
	//	//todo:`switch pos from pow,the time is not 1?
	//	h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())
	//	if nil != h {
	//		posconfig.EpochBaseTime = h.Time.Uint64()
	//	}
	//}
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

	if pluto, ok := self.engine.(*pluto.Pluto); ok {
		pluto.Authorize(eb, wallet.SignHash, key)
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
		if stop {
			return
		}
	} else {
		epochID, slotID = util.CalEpSlbyTd(h.Difficulty.Uint64())
		posconfig.FirstEpochId = epochID
		log.Info("backendTimerLoop first pos block exist :", "FirstEpochId", posconfig.FirstEpochId)
		// todo: need not reset the slot leader.
		//if epochID > posconfig.FirstEpochId+2 {
		//	stop := self.posRestartInit(s, localPublicKey)
		//	if stop {
		//		return
		//	}
		//}
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
		targetEpochLeaderID := epochID
		if isDefault {
			if epochID > posconfig.FirstEpochId+2 {
				log.Info("backendTimerLoop use default epoch leader.")
			}
			targetEpochLeaderID = 0
		}
		if sls.IsLocalPkInEpochLeaders(prePks) {
			leaderPub, err := sls.GetSlotLeader(targetEpochLeaderID, slotID)
			if err == nil {
				slotTime := (epochID*posconfig.SlotCount + slotID) * posconfig.SlotTime
				leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
				log.Info("leader ", "leader", leader)
				if leader == localPublicKey && len(self.worker.chainSlotTimer)< chainTimerSlotSize{
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

	epochID, slotID := util.CalEpochSlotID(h0.Time.Uint64())

	if slotID == posconfig.SlotCount-1 {
		epochID += 1
		slotID = 0
	} else {
		slotID += 1
	}

	leaderPub, _ := slotleader.GetSlotLeaderSelection().GetSlotLeader(0, slotID)
	leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
	log.Info("posStartInit leader ", "leader", leader)

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
		if !self.Mining() {
			return true
		}
		posconfig.FirstEpochId = epochID
		log.Info("backendTimerLoop :", "FirstEpochId", posconfig.FirstEpochId)

		self.worker.chainSlotTimer <- slotTime

	}

	for {

		h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())

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
			log.Info("backendTimerLoop download the first pos block :", "FirstEpochId", posconfig.FirstEpochId)

			break
		}

	}
	return false
}

// todo, reture true or false ,
//func (self *Miner) posRestartInit(s Backend, localPublicKey string) (stop bool)  {
//	//if chain is in restarting status,then return
//	res, _ := s.BlockChain().ChainRestartStatus()
//	if res {
//		return false
//	}
//
//	curBlk := s.BlockChain().CurrentBlock()
//	s.BlockChain().SetRestartBlock(curBlk, nil, true)
//	//if stop time is short,then follow normally processs
//	res, _ = s.BlockChain().ChainRestartStatus()
//	if !res {
//		s.BlockChain().SetChainRestartSuccess()
//		return false
//	}
//
//
//	//else restart process
//	// todo why dont use fir pos's epochID
//	h0 := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64() - 1)
//	if h0 == nil {
//		panic("last ppow block can't find")
//	}
//
//	epochID, slotID := util.CalEpochSlotID(h0.Time.Uint64())
//
//	if slotID == posconfig.SlotCount-1 {
//		epochID += 1
//		slotID = 0
//	} else {
//		slotID += 1
//	}
//
//	leaderPub, _ := slotleader.GetSlotLeaderSelection().GetSlotLeader(0, slotID)
//	leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
//	log.Info("posRestartInit leader ", "leader", leader)
//
//	// when restart and the chainquolity low,  if not the slot leader, wait until download the block.
//	if leader == localPublicKey {
//
//		cur := uint64(time.Now().Unix())
//
//		epochID, slotID := util.CalEpochSlotID(cur)
//		if slotID == posconfig.SlotCount-1 {
//			epochID += 1
//			slotID = 0
//		} else {
//			slotID += 1
//		}
//
//		slotTime := (epochID*posconfig.SlotCount + slotID) * posconfig.SlotTime
//
//		if slotTime > cur {
//			select {
//			case <-self.timerStop:
//				return true
//			case <-time.After(time.Duration(time.Second * time.Duration(slotTime-cur))):
//			}
//		}
//
//		self.worker.chainSlotTimer <- slotTime
//
//		for {
//
//			h := s.BlockChain().GetHeaderByNumber(curBlk.NumberU64() + 1)
//
//			if nil == h {
//				select {
//				case <-self.timerStop:
//					return true
//				case <-time.After(time.Duration(time.Second)):
//				}
//
//			} else {
//				preBlk := curBlk
//				curBlk := s.BlockChain().CurrentBlock()
//				s.BlockChain().SetRestartBlock(curBlk, preBlk, false)
//				//if stop time is short,then follow normally processs
//				res, _ = s.BlockChain().ChainRestartStatus()
//				log.Info("restart", "result", res)
//				break
//			}
//
//		}
//	} else {
//		for {
//
//			res, _ = s.BlockChain().ChainRestartStatus()
//			if !res {
//				select {
//					case <-self.timerStop:
//						return true
//					case <-time.After(time.Duration(time.Second)):
//				}
//			} else {
//				return false
//			}
//		}
//	}
//	return false
//}
