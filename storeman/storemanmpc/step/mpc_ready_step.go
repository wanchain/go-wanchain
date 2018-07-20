package step

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
)

type MpcReadyStep struct {
	BaseStep
}

func (ready *MpcReadyStep) InitStep(mpcprotocol.MpcResultInterface) error {
	return nil
}

func CreateMpcReadyStep(peers *[]mpcprotocol.PeerInfo) *MpcReadyStep {
	return &MpcReadyStep{*CreateBaseStep(peers, 0)}
}

func (ready *MpcReadyStep) CreateMessage() []mpcprotocol.StepMessage {
	data := make([]big.Int, 1)
	data[0].SetInt64(1)
	return []mpcprotocol.StepMessage{mpcprotocol.StepMessage{mpcprotocol.MPCMessage, nil, nil, data, nil}}
}

func (ready *MpcReadyStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := ready.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	return nil
}

func (ready *MpcReadyStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	return true
}
