package step

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
)

type GetMpcReadyStep struct {
	BaseStep
}

func (ready *GetMpcReadyStep) InitStep(mpcprotocol.MpcResultInterface) error {
	return nil
}

func CreateGetMpcReadyStep(peers *[]mpcprotocol.PeerInfo) *GetMpcReadyStep {
	return &GetMpcReadyStep{*CreateBaseStep(peers, 1)}
}

func (ready *GetMpcReadyStep) CreateMessage() []mpcprotocol.StepMessage {
	return nil
}

func (ready *GetMpcReadyStep) FinishStep(result mpcprotocol.MpcResultInterface,
	mpc mpcprotocol.StoremanManager) error {

	return ready.BaseStep.FinishStep()

}

func (ready *GetMpcReadyStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	return true
}
