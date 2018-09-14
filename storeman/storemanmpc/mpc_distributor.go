package storemanmpc

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"math/big"
	"sort"
	"sync"
	"io/ioutil"
	"math/rand"
	"time"
	"strconv"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/kms"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"github.com/wanchain/go-wanchain/rlp"
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"github.com/wanchain/go-wanchain/storeman/validator"
	"github.com/wanchain/go-wanchain/storeman/btc"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
)

type MpcContextCreater interface {
	CreateContext(int, uint64, []mpcprotocol.PeerInfo, ...MpcValue) (MpcInterface, error) //createContext
}

type MpcValue struct {
	Key       string
	Value     []big.Int
	ByteValue []byte
}

type MpcInterface interface {
	getMessage(*discover.NodeID, *mpcprotocol.MpcMessage, *[]mpcprotocol.PeerInfo) error
	mainMPCProcess(manager mpcprotocol.StoremanManager) error
	getMpcResult() []byte
	quit(error)
}

type P2pMessager interface {
	SendToPeer(*discover.NodeID, uint64, interface{}) error
	IsActivePeer(*discover.NodeID) bool
}

type mpcAccount struct {
	address      common.Address
	privateShare big.Int
	peers        []mpcprotocol.PeerInfo
}

type KmsInfo struct {
	AKID      string
	SecretKey string
	Region    string
}

type MpcDistributor struct {
	mu             sync.RWMutex
	Self           *discover.Node
	StoreManGroup  []discover.NodeID
	storeManIndex  map[discover.NodeID]byte
	mpcCreater     MpcContextCreater
	mpcMap         map[uint64]MpcInterface
	AccountManager *accounts.Manager
	P2pMessager    P2pMessager
	accMu          sync.Mutex
	mpcAccountMap  map[common.Address]*mpcAccount
	enableAwsKms   bool
	kmsInfo        KmsInfo
	password       string
}

func CreateMpcDistributor(accountManager *accounts.Manager, msger P2pMessager, aKID, secretKey, region, password string) *MpcDistributor {
	kmsInfo := KmsInfo{aKID, secretKey, region}
	mpc := &MpcDistributor{
		mu:             sync.RWMutex{},
		mpcCreater:     &MpcCtxFactory{},
		mpcMap:         make(map[uint64]MpcInterface),
		AccountManager: accountManager,
		accMu:          sync.Mutex{},
		mpcAccountMap:  make(map[common.Address]*mpcAccount),
		kmsInfo:        kmsInfo,
		password:       password,
		P2pMessager:    msger,
	}

	mpc.enableAwsKms = (aKID != "") && (secretKey != "") && (region != "")

	return mpc
}

func (mpcServer *MpcDistributor) createMPCTxSigner(ChainType string, ChainID *big.Int) (mpccrypto.MPCTxSigner, error) {
	log.Info("create MPC tx signer", "ChainType", ChainType, "ChainID", ChainID.Int64())
	mpcsyslog.Info("create MPC tx signer. ChainType:%s, ChainID:%d", ChainType, ChainID.Int64())

	if ChainType == "WAN" {
		return mpccrypto.CreateWanMPCTxSigner(ChainID), nil
	} else if ChainType == "ETH" {
		return mpccrypto.CreateEthMPCTxSigner(ChainID), nil
	}

	return nil, mpcprotocol.ErrChainTypeError
}

