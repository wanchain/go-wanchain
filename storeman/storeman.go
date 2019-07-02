package storeman

import (
	"context"
	"math/big"
	"path/filepath"
	"sync"

	"os"

	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"github.com/wanchain/go-wanchain/storeman/validator"
	"github.com/wanchain/go-wanchain/storeman/btc"
)

type Storeman struct {
	protocol       p2p.Protocol
	peers          map[discover.NodeID]*Peer
	storemanPeers  map[discover.NodeID]bool
	peerMu         sync.RWMutex  // Mutex to sync the active peer set
	quit           chan struct{} // Channel used for graceful exit
	mpcDistributor *storemanmpc.MpcDistributor
	cfg            *Config
}

type Config struct {
	StoremanNodes []*discover.Node
	Password      string
	DataPath      string
}

var DefaultConfig = Config{
	StoremanNodes: make([]*discover.Node, 0),
}

type StoremanKeepalive struct {
	version   int
	magic     int
	recipient discover.NodeID
}

type StoremanKeepaliveOk struct {
	version int
	magic   int
	status  int
}

const keepaliveMagic = 0x33

// MaxMessageSize returns the maximum accepted message size.
func (sm *Storeman) MaxMessageSize() uint32 {
	// TODO what is the max size of storeman???
	return uint32(1024 * 1024)
}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (sm *Storeman) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	mpcsyslog.Info("runMessageLoop begin")

	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			mpcsyslog.Err("runMessageLoop, peer:%s, err:%s", p.Peer.ID().String(), err.Error())
			return err
		}

		mpcsyslog.Info("runMessageLoop, received a msg, peer:%s, packet size:%d", p.Peer.ID().String(), packet.Size)
		if packet.Size > sm.MaxMessageSize() {
			mpcsyslog.Warning("runMessageLoop, oversized message received, peer:%s, packet size:%d", p.Peer.ID().String(), packet.Size)
		} else {
			err = sm.mpcDistributor.GetMessage(p.Peer.ID(), rw, &packet)
			if err != nil {
				mpcsyslog.Err("runMessageLoop, distributor handle msg fail, err:%s", err.Error())
			}
		}

		packet.Discard()
	}
}

type StoremanAPI struct {
	sm *Storeman
}

func (sa *StoremanAPI) Version(ctx context.Context) (v string) {
	return mpcprotocol.ProtocolVersionStr
}

func (sa *StoremanAPI) Peers(ctx context.Context) []*p2p.PeerInfo {
	var ps []*p2p.PeerInfo
	for _, p := range sa.sm.peers {
		ps = append(ps, p.Peer.Info())
	}

	return ps
}

func (sa *StoremanAPI) CreateMpcAccount(ctx context.Context, accType string) (common.Address, error) {
	mpcsyslog.Info("CreateMpcAccount begin, accType:%s", accType)

	if !mpcprotocol.CheckAccountType(accType) {
		return common.Address{}, mpcprotocol.ErrInvalidStmAccType
	}

	if len(sa.sm.peers) < len(sa.sm.storemanPeers)-1 {
		return common.Address{}, mpcprotocol.ErrTooLessStoreman
	}

	if len(sa.sm.storemanPeers) > 22 {
		return common.Address{}, mpcprotocol.ErrTooMoreStoreman
	}

	addr, err := sa.sm.mpcDistributor.CreateRequestStoremanAccount(accType)
	if err == nil {
		mpcsyslog.Info("CreateMpcAccount end, addr:%s", addr.String())
	} else {
		mpcsyslog.Err("CreateMpcAccount end, err:%s", err.Error())
	}

	return addr, err
}


