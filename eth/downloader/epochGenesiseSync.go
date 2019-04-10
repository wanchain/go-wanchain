
package downloader

import (
	"time"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
	"math/rand"
	"sync"
)


type epochGenesisReq struct {
	epochid  *big.Int              			// epochid items to download
//	tasks    map[*big.Int]*epochGenesisReq 			// Download tasks to track previous attempts
	timeout  time.Duration              	// Maximum round trip time for this to complete
	timer    *time.Timer                	// Timer to fire when the RTT timeout expires
	peer     *peerConnection            	// Peer that we're requesting from
}

// fetchHeight retrieves the head header of the remote peer to aid in estimating
// the total time a pending synchronisation would take.
func (d *Downloader) fetchEpochGenesises(startEpochid uint64,endEpochid uint64) (error) {

	var pend sync.WaitGroup
	fbchan  := make(chan uint64)

	d.epochGenesisFbCh = fbchan

	for i := startEpochid;i <= endEpochid;i++ {
		pend.Add(1)
		d.epochGenesisSyncStart <- i
	}

	for {
		select {
			case <- fbchan:
				pend.Done()

		}
	}

	pend.Wait()

	return nil
}

func (d *Downloader) epochGenesisFetcher() {

	var (
		active   = make(map[string]*epochGenesisReq) // Currently in-flight requests
		timeout  = make(chan *epochGenesisReq)       // Timed out active requests
	)

	peerDrop := make(chan *peerConnection, 1024)
	peerSub := d.peers.SubscribePeerDrops(peerDrop)
	defer peerSub.Unsubscribe()

	for {

		select {

			case epochid := <-d.epochGenesisSyncStart:
				req :=d.sendEpochGenesisReq(epochid,active,timeout)
				// Start a timer to notify the sync loop if the peer stalled.
				req.timer = time.AfterFunc(req.timeout, func() {
					select {
						case timeout <- req:
					}
				})

			case pack := <-d.epochGenesisCh:


				response := pack.(*epochGenesisPack).epochGenesis

				err := d.blockchain.SetEpochGenesis(response)
				log.Debug("got epoch genesis data", "peer", pack.PeerId(), "epochid", response.EpochId)

				if err != nil {
					log.Warn("get epoch genesis data write error", "err", err)
					continue
				}

				req := active[pack.PeerId()]
				if req == nil {
					log.Debug("Unrequested epoch genesis data", "peer", pack.PeerId(), "len", pack.Items())
					continue
				}
				// Finalize the request and queue up for processing
				req.timer.Stop()
				req.peer.SetEpochGenesisDataIdle(1)

				delete(active, pack.PeerId())

				if d.epochGenesisFbCh != nil {
					d.epochGenesisFbCh <- response.EpochId
				}

				// Handle dropped peer connections:
			case p := <-peerDrop:
				// Skip if no request is currently pending
				req := active[p.id]
				if req == nil {
					continue
				}
				// Finalize the request and queue up for processing
				req.timer.Stop()

				delete(active, req.peer.id)
				req.peer.SetEpochGenesisDataIdle(1)

				// Handle timed-out requests:
			case req := <-timeout:
				// If the peer is already requesting something else, ignore the stale timeout.
				// This can happen when the timeout and the delivery happens simultaneously,
				// causing both pathways to trigger.
				if active[req.peer.id] != req || req==nil {
					continue
				}

				delete(active, req.peer.id)
				req.peer.SetEpochGenesisDataIdle(1)
				d.epochGenesisSyncStart <- req.epochid.Uint64()

			case <-d.quitCh:
				return
			case <-d.cancelCh:
				return

		}
	}
}


func (d *Downloader) sendEpochGenesisReq(epochid uint64,active map[string]*epochGenesisReq,timeout chan *epochGenesisReq) *epochGenesisReq {

	newPeer := make(chan *peerConnection, 1024)
	peerSub := d.peers.SubscribeNewPeers(newPeer)
	defer peerSub.Unsubscribe()

	req := &epochGenesisReq{epochid: big.NewInt(int64(epochid)), timeout: d.requestTTL()}
	for {

		peers, _ := d.peers.EpochGenesisIdlePeers()
		if len(peers) == 0 {
			continue
		}

		idx := rand.Intn(len(peers))
		req.peer = peers[idx]

		err := req.peer.FetchEpochGenesisData(epochid)
		if err != nil {
			continue
		} else {
			break
		}
	}

	active[req.peer.id] = req

	return req
}
