package storemanmpc

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
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

func (mpcCtx *MpcContext) getMessage(PeerID *discover.NodeID, msg *mpcprotocol.MpcMessage, peers *[]mpcprotocol.PeerInfo) error {
	mpcCtx.MapStepChan[msg.StepID] <- &mpcprotocol.StepMessage{0, PeerID, peers, msg.Data, msg.BytesData}
	return nil
}

func createMpcContext(contextID uint64, peers []mpcprotocol.PeerInfo, mpcResult mpcprotocol.MpcResultInterface) *MpcContext {
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
	if err == nil{
		mpcsyslog.Info("MpcContext.quit")
	} else {
		mpcsyslog.Err("MpcContext.quit, err:%s", err.Error())
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
	mpcsyslog.Debug("mainMPCProcess begin, ctxid:%d", mpcCtx.ContextID)
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

			mpcsyslog.Debug("step init finished. ctxid:%d, stepId:%d", mpcCtx.ContextID, i)
			msg := mpcCtx.MpcSteps[i].CreateMessage()
			if msg != nil {
				for _, item := range msg {
					mpcMsg := &mpcprotocol.MpcMessage{ContextID: mpcCtx.ContextID,
						StepID:    uint64(i),
						Data:      item.Data,
						BytesData: item.BytesData}
					StoremanManager.SetMessagePeers(mpcMsg, item.Peers)
					if item.PeerID != nil {
						StoremanManager.P2pMessage(item.PeerID, item.Msgcode, mpcMsg)
						mpcsyslog.Debug("step send a p2p msg. ctxid:%d, stepId:%d", mpcCtx.ContextID, i)
					} else {
						StoremanManager.BoardcastMessage(peerIDs, item.Msgcode, mpcMsg)
						mpcsyslog.Debug("step boardcast a p2p msg. ctxid:%d, stepId:%d", mpcCtx.ContextID, i)
					}
				}
			}

			mpcsyslog.Debug("step send p2p msg finished. ctxid:%d, stepId:%d", mpcCtx.ContextID, i)
			err = mpcCtx.MpcSteps[i].FinishStep(mpcCtx.mpcResult, StoremanManager)
			if err != nil {
				mpcErr = err
				break
			}

			mpcsyslog.Info("step mssage finished. ctxid:%d, stepId:%d", mpcCtx.ContextID, i)
			log.Info("step mssage finished", "ctxid", mpcCtx.ContextID, "stepId", i)
		}
	}

	if mpcErr != nil {
		mpcsyslog.Err("mainMPCProcess fail. err:%s", mpcErr.Error())
		mpcMsg := &mpcprotocol.MpcMessage{ContextID: mpcCtx.ContextID,
			StepID: 0,
			Peers:  []byte(mpcErr.Error())}
		StoremanManager.BoardcastMessage(peerIDs, mpcprotocol.MPCError, mpcMsg)
	}

	mpcCtx.quit(nil)
	mpcsyslog.Info("MpcContext finished. ctx ID:%d", mpcCtx.ContextID)
	log.Info("MpcContext finished", "ctx ID", mpcCtx.ContextID)
	return mpcErr
}
