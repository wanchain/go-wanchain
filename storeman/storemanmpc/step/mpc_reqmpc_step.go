package step

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	"math/rand"
	"time"
)

type RequestMpcStep struct {
	BaseStep
	messageType int64
	address     big.Int
	mpcM        []byte
	message     map[discover.NodeID]bool
}

func (req *RequestMpcStep) InitStep(result mpcprotocol.MpcResultInterface) error {
	log.SyslogInfo("RequestMpcStep.InitStep begin")

	if req.messageType == mpcprotocol.MpcGPKLeader {
		findMap := make(map[uint64]bool)
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < len(*req.peers); i++ {
			for {
				(*req.peers)[i].Seed = (uint64)(rand.Intn(0x0FFFFFE) + 1)
				_, exist := findMap[(*req.peers)[i].Seed]
				if exist {
					continue
				}

				findMap[(*req.peers)[i].Seed] = true
				break
			}
		}

	} else if req.messageType == mpcprotocol.MpcSignLeader {
		addr, err := result.GetValue(mpcprotocol.MpcAddress)
		if err != nil {
			return err
		}

		req.address = addr[0]
		req.mpcM, err = result.GetByteValue(mpcprotocol.MpcM)
		if err != nil {
			return err
		}

	}

	return nil
}

func CreateRequestMpcStep(peers *[]mpcprotocol.PeerInfo, messageType int64) *RequestMpcStep {
	return &RequestMpcStep{BaseStep: *CreateBaseStep(peers, len(*peers)-1),
		messageType: messageType,
		message:     make(map[discover.NodeID]bool)}
}

func (req *RequestMpcStep) CreateMessage() []mpcprotocol.StepMessage {
	msg := mpcprotocol.StepMessage{
		Msgcode:   mpcprotocol.RequestMPC,
		PeerID:    nil,
		Peers:     req.peers,
		Data:      nil,
		BytesData: nil}
	msg.Data = make([]big.Int, 1)
	msg.Data[0].SetInt64(req.messageType)
	if req.messageType == mpcprotocol.MpcSignLeader {
		msg.Data = append(msg.Data, req.address)
		msg.BytesData = make([][]byte, 1)
		msg.BytesData[0] = req.mpcM
	} else if req.messageType == mpcprotocol.MpcGPKLeader {

	}

	return []mpcprotocol.StepMessage{msg}
}

func (req *RequestMpcStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := req.BaseStep.FinishStep()
	if err != nil {
		return err
	}

	data := make([]big.Int, 1)
	data[0].SetInt64(req.messageType)
	result.SetValue(mpcprotocol.MPCAction, data)
	return nil
}

func (req *RequestMpcStep) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.SyslogInfo("RequestMpcStep.HandleMessage begin, peerID:%s", msg.PeerID.String())
	_, exist := req.message[*msg.PeerID]
	if exist {
		log.SyslogErr("RequestMpcStep.HandleMessage, get message from peerID fail. peer:%s", msg.PeerID.String())
		return false
	}

	req.message[*msg.PeerID] = true
	return true
}
