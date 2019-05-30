package miner

import (
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus/pluto"
	//"github.com/wanchain/go-wanchain/common/hexutil"
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
	"time"
)

func posWhiteList() {

}
func PosInit(s Backend) *epochLeader.Epocher {
	log.Debug("PosInit is running")


	//if posconfig.EpochBaseTime == 0 {
	//	h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())
	//	if nil != h {
	//		posconfig.EpochBaseTime = h.Time.Uint64()
	//	}
	//}
	h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())
	if nil != h {
		epochId,_ := util.CalEpSlbyTd(h.Difficulty.Uint64())
		posconfig.FirstEpochId = epochId
	}
	epochSelector := epochLeader.NewEpocher(s.BlockChain())

	//todo,maybe init do not need epochid
	err := epochSelector.SelectLeadersLoop(0)
	//todo system should not startup if there are error,jia
	if err != nil {
		panic("PosInit")
	}

	cfm.InitCFM(s.BlockChain())

	slotleader.SlsInit()
	sls := slotleader.GetSlotLeaderSelection()
	sls.Init(s.BlockChain(), nil, nil)

	incentive.Init(epochSelector.GetEpochProbability, epochSelector.SetEpochIncentive, epochSelector.GetRBProposerGroup)

	s.BlockChain().SetSlSelector(sls)
	s.BlockChain().SetRbSelector(epochSelector)

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


	//todo:`switch pos from pow,the time is not 1?
	h := s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())
	if nil == h {
		leaderPub  := slotleader.GetSlotLeaderSelection().GetEpochFirstLeadersPK()
		leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub[0]))
		if leader == localPublicKey {
			cur := uint64(time.Now().Unix())
			sleepTime := cur%posconfig.SlotTime
			if sleepTime != 0 {
				select {
				case <-self.timerStop:
					return
				case <-time.After(time.Duration(time.Second * time.Duration(sleepTime))):
				}
			}
			util.CalEpochSlotIDByNow()
			epochId,_ := util.GetEpochSlotID()
			posconfig.FirstEpochId = epochId
			log.Info("************** backendTimerLoop :", "posconfig.FirstEpochId", posconfig.FirstEpochId)
			self.worker.chainSlotTimer <- struct{}{}
		}
		
		//set current block as the restart condition
		s.BlockChain().SetRestartBlock(s.BlockChain().CurrentBlock(),nil,true)
		//reset initial sma
		sls := slotleader.GetSlotLeaderSelection()
		res,_ := s.BlockChain().ChainRestartStatus()
		if res  {
			sls.Init(s.BlockChain(), nil, nil)
		}
		

		for {
			if nil == h {
				select {
				case <-self.timerStop:
					return
				case <-time.After(time.Duration(time.Second)):
				}
			} else {
				break
			}
			h = s.BlockChain().GetHeaderByNumber(s.BlockChain().Config().PosFirstBlock.Uint64())
		}
	}else{
		util.CalEpochSlotIDByNow()
		epochId,_ := util.CalEpSlbyTd(h.Difficulty.Uint64())
		posconfig.FirstEpochId = epochId
		log.Info("************** backendTimerLoop else :", "posconfig.FirstEpochId", posconfig.FirstEpochId)
	}
	posconfig.Pow2PosUpgradeBlockNumber = s.BlockChain().Config().PosFirstBlock.Uint64()

	for {
		cur := uint64(time.Now().Unix())
		sleepTime := cur%posconfig.SlotTime
		if sleepTime != 0 {
			select {
			case <-self.timerStop:
				randombeacon.GetRandonBeaconInst().Stop()
				return
			case <-time.After(time.Duration(time.Second * time.Duration(sleepTime))):
			}
		}
		util.CalEpochSlotIDByNow()
		epochid, slotid := util.GetEpochSlotID()
		log.Debug("get current period", "epochid", epochid, "slotid", slotid)

		slotleader.GetSlotLeaderSelection().Loop(rc, key, epochid, slotid)

		leaderPub, err := slotleader.GetSlotLeaderSelection().GetSlotLeader(epochid, slotid)
		if err == nil {
			leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
			log.Info("leader ","leader",leader)
			if leader == localPublicKey {
				self.worker.chainSlotTimer <- struct{}{}
			}
		}

		// get state of k blocks ahead the last block
		stateDb, err := s.BlockChain().State()
		if err == nil {
			// random beacon loop
			randombeacon.GetRandonBeaconInst().Loop(stateDb, rc, epochid, slotid)
		} else {
			log.SyslogErr("Failed to get stateDb", "err", err)
		}

		time.Sleep(time.Second)
	}
	return
}
