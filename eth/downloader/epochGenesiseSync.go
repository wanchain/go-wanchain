
package downloader

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"time"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
	"math/rand"
	"errors"
)

const repeatLimit  = 10

type epochGenesisReq struct {
	epochid  *big.Int              			// epochid items to download
	timeout  time.Duration              	// Maximum round trip time for this to complete
	timer    *time.Timer                	// Timer to fire when the RTT timeout expires
	peer     *peerConnection            	// Peer that we're requesting from
}

func (d *Downloader) fetchEpochGenesises(origin uint64, originHash *common.Hash,  latest *types.Header) error {
	localBlk := d.blockchain.GetBlockByHash(*originHash)
	beginEpid,_:= d.blockchain.GetBlockEpochIdAndSlotId(localBlk)


	endblk := types.NewBlockWithHeader(latest)
	endEpid,_:= d.blockchain.GetBlockEpochIdAndSlotId(endblk)

	return d.fetchEpochGenesisesBetween(beginEpid, endEpid - 1)
}

func (d *Downloader) fetchEpochGenesisesBetween(startEpochid uint64,endEpochid uint64) (error) {

	if d.epochGenesisFbCh != nil {
		return nil
	}

	if startEpochid == 0 {
		startEpochid = 1
	}

	fbchan  := make(chan int64,1)
	d.epochGenesisFbCh = fbchan

	for i := startEpochid;i < endEpochid;i++ {

		if i==0 || d.blockchain.IsExistEpochGenesis(i) {
			continue
		}

		d.epochGenesisSyncStart <- i

		select {
		case epid := <- fbchan:
			if epid >= 0 {
				log.Info("got epoch data", "", epid)
			} else {
				log.Info("failed to get epoch data", "", epid)
				return errors.New("failed to get epoch data")
			}
		}
	}

	d.epochGenesisFbCh = nil

	return nil
}

func (d *Downloader) epochGenesisFetcher() {

	var (
		active   = make(map[string]*epochGenesisReq) // Currently in-flight requests
		timeout  = make(chan *epochGenesisReq)       // Timed out active requests

		repeatCount   = make(map[uint64]uint64)
	)

	peerDrop := make(chan *peerConnection, 1024)
	peerSub := d.peers.SubscribePeerDrops(peerDrop)
	defer peerSub.Unsubscribe()

	for {

		select {

			case epochid := <-d.epochGenesisSyncStart:

				log.Debug("****fetching", "epochId", epochid)
				if repeatCount[epochid] > repeatLimit {

					if d.epochGenesisFbCh != nil {
						d.epochGenesisFbCh <- int64(-1)
					}

					continue
				}

				repeatCount[epochid] = repeatCount[epochid] + 1

				req := d.sendEpochGenesisReq(epochid,active)
				// Start a timer to notify the sync loop if the peer stalled.
				req.timer = time.AfterFunc(req.timeout, func() {
					select {
						case timeout <- req:
					}
				})



			case pack := <-d.epochGenesisCh:

				req := active[pack.PeerId()]
				if req == nil {
					log.Debug("Unrequested epoch genesis data", "peer", pack.PeerId(), "len", pack.Items())
					continue
				}

				response := pack.(*epochGenesisPack).epochGenesis
				log.Info("got epoch genesis data", "peer", pack.PeerId(), "epochid", response.EpochId)

				err := d.blockchain.SetEpochGenesis(response)

				if err != nil {
					log.Debug("epoch genesis data error,try again", "peer", pack.PeerId(), "len", pack.Items())
					d.epochGenesisSyncStart <- req.epochid.Uint64()
				} else {
					if d.epochGenesisFbCh != nil {
						d.epochGenesisFbCh <- int64(response.EpochId)
					}
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

				d.epochGenesisSyncStart <- req.epochid.Uint64()
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

		}
	}
}


func (d *Downloader) sendEpochGenesisReq(epochid uint64,active map[string]*epochGenesisReq) *epochGenesisReq {

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

	if active != nil {
		active[req.peer.id] = req
	}

	return req
}
