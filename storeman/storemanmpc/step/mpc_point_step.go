package step

import (
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
)

type MpcPointStep struct {
	BaseMpcStep
	resultKeys []string
	signNum    int
}

func CreateMpcPointStep(peers *[]mpcprotocol.PeerInfo, preValueKeys []string, resultKeys []string) *MpcPointStep {
	log.SyslogInfo("CreateMpcPointStep begin")

	signNum := len(preValueKeys)
	mpc := &MpcPointStep{*CreateBaseMpcStep(peers, signNum), resultKeys, signNum}

	for i := 0; i < signNum; i++ {
		mpc.messages[i] = createPointGenerator(preValueKeys[i])
	}

	return mpc
}

func (ptStep *MpcPointStep) CreateMessage() []mpcprotocol.StepMessage {
	log.SyslogInfo("MpcPointStep.CreateMessage begin")

	message := make([]mpcprotocol.StepMessage, 1)
	message[0].MsgCode = mpcprotocol.MPCMessage
	message[0].PeerID = nil

	for i := 0; i < ptStep.signNum; i++ {
		pointer := ptStep.messages[i].(*mpcPointGenerator)
		message[0].Data = append(message[0].Data, pointer.seed[:]...)
	}

	return message
}

func (ptStep *MpcPointStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.SyslogInfo("MpcPointStep.HandleMessage begin, peerID:%s", msg.PeerID.String())

	seed := ptStep.getPeerSeed(msg.PeerID)
	if seed == 0 {
		log.SyslogErr("MpcPointStep.HandleMessage, get peer seed fail. peer:%s", msg.PeerID.String())
		return false
	}

	if len(msg.Data) != 2*ptStep.signNum {
		log.SyslogErr("MpcPointStep HandleMessage, msg data len doesn't match requiremant, dataLen:%d", len(msg.Data))
		return false
	}

	for i := 0; i < ptStep.signNum; i++ {
		pointer := ptStep.messages[i].(*mpcPointGenerator)
		_, exist := pointer.message[seed]
		if exist {
			log.SyslogErr("MpcPointStep.HandleMessage, get msg from seed fail. peer:%s", msg.PeerID.String())
			return false
		}

		pointer.message[seed] = [2]big.Int{msg.Data[2*i+0], msg.Data[2*i+1]}
	}

	return true
}

func (ptStep *MpcPointStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.SyslogInfo("MpcPointStep.FinishStep begin")
	err := ptStep.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	for i := 0; i < ptStep.signNum; i++ {
		pointer := ptStep.messages[i].(*mpcPointGenerator)
		err = result.SetValue(ptStep.resultKeys[i], pointer.result[:])
		if err != nil {
			log.SyslogErr("MpcPointStep.FinishStep, SetValue fail. err:%s", err.Error())
			return err
		}
	}

	log.SyslogInfo("MpcPointStep.FinishStep succeed")
	return nil
}
