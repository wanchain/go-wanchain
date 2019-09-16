package step

import mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"

type MpcGPKStep struct {
	MpcPoint_Step
	accType string
}

func CreateMpcGPKStep(peers *[]mpcprotocol.PeerInfo, accType string) *MpcGPKStep {
	mpc := &MpcGPKStep{MpcPoint_Step: *CreateMpcPoint_Step(peers,
		[]string{mpcprotocol.MpcPublicShare},
		[]string{mpcprotocol.PublicKeyResult}),
		accType: accType}
	return mpc
}

func (addStep *MpcGPKStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := addStep.MpcPoint_Step.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	return mpc.CreateKeystore(result, addStep.peers, addStep.accType)
}
