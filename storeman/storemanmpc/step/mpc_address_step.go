package step

import mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"

type MpcAddressStep struct {
	MpcPoint_Step
}

func CreateMpcAddressStep(peers *[]mpcprotocol.PeerInfo) *MpcAddressStep {
	mpc := &MpcAddressStep{MpcPoint_Step: *CreateMpcPoint_Step(peers, mpcprotocol.MpcPublicShare, mpcprotocol.PublicKeyResult)}
	return mpc
}

func (addStep *MpcAddressStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := addStep.MpcPoint_Step.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	return mpc.CreateKeystore(result, addStep.peers)
}
