package step

import (
	mpcprotocol "github.com/wanchain/go-wanchain/schnorr/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
)

type AcknowledgeMpcStep struct {
	BaseStep
	messageType int64
}

func CreateAcknowledgeMpcStep(peers *[]mpcprotocol.PeerInfo, messageType int64) *AcknowledgeMpcStep {
	log.SyslogInfo("CreateAcknowledgeMpcStep begin")

	return &AcknowledgeMpcStep{*CreateBaseStep(peers, 0), messageType}
}

func (ack *AcknowledgeMpcStep) InitStep(mpcprotocol.MpcResultInterface) error {
	return nil
}

func (ack *AcknowledgeMpcStep) CreateMessage() []mpcprotocol.StepMessage {
	log.SyslogInfo("AcknowledgeMpcStep.CreateMessage begin")

	data := make([]big.Int, 1)
	data[0].SetInt64(ack.messageType)
	return []mpcprotocol.StepMessage{mpcprotocol.StepMessage{
		Msgcode:mpcprotocol.MPCMessage,
		PeerID:nil,
		Peers:nil,
		Data:data,
		BytesData:nil}}
}

func (ack *AcknowledgeMpcStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.SyslogInfo("AcknowledgeMpcStep.FinishStep begin")

	err := ack.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	data := make([]big.Int, 1)
	data[0].SetInt64(ack.messageType)
	result.SetValue(mpcprotocol.MPCActoin, data)

	log.SyslogInfo("AcknowledgeMpcStep.FinishStep succeed")
	return nil
}

func (ack *AcknowledgeMpcStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	return true
}
