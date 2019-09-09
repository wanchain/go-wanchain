package schnorr

import (
	"fmt"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	set "gopkg.in/fatih/set.v0"
	"log"
	"time"
)

// Creating new p2p server
func NewServer(name string, port int) (*p2p.Server, error) {
	pkey, err := crypto.GenerateKey()
	if err != nil {
		log.Printf("Generate private key failed with err: %v", err)
		return nil, err
	}

	cfg := p2p.Config{
		PrivateKey:      pkey,
		Name:            name,
		MaxPeers:        1,
		Protocols:       []p2p.Protocol{proto},
		EnableMsgEvents: true,
	}

	if port > 0 {
		cfg.ListenAddr = fmt.Sprintf(":%d", port)
	}
	srv := &p2p.Server{
		Config: cfg,
	}

	err = srv.Start()
	if err != nil {
		log.Printf("Start server failed with err: %v", err)
		return nil, err
	}

	return srv, nil
}

func ConnectToPeer(srv *p2p.Server, enode string) error {
	// Parsing the enode url
	node, err := discover.ParseNode(enode)
	if err != nil {
		log.Printf("Failed to parse enode url with err: %v", err)
		return err
	}

	// Connecting to the peer
	srv.AddPeer(node)

	return nil
}

func SubscribeToEvents(srv *p2p.Server, communicated chan<- bool) {
	// Subscribing to the peer events
	peerEvent := make(chan *p2p.PeerEvent)
	eventSub := srv.SubscribeEvents(peerEvent)

	for {
		select {
		case event := <-peerEvent:
			if event.Type == p2p.PeerEventTypeMsgRecv {
				log.Println("Received message received notification")
				communicated <- true
			}
		case <-eventSub.Err():
			log.Println("subscription closed")

			// Closing the channel so that server gets stopped since
			// there won't be any more events coming in
			close(communicated)
		}
	}
}

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
	//log.SyslogInfo("storeman peer start", "peer", p.ID().String())
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (p *Peer) update() {
	// Start the tickers for the updates
	keepalive := time.NewTicker(KeepaliveCycle * time.Second)

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
	//log.SyslogInfo("storeman peer stop", "peer", p.ID().String())
}

func (p *Peer) sendKeepalive() {
	//p2p.Send(p.ws, mpcprotocol.KeepaliveCode, StoremanKeepalive{version: 1, magic: keepaliveMagic, recipient: p.Peer.ID()})
}

func (p *Peer) sendKeepaliveOk() {
	//p2p.Send(p.ws, mpcprotocol.KeepaliveOkCode, StoremanKeepaliveOk{version: 1, magic: keepaliveMagic, status: 0})
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (p *Peer) handshake() error {
	// Send the handshake status message asynchronously
	//errc := make(chan error, 1)
	//go func() {
	////	errc <- p2p.Send(p.ws, mpcprotocol.StatusCode, mpcprotocol.ProtocolVersion)
	//}()
	//// Fetch the remote status packet and verify protocol match
	//packet, err := p.ws.ReadMsg()
	//if err != nil {
	////	log.SyslogErr("storeman peer read msg fail", "peer", p.ID().String(), "err", err.Error())
	//	return err
	//}
	//defer packet.Discard()
	//
	//log.SyslogInfo("storeman received handshake", "peer", p.ID().String(), "code", packet.Code)
	//if packet.Code != mpcprotocol.StatusCode {
	//	log.SyslogErr("storeman peer sent packet before status packet", "peer", p.ID().String(), "code", packet.Code)
	//	return fmt.Errorf("storman peer [%s] sent packet %x before status packet", p.ID().String(), packet.Code)
	//}
	//s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	//peerVersion, err := s.Uint()
	//if err != nil {
	//	log.SyslogErr("storman peer sent bad status message", "peer", p.ID().String(), "err", err)
	//	return fmt.Errorf("storman peer [%s] sent bad status message: %v", p.ID().String(), err)
	//}
	//if peerVersion != mpcprotocol.ProtocolVersion {
	//	log.SyslogErr("storman peer: protocol version dont mismatch", "peer", p.ID().String(), "actual version", peerVersion, "expect version", mpcprotocol.ProtocolVersion)
	//	return fmt.Errorf("storman peer [%s]: protocol version mismatch %d != %d", p.ID().String(), peerVersion, mpcprotocol.ProtocolVersion)
	//}
	//// Wait until out own status is consumed too
	//if err := <-errc; err != nil {
	//	log.SyslogErr("storman peer failed to send status packet", "peer", p.ID().String(), "err", err)
	//	return fmt.Errorf("storman peer [%s] failed to send status packet: %v", p.ID().String(), err)
	//}
	return nil
}

func (p *Peer) ID() discover.NodeID {
	id := p.Peer.ID()
	return id
}
