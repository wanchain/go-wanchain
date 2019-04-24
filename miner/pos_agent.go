package miner

import (
	"encoding/hex"
	"fmt"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
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
	log.Info("backendTimerLoop is running!!!!!!")
	g := s.BlockChain().GetHeaderByNumber(0)
	posconfig.GenesisPK = hexutil.Encode(g.Extra)[2:]
	cfm.InitCFM(s.BlockChain())
	slotleader.SlsInit()

	if posconfig.EpochBaseTime == 0 {
		h := s.BlockChain().GetHeaderByNumber(1)
		if nil != h {
			posconfig.EpochBaseTime = h.Time.Uint64()
		}
	}

	epochSelector := epochLeader.NewEpocher(s.BlockChain())

	eerr := epochSelector.SelectLeadersLoop(0)

	sls := slotleader.GetSlotLeaderSelection()
	sls.Init(s.BlockChain(), nil, nil)

	incentive.Init(epochSelector.GetEpochProbability, epochSelector.SetEpochIncentive, epochSelector.GetRBProposerGroup)
	fmt.Println("posInit: ", eerr)

	s.BlockChain().SetSlSelector(sls)
	s.BlockChain().SetRbSelector(epochSelector)

	s.BlockChain().SetSlotValidator(sls)

	return epochSelector
}
func posInitMiner(s Backend, key *keystore.Key) {
	log.Info("timer backendTimerLoop is running!!!!!!")

	// config
	if key != nil {
		posconfig.Cfg().MinerKey = key
	}
	epochSelector := epochLeader.NewEpocher(s.BlockChain())
	randombeacon.GetRandonBeaconInst().Init(epochSelector)
	if posconfig.EpochBaseTime == 0 {
		h := s.BlockChain().GetHeaderByNumber(1)
		if nil != h {
			posconfig.EpochBaseTime = h.Time.Uint64()
		}
	}
}

// backendTimerLoop is pos main time loop
func (self *Miner) backendTimerLoop(s Backend) {
	log.Info("backendTimerLoop is running!!!!!!")
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
	posInitMiner(s, key)
	// get rpcClient
	url := posconfig.Cfg().NodeCfg.IPCEndpoint()
	rc, err := rpc.Dial(url)
	if err != nil {
		fmt.Println("err:", err)
		panic(err)
	}

	// if there is no block at all
	h := s.BlockChain().GetHeaderByNumber(1)
	if nil == h {
		leaderPub, err := slotleader.GetSlotLeaderSelection().GetSlotLeader(0, 0)
		if err == nil {
			leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
			if leader == localPublicKey {
				self.worker.chainSlotTimer <- struct{}{}
			}
		}
	}
	for {
		// wait until block1
		h := s.BlockChain().GetHeaderByNumber(1)
		if nil == h {
			select {
			case <-self.timerStop:
				randombeacon.GetRandonBeaconInst().Stop()
				return
			case <-time.After(time.Duration(time.Second)):
				continue
			}

			continue
		} else {
			posconfig.EpochBaseTime = h.Time.Uint64()
			cur := uint64(time.Now().Unix())
			if cur < posconfig.EpochBaseTime+posconfig.SlotTime {
				time.Sleep(time.Duration((posconfig.EpochBaseTime + posconfig.SlotTime - cur)) * time.Second)
			}
		}

		util.CalEpochSlotIDByNow()
		epochid, slotid := util.GetEpochSlotID()
		log.Debug("get current period", "epochid", epochid, "slotid", slotid)

		slotleader.GetSlotLeaderSelection().Loop(rc, key, epochid, slotid)

		leaderPub, err := slotleader.GetSlotLeaderSelection().GetSlotLeader(epochid, slotid)
		if err == nil {
			leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
			if leader == localPublicKey {
				self.worker.chainSlotTimer <- struct{}{}
			}
		}

		// get state of k blocks ahead the last block
		stateDb, err := s.BlockChain().State()
		if err != nil {
			log.Error("Failed to get stateDb", "err", err)
		}

		// random beacon loop
		randombeacon.GetRandonBeaconInst().Loop(stateDb, rc, epochid, slotid)

		cur := uint64(time.Now().Unix())
		sleepTime := posconfig.SlotTime - (cur - posconfig.EpochBaseTime - (epochid*posconfig.SlotCount+slotid)*posconfig.SlotTime)
		log.Debug("timeloop sleep", "sleepTime", sleepTime)
		if sleepTime < 0 {
			sleepTime = 0
		}
		select {
		case <-self.timerStop:
			randombeacon.GetRandonBeaconInst().Stop()
			return
		case <-time.After(time.Duration(time.Second * time.Duration(sleepTime))):
			continue
		}
	}
	return
}
