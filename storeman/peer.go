package storeman

import (
	"fmt"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"github.com/wanchain/go-wanchain/rlp"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	set "gopkg.in/fatih/set.v0"
	"time"
)

// peer represents a whisper protocol peer connection.
type Peer struct {
	host    *Storeman
	Peer    *p2p.Peer
	ws      p2p.MsgReadWriter
	trusted bool

	known *set.Set // Messages already known by the peer to avoid wasting bandwidth

	quit chan struct{}
}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
func newPeer(host *Storeman, remote *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	return &Peer{
		host:    host,
		Peer:    remote,
		ws:      rw,
		trusted: false,
		known:   set.New(),
		quit:    make(chan struct{}),
	}
}

// start initiates the peer updater, periodically broadcasting the whisper packets
// into the network.
func (p *Peer) start() {
	log.SyslogInfo("storeman peer start", "peer", p.ID().String())
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (p *Peer) update() {
	// Start the tickers for the updates
	keepalive := time.NewTicker(mpcprotocol.KeepaliveCycle * time.Second)

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
	log.SyslogInfo("storeman peer stop", "peer", p.ID().String())
}

func (p *Peer) sendKeepalive() {
	p2p.Send(p.ws, mpcprotocol.KeepaliveCode, StrmanKeepAlive{version: 1, magic: keepaliveMagic, recipient: p.Peer.ID()})
}

func (p *Peer) sendKeepaliveOk() {
	p2p.Send(p.ws, mpcprotocol.KeepaliveOkCode, StrmanKeepAliveOk{version: 1, magic: keepaliveMagic, status: 0})
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (p *Peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.ws, mpcprotocol.StatusCode, mpcprotocol.PVer)
	}()
	// Fetch the remote status packet and verify protocol match
	packet, err := p.ws.ReadMsg()
	if err != nil {
		log.SyslogErr("storeman peer read msg fail", "peer", p.ID().String(), "err", err.Error())
		return err
	}
	defer packet.Discard()

	log.SyslogInfo("storeman received handshake", "peer", p.ID().String(), "code", packet.Code)
	if packet.Code != mpcprotocol.StatusCode {
		log.SyslogErr("storeman peer sent packet before status packet", "peer", p.ID().String(), "code", packet.Code)
		return fmt.Errorf("storeman peer [%s] sent packet %x before status packet", p.ID().String(), packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	peerVersion, err := s.Uint()
	if err != nil {
		log.SyslogErr("storeman peer sent bad status message", "peer", p.ID().String(), "err", err)
		return fmt.Errorf("storeman peer [%s] sent bad status message: %v", p.ID().String(), err)
	}
	if peerVersion != mpcprotocol.PVer {
		log.SyslogErr("storeman peer: protocol version not mismatch",
			"peer", p.ID().String(),
			"actual version", peerVersion,
			"expect version", mpcprotocol.PVer)

		return fmt.Errorf("storeman peer [%s]: protocol version mismatch %d != %d",
			p.ID().String(),
			peerVersion,
			mpcprotocol.PVer)

	}
	// Wait until out own status is consumed too
	if err := <-errc; err != nil {
		log.SyslogErr("storeman peer failed to send status packet", "peer", p.ID().String(), "err", err)
		return fmt.Errorf("storeman peer [%s] failed to send status packet: %v", p.ID().String(), err)
	}
	return nil
}

func (p *Peer) ID() discover.NodeID {
	id := p.Peer.ID()
	return id
}