func (mpcServer *MpcDistributor) GetMessage(PeerID discover.NodeID, rw p2p.MsgReadWriter, msg *p2p.Msg) error {
	switch msg.Code {
	case mpcprotocol.StatusCode:
		// this should not happen, but no need to panic; just ignore this message.
		log.Warn("unxepected status message received", "peer", PeerID)

	case mpcprotocol.KeepaliveCode:
		// this should not happen, but no need to panic; just ignore this message.
		//		log.Warn("Keepalive message received", "peer", PeerID)

	case mpcprotocol.KeepaliveOkCode:
		// this should not happen, but no need to panic; just ignore this message.
		//		log.Warn("KeepaliveOk message received", "peer", PeerID)

	case mpcprotocol.MPCError:
		var mpcMessage mpcprotocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			mpcsyslog.Err("MpcDistributor.GetMessage, rlp decode MPCError msg fail. err:%s", err.Error())
			return err
		}

		errText := string(mpcMessage.Peers[:])
		mpcsyslog.Err("MpcDistributor.GetMessage, MPCError message received. peer:%s, err:%s", PeerID.String(), errText)
		log.Error("Mpc error message received", "peer", PeerID, "error", errText)
		go mpcServer.QuitMpcContext(&mpcMessage)

	case mpcprotocol.RequestMPC:
		log.Debug("RequestMPC message received", "peer", PeerID)
		mpcsyslog.Debug("MpcDistributor.GetMessage, RequestMPC message received. peer:%s", PeerID.String())
		var mpcMessage mpcprotocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			mpcsyslog.Err("MpcDistributor.GetMessage, rlp decode RequestMPC msg fail. err:%s", err.Error())
			return err
		}

		//create context
		go func() {
			err := mpcServer.createMpcContext(&mpcMessage)

			if err != nil {
				mpcsyslog.Err("createMpcContext fail. err:%s", err.Error())
				log.Error("createMpcContext fail", "Error", err)
			}
		}()

	case mpcprotocol.MPCMessage:
		var mpcMessage mpcprotocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			mpcsyslog.Err("GetP2pMessage fail. err:%s", err.Error())
			log.Error("GetP2pMessage fail", "error", err)
			return err
		}

		mpcsyslog.Debug("MpcDistributor.GetMessage, MPCMessage message received. peer:%s", PeerID.String())
		go mpcServer.getMpcMessage(&PeerID, &mpcMessage)

	default:
		// New message types might be implemented in the future versions of Whisper.
		// For forward compatibility, just ignore.
	}

	return nil
}

func (mpcServer *MpcDistributor) InitStoreManGroup() {
	sort.Sort(mpcprotocol.SliceStoremanGroup(mpcServer.StoreManGroup))
	mpcServer.storeManIndex = make(map[discover.NodeID]byte)
	for i := 0; i < len(mpcServer.StoreManGroup); i++ {
		mpcServer.storeManIndex[mpcServer.StoreManGroup[i]] = byte(i)
	}
}

func GetPrivateShare(ks *keystore.KeyStore, address common.Address, enableKms bool, kmsInfo *KmsInfo, password string) (*keystore.Key, int, error) {
	account := accounts.Account{Address: address}
	account, err := ks.Find(account)
	if err != nil {
		mpcsyslog.Err("find account from keystore fail. addr:%s, err:%s", address.String(), err.Error())
		return nil, 0x00, err
	}

	var keyjson []byte
	if enableKms {
		keyjson, err = kms.DecryptFileToBuffer(account.URL.Path, kmsInfo.AKID, kmsInfo.SecretKey, kmsInfo.Region)
	} else {
		keyjson, err = ioutil.ReadFile(account.URL.Path)
	}

	if err != nil {
		mpcsyslog.Err("get account keyjson fail. addr:%s, path:%s, err:%s", address.String(), account.URL.Path, err.Error())
		return nil, 0x01, err
	}

	key, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		mpcsyslog.Err("decrypt account keyjson fail. addr:%s, path:%s, err:%s", address.String(), account.URL.Path, err.Error())
		return nil, 0x011, err
	}

	return key, 0x111, nil
}

func (mpcServer *MpcDistributor) loadStoremanAddress(address *common.Address) (*MpcValue, []mpcprotocol.PeerInfo, error) {
	mpcServer.accMu.Lock()
	defer mpcServer.accMu.Unlock()
	value, exist := mpcServer.mpcAccountMap[*address]
	if !exist {
		ks := mpcServer.AccountManager.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		key, _, err := GetPrivateShare(ks, *address, mpcServer.enableAwsKms, &mpcServer.kmsInfo, mpcServer.password)
		if err != nil {
			return nil, nil, err
		}

		b := make([]byte, 8)
		peers := make([]mpcprotocol.PeerInfo, len(mpcServer.StoreManGroup))
		for i := 0; i < len(mpcServer.StoreManGroup); i++ {
			copy(b[5:], key.WAddress[i*3:])
			seed := binary.BigEndian.Uint64(b)
			peers[i].PeerID = mpcServer.StoreManGroup[i]
			peers[i].Seed = seed
		}

		value = &mpcAccount{*address, *key.PrivateKey.D, peers}
		mpcServer.mpcAccountMap[*address] = value
	}

	return &MpcValue{mpcprotocol.MpcPrivateShare, []big.Int{value.privateShare}, nil}, value.peers, nil
}

