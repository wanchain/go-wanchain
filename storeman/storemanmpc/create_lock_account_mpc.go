package storemanmpc

import (
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc/step"
	"github.com/wanchain/go-wanchain/log"
)

//send create LockAccount from leader
func requestCreateLockAccountMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	requestMpc := step.CreateRequestMpcStep(&mpc.peers, mpcprotocol.MpcCreateLockAccountLeader)
	mpcReady := step.CreateMpcReadyStep(&mpc.peers)
	return generateCreateLockAccountMpc(mpc, requestMpc, mpcReady)

}

//get message from leader and create Context
func acknowledgeCreateLockAccountMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	log.Warn("-----------------acknowledgeCreateLockAccountMpc begin.")
	for _, preSetValuebyteData := range preSetValue {
		log.Warn("-----------------acknowledgeCreateLockAccountMpc", "byteValue", string(preSetValuebyteData.ByteValue[:]))
	}

	findMap := make(map[uint64]bool)
	for _, item := range peers {
		if item.Seed > 0xffffff {
			mpcsyslog.Err("acknowledgeCreateLockAccountMpc fail. err:%s", mpcprotocol.ErrMpcSeedOutRange.Error())
			return nil, mpcprotocol.ErrMpcSeedOutRange
		}

		_, exist := findMap[item.Seed]
		if exist {
			mpcsyslog.Err("acknowledgeCreateLockAccountMpc fail. err:%s", mpcprotocol.ErrMpcSeedDuplicate.Error())
			return nil, mpcprotocol.ErrMpcSeedDuplicate
		}

		findMap[item.Seed] = true
	}

	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(&mpc.peers, mpcprotocol.MpcCreateLockAccountPeer)
	mpcReady := step.CreateGetMpcReadyStep(&mpc.peers)
	return generateCreateLockAccountMpc(mpc, AcknowledgeMpc, mpcReady)
}

func generateCreateLockAccountMpc(mpc *MpcContext, firstStep MpcStepFunc, readyStep MpcStepFunc) (*MpcContext, error) {
	var accTypeStr string
	accType, err := mpc.mpcResult.GetByteValue(mpcprotocol.MpcStmAccType)
	if err != nil {
		return nil, err
	} else if accType == nil {
		accTypeStr = ""
	} else {
		accTypeStr = string(accType[:])
	}

	JRSS := step.CreateMpcJRSS_Step(mpcprotocol.MPCDegree, &mpc.peers)
	PublicKey := step.CreateMpcAddressStep(&mpc.peers, accTypeStr)
	ackAddress := step.CreateAckMpcAccountStep(&mpc.peers)
	mpc.setMpcStep(firstStep, readyStep, JRSS, PublicKey, ackAddress)
	return mpc, nil
}
