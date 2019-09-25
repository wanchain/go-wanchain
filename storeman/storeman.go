package storeman

import (
	"context"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	"path/filepath"
	"sync"

	"os"

	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/storeman/validator"
)

type Config struct {
	StoremanNodes []*discover.Node
	Password      string
	DataPath      string
}

var DefaultConfig = Config{
	StoremanNodes: make([]*discover.Node, 0),
}

type StrmanKeepAlive struct {
	version   int
	magic     int
	recipient discover.NodeID
}

type StrmanKeepAliveOk struct {
	version int
	magic   int
	status  int
}

const keepaliveMagic = 0x33

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New(cfg *Config, accountManager *accounts.Manager, aKID, secretKey, region string) *Storeman {
	storeman := &Storeman{
		peers: make(map[discover.NodeID]*Peer),
		quit:  make(chan struct{}),
		cfg:   cfg,
	}

	storeman.mpcDistributor = storemanmpc.CreateMpcDistributor(accountManager,
		storeman,
		aKID,
		secretKey,
		region,
		cfg.Password)

	dataPath := filepath.Join(cfg.DataPath, "storeman", "data")
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dataPath, 0700); err != nil {
			log.SyslogErr("make Storeman path fail", "err", err.Error())
		}
	}

	validator.NewDatabase(dataPath)
	// p2p storeman sub protocol handler
	storeman.protocol = p2p.Protocol{
		Name:    mpcprotocol.PName,
		Version: uint(mpcprotocol.PVer),
		Length:  mpcprotocol.NumberOfMessageCodes,
		Run:     storeman.HandlePeer,
		NodeInfo: func() interface{} {
			return map[string]interface{}{
				"version": mpcprotocol.PVerStr,
			}
		},
	}

	return storeman
}

////////////////////////////////////
// Storeman
////////////////////////////////////
type Storeman struct {
	protocol       p2p.Protocol
	peers          map[discover.NodeID]*Peer
	storemanPeers  map[discover.NodeID]bool
	peerMu         sync.RWMutex  // Mutex to sync the active peer set
	quit           chan struct{} // Channel used for graceful exit
	mpcDistributor *storemanmpc.MpcDistributor
	cfg            *Config
}

// MaxMessageSize returns the maximum accepted message size.
func (sm *Storeman) MaxMessageSize() uint32 {
	// TODO what is the max size of storeman???
	return uint32(1024 * 1024)
}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (sm *Storeman) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	log.SyslogInfo("runMessageLoop begin")

	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			log.SyslogErr("runMessageLoop", "peer", p.Peer.ID().String(), "err", err.Error())
			return err
		}

		log.SyslogInfo("runMessageLoop, received a msg", "peer", p.Peer.ID().String(), "packet size", packet.Size)
		if packet.Size > sm.MaxMessageSize() {
			log.SyslogWarning("runMessageLoop, oversized message received", "peer", p.Peer.ID().String(), "packet size", packet.Size)
		} else {
			err = sm.mpcDistributor.GetMessage(p.Peer.ID(), rw, &packet)
			if err != nil {
				log.SyslogErr("runMessageLoop, distributor handle msg fail", "err", err.Error())
			}
		}

		packet.Discard()
	}
}

// APIs returns the RPC descriptors the Whisper implementation offers
func (sm *Storeman) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: mpcprotocol.PName,
			Version:   mpcprotocol.PVerStr,
			Service:   &StoremanAPI{sm: sm},
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
func (sm *Storeman) Start(server *p2p.Server) error {
	sm.mpcDistributor.Self = server.Self()
	sm.mpcDistributor.StoreManGroup = make([]discover.NodeID, len(server.StoremanNodes))
	sm.storemanPeers = make(map[discover.NodeID]bool)
	for i, item := range server.StoremanNodes {
		sm.mpcDistributor.StoreManGroup[i] = item.ID
		sm.storemanPeers[item.ID] = true
	}
	sm.mpcDistributor.InitStoreManGroup()
	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (sm *Storeman) Stop() error {
	return nil
}

func (sm *Storeman) SendToPeer(peerID *discover.NodeID, msgcode uint64, data interface{}) error {
	sm.peerMu.RLock()
	defer sm.peerMu.RUnlock()
	peer, exist := sm.peers[*peerID]
	if exist {
		return p2p.Send(peer.ws, msgcode, data)
	} else {
		log.SyslogErr("peer not find", "peer", peerID.String())
	}
	return nil
}

func (sm *Storeman) IsActivePeer(peerID *discover.NodeID) bool {
	sm.peerMu.RLock()
	defer sm.peerMu.RUnlock()
	_, exist := sm.peers[*peerID]
	return exist
}

// HandlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (sm *Storeman) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	if _, exist := sm.storemanPeers[peer.ID()]; !exist {
		return errors.New("Peer is not in storemangroup")
	}

	log.SyslogInfo("handle new peer", "remoteAddr", peer.RemoteAddr().String(), "peerID", peer.ID().String())

	// Create the new peer and start tracking it
	storemanPeer := newPeer(sm, peer, rw)

	sm.peerMu.Lock()
	sm.peers[storemanPeer.ID()] = storemanPeer
	sm.peerMu.Unlock()

	defer func() {
		sm.peerMu.Lock()
		delete(sm.peers, storemanPeer.ID())
		sm.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := storemanPeer.handshake(); err != nil {
		log.SyslogErr("storemanPeer.handshake failed", "peerID", peer.ID().String(), "err", err.Error())
		return err
	}

	storemanPeer.start()
	defer storemanPeer.stop()

	return sm.runMessageLoop(storemanPeer, rw)
}

