package storemanmpc

import (
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc/step"
)

//send create LockAccount from leader
func reqGPKMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	requestMpc := step.CreateRequestMpcStep(&mpc.peers, mpcprotocol.MpcGPKLeader)
	mpcReady := step.CreateMpcReadyStep(&mpc.peers)
	return genCreateGPKMpc(mpc, requestMpc, mpcReady)

}

//get message from leader and create Context
func ackGPKMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	log.SyslogInfo("acknowledgeCreateLockAccountMpc begin.")
	for _, preSetValuebyteData := range preSetValue {
		log.SyslogInfo("acknowledgeCreateLockAccountMpc", "byteValue", string(preSetValuebyteData.ByteValue[:]))
	}

	findMap := make(map[uint64]bool)
	for _, item := range peers {
		if item.Seed > 0xffffff {
			log.SyslogErr("acknowledgeCreateLockAccountMpc fail", "err", mpcprotocol.ErrMpcSeedOutRange.Error())
			return nil, mpcprotocol.ErrMpcSeedOutRange
		}

		_, exist := findMap[item.Seed]
		if exist {
			log.SyslogErr("acknowledgeCreateLockAccountMpc fail", "err", mpcprotocol.ErrMpcSeedDuplicate.Error())
			return nil, mpcprotocol.ErrMpcSeedDuplicate
		}

		findMap[item.Seed] = true
	}

	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(&mpc.peers, mpcprotocol.MpcGPKPeer)
	mpcReady := step.CreateGetMpcReadyStep(&mpc.peers)
	return genCreateGPKMpc(mpc, AcknowledgeMpc, mpcReady)
}

func genCreateGPKMpc(mpc *MpcContext, firstStep MpcStepFunc, readyStep MpcStepFunc) (*MpcContext, error) {
	accTypeStr := ""
	skShare := step.CreateMpcSKShare_Step(mpcprotocol.MPCDegree, &mpc.peers)
	gpk := step.CreateMpcGPKStep(&mpc.peers, accTypeStr)
	ackGpk := step.CreateAckMpcGPKStep(&mpc.peers)
	mpc.setMpcStep(firstStep, readyStep, skShare, gpk, ackGpk)
	return mpc, nil
}
