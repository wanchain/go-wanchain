
package downloader

import (
	"sync"
	"time"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
	"math/rand"
)


type epochGenesisReq struct {
	epochid  *big.Int              			// epochid items to download
	tasks    map[*big.Int]*epochGenesisReq 			// Download tasks to track previous attempts
	timeout  time.Duration              	// Maximum round trip time for this to complete
	timer    *time.Timer                	// Timer to fire when the RTT timeout expires
	peer     *peerConnection            	// Peer that we're requesting from
	dropped  bool                       	// Flag whether the peer dropped off early
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

		select {


		case pack := <-d.epochGenesisCh:

			req := active[pack.PeerId()]
			if req == nil {
				log.Debug("Unrequested epoch genesis data", "peer", pack.PeerId(), "len", pack.Items())
				continue
			}
			// Finalize the request and queue up for processing
			req.timer.Stop()
			response := pack.(*epochGenesisPack).epochGenesis

			err := d.blockchain.SetEpochGenesis(response)
			if err != nil {
				log.Warn("get epoch genesis data write error", "err", err)
				continue
			}

			delete(active, pack.PeerId())
			req.peer.SetEpochGenesisDataIdle(1)

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

			delete(active, p.id)

		// Handle timed-out requests:
		case req := <-timeout:
			// If the peer is already requesting something else, ignore the stale timeout.
			// This can happen when the timeout and the delivery happens simultaneously,
			// causing both pathways to trigger.
			if active[req.peer.id] != req {
				continue
			}

			delete(active, req.peer.id)

		case req := <-d.trackEpochGenesisReq:
			if old := active[req.peer.id]; old != nil {
				log.Warn("Busy peer assigned new state fetch", "peer", old.peer.id)

				// Make sure the previous one doesn't get siletly lost
				old.timer.Stop()
				old.dropped = true
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
	cancel     chan struct{}  				// Channel to signal a termination request
	cancelOnce sync.Once      				// Ensures cancel only ever gets called once
	done       chan struct{}  				// Channel to signal termination completion
	err        error          				// Any error hit during sync (set before completion)
}

func newepochGenesisSync(d *Downloader, epochid uint64) *epochGenesisSync {
	return &epochGenesisSync{
		epochid: epochid,
		d:       d,
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
			continue
		}

		idx := rand.Intn(len(peers))
		request.peer = peers[idx]

		select {

			case s.d.trackEpochGenesisReq <- req:
				err := req.peer.FetchEpochGenesisData(s.epochid)
				if err == nil {
					delete(req.tasks,big.NewInt(int64(s.epochid)))
				} else {
					continue
				}

			case <-s.cancel:
		}

		select {
			case <-newPeer:
				// New peer arrived, try to assign it download tasks

			case <-s.cancel:
				return errCancelEpochGenesisFetch
		}

	}

	return nil
}
