package step

import mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"

type MpcAckRSStep struct {
	MpcPoint_Step
	accType string
}

func CreateAckMpcRSStep(peers *[]mpcprotocol.PeerInfo, accType string) *MpcAckRSStep {
	mpc := &MpcAckRSStep{MpcPoint_Step: *CreateMpcPoint_Step(peers,
		[]string{mpcprotocol.MpcPublicShare},
		[]string{mpcprotocol.PublicKeyResult}),
		accType: accType}
	return mpc
}

func (mars *MpcAckRSStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := mars.MpcPoint_Step.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	return nil
}
