package step

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
)

type MpcPoint_Step struct {
	BaseMpcStep
	resultKeys []string
	signNum int
}

func CreateMpcPoint_Step(peers *[]mpcprotocol.PeerInfo, preValueKeys []string, resultKeys []string) *MpcPoint_Step {
	mpcsyslog.Info("CreateMpcPoint_Step begin")

	signNum := len(preValueKeys)
	mpc := &MpcPoint_Step{*CreateBaseMpcStep(peers, signNum), resultKeys, signNum}

	for i := 0; i < signNum; i++ {
		mpc.messages[i] = createPointGenerator(preValueKeys[i])
	}

	return mpc
}

func (ptStep *MpcPoint_Step) CreateMessage() []mpcprotocol.StepMessage {
	mpcsyslog.Info("MpcPoint_Step.CreateMessage begin")

	message := make([]mpcprotocol.StepMessage, 1)
	message[0].Msgcode = mpcprotocol.MPCMessage
	message[0].PeerID = nil

	for i := 0; i < ptStep.signNum; i++ {
		pointer := ptStep.messages[i].(*mpcPointGenerator)
		message[0].Data = append(message[0].Data, pointer.seed[:]...)
	}

	return message
}

func (ptStep *MpcPoint_Step) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	mpcsyslog.Info("MpcPoint_Step.HandleMessage begin, peerID:%s", msg.PeerID.String())

	seed := ptStep.getPeerSeed(msg.PeerID)
	if seed == 0 {
		mpcsyslog.Err("MpcPoint_Step.HandleMessage, get peer seed fail. peer:%s", msg.PeerID.String())
		return false
	}

	if len(msg.Data) != 2*ptStep.signNum {
		mpcsyslog.Err("MpcPoint_Step HandleMessage, msg data len doesn't match requiremant, dataLen:%d", len(msg.Data))
		return false
	}

	for i := 0; i < ptStep.signNum; i++ {
		pointer := ptStep.messages[i].(*mpcPointGenerator)
		_, exist := pointer.message[seed]
		if exist {
			mpcsyslog.Err("MpcPoint_Step.HandleMessage, get msg from seed fail. peer:%s", msg.PeerID.String())
			return false
		}

		pointer.message[seed] = [2]big.Int{msg.Data[2*i + 0], msg.Data[2*i + 1]}
	}

	return true
}

func (ptStep *MpcPoint_Step) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	mpcsyslog.Info("MpcPoint_Step.FinishStep begin")
	err := ptStep.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	for i := 0; i < ptStep.signNum; i++  {
		pointer := ptStep.messages[i].(*mpcPointGenerator)
		err = result.SetValue(ptStep.resultKeys[i], pointer.result[:])
		if err != nil {
			mpcsyslog.Err("MpcPoint_Step.FinishStep, SetValue fail. err:%s", err.Error())
			return err
		}
	}

	mpcsyslog.Info("MpcPoint_Step.FinishStep succeed")
	return nil
}

