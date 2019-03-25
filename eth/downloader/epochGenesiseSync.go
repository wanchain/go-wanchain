
package downloader

import (
	"time"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
	"math/rand"
)


type epochGenesisReq struct {
	epochid  *big.Int              			// epochid items to download
//	tasks    map[*big.Int]*epochGenesisReq 			// Download tasks to track previous attempts
	timeout  time.Duration              	// Maximum round trip time for this to complete
	timer    *time.Timer                	// Timer to fire when the RTT timeout expires
	peer     *peerConnection            	// Peer that we're requesting from
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
				}
				// Finalize the request and queue up for processing
				req.timer.Stop()
				req.peer.SetEpochGenesisDataIdle(1)

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

				delete(active, req.peer.id)
				req.peer.SetEpochGenesisDataIdle(1)

				// Handle timed-out requests:
			case req := <-timeout:
				// If the peer is already requesting something else, ignore the stale timeout.
				// This can happen when the timeout and the delivery happens simultaneously,
				// causing both pathways to trigger.
				if active[req.peer.id] != req {
					continue
				}

				delete(active, req.peer.id)
				req.peer.SetEpochGenesisDataIdle(1)
				d.epochGenesisSyncStart <- req.epochid.Uint64()

			case <-d.quitCh:
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
