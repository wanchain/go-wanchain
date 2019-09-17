package storemanmpc

import (
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc/step"
)

//send create LockAccount from leader
func reqSignMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	reqMpc := step.CreateRequestMpcStep(&mpc.peers, mpcprotocol.MpcSignLeader)
	mpcReady := step.CreateMpcReadyStep(&mpc.peers)
	return generateTxSignMpc(mpc, reqMpc, mpcReady)
}

//get message from leader and create Context
func ackSignMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	ackMpc := step.CreateAckMpcStep(&mpc.peers, mpcprotocol.MpcSignPeer)
	mpcReady := step.CreateGetMpcReadyStep(&mpc.peers)
	return generateTxSignMpc(mpc, ackMpc, mpcReady)
}

func generateTxSignMpc(mpc *MpcContext, firstStep MpcStepFunc, readyStep MpcStepFunc) (*MpcContext, error) {
	log.SyslogInfo("generateTxSignMpc begin")

	accTypeStr := ""
	skShare := step.CreateMpcRSKShareStep(mpcprotocol.MPCDegree, &mpc.peers)
	RStep := step.CreateMpcRStep(&mpc.peers, accTypeStr)
	SStep := step.CreateMpcSStep(&mpc.peers, []string{mpcprotocol.MpcPrivateShare}, []string{mpcprotocol.MpcS})
	ackRSStep := step.CreateAckMpcRSStep(&mpc.peers, accTypeStr)
	mpc.setMpcStep(firstStep, readyStep, skShare, RStep, SStep, ackRSStep)

	return mpc, nil
}