func (mpcServer *MpcDistributor) SetMessagePeers(mpcMessage *mpcprotocol.MpcMessage, peers *[]mpcprotocol.PeerInfo) {
	if peers == nil || len(*peers) == 0 {
		return
	}

	mpcMessage.Peers = make([]byte, len(*peers)*4)
	for i, peer := range *peers {
		mpcMessage.Peers[i*4] = mpcServer.storeManIndex[peer.PeerID]
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, peer.Seed)
		copy(mpcMessage.Peers[i*4+1:], b[5:])
	}
}

func (mpcServer *MpcDistributor) getMessagePeers(mpcMessage *mpcprotocol.MpcMessage) *[]mpcprotocol.PeerInfo {
	peerLen := len(mpcMessage.Peers)
	if peerLen == 0 || peerLen%4 != 0 {
		return nil
	}

	peerLen = peerLen / 4
	peers := make([]mpcprotocol.PeerInfo, peerLen)
	b := make([]byte, 8)
	for i := 0; i < peerLen; i++ {
		peerIndex := int(mpcMessage.Peers[i*4])
		if peerIndex < len(mpcServer.StoreManGroup) {
			peers[i].PeerID = mpcServer.StoreManGroup[peerIndex]
			copy(b[5:], mpcMessage.Peers[i*4+1:])
			peers[i].Seed = binary.BigEndian.Uint64(b)
		}
	}

	return &peers
}

func (mpcServer *MpcDistributor) selectPeers(ctxType int, allPeers []mpcprotocol.PeerInfo, preSetValue ...MpcValue) []mpcprotocol.PeerInfo {
	var peers []mpcprotocol.PeerInfo
	if ctxType == mpcprotocol.MpcCreateLockAccountLeader {
		peers = allPeers
	} else {
		peers = make([]mpcprotocol.PeerInfo, mpcprotocol.MPCDegree*2+1)
		storemanLen := len(mpcServer.StoreManGroup)
		selectIndex := 0
		rand.Seed(time.Now().UnixNano())
		for _, item := range preSetValue {
			if item.Key == mpcprotocol.MpcTxHash {
				sel := big.NewInt(0)
				hash := item.Value[0]
				sel.Mod(&hash, big.NewInt(int64(storemanLen)))
				selectIndex = (int(sel.Uint64()) + rand.Int()) % storemanLen
				break
			}
		}

		j := 1
		for i := 0; j < len(peers) && i < len(allPeers); i++ {
			sel := (i + selectIndex) % storemanLen
			if mpcServer.P2pMessager.IsActivePeer(&(allPeers[sel].PeerID)) {
				peers[j] = allPeers[sel]
				mpcsyslog.Info("select peers. index:%d, peer:%s", j, peers[j].PeerID.String())
				log.Debug("select peers", "index", j, "peer id", peers[j].PeerID)
				j++
			}
		}

		index := int(mpcServer.storeManIndex[mpcServer.Self.ID])
		peers[0] = allPeers[index]
		mpcsyslog.Info("select peers. index:%d, peer:%s", 0, peers[0].PeerID.String())
		log.Debug("select peers", "index", 0, "peer id", peers[0].PeerID)
	}

	return peers
}

func (mpcServer *MpcDistributor) CreateRequestStoremanAccount(accType string) (common.Address, error) {
	mpcsyslog.Debug("CreateRequestStoremanAccount begin")
	log.Warn("-----------------CreateRequestStoremanAccount begin", "accType", accType)
	preSetValue := make([]MpcValue, 0, 1)
	preSetValue = append(preSetValue, MpcValue{Key:mpcprotocol.MpcStmAccType, ByteValue:[]byte(accType)})

	value, err := mpcServer.createRequestMpcContext(mpcprotocol.MpcCreateLockAccountLeader, preSetValue...)
	if err != nil {
		return common.Address{}, err
	} else {
		return common.BytesToAddress(value), err
	}
}

