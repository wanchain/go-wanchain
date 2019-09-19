package storemanmpc

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"sync"
)

type MemStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
	Self uint64 `json:"self"`
}

type MpcStepFunc interface {
	mpcprotocol.GetMessageInterface
	InitMessageLoop(mpcprotocol.GetMessageInterface) error
	Quit(error)
	InitStep(mpcprotocol.MpcResultInterface) error
	CreateMessage() []mpcprotocol.StepMessage
	FinishStep(mpcprotocol.MpcResultInterface, mpcprotocol.StoremanManager) error
	GetMessageChan() chan *mpcprotocol.StepMessage
	SetWaitAll(bool)
}

type MpcContext struct {
	ContextID   uint64 //Unique id for every content
	quitMu      sync.Mutex
	bQuit       bool
	peers       []mpcprotocol.PeerInfo
	mpcResult   mpcprotocol.MpcResultInterface
	MpcSteps    []MpcStepFunc
	MapStepChan map[uint64]chan *mpcprotocol.StepMessage
}

func (mpcCtx *MpcContext) getMpcResult() []byte {
	value, err := mpcCtx.mpcResult.GetByteValue(mpcprotocol.MpcContextResult)
	if err != nil {
		return []byte{}
	} else {
		return value
	}
}

func (mpcCtx *MpcContext) getMessage(PeerID *discover.NodeID,
	msg *mpcprotocol.MpcMessage,
	peers *[]mpcprotocol.PeerInfo) error {

	mpcCtx.MapStepChan[msg.StepID] <- &mpcprotocol.StepMessage{MsgCode: 0,
		PeerID:    PeerID,
		Peers:     peers,
		Data:      msg.Data,
		BytesData: msg.BytesData}
	return nil
}

func createMpcContext(contextID uint64,
	peers []mpcprotocol.PeerInfo,
	mpcResult mpcprotocol.MpcResultInterface) *MpcContext {

	mpc := &MpcContext{
		ContextID:   contextID,
		peers:       peers,
		bQuit:       false,
		quitMu:      sync.Mutex{},
		mpcResult:   mpcResult,
		MapStepChan: make(map[uint64]chan *mpcprotocol.StepMessage),
	}

	return mpc
}

func (mpcCtx *MpcContext) setMpcStep(mpcSteps ...MpcStepFunc) {
	mpcCtx.MpcSteps = mpcSteps
	for i, step := range mpcSteps {
		mpcCtx.MapStepChan[uint64(i)] = step.GetMessageChan()
	}
}

func (mpcCtx *MpcContext) quit(err error) {
	if err == nil {
		log.SyslogInfo("MpcContext.quit")
	} else {
		log.SyslogErr("MpcContext.quit", "err", err.Error())
	}

	mpcCtx.quitMu.Lock()
	defer mpcCtx.quitMu.Unlock()
	if mpcCtx.bQuit {
		return
	}
	mpcCtx.bQuit = true
	for i := 0; i < len(mpcCtx.MpcSteps); i++ {
		mpcCtx.MpcSteps[i].Quit(err)
	}
}

func (mpcCtx *MpcContext) mainMPCProcess(StoremanManager mpcprotocol.StoremanManager) error {
	log.SyslogInfo("mainMPCProcess begin", "ctxid", mpcCtx.ContextID)
	mpcErr := error(nil)
	for _, mpcCt := range mpcCtx.MpcSteps {
		err := mpcCt.InitMessageLoop(mpcCt)
		if err != nil {
			mpcErr = err
			break
		}
	}

	peerIDs := make([]discover.NodeID, 0)
	for _, item := range mpcCtx.peers {
		peerIDs = append(peerIDs, item.PeerID)
	}

	if mpcErr == nil {
		mpcCtx.mpcResult.Initialize()
		for i := 0; i < len(mpcCtx.MpcSteps); i++ {
			err := mpcCtx.MpcSteps[i].InitStep(mpcCtx.mpcResult)
			if err != nil {
				mpcErr = err
				break
			}

			log.SyslogInfo("step init finished", "ctxid", mpcCtx.ContextID, "stepId", i)
			msg := mpcCtx.MpcSteps[i].CreateMessage()
			if msg != nil {
				for _, item := range msg {
					mpcMsg := &mpcprotocol.MpcMessage{ContextID: mpcCtx.ContextID,
						StepID:    uint64(i),
						Data:      item.Data,
						BytesData: item.BytesData}
					StoremanManager.SetMessagePeers(mpcMsg, item.Peers)
					if item.PeerID != nil {
						StoremanManager.P2pMessage(item.PeerID, item.MsgCode, mpcMsg)
						log.SyslogInfo("step send a p2p msg", "ctxid", mpcCtx.ContextID, "stepId", i)
					} else {
						StoremanManager.BroadcastMessage(peerIDs, item.MsgCode, mpcMsg)
						log.SyslogInfo("step boardcast a p2p msg", "ctxid", mpcCtx.ContextID, "stepId", i)
					}
				}
			}

			log.SyslogInfo("step send p2p msg finished", "ctxid", mpcCtx.ContextID, "stepId", i)
			err = mpcCtx.MpcSteps[i].FinishStep(mpcCtx.mpcResult, StoremanManager)
			if err != nil {
				mpcErr = err
				break
			}

			log.SyslogInfo("step mssage finished", "ctxid", mpcCtx.ContextID, "stepId", i)
		}
	}

	if mpcErr != nil {
		log.SyslogErr("mainMPCProcess fail", "err", mpcErr.Error())
		mpcMsg := &mpcprotocol.MpcMessage{ContextID: mpcCtx.ContextID,
			StepID: 0,
			Peers:  []byte(mpcErr.Error())}
		StoremanManager.BroadcastMessage(peerIDs, mpcprotocol.MPCError, mpcMsg)
	}

	mpcCtx.quit(nil)
	log.SyslogInfo("MpcContext finished", "ctx ID", mpcCtx.ContextID)
	return mpcErr
}
