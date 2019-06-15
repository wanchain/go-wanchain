
package downloader

import (
	"errors"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/util"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

const repeatLimit  = 10

type epochGenesisReq struct {
	epochId  *big.Int              			// epochId items to download
	timeout  time.Duration              	// Maximum round trip time for this to complete
	timer    *time.Timer                	// Timer to fire when the RTT timeout expires
	peer     *peerConnection            	// Peer that we're requesting from
}

func (d *Downloader) fetchEpochGenesises(startEpoch uint64, latest *types.Header) error {
	endblk := types.NewBlockWithHeader(latest)
	endEpid, _:= util.CalEpSlbyTd(endblk.Difficulty().Uint64())

	return d.fetchEpochGenesisesBetween(startEpoch, endEpid )
}

func (d *Downloader) fetchEpochGenesisesBetween(startEpochid uint64,endEpochid uint64) (error) {
	if d.epochGenesisFbCh != nil {
		return nil
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
			case epochId := <-d.epochGenesisSyncStart:
				log.Info("****fetching", "epochId", epochId)
				if repeatCount[epochId] > repeatLimit {

					if d.epochGenesisFbCh != nil {
						d.epochGenesisFbCh <- int64(-1)
					}

					continue
				}

				repeatCount[epochId] = repeatCount[epochId] + 1

				req := d.sendEpochGenesisReq(epochId,active)
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
				log.Info("got epoch genesis data", "peer", pack.PeerId(), "epochId", response.EpochId)

				var err error = nil
				rt := d.blockchain.PreVerifyEpochGenesis(response, pack.(*epochGenesisPack).whiteHeader)
				if rt < 0 {
					err = errors.New("PreVerifyEpochGenesis failed rt=" + strconv.Itoa(int(rt)) + " epochId" + strconv.FormatUint(response.EpochId, 10))
				} else if rt == 0 {
					err = d.blockchain.SetEpochGenesis(response, pack.(*epochGenesisPack).whiteHeader)
				} else {
					err = errors.New("p")
				}

				if err != nil {
					log.Warn("epoch genesis data error,try again", "peer", pack.PeerId(), "len", pack.Items())
					d.epochGenesisSyncStart <- req.epochId.Uint64()
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

				d.epochGenesisSyncStart <- req.epochId.Uint64()
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
				d.epochGenesisSyncStart <- req.epochId.Uint64()

			case <-d.quitCh:
				return

		}
	}
}


func (d *Downloader) sendEpochGenesisReq(epochId uint64,active map[string]*epochGenesisReq) *epochGenesisReq {
	newPeer := make(chan *peerConnection, 1024)
	peerSub := d.peers.SubscribeNewPeers(newPeer)
	defer peerSub.Unsubscribe()

	req := &epochGenesisReq{epochId: big.NewInt(int64(epochId)), timeout: d.requestTTL()}
	for {

		peers, _ := d.peers.EpochGenesisIdlePeers()
		if len(peers) == 0 {
			continue
		}

		idx := rand.Intn(len(peers))
		req.peer = peers[idx]

		err := req.peer.FetchEpochGenesisData(epochId)
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
