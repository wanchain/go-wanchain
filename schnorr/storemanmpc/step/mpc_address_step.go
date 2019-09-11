package step

import mpcprotocol "github.com/wanchain/go-wanchain/schnorr/storemanmpc/protocol"

type MpcAddressStep struct {
	MpcPoint_Step
	accType string
}

func CreateMpcAddressStep(peers *[]mpcprotocol.PeerInfo, accType string) *MpcAddressStep {
	mpc := &MpcAddressStep{MpcPoint_Step: *CreateMpcPoint_Step(peers, []string{mpcprotocol.MpcPublicShare}, []string{mpcprotocol.PublicKeyResult}), accType:accType}
	return mpc
}

func (addStep *MpcAddressStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := addStep.MpcPoint_Step.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	return mpc.CreateKeystore(result, addStep.peers, addStep.accType)
}