////////////////////////////////////
// StoremanAPI
////////////////////////////////////
type StoremanAPI struct {
	sm *Storeman
}

func (sa *StoremanAPI) Version(ctx context.Context) (v string) {
	return mpcprotocol.PVerStr
}

func (sa *StoremanAPI) Peers(ctx context.Context) []*p2p.PeerInfo {
	var ps []*p2p.PeerInfo
	for _, p := range sa.sm.peers {
		ps = append(ps, p.Peer.Info())
	}

	return ps
}

func (sa *StoremanAPI) CreateGPK(ctx context.Context) (pk hexutil.Bytes, err error) {

	log.SyslogInfo("CreateGPK begin")
	log.SyslogInfo("CreateGPK begin", "peers", len(sa.sm.peers), "storeman peers", len(sa.sm.storemanPeers))
	if len(sa.sm.peers) < len(sa.sm.storemanPeers)-1 {
		return []byte{}, mpcprotocol.ErrTooLessStoreman
	}

	if len(sa.sm.storemanPeers)+1 < mpcprotocol.MpcSchnrNodeNumber {
		return []byte{}, mpcprotocol.ErrTooLessStoreman
	}

	gpk, err := sa.sm.mpcDistributor.CreateRequestGPK()
	if err == nil {
		log.SyslogInfo("CreateGPK end", "gpk", gpk)
	} else {
		log.SyslogErr("CreateGPK end", "err", err.Error())
	}

	return gpk, err
}

//func (sa *StoremanAPI) SignData(ctx context.Context, data mpcprotocol.SendData) (R []byte, s []byte, err error) {
//	//Todo  check the input parameter
//
//	if len(sa.sm.peers) < mpcprotocol.MPCDegree*2 {
//		return []byte{}, []byte{}, mpcprotocol.ErrTooLessStoreman
//	}
//
//	PKBytes := data.PKBytes
//	pk := crypto.ToECDSAPub(PKBytes)
//	from := crypto.PubkeyToAddress(*pk)
//
//	signed, err := sa.sm.mpcDistributor.CreateReqMpcSign(data.Data, from)
//	if err == nil {
//		log.SyslogInfo("SignMpcTransaction end", "signed", common.ToHex(signed))
//	} else {
//		log.SyslogErr("SignMpcTransaction end", "err", err.Error())
//	}
//
//	return []byte{}, []byte{}, nil
//}

func (sa *StoremanAPI) SignData(ctx context.Context, data mpcprotocol.SendData) (result mpcprotocol.SignedResult, err error) {
	//Todo  check the input parameter

	if len(sa.sm.peers) < mpcprotocol.MPCDegree*2 {
		return mpcprotocol.SignedResult{R: []byte{}, S: []byte{}}, mpcprotocol.ErrTooLessStoreman
	}

	PKBytes := data.PKBytes
	pk := crypto.ToECDSAPub(PKBytes)
	from := crypto.PubkeyToAddress(*pk)

	signed, err := sa.sm.mpcDistributor.CreateReqMpcSign(data.Data, from)

	// signed   R // s
	if err == nil {
		log.SyslogInfo("SignMpcTransaction end", "signed", common.ToHex(signed))
	} else {
		log.SyslogErr("SignMpcTransaction end", "err", err.Error())
		return mpcprotocol.SignedResult{R: []byte{}, S: []byte{}}, err
	}

	return mpcprotocol.SignedResult{R: signed[0:66], S: signed[65:]}, nil
}

func (sa *StoremanAPI) AddValidData(ctx context.Context, data mpcprotocol.SendData) error {
	return validator.AddValidData(&data)
}

// non leader node polling the data received from leader node
//func (sa *StoremanAPI) GetDataForApprove(ctx context.Context) ([]mpcprotocol.SendData, error) {
//	return nil, nil
//}
//
//// non leader node ApproveData, and make sure that the data is really required to be signed by them.
//func (sa *StoremanAPI) ApproveData(ctx context.Context, data []mpcprotocol.SendData) []bool {
//	return []bool{true}
//}