func (sa *StoremanAPI) SignMpcTransaction(ctx context.Context, tx mpcprotocol.SendTxArgs) (hexutil.Bytes, error) {
	if tx.To == nil ||
		tx.Gas == nil ||
		tx.GasPrice == nil ||
		tx.Value == nil ||
		tx.Nonce == nil ||
		tx.ChainID == nil {
		return nil, mpcprotocol.ErrInvalidMpcTx
	}

	mpcsyslog.Info("SignMpcTransaction begin, txInfo:%s", tx.String())

	if len(sa.sm.peers) < mpcprotocol.MPCDegree*2 {
		return nil, mpcprotocol.ErrTooLessStoreman
	}

	trans := types.NewTransaction(uint64(*tx.Nonce), *tx.To, (*big.Int)(tx.Value), (*big.Int)(tx.Gas), (*big.Int)(tx.GasPrice), tx.Data)
	signed, err := sa.sm.mpcDistributor.CreateRequestMpcSign(trans, tx.From, tx.ChainType, tx.SignType, (*big.Int)(tx.ChainID))
	if err == nil {
		mpcsyslog.Info("SignMpcTransaction end, signed:%s", common.ToHex(signed))
	} else {
		mpcsyslog.Err("SignMpcTransaction end, err:%s", err.Error())
	}

	return signed, err
}

func (sa *StoremanAPI) SignMpcBtcTransaction(ctx context.Context, args btc.MsgTxArgs) ([]hexutil.Bytes, error) {
	mpcsyslog.Info("SignMpcBtcTransaction begin, txInfo:%s", args.String())

	if len(sa.sm.peers) < mpcprotocol.MPCDegree*2 {
		return nil, mpcprotocol.ErrTooLessStoreman
	}

	msgTx, err := btc.GetMsgTxFromMsgTxArgs(&args)
	if err != nil {
		return nil, err
	}

	if len(msgTx.TxIn) == 0 {
		mpcsyslog.Err("SignMpcBtcTransaction, invalid btc MsgTxArgs, doesn't have TxIn")
		return nil, errors.New("invalid btc MsgTxArgs, doesn't have TxIn")
	}

	signeds, err := sa.sm.mpcDistributor.CreateRequestBtcMpcSign(&args)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(signeds); i++ {
		mpcsyslog.Info("SignMpcBtcTransaction end, signed:%s", common.ToHex(signeds[i]))
	}

	return signeds, err
}


// APIs returns the RPC descriptors the Whisper implementation offers
//AddValidMpcTx stores raw data of cross chain transaction for MPC signing verification
func (sa *StoremanAPI) AddValidMpcTx(ctx context.Context, tx mpcprotocol.SendTxArgs) error {
	return validator.AddValidMpcTx(&tx)
}


func (sa *StoremanAPI) AddValidMpcBtcTx(ctx context.Context, args btc.MsgTxArgs) error {
	mpcsyslog.Info("AddValidMpcBtcTx, txInfo:%s", args.String())
	return validator.AddValidMpcBtcTx(&args)
}

// APIs returns the RPC descriptors the Whisper implementation offers
func (sm *Storeman) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: mpcprotocol.ProtocolName,
			Version:   mpcprotocol.ProtocolVersionStr,
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
		mpcsyslog.Err("peer not find. peer:%s", peerID.String())
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

	mpcsyslog.Info("handle new peer, remoteAddr:%s, peerID:%s", peer.RemoteAddr().String(), peer.ID().String())

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
		mpcsyslog.Err("storemanPeer.handshake failed. peerID:%s. err:%s", peer.ID().String(), err.Error())
		return err
	}

	storemanPeer.start()
	defer storemanPeer.stop()

	return sm.runMessageLoop(storemanPeer, rw)
}

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New(cfg *Config, accountManager *accounts.Manager, aKID, secretKey, region string) *Storeman {
	storeman := &Storeman{
		peers: make(map[discover.NodeID]*Peer),
		quit:  make(chan struct{}),
		cfg:   cfg,
	}

	storeman.mpcDistributor = storemanmpc.CreateMpcDistributor(accountManager, storeman, aKID, secretKey, region, cfg.Password)
	dataPath := filepath.Join(cfg.DataPath, "storeman", "data")
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dataPath, 0700); err != nil {
			mpcsyslog.Err("make Stroreman path fail. err:%s", err.Error())
		}
	}

	validator.NewDatabase(dataPath)
	// p2p storeman sub protocol handler
	storeman.protocol = p2p.Protocol{
		Name:    mpcprotocol.ProtocolName,
		Version: uint(mpcprotocol.ProtocolVersion),
		Length:  mpcprotocol.NumberOfMessageCodes,
		Run:     storeman.HandlePeer,
		NodeInfo: func() interface{} {
			return map[string]interface{}{
				"version": mpcprotocol.ProtocolVersionStr,
			}
		},
	}

	return storeman
}
