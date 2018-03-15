package storeman

import (
	"gopkg.in/fatih/set.v0"
	"github.com/wanchain/go-wanchain/p2p"

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
	//go p.update()
	//log.Trace("start", "peer", p.ID())
}

// stop terminates the peer updater, stopping message forwarding to it.
func (p *Peer) stop() {
	//close(p.quit)
	//log.Trace("stop", "peer", p.ID())
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (p *Peer) handshake() error {
	// Send the handshake status message asynchronously
	//errc := make(chan error, 1)
	//go func() {
	//	errc <- p2p.Send(p.ws, statusCode, ProtocolVersion)
	//}()
	//// Fetch the remote status packet and verify protocol match
	//packet, err := p.ws.ReadMsg()
	//if err != nil {
	//	return err
	//}
	//if packet.Code != statusCode {
	//	return fmt.Errorf("peer [%x] sent packet %x before status packet", p.ID(), packet.Code)
	//}
	//s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	//peerVersion, err := s.Uint()
	//if err != nil {
	//	return fmt.Errorf("peer [%x] sent bad status message: %v", p.ID(), err)
	//}
	//if peerVersion != ProtocolVersion {
	//	return fmt.Errorf("peer [%x]: protocol version mismatch %d != %d", p.ID(), peerVersion, ProtocolVersion)
	//}
	//// Wait until out own status is consumed too
	//if err := <-errc; err != nil {
	//	return fmt.Errorf("peer [%x] failed to send status packet: %v", p.ID(), err)
	//}
	return nil
}