func (mpcServer *MpcDistributor) CreateRequestMpcSign(tx *types.Transaction, from common.Address, chainType string, SignType string, chianID *big.Int) (hexutil.Bytes, error) {
	mpcsyslog.Debug("CreateRequestMpcSign begin")
	if chianID == nil {
		mpcsyslog.Err("CreateRequestMpcSign fail. err:%s", mpcprotocol.ErrChainID.Error())
		return nil, mpcprotocol.ErrChainID
	}

	signer, err := mpcServer.createMPCTxSigner(chainType, chianID)
	if err != nil {
		mpcsyslog.Err("createMPCTxSigner fail. err:%s", err.Error())
		return nil, err
	}

	txHash := signer.Hash(tx)
	txbytes, err := rlp.EncodeToBytes(tx)
	if err != nil {
		mpcsyslog.Err("CreateRequestMpcSign. rlp EncodeToBytes fail. err:%s", err.Error())
		return nil, err
	}

	value, err := mpcServer.createRequestMpcContext(mpcprotocol.MpcTXSignLeader, MpcValue{mpcprotocol.MpcTxHash, []big.Int{*txHash.Big()}, nil},
		MpcValue{mpcprotocol.MpcAddress, []big.Int{*from.Big()}, nil}, MpcValue{mpcprotocol.MpcTransaction, nil, txbytes},
		MpcValue{mpcprotocol.MpcChainType, nil, []byte(chainType)}, MpcValue{mpcprotocol.MpcSignType, nil, []byte(SignType)},
		MpcValue{mpcprotocol.MpcChainID, []big.Int{*chianID}, nil})

	return value, err
}

func (mpcServer *MpcDistributor) CreateRequestBtcMpcSign(args *btc.MsgTxArgs) (hexutil.Bytes, error) {
	mpcsyslog.Debug("CreateRequestBtcMpcSign begin")

	txBytesData, err := rlp.EncodeToBytes(args)
	if err != nil {
		log.Error("CreateRequestBtcMpcSign, rlp encode tx fail.", "err", err)
		return nil, err
	}

	txHashes, err := btc.GetHashedForEachTxIn(args)
	if err != nil {
		return nil, err
	}

	preSetValues := []MpcValue{}
	for i := 0; i < len(args.TxIn); i++ {
		preSetValues = append(preSetValues, MpcValue{mpcprotocol.MpcTxHash + "_" + strconv.Itoa(i), []big.Int{*txHashes[i].Big()}, nil})
	}

	preSetValues = append(preSetValues, MpcValue{mpcprotocol.MpcAddress, []big.Int{*args.From.Big()}, nil})
	preSetValues = append(preSetValues, MpcValue{mpcprotocol.MpcTransaction, nil, txBytesData})
	preSetValues = append(preSetValues, MpcValue{mpcprotocol.MpcChainType, nil, []byte("BTC")})

	value, err := mpcServer.createRequestMpcContext(mpcprotocol.MpcTXSignLeader, preSetValues...)
	return value, err
}

//func (mpcServer *MpcDistributor) CreateRequestMpcSignHash(txHash common.Hash, from common.Address) (hexutil.Bytes, error) {
//	value, err := mpcServer.createRequestMpcContext(mpcprotocol.MpcTXSignLeader, MpcValue{mpcprotocol.MpcTxHash, []big.Int{*txHash.Big()}, nil},
//		MpcValue{mpcprotocol.MpcAddress, []big.Int{*from.Big()}, nil}, MpcValue{mpcprotocol.MpcSignType, nil, []byte("hash")})
//	log.Debug("mpc sign result", "tx raw data", common.ToHex(value))
//	return value, err
//}

