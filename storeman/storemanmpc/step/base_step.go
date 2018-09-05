package step

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"time"
)

type BaseStep struct {
	peers   *[]mpcprotocol.PeerInfo
	msgChan chan *mpcprotocol.StepMessage
	finish  chan error
	waiting int
}

func CreateBaseStep(peers *[]mpcprotocol.PeerInfo, wait int) *BaseStep {
	step := &BaseStep{peers: peers, msgChan: make(chan *mpcprotocol.StepMessage, len(*peers)+3), finish: make(chan error, 3)}
	if wait >= 0 {
		step.waiting = wait
	} else {
		step.waiting = len(*peers)
	}

	return step
}

func (step *BaseStep) InitMessageLoop(msger mpcprotocol.GetMessageInterface) error {
	mpcsyslog.Debug("BaseStep.InitMessageLoop begin")
	if step.waiting <= 0 {
		step.finish <- nil
	} else {
		go func() {
			log.Debug("InitMessageLoop begin")
			mpcsyslog.Debug("InitMessageLoop begin")

			for {
				err := step.HandleMessage(msger)
				if err != nil {
					if err != mpcprotocol.ErrQuit {
						log.Error("InitMessageLoop fail", "get message err", err)
					}

					break
				}
			}

			log.Debug("InitMessageLoop end")
		}()
	}

	return nil
}

func (step *BaseStep) Quit(err error) {
	step.msgChan <- nil
	step.finish <- err
}

func (step *BaseStep) FinishStep() error {
	select {
	case err := <-step.finish:
		if err != nil {
			mpcsyslog.Err("BaseStep.FinishStep, get a step finish error. err:%s", err.Error())
		}

		step.msgChan <- nil
		return err
	case <-time.After(mpcprotocol.MPCTimeOut):
		mpcsyslog.Err("BaseStep.FinishStep, wait step finish timeout")
		step.msgChan <- nil
		return mpcprotocol.ErrTimeOut
	}
}

func (step *BaseStep) GetMessageChan() chan *mpcprotocol.StepMessage {
	return step.msgChan
}

func (step *BaseStep) HandleMessage(msger mpcprotocol.GetMessageInterface) error {
	var msg *mpcprotocol.StepMessage
	select {
	case msg = <-step.msgChan:
		if msg == nil {
			log.Info("-----------------HandleMessage. BaseStep get a quit msg")
			mpcsyslog.Info("BaseStep get a quit msg")
			return mpcprotocol.ErrQuit
		}

		if step.waiting > 0 && msger.HandleMessage(msg) {
			step.waiting--
			if step.waiting <= 0 {
				step.finish <- nil
			}
		}
	}

	return nil
}

func (step *BaseStep) getPeerIndex(peerID *discover.NodeID) int {
	for i, item := range *step.peers {
		if item.PeerID == *peerID {
			return i
		}
	}

	return -1
}

func (step *BaseStep) getPeerSeed(peerID *discover.NodeID) uint64 {
	for _, item := range *step.peers {
		if item.PeerID == *peerID {
			return item.Seed
		}
	}

	return 0
}
