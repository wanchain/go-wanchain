package storemanmpc

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/storeman/btc"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc/step"
)

//send create LockAccount from leader
func reqSignMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	requestMpc := step.CreateRequestMpcStep(&mpc.peers, mpcprotocol.MpcSignLeader)
	mpcReady := step.CreateMpcReadyStep(&mpc.peers)
	return generateTxSignMpc(mpc, requestMpc, mpcReady)
}

//get message from leader and create Context
func ackSignMpc(mpcID uint64, peers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) (*MpcContext, error) {
	result := createMpcBaseMpcResult()
	result.InitializeValue(preSetValue...)
	mpc := createMpcContext(mpcID, peers, result)
	AcknowledgeMpc := step.CreateAcknowledgeMpcStep(&mpc.peers, mpcprotocol.MpcSignPeer)
	mpcReady := step.CreateGetMpcReadyStep(&mpc.peers)
	return generateTxSignMpc(mpc, AcknowledgeMpc, mpcReady)
}

func generateTxSignMpc(mpc *MpcContext, firstStep MpcStepFunc, readyStep MpcStepFunc) (*MpcContext, error) {
	log.SyslogInfo("generateTxSignMpc begin")

	signNum, err := getSignNumFromTxInfo(mpc)
	if err != nil {
		return nil, err
	}

	JRJZ := step.CreateTXSignJR_JZ_Step(mpcprotocol.MPCDegree, &mpc.peers, signNum)

	pointStepPreValueKeys := mpcprotocol.GetPreSetKeyArr(mpcprotocol.MpcSignA0, signNum)
	pointStepResultKeys := mpcprotocol.GetPreSetKeyArr(mpcprotocol.MpcSignAPoint, signNum)
	AGPoint := step.CreateMpcPoint_Step(&mpc.peers, pointStepPreValueKeys, pointStepResultKeys)

	lagStepPreValueKeys := mpcprotocol.GetPreSetKeyArr(mpcprotocol.MpcSignARSeed, signNum)
	lagStepResultKeys := mpcprotocol.GetPreSetKeyArr(mpcprotocol.MpcSignARResult, signNum)
	ARLag := step.CreateTXSign_Lagrange_Step(&mpc.peers, lagStepPreValueKeys, lagStepResultKeys)

	TXSignLag := step.CreateTxSign_CalSignStep(&mpc.peers, mpcprotocol.MpcTxSignResult, signNum)
	mpc.setMpcStep(firstStep, readyStep, JRJZ, AGPoint, ARLag, TXSignLag)
	return mpc, nil
}

func getSignNumFromTxInfo(mpc *MpcContext) (int, error) {
	signNum := 1
	chainType, err := mpc.mpcResult.GetByteValue(mpcprotocol.MpcChainType)
	if err != nil {
		log.SyslogErr("getSignNumFromTxInfo, get chainType fail", "err", err.Error())
		return 0, err
	}

	if string(chainType) == "BTC" {
		btcTxData, err := mpc.mpcResult.GetByteValue(mpcprotocol.MpcTransaction)
		if err != nil {
			log.SyslogErr("getSignNumFromTxInfo, get tx rlp date fail", "err", err.Error())
			return 0, err
		}

		var args btc.MsgTxArgs
		err = rlp.DecodeBytes(btcTxData, &args)
		if err != nil {
			log.SyslogErr("getSignNumFromTxInfo, decode tx rlp data fail", "err", err.Error())
			return 0, err
		}

		signNum = len(args.TxIn)
	}

	log.SyslogInfo("getSignNumFromTxInfo, succeed", "signNum", signNum)
	return signNum, nil
}
