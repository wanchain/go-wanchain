package step

import mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"

type MpcGPKStep struct {
	MpcPointStep
	accType string
}

func CreateMpcGPKStep(peers *[]mpcprotocol.PeerInfo, accType string) *MpcGPKStep {
	mpc := &MpcGPKStep{
		MpcPointStep: *CreateMpcPointStep(
			peers,
			[]string{mpcprotocol.MpcPublicShare},
			[]string{mpcprotocol.PublicKeyResult}),
		accType: accType}
	return mpc
}

func (addStep *MpcGPKStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := addStep.MpcPointStep.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	return mpc.CreateKeystore(result, addStep.peers, addStep.accType)
}
