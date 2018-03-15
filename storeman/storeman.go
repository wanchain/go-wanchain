package storeman

import (
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/rpc"
	"sync"
	"fmt"
)


type Storeman struct {
	protocol p2p.Protocol
	peers  map[*Peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set
}
type Config struct {
}
var DefaultConfig = Config{
}

const (
	ProtocolName = "storeman"
	ProtocolVersion = 1
	ProtocolVersionStr = "1.0"
	NumberOfMessageCodes = 3

)


// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (sm *Storeman) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		return nil
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
		return err
	}
	storemanPeer.start()
	defer storemanPeer.stop()

	return sm.runMessageLoop(storemanPeer, rw)
}

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New(cfg *Config) *Storeman {
	storeman := &Storeman{
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
