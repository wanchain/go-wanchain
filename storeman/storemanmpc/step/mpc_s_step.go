package step

import (
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
)

type MpcSStep struct {
	BaseMpcStep
	resultKeys []string
	signNum    int
}

func CreateMpcSStep(peers *[]mpcprotocol.PeerInfo, preValueKeys []string, resultKeys []string) *MpcSStep {

	log.SyslogInfo("CreateMpcSStep begin")
	signNum := len(preValueKeys)
	mpc := &MpcSStep{*CreateBaseMpcStep(peers, signNum), resultKeys, signNum}

	for i := 0; i < signNum; i++ {
		mpc.messages[i] = createSGenerator(preValueKeys[i])
	}
	return mpc
}

func (msStep *MpcSStep) CreateMessage() []mpcprotocol.StepMessage {
	log.SyslogInfo("MpcSStep.CreateMessage begin")

	message := make([]mpcprotocol.StepMessage, 1)
	message[0].Msgcode = mpcprotocol.MPCMessage
	message[0].PeerID = nil

	for i := 0; i < msStep.signNum; i++ {
		pointer := msStep.messages[i].(*mpcSGenerator)
		message[0].Data = append(message[0].Data, pointer.seed)
	}

	return message
}

func (msStep *MpcSStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.SyslogInfo("MpcSStep.HandleMessage begin, peerID:%s", msg.PeerID.String())

	seed := msStep.getPeerSeed(msg.PeerID)
	if seed == 0 {
		log.SyslogErr("MpcSStep.HandleMessage, get peer seed fail. peer:%s", msg.PeerID.String())
		return false
	}

	if len(msg.Data) != msStep.signNum {
		log.SyslogErr("MpcSStep HandleMessage, msg data len doesn't match requiremant, dataLen:%d", len(msg.Data))
		return false
	}

	for i := 0; i < msStep.signNum; i++ {
		pointer := msStep.messages[i].(*mpcSGenerator)
		_, exist := pointer.message[seed]
		if exist {
			log.SyslogErr("MpcSStep.HandleMessage, get msg from seed fail. peer:%s", msg.PeerID.String())
			return false
		}

		pointer.message[seed] = msg.Data[2*i]
	}

	return true
}

func (msStep *MpcSStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.SyslogInfo("MpcSStep.FinishStep begin")
	err := msStep.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	for i := 0; i < msStep.signNum; i++ {
		pointer := msStep.messages[i].(*mpcSGenerator)
		// MpcS
		err = result.SetValue(msStep.resultKeys[i], []big.Int{pointer.result})
		if err != nil {
			log.SyslogErr("MpcSStep.FinishStep, SetValue fail. err:%s", err.Error())
			return err
		}
	}

	log.SyslogInfo("MpcSStep.FinishStep succeed")
	return nil
}
