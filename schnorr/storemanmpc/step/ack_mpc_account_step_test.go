package step

import (
	"testing"
	mpcprotocol "github.com/wanchain/go-wanchain/schnorr/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"math/big"
	"github.com/wanchain/go-wanchain/common"
	"errors"
	"bytes"
)

var peers []mpcprotocol.PeerInfo

var msg1 mpcprotocol.StepMessage
var msg2 mpcprotocol.StepMessage
var msg3 mpcprotocol.StepMessage
var msgWrong1 mpcprotocol.StepMessage
var msgWrong2 mpcprotocol.StepMessage

type tmpMpcResult struct {
}

var mpcAddr = "0x0000000000000000000000000000000000000055"
var mpcAddrBytes = common.FromHex(mpcAddr)
var wrongMpcAddrBytes1 = common.FromHex("0x0000000000000000000000000000000000000044")
var wrongMpcAddrBytes2 = common.FromHex("0x00000000000000000000000000000000000044")


func (ret *tmpMpcResult) Initialize() error {
	return nil
}

func (ret *tmpMpcResult) SetValue(key string, value []big.Int) error {
	return nil
}

func (ret *tmpMpcResult) GetValue(key string) ([]big.Int, error) {
	return nil, nil
}

func (ret *tmpMpcResult) SetByteValue(key string, value []byte) error {
	return nil
}

func (ret *tmpMpcResult) GetByteValue(key string) ([]byte, error) {
	if key == mpcprotocol.MpcContextResult {
		return mpcAddrBytes, nil
	}

	return nil, nil
}


type tmpMpcResultWrong1 struct {
	tmpMpcResult
}

type tmpMpcResultWrong2 struct {
	tmpMpcResult
}

func (ret *tmpMpcResultWrong1) GetByteValue(key string) ([]byte, error) {
	return nil, errors.New("invalid key")
}

func (ret *tmpMpcResultWrong2) GetByteValue(key string) ([]byte, error) {
	if key == mpcprotocol.MpcContextResult {
		return common.FromHex("0x00000000000000000055"), nil
	}

	return nil, nil
}


var mpcResult tmpMpcResult
var mpcResultWrong1 tmpMpcResultWrong1
var mpcResultWrong2 tmpMpcResultWrong2

func Init()  {
	if len(peers) != 0 {
		return
	}

	nodeId1, _ := discover.HexID("0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001")
	nodeId2, _ := discover.HexID("0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002")
	nodeId3, _ := discover.HexID("0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003")

	peers = append(peers, mpcprotocol.PeerInfo{nodeId1, 1})
	peers = append(peers, mpcprotocol.PeerInfo{nodeId2, 2})
	peers = append(peers, mpcprotocol.PeerInfo{nodeId3, 3})

	bytesData := make([][]byte, 1, 1)
	bytesData[0] = mpcAddrBytes
	msg1 = mpcprotocol.StepMessage{mpcprotocol.MPCMessage, &peers[0].PeerID, &peers, nil, bytesData}
	msg2 = mpcprotocol.StepMessage{mpcprotocol.MPCMessage, &peers[1].PeerID, &peers, nil, bytesData}
	msg3 = mpcprotocol.StepMessage{mpcprotocol.MPCMessage, &peers[2].PeerID, &peers, nil, bytesData}

	bytesData[0] = wrongMpcAddrBytes1
	msgWrong1 = mpcprotocol.StepMessage{mpcprotocol.MPCMessage, &peers[0].PeerID, &peers, nil, bytesData}

	bytesData[0] = wrongMpcAddrBytes2
	msgWrong2 = mpcprotocol.StepMessage{mpcprotocol.MPCMessage, &peers[0].PeerID, &peers, nil, bytesData}
}

func TestInitStep(t *testing.T) {

	Init()
	step := CreateAckMpcAccountStep(&peers)

	err := step.InitStep(&mpcResultWrong1)
	if err == nil {
		t.Error("should return error")
	}

	err = step.InitStep(&mpcResultWrong2)
	if err != mpcprotocol.ErrInvalidMPCAddr {
		t.Error("should return 'invalid mpc address'")
	}

	err = step.InitStep(&mpcResult)
	if err != nil {
		t.Error("InitStep should succeed")
	}

	if !bytes.Equal(step.mpcAddr, mpcAddrBytes) {
		t.Error("invalid step's mpcAddr")
	}
}


func TestHandleMessage(t *testing.T) {
	Init()
	step := CreateAckMpcAccountStep(&peers)
	step.InitStep(&mpcResult)

	bSuc := step.HandleMessage(&msg1)
	if !bSuc {
		t.Error("step HandleMessage should succeed")
	}

	bSuc = step.HandleMessage(&msg1)
	if bSuc {
		t.Error("step repeat HandleMessage should fail")
	}

	bSuc = step.HandleMessage(&msg2)
	if !bSuc {
		t.Error("step HandleMessage should succeed")
	}
}

func TestFinishStep(t *testing.T) {
	Init()

	{
		step := CreateAckMpcAccountStep(&peers)
		step.InitStep(&mpcResult)

		step.HandleMessage(&msg1)
		step.HandleMessage(&msg2)
		step.HandleMessage(&msg3)

		err := step.FinishStep(&mpcResult, nil)
		if err != nil {
			t.Error("step FinishStep fail")
		}
	}

	{
		step := CreateAckMpcAccountStep(&peers)
		step.InitStep(&mpcResult)

		step.HandleMessage(&msg1)
		step.HandleMessage(&msg2)
		step.HandleMessage(&msgWrong1)

		err := step.FinishStep(&mpcResult, nil)
		if err == nil {
			t.Error("step FinishStep should fail")
		}
	}

	{
		step := CreateAckMpcAccountStep(&peers)
		step.InitStep(&mpcResult)

		step.HandleMessage(&msg1)
		step.HandleMessage(&msg2)
		step.HandleMessage(&msgWrong2)

		err := step.FinishStep(&mpcResult, nil)
		if err == nil {
			t.Error("step FinishStep should fail")
		}
	}
}










