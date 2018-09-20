package step

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/log"
)

type GetMpcReadyStep struct {
	BaseStep
}

func (ready *GetMpcReadyStep) InitStep(mpcprotocol.MpcResultInterface) error {
	log.Warn("-----------------GetMpcReadyStep.InitStep begin")
	return nil
}

func CreateGetMpcReadyStep(peers *[]mpcprotocol.PeerInfo) *GetMpcReadyStep {
	log.Warn("-----------------CreateGetMpcReadyStep begin")
	return &GetMpcReadyStep{*CreateBaseStep(peers, 1)}
}

func (ready *GetMpcReadyStep) CreateMessage() []mpcprotocol.StepMessage {
	log.Warn("-----------------GetMpcReadyStep.CreateMessage begin")
	return nil
}

func (ready *GetMpcReadyStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.Warn("-----------------GetMpcReadyStep.FinishStep begin")
	return ready.BaseStep.FinishStep()
}

func (ready *GetMpcReadyStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.Warn("-----------------GetMpcReadyStep.HandleMessage begin")
	return true
}
