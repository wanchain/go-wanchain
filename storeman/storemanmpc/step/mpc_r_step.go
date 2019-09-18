package step

import mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"

type MpcRStep struct {
	MpcPointStep
	accType string
}

func CreateMpcRStep(peers *[]mpcprotocol.PeerInfo, accType string) *MpcRStep {
	mpc := &MpcRStep{MpcPointStep: *CreateMpcPointStep(peers,
		[]string{mpcprotocol.RMpcPublicShare},
		[]string{mpcprotocol.RPublicKeyResult}),
		accType: accType}
	return mpc
}

func (addStep *MpcRStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := addStep.MpcPointStep.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	return nil
}
