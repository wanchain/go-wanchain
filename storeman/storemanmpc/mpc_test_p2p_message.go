package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc/step"
)

//send create LockAccount from leader
func testCreatep2pMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	requestMpc := step.CreateRequestMpcStep(&mpc.peers, mpcprotocol.MpcGPKLeader)
	mpcReady := step.CreateMpcReadyStep(&mpc.peers)
	return generateCreateTestMpc(mpc, requestMpc, mpcReady)
}

//get message from leader and create Context
func acknowledgeCreatep2pMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	findMap := make(map[uint64]bool)
	for _, item := range peers {
		if item.Seed > 0xffffff {
			return nil, mpcprotocol.ErrMpcSeedOutRange
		}
		_, exist := findMap[item.Seed]
		if exist {
			return nil, mpcprotocol.ErrMpcSeedDuplicate
		}
		findMap[item.Seed] = true
	}

	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(&mpc.peers, mpcprotocol.MpcGPKPeer)
	mpcReady := step.CreateGetMpcReadyStep(&mpc.peers)
	return generateCreateTestMpc(mpc, AcknowledgeMpc, mpcReady)
}

func generateCreateTestMpc(mpc *MpcContext, firstStep MpcStepFunc, readyStep MpcStepFunc) (*MpcContext, error) {
	test := 1000
	mpcTest := make([]MpcStepFunc, test+2)
	mpcTest[0] = firstStep
	mpcTest[1] = readyStep
	for i := 0; i < test; i++ {
		mpcTest[i+2] = step.CreateMpcSKShare_Step(mpcprotocol.MPCDegree, &mpc.peers)
	}

	mpc.setMpcStep(mpcTest...)
	return mpc, nil
}