func (mpcServer *MpcDistributor) createRequestMpcContext(ctxType int, preSetValue ...MpcValue) (hexutil.Bytes, error) {
	mpcsyslog.Debug("createRequestMpcContext begin")
	var mpcID uint64
	var err error
	for {
		mpcID, err = mpccrypto.UintRand(uint64(1<<64 - 1))
		if err != nil {
			mpcsyslog.Err("UnitRand fail. err:%s", err.Error())
			return nil, err
		}

		mpcServer.mu.RLock()
		_, exist := mpcServer.mpcMap[mpcID]
		mpcServer.mu.RUnlock()
		if !exist {
			break
		}
	}

	peers := []mpcprotocol.PeerInfo{}
	if ctxType == mpcprotocol.MpcTXSignLeader {
		address := common.Address{}
		for _, item := range preSetValue {
			if item.Key == mpcprotocol.MpcAddress {
				address = common.BigToAddress(&item.Value[0])
				break
			}
		}

		value, peers1, err := mpcServer.loadStoremanAddress(&address)
		if err != nil {
			mpcsyslog.Err("load storeman address fail. address:%s", address.String())
			return []byte{}, err
		}

		peers = peers1
		preSetValue = append(preSetValue, *value)
	} else {
		for i := 0; i < len(mpcServer.StoreManGroup); i++ {
			peers = append(peers, mpcprotocol.PeerInfo{PeerID:mpcServer.StoreManGroup[i], Seed:0})
		}
	}

	mpc, err := mpcServer.mpcCreater.CreateContext(ctxType, mpcID, mpcServer.selectPeers(ctxType, peers, preSetValue...), preSetValue...)
	if err != nil {
		mpcsyslog.Err("CreateContext fail. err:%s", err.Error())
		log.Error("CreateContext fail", "error", err)
		return []byte{}, err
	}

	mpcsyslog.Info("CreateRequestMpcContext. ctxType:%d, mpcID:%d", ctxType, mpcID)
	log.Info("CreateRequestMpcContext", "ctxType", ctxType, "ctxId", mpcID)
	mpcServer.addMpcContext(mpcID, mpc)
	defer mpcServer.removeMpcContext(mpcID)
	err = mpc.mainMPCProcess(mpcServer)
	if err != nil {
		mpcsyslog.Err("mainMPCProcess fail. err:%s", err.Error())
		log.Error("mainMPCProcess fail", "error", err)
		return []byte{}, err
	}

	result := mpc.getMpcResult()
	return result, nil
}

func (mpcServer *MpcDistributor) QuitMpcContext(msg *mpcprotocol.MpcMessage) {
	mpcServer.mu.RLock()
	mpc, exist := mpcServer.mpcMap[msg.ContextID]
	mpcServer.mu.RUnlock()
	if exist {
		mpc.quit(errors.New(string(msg.Peers[:])))
	}
}

