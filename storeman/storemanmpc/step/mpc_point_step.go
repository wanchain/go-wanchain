package step

import (
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
)

type MpcPoint_Step struct {
	BaseMpcStep
	resultKey string
}

func CreateMpcPoint_Step(peers *[]mpcprotocol.PeerInfo, preValueKey string, resultKey string) *MpcPoint_Step {
	mpc := &MpcPoint_Step{*CreateBaseMpcStep(peers, 1), resultKey}
	mpc.messages[0] = createPointGenerator(preValueKey)
	return mpc
}

func (ptStep *MpcPoint_Step) CreateMessage() []mpcprotocol.StepMessage {
	message := make([]mpcprotocol.StepMessage, 1)
	pointer := ptStep.messages[0].(*mpcPointGenerator)
	message[0].Msgcode = mpcprotocol.MPCMessage
	message[0].PeerID = nil
	message[0].Data = pointer.seed[:]
	return message
}

func (ptStep *MpcPoint_Step) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := ptStep.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	pointer := ptStep.messages[0].(*mpcPointGenerator)
	err = result.SetValue(ptStep.resultKey, pointer.result[:])
	log.Debug("mpc point finish", "x", pointer.result[0].String(), "y", pointer.result[1].String())
	if err != nil {
		mpcsyslog.Err("MpcPoint_Step.FinishStep, SetValue fail. err:%s", err.Error())
		return err
	}

	return nil
}

func (ptStep *MpcPoint_Step) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.Debug("mpc point handle message", "peerID", msg.PeerID)
	seed := ptStep.getPeerSeed(msg.PeerID)
	if seed == 0 {
		mpcsyslog.Err("MpcPoint_Step.HandleMessage, get peer seed fail. peer:%s", msg.PeerID.String())
		log.Error("MpcPoint_Step Not Find:", "peerID", msg.PeerID)
	}

	pointer := ptStep.messages[0].(*mpcPointGenerator)
	_, exist := pointer.message[seed]
	if exist {
		mpcsyslog.Err("MpcPoint_Step.HandleMessage, get msg from seed fail. peer:%s", msg.PeerID.String())
		return false
	}

	pointer.message[seed] = [2]big.Int{msg.Data[0], msg.Data[1]}
	return true
}
