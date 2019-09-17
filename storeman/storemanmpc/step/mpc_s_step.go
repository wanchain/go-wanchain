package step

import mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"

type MpcSStep struct {
	MpcPoint_Step
	accType string
}

func CreateMpcSStep(peers *[]mpcprotocol.PeerInfo, accType string) *MpcSStep {
	mpc := &MpcSStep{MpcPoint_Step: *CreateMpcPoint_Step(peers,
		[]string{mpcprotocol.MpcPublicShare},
		[]string{mpcprotocol.PublicKeyResult}),
		accType: accType}
	return mpc
}

func (addStep *MpcSStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := addStep.MpcPoint_Step.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	return nil
}