func (mpcServer *MpcDistributor) createMpcContext(mpcMessage *mpcprotocol.MpcMessage, preSetValue ...MpcValue) error {
	log.Warn("-----------------createMpcContext begin");
	for _, byteData := range mpcMessage.BytesData {
		log.Warn("-----------------createMpcContext", "byteData", string(byteData[:]))
	}

	mpcServer.mu.RLock()
	_, exist := mpcServer.mpcMap[mpcMessage.ContextID]
	mpcServer.mu.RUnlock()
	if exist {
		mpcsyslog.Err("createMpcContext fail. err:%s", mpcprotocol.ErrMpcContextExist.Error())
		return mpcprotocol.ErrMpcContextExist
	}

	var ctxType int
	nType := mpcMessage.Data[0].Int64()
	if nType == mpcprotocol.MpcCreateLockAccountLeader {
		ctxType = mpcprotocol.MpcCreateLockAccountPeer
	} else {
		ctxType = mpcprotocol.MpcTXSignPeer
	}

	log.Debug("createMpcContext", "ctxType", ctxType, "ctxId", mpcMessage.ContextID)
	if ctxType == mpcprotocol.MpcTXSignPeer {
		signer, err := mpcServer.createMPCTxSigner(string(mpcMessage.BytesData[0]), &mpcMessage.Data[3])
		if err != nil {
			mpcsyslog.Err("createMPCTxSigner fail. err:%s", err.Error())
			log.Error("createMPCTxSigner fail", "error", err)
			return err
		}

		verifyResult := validator.ValidateTx(signer, mpcMessage.BytesData[1], mpcMessage.Data[1].Bytes())
		if !verifyResult {
			mpcMsg := &mpcprotocol.MpcMessage{ContextID: mpcMessage.ContextID,
				StepID: 0,
				Peers:  []byte(mpcprotocol.ErrFailedTxVerify.Error())}
			peerInfo := mpcServer.getMessagePeers(mpcMessage)
			peerIDs := make([]discover.NodeID, 0)
			for _, item := range *peerInfo {
				peerIDs = append(peerIDs, item.PeerID)
			}

			mpcServer.BoardcastMessage(peerIDs, mpcprotocol.MPCError, mpcMsg)
			return mpcprotocol.ErrFailedTxVerify
		}

		if len(mpcMessage.Data) > 1 {
			preSetValue = append(preSetValue, MpcValue{mpcprotocol.MpcTxHash, []big.Int{mpcMessage.Data[1]}, nil})
			log.Debug("createMpcContext", "from", common.BigToAddress(&mpcMessage.Data[2]), "txHash", common.BigToHash(&mpcMessage.Data[1]))
			mpcsyslog.Debug("createMpcContext. from:%s, txHash:%s", common.BigToAddress(&mpcMessage.Data[2]).String(), common.BigToHash(&mpcMessage.Data[1]).String())
			address := common.BigToAddress(&mpcMessage.Data[2])
			value, _, err := mpcServer.loadStoremanAddress(&address)
			if err != nil {
				return err
			}

			preSetValue = append(preSetValue, *value)
		}
	} else if ctxType == mpcprotocol.MpcCreateLockAccountPeer {
		if len(mpcMessage.BytesData) == 0 {
			return mpcprotocol.ErrInvalidStmAccType
		}

		accType := string(mpcMessage.BytesData[0][:])
		if !mpcprotocol.CheckAccountType(accType) {
			return mpcprotocol.ErrInvalidStmAccType
		}

		preSetValue = append(preSetValue, MpcValue{mpcprotocol.MpcStmAccType, nil, mpcMessage.BytesData[0]})
	}

	mpc, err := mpcServer.mpcCreater.CreateContext(ctxType, mpcMessage.ContextID, *mpcServer.getMessagePeers(mpcMessage), preSetValue...)
	if err == nil {
		go func() {
			mpcServer.addMpcContext(mpcMessage.ContextID, mpc)
			defer mpcServer.removeMpcContext(mpcMessage.ContextID)
			err = mpc.mainMPCProcess(mpcServer)
		}()
	}

	return err
}

func (mpcServer *MpcDistributor) addMpcContext(mpcID uint64, mpc MpcInterface) {
	log.Debug("addMpcContext", "ctxId", mpcID)
	mpcsyslog.Debug("addMpcContext. ctxId:%d", mpcID)
	mpcServer.mu.Lock()
	defer mpcServer.mu.Unlock()
	mpcServer.mpcMap[mpcID] = mpc
}

func (mpcServer *MpcDistributor) removeMpcContext(mpcID uint64) {
	log.Debug("removeMpcContext", "ctxId", mpcID)
	mpcsyslog.Debug("removeMpcContext. ctxId:%d", mpcID)
	mpcServer.mu.Lock()
	defer mpcServer.mu.Unlock()
	delete(mpcServer.mpcMap, mpcID)
}

func (mpcServer *MpcDistributor) getMpcMessage(PeerID *discover.NodeID, mpcMessage *mpcprotocol.MpcMessage) error {
	log.Debug("getMpcMessage", "peerid", PeerID, "ctxId", mpcMessage.ContextID, "stepID", mpcMessage.StepID)
	mpcsyslog.Debug("getMpcMessage. peerid:%s, ctxId:%d, stepID:%d", PeerID.String(), mpcMessage.ContextID, mpcMessage.StepID)
	mpcServer.mu.RLock()
	mpc, exist := mpcServer.mpcMap[mpcMessage.ContextID]
	mpcServer.mu.RUnlock()
	if exist {
		return mpc.getMessage(PeerID, mpcMessage, mpcServer.getMessagePeers(mpcMessage))
	}

	return nil
}

