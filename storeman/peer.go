package storeman

import (
	"fmt"
	//"time"

	//"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"github.com/wanchain/go-wanchain/rlp"
	set "gopkg.in/fatih/set.v0"
	"time"
)
// peer represents a whisper protocol peer connection.
type Peer struct {
	host    *Storeman
	peer    *p2p.Peer
	ws      p2p.MsgReadWriter
	trusted bool

	known *set.Set // Messages already known by the peer to avoid wasting bandwidth

	quit chan struct{}
}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
func newPeer(host *Storeman, remote *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	return &Peer{
		host:    host,
		peer:    remote,
		ws:      rw,
		trusted: false,
		known:   set.New(),
		quit:    make(chan struct{}),
	}
}
// start initiates the peer updater, periodically broadcasting the whisper packets
// into the network.
func (p *Peer) start() {
	go p.update()
	fmt.Println("storeman peer start...")
	log.Trace("start", "peer", p.ID())
}
// update executes periodic operations on the peer, including message transmission
// and expiration.
func (p *Peer) update() {
	// Start the tickers for the updates
	keepalive := time.NewTicker(keepaliveCycle *time.Second)

	// Loop and transmit until termination is requested
	for {
		select {
		case <-keepalive.C:
			p.sendKeepalive()


		case <-p.quit:
			return
		}
	}
}
// stop terminates the peer updater, stopping message forwarding to it.
func (p *Peer) stop() {
	close(p.quit)
	log.Trace("stop", "peer", p.ID())
}
func (p *Peer) sendKeepalive() {
	fmt.Println("sendKeepalive to ", p.peer.RemoteAddr(), " from ", p.peer.LocalAddr())
	p2p.Send(p.ws, keepaliveCode, StoremanKeepalive{version: 1, magic: keepaliveMagic, recipient: p.peer.ID()})
}
func (p *Peer) sendKeepaliveOk() {
	fmt.Println("sendKeepaliveOk to ", p.peer.RemoteAddr(), " from ", p.peer.LocalAddr())
	p2p.Send(p.ws, keepaliveOkCode, StoremanKeepaliveOk{version: 1, magic: keepaliveMagic, status: 0})
}
// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (p *Peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.ws, statusCode, ProtocolVersion)
	}()
	// Fetch the remote status packet and verify protocol match
	packet, err := p.ws.ReadMsg()
	if err != nil {
		fmt.Println("p.ws.ReadMsg: ", err)
		return err
	}
	fmt.Println("Received handshake, packet.Code is ", packet.Code)
	if packet.Code != statusCode {
		return fmt.Errorf("peer [%x] sent packet %x before status packet", p.ID(), packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("peer [%x] sent bad status message: %v", p.ID(), err)
	}
	if peerVersion != ProtocolVersion {
		return fmt.Errorf("peer [%x]: protocol version mismatch %d != %d", p.ID(), peerVersion, ProtocolVersion)
	}
	// Wait until out own status is consumed too
	if err := <-errc; err != nil {
		return fmt.Errorf("peer [%x] failed to send status packet: %v", p.ID(), err)
	}
	return nil
}
func (p *Peer) ID() discover.NodeID {
	id := p.peer.ID()
	return id
}

