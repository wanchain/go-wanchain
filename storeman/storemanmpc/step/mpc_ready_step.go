package step

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	"github.com/wanchain/go-wanchain/log"
)

type MpcReadyStep struct {
	BaseStep
}

func (ready *MpcReadyStep) InitStep(mpcprotocol.MpcResultInterface) error {
	log.Warn("-----------------MpcReadyStep.InitStep begin")

	return nil
}

func CreateMpcReadyStep(peers *[]mpcprotocol.PeerInfo) *MpcReadyStep {
	log.Warn("-----------------CreateMpcReadyStep begin")

	return &MpcReadyStep{*CreateBaseStep(peers, 0)}
}

func (ready *MpcReadyStep) CreateMessage() []mpcprotocol.StepMessage {
	log.Warn("-----------------MpcReadyStep.CreateMessage begin")

	data := make([]big.Int, 1)
	data[0].SetInt64(1)
	return []mpcprotocol.StepMessage{mpcprotocol.StepMessage{Msgcode:
		mpcprotocol.MPCMessage,
		PeerID:nil,
		Peers:nil,
		Data:data,
		BytesData:nil}}
}

func (ready *MpcReadyStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.Warn("-----------------MpcReadyStep.FinishStep begin")

	err := ready.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	return nil
}

func (ready *MpcReadyStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.Warn("-----------------MpcReadyStep.MpcReadyStep begin")

	return true
}