func (mpcServer *MpcDistributor) getOwnerP2pMessage(PeerID *discover.NodeID, code uint64, msg interface{}) error {
	switch code {
	case mpcprotocol.MPCMessage:
		mpcMessage := msg.(*mpcprotocol.MpcMessage)
		mpcServer.getMpcMessage(PeerID, mpcMessage)
	case mpcprotocol.RequestMPCNonce:
		// do nothing
	}

	return nil
}

func (mpcServer *MpcDistributor) SelfNodeId() *discover.NodeID {
	return &mpcServer.Self.ID
}

func (mpcServer *MpcDistributor) P2pMessage(peerID *discover.NodeID, code uint64, msg interface{}) error {
	if *peerID == mpcServer.Self.ID {
		mpcServer.getOwnerP2pMessage(&mpcServer.Self.ID, code, msg)
	} else {
		err := mpcServer.P2pMessager.SendToPeer(peerID, code, msg)
		if err != nil {
			mpcsyslog.Err("BoardcastMessage fail. err:%s", err.Error())
			log.Error("BoardcastMessage fail", "error", err)
		}
	}

	return nil
}

func (mpcServer *MpcDistributor) BoardcastMessage(peers []discover.NodeID, code uint64, msg interface{}) error {
	if peers == nil {
		for _, peer := range mpcServer.StoreManGroup {
			if peer == mpcServer.Self.ID {
				mpcServer.getOwnerP2pMessage(&mpcServer.Self.ID, code, msg)
			} else {
				err := mpcServer.P2pMessager.SendToPeer(&peer, code, msg)
				if err != nil {
					mpcsyslog.Err("BoardcastMessage fail. peer:%s, err:%s", peer.String(), err.Error())
					log.Error("BoardcastMessage fail", "peerID", peer.String(), "error", err)
				}
			}
		}
	} else {
		for _, peerID := range peers {
			if peerID == mpcServer.Self.ID {
				mpcServer.getOwnerP2pMessage(&mpcServer.Self.ID, code, msg)
			} else {
				err := mpcServer.P2pMessager.SendToPeer(&peerID, code, msg)
				if err != nil {
					mpcsyslog.Err("BoardcastMessage fail. peer:%s, err:%s", peerID.String(), err.Error())
					log.Error("BoardcastMessage fail", "peerID", peerID.String(), "error", err)
				}
			}
		}
	}

	return nil
}

func (mpcServer *MpcDistributor) newStoremanKeyStore(pKey *ecdsa.PublicKey, pShare *big.Int, seeds []uint64, passphrase string, accType string) (accounts.Account, error) {
	ks := mpcServer.AccountManager.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	account, err := ks.NewStoremanAccount(pKey, pShare, seeds, passphrase, accType)
	if err != nil {
		mpcsyslog.Err("NewStoremanKeyStore fail. err:%s", err.Error())
		log.Error("NewStoremanKeyStore fail", "err", err)
	} else {
		mpcsyslog.Debug("newStoremanKeyStore success, addr:%s", account.Address.String())
		log.Debug("newStoremanKeyStore success", "address", account.Address)
	}

	return account, err
}

func (mpcServer *MpcDistributor) CreateKeystore(result mpcprotocol.MpcResultInterface, peers *[]mpcprotocol.PeerInfo, accType string) error {
	mpcsyslog.Debug("MpcDistributor.CreateKeystore begin")
	point, err := result.GetValue(mpcprotocol.PublicKeyResult)
	if err != nil {
		mpcsyslog.Err("CreateKeystore fail. get PublicKeyResult fail")
		return err
	}

	private, err := result.GetValue(mpcprotocol.MpcPrivateShare)
	if err != nil {
		mpcsyslog.Err("CreateKeystore fail. get MpcPrivateShare fail")
		return err
	}

	result1 := new(ecdsa.PublicKey)
	result1.Curve = crypto.S256()
	result1.X = &point[0]
	result1.Y = &point[1]
	seed := make([]uint64, len(*peers))
	for i, item := range *peers {
		seed[i] = item.Seed
	}

	account, err := mpcServer.newStoremanKeyStore(result1, &private[0], seed, mpcServer.password, accType)
	if err != nil {
		return err
	}

	result.SetByteValue(mpcprotocol.MpcContextResult, account.Address[:])
	return nil
}

