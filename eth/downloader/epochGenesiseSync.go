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
	"fmt"
	"hash"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wanchain/go-wanchain/crypto/sha3"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/trie"
	"math/big"
	"github.com/wanchain/go-wanchain/core/types"
	"errors"
)


type epochGenesisReq struct {
	epochid  *big.Int              			// epochid items to download
	tasks    map[*big.Int]*epochGenesisTask // Download tasks to track previous attempts
	timeout  time.Duration              	// Maximum round trip time for this to complete
	timer    *time.Timer                	// Timer to fire when the RTT timeout expires
	peer     *peerConnection            	// Peer that we're requesting from
	response []*types.EpochGenesis                   	// Response data of the peer (nil for timeouts)
	dropped  bool                       	// Flag whether the peer dropped off early
}

// timedOut returns if this request timed out.
func (req *epochGenesisReq) timedOut() bool {
	return req.response == nil
}

type epochGenesisSyncStats struct {
	processed  uint64 // Number of state entries processed
	duplicate  uint64 // Number of state entries downloaded twice
	unexpected uint64 // Number of non-requested state entries received
	pending    uint64 // Number of still pending state entries
}


func (d *Downloader) syncEpochGenesis(epochid uint64) *epochGenesisSync {

	s := newepochGenesisSync(d, epochid)

	select {

	case d.epochGenesisSyncStart <- s:

	case <-d.quitCh:
		s.err = errCancelEpochGenesisFetch
		close(s.done)
	}

	return s
}

func (d *Downloader) epochGenesisFetcher() {
	for {
		select {
		case s := <-d.epochGenesisSyncStart:
			for next := s; next != nil; {
				next = d.runEpochGenesisSync(next)
			}
		case <-d.epochGenesisCh:

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

		case next := <-d.epochGenesisSyncStart:
			return next

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
				//case timeout <- req:
				case <-s.done:

				}
			})
			//active[req.peer.id] = req
		}
	}
}


type epochGenesisSync struct {
	epochid	uint64
	d *Downloader 							// Downloader instance to access and manage current peerset
	keccak hash.Hash                  		// Keccak256 hasher to verify deliveries with
	tasks  map[*big.Int]*epochGenesisTask 	// Set of tasks currently queued for retrieval

	numUncommitted   int
	bytesUncommitted int

	deliver    chan *epochGenesisReq // Delivery channel multiplexing peer responses

	cancel     chan struct{}  // Channel to signal a termination request
	cancelOnce sync.Once      // Ensures cancel only ever gets called once
	done       chan struct{}  // Channel to signal termination completion
	err        error          // Any error hit during sync (set before completion)
}


type epochGenesisTask struct {
	attempts map[string]struct{}
}


func newepochGenesisSync(d *Downloader, epochid uint64) *epochGenesisSync {
	return &epochGenesisSync{
		epochid: epochid,
		d:       d,
		keccak:  sha3.NewKeccak256(),
		tasks:   make(map[*big.Int]*epochGenesisTask),
		deliver: make(chan *epochGenesisReq),
		cancel:  make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// run starts the task assignment and response processing loop, blocking until
// it finishes, and finally notifying any goroutines waiting for the loop to
// finish.
func (s *epochGenesisSync) run() {
	s.err = s.loop()
	close(s.done)
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


func (s *epochGenesisSync) loop() error {
	// Listen for new peer events to assign tasks to them
	newPeer := make(chan *peerConnection, 1024)
	peerSub := s.d.peers.SubscribeNewPeers(newPeer)
	defer peerSub.Unsubscribe()

	// {

		peers, _ := s.d.peers.NodeDataIdlePeers()

		if len(peers) == 0 {
			return errors.New("failed found peer when send epoch genesis request")
		}

		p := peers[0]

		req := &epochGenesisReq{epochid:big.NewInt(int64(s.epochid)),peer: p, timeout: s.d.requestTTL()}

		select {

			case s.d.trackEpochGenesisReq <- req:
				req.peer.FetchEpochGenesisData(s.epochid)
			case <-s.cancel:
		}


		select {
		case <-newPeer:
			// New peer arrived, try to assign it download tasks

		case <-s.cancel:
			return errCancelEpochGenesisFetch

		case req := <-s.deliver:
			stale, err := s.process(req)
			if err != nil {
				log.Warn("Node data write error", "err", err)
				return err
			}

			if !stale {
				req.peer.SetNodeDataIdle(len(req.response))
			}
		}
	//}

	return nil
}


func (s *epochGenesisSync) process(req *epochGenesisReq) (bool, error) {
	// Collect processing stats and update progress if valid data was received
	duplicate, unexpected := 0, 0

	defer func(start time.Time) {
		if duplicate > 0 || unexpected > 0 {
			s.updateStats(0, duplicate, unexpected, time.Since(start))
		}
	}(time.Now())

	// Iterate over all the delivered data and inject one-by-one into the trie
	progress, stale := false, len(req.response) > 0

	for _, blob := range req.response {
		prog, hash, err := s.processNodeData(blob)
		switch err {
		case nil:
			s.numUncommitted++
			s.bytesUncommitted += 1//len(blob)
			progress = progress || prog
		case trie.ErrNotRequested:
			unexpected++
		case trie.ErrAlreadyProcessed:
			duplicate++
		default:
			return stale, fmt.Errorf("invalid epoch node %s: %v", hash.String(), err)
		}
		// If the node delivered a requested item, mark the delivery non-stale
		if _, ok := req.tasks[hash]; ok {
			delete(req.tasks, hash)
			stale = false
		}
	}

	// If we're inside the critical section, reset fail counter since we progressed.
	if progress && atomic.LoadUint32(&s.d.fsPivotFails) > 1 {
		log.Trace("Fast-sync progressed, resetting fail counter", "previous", atomic.LoadUint32(&s.d.fsPivotFails))
		atomic.StoreUint32(&s.d.fsPivotFails, 1) // Don't ever reset to 0, as that will unlock the pivot block
	}

	// Put unfulfilled tasks back into the retry queue
	npeers := s.d.peers.Len()
	for epid, task := range req.tasks {
		// If the node did deliver something, missing items may be due to a protocol
		// limit or a previous timeout + delayed delivery. Both cases should permit
		// the node to retry the missing items (to avoid single-peer stalls).
		if len(req.response) > 0 || req.timedOut() {
			delete(task.attempts, req.peer.id)
		}
		// If we've requested the node too many times already, it may be a malicious
		// sync where nobody has the right data. Abort.
		if len(task.attempts) >= npeers {
			return stale, fmt.Errorf("epoch node %s failed with all peers (%d tries, %d peers)", len(task.attempts), npeers)
		}
		// Missing item, place into the retry queue.
		s.tasks[epid] = task
	}
	return stale, nil
}


func (s *epochGenesisSync) processNodeData(genesis *types.EpochGenesis) (bool, *big.Int, error) {

	return true,nil,nil
}



func (s *epochGenesisSync) updateStats(written, duplicate, unexpected int, duration time.Duration) {

}
