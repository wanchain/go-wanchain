package storeman

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"sync"
	"fmt"
)


type Storeman struct {
	protocol p2p.Protocol
	peers  map[*Peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set
	quit         chan struct{}  // Channel used for graceful exit
}
type Config struct {
}
var DefaultConfig = Config{
}

const (
	ProtocolName = "storeman"
	ProtocolVersion = uint64(1)
	ProtocolVersionStr = "1.0"
	NumberOfMessageCodes = 3

	statusCode           = 0 // used by storeman protocol
	messagesCode         = 1 // normal whisper message
	keepaliveCode		=2
	keepaliveOkCode		=3
	txAuthenCode		=5
	txAuthenResultCode	=6

	keepaliveCycle = 5

)

type StoremanKeepalive struct {
	version 	int
	magic 		int
	recipient 	discover.NodeID
}

type StoremanKeepaliveOk struct {
	version 	int
	magic 		int
	status	 	int
}
const keepaliveMagic = 0x33

// MaxMessageSize returns the maximum accepted message size.
func (sm *Storeman) MaxMessageSize() uint32 {
	// TODO what is the max size of storeman???
	return uint32(1024)
}
// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (sm *Storeman) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			log.Warn("Storeman message loop", "peer", p.peer.ID(), "err", err)
			return err
		}
		if packet.Size > sm.MaxMessageSize() {
			log.Warn("oversized message received", "peer", p.peer.ID())
			return errors.New("oversized message received")
		}

		switch packet.Code {
		case statusCode:
			// this should not happen, but no need to panic; just ignore this message.
			log.Warn("unxepected status message received", "peer", p.peer.ID())
		case keepaliveCode:
			p.sendKeepaliveOk()

		default:
			// New message types might be implemented in the future versions of Whisper.
			// For forward compatibility, just ignore.
		}

		packet.Discard()
	}
}
type StoremanAPI struct{}
func (sa *StoremanAPI)Version()(v string){
	return ProtocolVersionStr
}
// APIs returns the RPC descriptors the Whisper implementation offers
func (sm *Storeman) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   &StoremanAPI{},
			Public:    true,
		},
	}
}
// Protocols returns the whisper sub-protocols ran by this particular client.
func (sm *Storeman) Protocols() []p2p.Protocol {
	return []p2p.Protocol{sm.protocol}
}

// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (sm *Storeman) Start(*p2p.Server) error {
	fmt.Println("storeman start...")
	log.Info("storeman start...")
	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (sm *Storeman) Stop() error {
	//close(sm.quit)
	//log.Info("whisper stopped")
	return nil
}

// HandlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (sm *Storeman) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create the new peer and start tracking it
	storemanPeer := newPeer(sm, peer, rw)

	sm.peerMu.Lock()
	sm.peers[storemanPeer] = struct{}{}
	sm.peerMu.Unlock()

	defer func() {
		sm.peerMu.Lock()
		delete(sm.peers, storemanPeer)
		sm.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := storemanPeer.handshake(); err != nil {
		fmt.Println("storemanPeer.handshake failed: ", err)
		return err
	}
	storemanPeer.start()
	defer storemanPeer.stop()

	return sm.runMessageLoop(storemanPeer, rw)
}

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New(cfg *Config) *Storeman {
	storeman := &Storeman{
		peers:        make(map[*Peer]struct{}),
		quit:         make(chan struct{}),

	}

	// p2p storeman sub protocol handler
	storeman.protocol = p2p.Protocol{
		Name:    ProtocolName,
		Version: uint(ProtocolVersion),
		Length:  NumberOfMessageCodes,
		Run:     storeman.HandlePeer,
	}

	return storeman
}