func (mpcServer *MpcDistributor) SignTransaction(result mpcprotocol.MpcResultInterface, signNum int) error {
	chainType, err1 := result.GetByteValue(mpcprotocol.MpcChainType)

	if bytes.Equal(chainType, []byte("BTC")) {
		txSigns := []byte{}

		for i := 0; i < signNum; i++ {
			iStr := "_" + strconv.Itoa(i)

			R, err := result.GetValue(mpcprotocol.MpcTxSignResultR + iStr)
			if err != nil {
				mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s, i:%d", mpcprotocol.MpcTxSignResultR, i)
				log.Error("-----------------MpcDistributor SignTransaction, get value fail.", "key", mpcprotocol.MpcTxSignResultR, "i", i)
				return err
			}

			//V, err := result.GetValue(mpcprotocol.MpcTxSignResultV + iStr)
			//if err != nil {
			//	mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s", mpcprotocol.MpcTxSignResultV)
			//	return err
			//}

			S, err := result.GetValue(mpcprotocol.MpcTxSignResult + iStr)
			if err != nil {
				mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s, i:%d", mpcprotocol.MpcTxSignResult, i)
				log.Error("-----------------MpcDistributor SignTransaction, get value fail.", "key", mpcprotocol.MpcTxSignResult, "i", i)
				return err
			}

			sinature := btcec.Signature{&R[0], &S[0]}
			sign := sinature.Serialize()
			txSigns = append(txSigns, sign...)
			txSigns = append(txSigns, byte(txscript.SigHashAll))
		}

		result.SetByteValue(mpcprotocol.MpcContextResult, txSigns)
		return nil

	} else {

		R, err := result.GetValue(mpcprotocol.MpcTxSignResultR + "_0")
		if err != nil {
			mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s", mpcprotocol.MpcTxSignResultR)
			return err
		}

		V, err := result.GetValue(mpcprotocol.MpcTxSignResultV + "_0")
		if err != nil {
			mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s", mpcprotocol.MpcTxSignResultV)
			return err
		}

		S, err := result.GetValue(mpcprotocol.MpcTxSignResult + "_0")
		if err != nil {
			mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s", mpcprotocol.MpcTxSignResult)
			return err
		}

		SignType, err := result.GetByteValue(mpcprotocol.MpcSignType)
		if (err == nil && bytes.Equal(SignType, []byte("hash"))) || err1 != nil {
			txSign, err := mpccrypto.TransSignature(&R[0], &S[0], &V[0])
			if err != nil {
				mpcsyslog.Err("mpccrypto tans signature fail. err:%s", err.Error())
				return err
			}

			result.SetByteValue(mpcprotocol.MpcContextResult, txSign)
			txHash, err := result.GetValue(mpcprotocol.MpcTxHash + "_0")
			if err == nil {
				from, err := mpccrypto.SenderEcrecover(txHash[0].Bytes(), txSign)
				if err == nil {
					result.SetByteValue(mpcprotocol.MPCSignedFrom, from[:])
				}
			}

			return nil

		} else {
			chianID, err := result.GetValue(mpcprotocol.MpcChainID)
			if err != nil {
				mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s", mpcprotocol.MpcChainID)
				return err
			}

			signer, err := mpcServer.createMPCTxSigner(string(chainType[:]), &chianID[0])
			if err != nil {
				mpcsyslog.Err("MpcDistributor.SignTransaction, create mpc signer fail. err:%s", err.Error())
				return err
			}

			encodedTx, err := result.GetByteValue(mpcprotocol.MpcTransaction)
			if err != nil {
				mpcsyslog.Err("MpcDistributor.SignTransaction, GetValue fail. key:%s", mpcprotocol.MpcTransaction)
				return err
			}

			tx := new(types.Transaction)
			if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
				mpcsyslog.Err("rlp decode fail. err:%s", err.Error())
				return err
			}

			txSign, from, err := signer.SignTransaction(tx, &R[0], &S[0], &V[0])
			if err != nil {
				mpcsyslog.Err("mpc signatual fail. err:%s", err.Error())
				return err
			}

			result.SetByteValue(mpcprotocol.MpcContextResult, txSign)
			result.SetByteValue(mpcprotocol.MPCSignedFrom, from[:])
			return nil
		}
	}

}
