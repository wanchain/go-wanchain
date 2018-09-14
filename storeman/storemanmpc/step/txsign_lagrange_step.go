package step

import (
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
)

type TXSign_Lagrange_Step struct {
	BaseMpcStep
	resultKeys []string
	signNum int
}

func CreateTXSign_Lagrange_Step(peers *[]mpcprotocol.PeerInfo, preValueKeys []string, resultKeys []string) *TXSign_Lagrange_Step {
	signNum := len(preValueKeys)
	mpc := &TXSign_Lagrange_Step{*CreateBaseMpcStep(peers, signNum), resultKeys, signNum}

	for i := 0; i < signNum; i++ {
		mpc.messages[i] = createLagrangeGenerator(preValueKeys[i])
	}

	return mpc
}

func (lagStep *TXSign_Lagrange_Step) CreateMessage() []mpcprotocol.StepMessage {
	message := make([]mpcprotocol.StepMessage, 1)
	message[0].Msgcode = mpcprotocol.MPCMessage
	message[0].PeerID = nil
	message[0].Data = make([]big.Int, 1)

	for i := 0; i < lagStep.signNum; i++ {
		lag := lagStep.messages[i].(*lagrangeGenerator)
		message[0].Data = append(message[0].Data, lag.seed)
	}

	return message
}

func (lagStep *TXSign_Lagrange_Step) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := lagStep.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	for i := 0; i < lagStep.signNum; i++ {
		lag := lagStep.messages[i].(*lagrangeGenerator)
		err = result.SetValue(lagStep.resultKeys[i], []big.Int{lag.result})
		if err != nil {
			return err
		}
	}

	return nil
}

func (lagStep *TXSign_Lagrange_Step) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.Debug("lagrange step", "peerID", msg.PeerID)
	seed := lagStep.getPeerSeed(msg.PeerID)
	if seed == 0 {
		mpcsyslog.Err("TXSign_Lagrange_Step.HandleMessage, get seed fail. peer:%s", msg.PeerID.String())
		log.Error("lagrange step", "peerID", msg.PeerID)
	}

	for i := 0; i < lagStep.signNum; i++ {
		lag := lagStep.messages[i].(*lagrangeGenerator)
		_, exist := lag.message[seed]
		if exist {
			mpcsyslog.Err("TXSign_Lagrange_Step.HandleMessage, get msg fail. peer:%s", msg.PeerID.String())
			return false
		}

		lag.message[seed] = msg.Data[i]
	}

	return true
}
