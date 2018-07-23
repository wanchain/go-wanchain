package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc/step"
)

//send create LockAccount from leader
func requestTxSignMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	requestMpc := step.CreateRequestMpcStep(&mpc.peers, mpcprotocol.MpcTXSignLeader)
	mpcReady := step.CreateMpcReadyStep(&mpc.peers)
	return generateTxSignMpc(mpc, requestMpc, mpcReady)
}

//get message from leader and create Context
func acknowledgeTxSignMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(&mpc.peers, mpcprotocol.MpcTXSignPeer)
	mpcReady := step.CreateGetMpcReadyStep(&mpc.peers)
	return generateTxSignMpc(mpc, AcknowledgeMpc, mpcReady)
}

func generateTxSignMpc(mpc *MpcContext, firstStep MpcStepFunc, readyStep MpcStepFunc) (*MpcContext, error) {
	JRJZ := step.CreateTXSignJR_JZ_Step(mpcprotocol.MPCDegree, &mpc.peers)
	AGPoint := step.CreateMpcPoint_Step(&mpc.peers, mpcprotocol.MpcSignA0, mpcprotocol.MpcSignAPoint)
	ARLag := step.CreateTXSign_Lagrange_Step(&mpc.peers, mpcprotocol.MpcSignARSeed, mpcprotocol.MpcSignARResult)
	TXSignLag := step.CreateTxSign_CalSignStep(&mpc.peers, mpcprotocol.MpcTxSignResult)
	mpc.setMpcStep(firstStep, readyStep, JRJZ, AGPoint, ARLag, TXSignLag)
	return mpc, nil
}
