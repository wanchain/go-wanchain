// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package downloader

import (
	"sync"
	"time"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
	"github.com/wanchain/go-wanchain/core/types"
	"errors"
	"math/rand"
)


type epochGenesisReq struct {
	epochid  *big.Int              			// epochid items to download
	tasks    map[*big.Int]*epochGenesisReq 			// Download tasks to track previous attempts
	timeout  time.Duration              	// Maximum round trip time for this to complete
	timer    *time.Timer                	// Timer to fire when the RTT timeout expires
	peer     *peerConnection            	// Peer that we're requesting from
	response *types.EpochGenesis                   	// Response data of the peer (nil for timeouts)
	dropped  bool                       	// Flag whether the peer dropped off early
}

// timedOut returns if this request timed out.
func (req *epochGenesisReq) timedOut() bool {
	return req.response == nil
}

func (d *Downloader) epochGenesisFetcher() {
	for {
		select {

		case epochid := <-d.epochGenesisSyncStart:

			s := newepochGenesisSync(d, epochid)
			d.runEpochGenesisSync(s)

		case <-d.quitCh:
			return
		}
	}
}


func (d *Downloader) runEpochGenesisSync(s *epochGenesisSync) *epochGenesisSync {

	var (
		active   = make(map[string]*epochGenesisReq) // Currently in-flight requests
		finished []*epochGenesisReq                  // Completed or failed requests
		timeout  = make(chan *epochGenesisReq)       // Timed out active requests
	)

	defer func() {
		for _, req := range active {
			req.timer.Stop()
			req.peer.SetEpochGenesisDataIdle(1)
		}
	}()

	go s.run()
	defer s.Cancel()

	// Listen for peer departure events to cancel assigned tasks
	peerDrop := make(chan *peerConnection, 1024)
	peerSub := s.d.peers.SubscribePeerDrops(peerDrop)
	defer peerSub.Unsubscribe()

	for {
		// Enable sending of the first buffered element if there is one.
		var (
			deliverReq   *epochGenesisReq
			deliverReqCh chan *epochGenesisReq
		)

		if len(finished) > 0 {
			deliverReq = finished[0]
			deliverReqCh = s.deliver
		}

		select {

		case <-s.done:
			return nil

		case deliverReqCh <- deliverReq:
			finished = append(finished[:0], finished[1:]...)

		case pack := <-d.epochGenesisCh:

			req := active[pack.PeerId()]
			if req == nil {
				log.Debug("Unrequested epoch genesis data", "peer", pack.PeerId(), "len", pack.Items())
				continue
			}
			// Finalize the request and queue up for processing
			req.timer.Stop()
			req.response = pack.(*epochGenesisPack).epochGenesis

			finished = append(finished, req)

			delete(active, pack.PeerId())

			// Handle dropped peer connections:
		case p := <-peerDrop:
			// Skip if no request is currently pending
			req := active[p.id]
			if req == nil {
				continue
			}
			// Finalize the request and queue up for processing
			req.timer.Stop()
			req.dropped = true

			finished = append(finished, req)
			delete(active, p.id)

		// Handle timed-out requests:
		case req := <-timeout:
			// If the peer is already requesting something else, ignore the stale timeout.
			// This can happen when the timeout and the delivery happens simultaneously,
			// causing both pathways to trigger.
			if active[req.peer.id] != req {
				continue
			}
			// Move the timed out data back into the download queue
			finished = append(finished, req)
			delete(active, req.peer.id)

		case req := <-d.trackEpochGenesisReq:
			if old := active[req.peer.id]; old != nil {
				log.Warn("Busy peer assigned new state fetch", "peer", old.peer.id)

				// Make sure the previous one doesn't get siletly lost
				old.timer.Stop()
				old.dropped = true

				finished = append(finished, old)
			}

			// Start a timer to notify the sync loop if the peer stalled.
			req.timer = time.AfterFunc(req.timeout, func() {
				select {
					case timeout <- req:
					case <-s.done:

				}
			})

			active[req.peer.id] = req
		}
	}
}


type epochGenesisSync struct {
	epochid	uint64
	d *Downloader 							// Downloader instance to access and manage current peerset
	deliver    chan *epochGenesisReq 		// Delivery channel multiplexing peer responses
	cancel     chan struct{}  				// Channel to signal a termination request
	cancelOnce sync.Once      				// Ensures cancel only ever gets called once
	done       chan struct{}  				// Channel to signal termination completion
	err        error          				// Any error hit during sync (set before completion)
}

func newepochGenesisSync(d *Downloader, epochid uint64) *epochGenesisSync {
	return &epochGenesisSync{
		epochid: epochid,
		d:       d,
		deliver: make(chan *epochGenesisReq),
		cancel:  make(chan struct{}),
	}
}


// Wait blocks until the sync is done or canceled.
func (s *epochGenesisSync) Wait() error {
	<-s.done
	return s.err
}

// Cancel cancels the sync and waits until it has shut down.
func (s *epochGenesisSync) Cancel() error {
	s.cancelOnce.Do(func() { close(s.cancel) })
	return s.Wait()
}


func (s *epochGenesisSync) run() error {

	// Listen for new peer events to assign tasks to them
	newPeer := make(chan *peerConnection, 1024)
	peerSub := s.d.peers.SubscribeNewPeers(newPeer)
	defer peerSub.Unsubscribe()

	req := &epochGenesisReq{epochid:big.NewInt(int64(s.epochid)),timeout: s.d.requestTTL()}
	req.tasks = make( map[*big.Int]*epochGenesisReq)
	req.tasks[req.epochid] = req

	for _,request :=range req.tasks {

		peers, _ := s.d.peers.NodeDataIdlePeers()

		if len(peers) == 0 {
			return errors.New("failed found peer when send epoch genesis request")
		}

		idx := rand.Intn(len(peers))
		request.peer = peers[idx]

		select {

			case s.d.trackEpochGenesisReq <- req:
				err := req.peer.FetchEpochGenesisData(s.epochid)
				if err == nil {
					delete(req.tasks,big.NewInt(int64(s.epochid)))
				}

			case <-s.cancel:
		}

		select {
		case <-newPeer:
			// New peer arrived, try to assign it download tasks

		case <-s.cancel:
			return errCancelEpochGenesisFetch

		case req := <-s.deliver:
			err := s.d.blockchain.SetEpochGenesis(req.response)
			if err != nil {
				log.Warn("get epoch genesis data write error", "err", err)
				return err
			}

			req.peer.SetNodeDataIdle(1)
		}
	}

	return nil
}
