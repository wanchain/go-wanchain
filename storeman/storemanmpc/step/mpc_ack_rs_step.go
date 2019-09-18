package step

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
)

type MpcAckRSStep struct {
	BaseStep
	message    map[discover.NodeID]bool
	remoteMpcR map[discover.NodeID][]big.Int // R
	remoteMpcS map[discover.NodeID]big.Int   // S
	accType    string
	mpcR       [2]big.Int
	mpcS       big.Int
}

func CreateAckMpcRSStep(peers *[]mpcprotocol.PeerInfo, accType string) *MpcAckRSStep {
	mpc := &MpcAckRSStep{*CreateBaseStep(peers, -1),
		make(map[discover.NodeID]bool),
		make(map[discover.NodeID][]big.Int),
		make(map[discover.NodeID]big.Int),
		accType,
		[2]big.Int{*big.NewInt(0), *big.NewInt(0)},
		*big.NewInt(0)}
	return mpc
}

func (mars *MpcAckRSStep) InitStep(result mpcprotocol.MpcResultInterface) error {
	log.SyslogInfo("MpcAckRSStep.InitStep begin")
	value, err := result.GetValue(mpcprotocol.RPublicKeyResult)
	if err != nil {
		log.SyslogErr("ack mpc account step, init fail. err:%s", err.Error())
		return err
	}
	mars.mpcR[0], mars.mpcR[1] = value[0], value[1]

	sValue, err := result.GetValue(mpcprotocol.MpcS)
	if err != nil {
		log.SyslogErr("ack mpc account step, init fail. err:%s", err.Error())
		return err
	}
	mars.mpcS = sValue[0]
	return nil
}

func (mars *MpcAckRSStep) CreateMessage() []mpcprotocol.StepMessage {
	return []mpcprotocol.StepMessage{mpcprotocol.StepMessage{
		MsgCode:   mpcprotocol.MPCMessage,
		PeerID:    nil,
		Peers:     nil,
		Data:      []big.Int{mars.mpcS, mars.mpcR[0], mars.mpcR[1]},
		BytesData: nil}}
}

func (mars *MpcAckRSStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.SyslogInfo("MpcAckRSStep.FinishStep begin")
	err := mars.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	for _, mpcR := range mars.remoteMpcR {
		if mpcR == nil {
			return mpcprotocol.ErrInvalidMPCR
		}

		if mars.mpcR[0].Cmp(&mpcR[0]) != 0 || mars.mpcR[1].Cmp(&mpcR[1]) != 0 {

			return mpcprotocol.ErrInvalidMPCR
		}

	}

	for _, mpcS := range mars.remoteMpcS {
		if mars.mpcS.Cmp(&mpcS) != 0 {

			return mpcprotocol.ErrInvalidMPCS
		}

	}

	return nil
}

func (mars *MpcAckRSStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.SyslogInfo("MpcAckRSStep.HandleMessage begin")
	_, exist := mars.message[*msg.PeerID]
	if exist {
		log.SyslogErr("MpcAckRSStep.HandleMessage fail. peer doesn't exist in task peer group. peerID:%s",
			msg.PeerID.String())
		return false
	}

	if len(msg.Data) >= 3 {
		mars.remoteMpcR[*msg.PeerID] = []big.Int{msg.Data[1], msg.Data[2]}
		mars.remoteMpcS[*msg.PeerID] = msg.Data[0]
	}

	mars.message[*msg.PeerID] = true
	return true
}
