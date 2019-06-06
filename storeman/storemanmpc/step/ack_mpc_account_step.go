package step

import (
	"bytes"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
)

type AckMpcAccountStep struct {
	BaseStep
	message        map[discover.NodeID]bool
	mpcAddr        []byte
	remoteMpcAddrs map[discover.NodeID][]byte
}

func CreateAckMpcAccountStep(peers *[]mpcprotocol.PeerInfo) *AckMpcAccountStep {
	return &AckMpcAccountStep{*CreateBaseStep(peers, -1), make(map[discover.NodeID]bool), nil, make(map[discover.NodeID][]byte)}
}

func (ack *AckMpcAccountStep) InitStep(result mpcprotocol.MpcResultInterface) error {
	mpcsyslog.Info("AckMpcAccountStep.InitStep begin")
	mpcAddr, err := result.GetByteValue(mpcprotocol.MpcContextResult)
	if err != nil {
		mpcsyslog.Err("ack mpc account step, init fail. err:%s", err.Error())
		return err
	}

	if len(mpcAddr) != common.AddressLength {
		mpcsyslog.Err("ack mpc account step, invalid mpc address length. address:%s", common.ToHex(mpcAddr))
		return mpcprotocol.ErrInvalidMPCAddr
	}

	ack.mpcAddr = mpcAddr
	return nil
}

func (ack *AckMpcAccountStep) CreateMessage() []mpcprotocol.StepMessage {
	return []mpcprotocol.StepMessage{mpcprotocol.StepMessage{
		Msgcode:mpcprotocol.MPCMessage,
		PeerID:nil,
		Peers:nil,
		Data:nil,
		BytesData:[][]byte{ack.mpcAddr}}}
}

func (ack *AckMpcAccountStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	mpcsyslog.Info("AckMpcAccountStep.FinishStep begin")
	err := ack.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	if len(ack.remoteMpcAddrs) != len(*ack.BaseStep.peers) {
		mpcsyslog.Err("ack mpc account step, finish, invalid remote mpc address. peer num:%d, mpc addr num:%d", len(*ack.BaseStep.peers), len(ack.remoteMpcAddrs))
		return mpcprotocol.ErrInvalidMPCAddr
	}

	for peerID, mpcAddr := range ack.remoteMpcAddrs {
		if mpcAddr == nil {
			mpcsyslog.Err("ack mpc account step, finish, invalid remote mpc address: nil. peerID:%s", peerID.String())
			return mpcprotocol.ErrInvalidMPCAddr
		}

		if !bytes.Equal(ack.mpcAddr, mpcAddr) {
			mpcsyslog.Err("ack mpc account step, finish, invalid remote mpc address. local:%s, received:%s, peerID:%s", common.ToHex(ack.mpcAddr), common.ToHex(mpcAddr), peerID.String())
			return mpcprotocol.ErrInvalidMPCAddr
		}
	}

	return nil
}

func (ack *AckMpcAccountStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	mpcsyslog.Info("AckMpcAccountStep.HandleMessage begin")
	_, exist := ack.message[*msg.PeerID]
	if exist {
		mpcsyslog.Err("AckMpcAccountStep.HandleMessage fail. peer doesn't exist in task peer group. peerID:%s", msg.PeerID.String())
		return false
	}

	if len(msg.BytesData) >= 1 {
		ack.remoteMpcAddrs[*msg.PeerID] = msg.BytesData[0]
	}

	ack.message[*msg.PeerID] = true
	return true
}
