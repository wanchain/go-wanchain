package storemanmpc

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/awskms"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/p2p/discover"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/storeman/shcnorrmpc"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/storeman/validator"
	"io/ioutil"
	"math/big"
	"sort"
	"sync"
)

type MpcContextCreater interface {
	CreateContext(int, uint64, []mpcprotocol.PeerInfo, ...MpcValue) (MpcInterface, error) //createContext
}

type MpcValue struct {
	Key       string
	Value     []big.Int
	ByteValue []byte
}

func (v *MpcValue) String() string {
	strRet := "key=" + v.Key
	for i := range v.Value {
		strRet += ", value:" + v.Value[i].String()
	}

	if v.ByteValue != nil {
		strRet += ", value:" + common.ToHex(v.ByteValue)
	}

	return strRet
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

func CreateMpcDistributor(accountManager *accounts.Manager,
	msger P2pMessager,
	aKID,
	secretKey,
	region,
	password string) *MpcDistributor {

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

func GetPrivateShare(ks *keystore.KeyStore,
	address common.Address,
	enableKms bool,
	kmsInfo *KmsInfo,
	password string) (*keystore.Key, int, error) {

	account := accounts.Account{Address: address}
	account, err := ks.Find(account)
	if err != nil {
		log.SyslogErr("find account from keystore fail", "addr", address.String(), "err", err.Error())
		return nil, 0x00, err
	}

	var keyjson []byte
	if enableKms {
		keyjson, err = awskms.DecryptFileToBuffer(account.URL.Path, kmsInfo.AKID, kmsInfo.SecretKey, kmsInfo.Region)
	} else {
		keyjson, err = ioutil.ReadFile(account.URL.Path)
	}

	if err != nil {
		log.SyslogErr("get account keyjson fail",
			"addr", address.String(),
			"path", account.URL.Path,
			"err", err.Error())

		return nil, 0x01, err
	}

	key, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		log.SyslogErr("decrypt account keyjson fail",
			"addr", address.String(),
			"path", account.URL.Path,
			"err", err.Error())

		return nil, 0x011, err
	}

	return key, 0x111, nil
}

func (mpcServer *MpcDistributor) GetMessage(PeerID discover.NodeID, rw p2p.MsgReadWriter, msg *p2p.Msg) error {
	log.SyslogInfo("MpcDistributor GetMessage begin", "msgCode", msg.Code)

	switch msg.Code {
	case mpcprotocol.StatusCode:
		// this should not happen, but no need to panic; just ignore this message.
		log.SyslogInfo("unexpected status message received", "peer", PeerID.String())

	case mpcprotocol.KeepaliveCode:
		// this should not happen, but no need to panic; just ignore this message.

	case mpcprotocol.KeepaliveOkCode:
		// this should not happen, but no need to panic; just ignore this message.

	case mpcprotocol.MPCError:
		var mpcMessage mpcprotocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.SyslogErr("MpcDistributor.GetMessage, rlp decode MPCError msg fail", "err", err.Error())
			return err
		}

		errText := string(mpcMessage.Peers[:])
		log.SyslogErr("MpcDistributor.GetMessage, MPCError message received", "peer", PeerID.String(), "err", errText)
		go mpcServer.QuitMpcContext(&mpcMessage)

	case mpcprotocol.RequestMPC:
		log.SyslogInfo("MpcDistributor.GetMessage, RequestMPC message received", "peer", PeerID.String())
		var mpcMessage mpcprotocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.SyslogErr("MpcDistributor.GetMessage, rlp decode RequestMPC msg fail", "err", err.Error())
			return err
		}

		//create context
		go func() {
			err := mpcServer.createMpcCtx(&mpcMessage)

			if err != nil {
				log.SyslogErr("createMpcContext fail", "err", err.Error())
			}
		}()

	case mpcprotocol.MPCMessage:
		var mpcMessage mpcprotocol.MpcMessage
		err := rlp.Decode(msg.Payload, &mpcMessage)
		if err != nil {
			log.SyslogErr("GetP2pMessage fail", "err", err.Error())
			return err
		}

		log.SyslogInfo("MpcDistributor.GetMessage, MPCMessage message received", "peer", PeerID.String())
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

func (mpcServer *MpcDistributor) CreateRequestGPK() ([]byte, error) {
	log.SyslogInfo("CreateRequestGPK begin")

	preSetValue := make([]MpcValue, 0, 1)
	value, err := mpcServer.createRequestMpcContext(mpcprotocol.MpcGPKLeader,
		preSetValue...)

	if err != nil {
		return []byte{}, err
	} else {
		return value, err
	}
}

func (mpcServer *MpcDistributor) CreateReqMpcSign(data []byte, from common.Address) ([]byte, error) {

	log.SyslogInfo("CreateReqMpcSign begin")

	value, err := mpcServer.createRequestMpcContext(mpcprotocol.MpcSignLeader,
		MpcValue{mpcprotocol.MpcAddress, nil, from[:]},
		MpcValue{mpcprotocol.MpcM, nil, data})

	//Todo update the return value
	return value, err
}

func (mpcServer *MpcDistributor) createRequestMpcContext(ctxType int, preSetValue ...MpcValue) (hexutil.Bytes, error) {
	log.SyslogInfo("MpcDistributor createRequestMpcContext begin")
	mpcID, err := mpcServer.getMpcID()
	if err != nil {
		return nil, err
	}

	peers := []mpcprotocol.PeerInfo{}
	if ctxType == mpcprotocol.MpcSignLeader {
		address := common.Address{}
		for _, item := range preSetValue {
			if item.Key == mpcprotocol.MpcAddress {
				copy(address[:], item.ByteValue)
				break
			}
		}
		// peers1: the peers which are used to create the group public key, used to build the sign data.
		value, value1, peers1, err := mpcServer.loadStoremanAddress(&address)
		if err != nil {

			log.SyslogErr("MpcDistributor createRequestMpcContext, loadStoremanAddress fail",
				"address", address.String(),
				"err", err.Error())

			return []byte{}, err
		}

		// mpc private share
		preSetValue = append(preSetValue, *value)

		// mpc gpk for sign
		preSetValue = append(preSetValue, *value1)

		peers = peers1
	} else {
		for i := 0; i < len(mpcServer.StoreManGroup); i++ {
			peers = append(peers, mpcprotocol.PeerInfo{PeerID: mpcServer.StoreManGroup[i], Seed: 0})
		}
	}
	mpc, err := mpcServer.mpcCreater.CreateContext(ctxType,
		mpcID,
		peers,
		preSetValue...)

	if err != nil {
		log.SyslogErr("MpcDistributor createRequestMpcContext, CreateContext fail", "err", err.Error())
		return []byte{}, err
	}

	log.SyslogInfo("MpcDistributor createRequestMpcContext", "ctxType", ctxType, "mpcID", mpcID)

	mpcServer.addMpcContext(mpcID, mpc)
	defer mpcServer.removeMpcContext(mpcID)
	err = mpc.mainMPCProcess(mpcServer)
	if err != nil {
		log.SyslogErr("MpcDistributor createRequestMpcContext, mainMPCProcess fail", "err", err.Error())
		return []byte{}, err
	}

	result := mpc.getMpcResult()

	log.SyslogInfo("MpcDistributor createRequestMpcContext, succeed", "result", common.ToHex(result))
	return result, nil
}

func (mpcServer *MpcDistributor) loadStoremanAddress(address *common.Address) (*MpcValue, *MpcValue, []mpcprotocol.PeerInfo, error) {
	log.SyslogInfo("MpcDistributor.loadStoremanAddress begin", "address", address.String())

	mpcServer.accMu.Lock()
	defer mpcServer.accMu.Unlock()
	value, exist := mpcServer.mpcAccountMap[*address]

	var key *keystore.Key
	var err error
	if !exist {
		ks := mpcServer.AccountManager.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		key, _, err = GetPrivateShare(ks, *address, mpcServer.enableAwsKms, &mpcServer.kmsInfo, mpcServer.password)
		if err != nil {
			return nil, nil, nil, err
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

	return &MpcValue{mpcprotocol.MpcPrivateShare, []big.Int{value.privateShare}, nil},
		&MpcValue{mpcprotocol.PublicKeyResult, []big.Int{*(key.PrivateKey.PublicKey.X), *(key.PrivateKey.PublicKey.Y)}, nil},
		value.peers,
		nil
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

//func (mpcServer *MpcDistributor) selectPeers(ctxType int,
//	allPeers []mpcprotocol.PeerInfo,
//	preSetValue ...MpcValue) []mpcprotocol.PeerInfo {
//
//	var peers []mpcprotocol.PeerInfo
//	if ctxType == mpcprotocol.MpcGPKLeader {
//		peers = allPeers
//	} else {
//		peers = make([]mpcprotocol.PeerInfo, mpcprotocol.MPCDegree*2+1)
//		storemanLen := len(mpcServer.StoreManGroup)
//		selectIndex := 0
//		rand.Seed(time.Now().UnixNano())
//		for _, item := range preSetValue {
//			if strings.Index(item.Key, mpcprotocol.MpcTxHash) == 0 {
//				sel := big.NewInt(0)
//				hash := item.Value[0]
//				sel.Mod(&hash, big.NewInt(int64(storemanLen)))
//				selectIndex = (int(sel.Uint64()) + rand.Int()) % storemanLen
//				break
//			}
//		}
//
//		j := 1
//		for i := 0; j < len(peers) && i < len(allPeers); i++ {
//			sel := (i + selectIndex) % storemanLen
//			if mpcServer.P2pMessager.IsActivePeer(&(allPeers[sel].PeerID)) {
//				peers[j] = allPeers[sel]
//				log.SyslogInfo("select peers", "index", j, "peer", peers[j].PeerID.String())
//				j++
//			}
//		}
//
//		index := int(mpcServer.storeManIndex[mpcServer.Self.ID])
//		peers[0] = allPeers[index]
//		log.SyslogInfo("select peers", "index", 0, "peer", peers[0].PeerID.String())
//	}
//
//	return peers
//}

func (mpcServer *MpcDistributor) getMpcID() (uint64, error) {
	var mpcID uint64
	var err error
	for {
		mpcID, err = shcnorrmpc.UintRand(uint64(1<<64 - 1))
		if err != nil {
			log.SyslogErr("MpcDistributor getMpcID, UnitRand fail", "err", err.Error())
			return 0, err
		}

		mpcServer.mu.RLock()
		_, exist := mpcServer.mpcMap[mpcID]
		mpcServer.mu.RUnlock()
		if !exist {
			return mpcID, nil
		}
	}
}

func (mpcServer *MpcDistributor) QuitMpcContext(msg *mpcprotocol.MpcMessage) {
	mpcServer.mu.RLock()
	mpc, exist := mpcServer.mpcMap[msg.ContextID]
	mpcServer.mu.RUnlock()
	if exist {
		mpc.quit(errors.New(string(msg.Peers[:])))
	}
}

func (mpcServer *MpcDistributor) createMpcCtx(mpcMessage *mpcprotocol.MpcMessage, preSetValue ...MpcValue) error {
	log.SyslogInfo("MpcDistributor createMpcCtx begin")

	mpcServer.mu.RLock()
	_, exist := mpcServer.mpcMap[mpcMessage.ContextID]
	mpcServer.mu.RUnlock()
	if exist {
		log.SyslogErr("createMpcCtx fail", "err", mpcprotocol.ErrMpcContextExist.Error())
		return mpcprotocol.ErrMpcContextExist
	}

	var ctxType int
	nType := mpcMessage.Data[0].Int64()
	if nType == mpcprotocol.MpcGPKLeader {
		ctxType = mpcprotocol.MpcGPKPeer
	} else {
		ctxType = mpcprotocol.MpcSignPeer
	}

	log.SyslogInfo("createMpcCtx", "ctxType", ctxType, "ctxId", mpcMessage.ContextID)
	if ctxType == mpcprotocol.MpcSignPeer {
		log.SyslogInfo("createMpcCtx MpcSignPeer")
		mpcM := mpcMessage.BytesData[0]
		address := mpcMessage.BytesData[1]

		add := common.Address{}
		copy(add[:], address)

		log.SyslogInfo("createMpcCtx", "address", address, "mpcM", mpcM)
		// load account
		MpcPrivateShare, _, _, err := mpcServer.loadStoremanAddress(&add)
		if err != nil {
			return err
		}

		preSetValue = append(preSetValue, MpcValue{mpcprotocol.MpcAddress, nil, address})
		preSetValue = append(preSetValue, MpcValue{mpcprotocol.MpcM, nil, mpcM})
		preSetValue = append(preSetValue, *MpcPrivateShare)

		//verifyResult := validator.ValidateTx(signer, address, chainType, &chainId, txBytesData, txHash.Bytes())
		verifyResult := validator.ValidateData(mpcM[:])

		if !verifyResult {
			mpcMsg := &mpcprotocol.MpcMessage{ContextID: mpcMessage.ContextID,
				StepID: 0,
				Peers:  []byte(mpcprotocol.ErrFailedTxVerify.Error())}
			peerInfo := mpcServer.getMessagePeers(mpcMessage)
			peerIDs := make([]discover.NodeID, 0)
			for _, item := range *peerInfo {
				peerIDs = append(peerIDs, item.PeerID)
			}

			mpcServer.BroadcastMessage(peerIDs, mpcprotocol.MPCError, mpcMsg)

			log.SyslogErr("createMpcContext, verify tx fail", "ContextID", mpcMessage.ContextID)
			return mpcprotocol.ErrFailedTxVerify
		}

	} else if ctxType == mpcprotocol.MpcGPKPeer {
		//ToDo add log info
		//ToDo change reqMPC message sent
	}

	mpc, err := mpcServer.mpcCreater.CreateContext(ctxType,
		mpcMessage.ContextID,
		*mpcServer.getMessagePeers(mpcMessage),
		preSetValue...)

	if err != nil {
		log.SyslogErr("createMpcContext, createContext fail", "err", err.Error())
		return err
	}

	go func() {
		mpcServer.addMpcContext(mpcMessage.ContextID, mpc)
		defer mpcServer.removeMpcContext(mpcMessage.ContextID)
		err = mpc.mainMPCProcess(mpcServer)
	}()

	return nil
}

func (mpcServer *MpcDistributor) addMpcContext(mpcID uint64, mpc MpcInterface) {
	log.SyslogInfo("addMpcContext", "ctxId", mpcID)

	mpcServer.mu.Lock()
	defer mpcServer.mu.Unlock()
	mpcServer.mpcMap[mpcID] = mpc
}

func (mpcServer *MpcDistributor) removeMpcContext(mpcID uint64) {
	log.SyslogInfo("removeMpcContext", "ctxId", mpcID)

	mpcServer.mu.Lock()
	defer mpcServer.mu.Unlock()
	delete(mpcServer.mpcMap, mpcID)
}

func (mpcServer *MpcDistributor) getMpcMessage(PeerID *discover.NodeID, mpcMessage *mpcprotocol.MpcMessage) error {
	log.SyslogInfo("getMpcMessage",
		"peerid", PeerID.String(),
		"ctxId", mpcMessage.ContextID,
		"stepID", mpcMessage.StepID)

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
			log.SyslogErr("BroadcastMessage fail", "err", err.Error())
		}
	}

	return nil
}

func (mpcServer *MpcDistributor) BroadcastMessage(peers []discover.NodeID, code uint64, msg interface{}) error {
	if peers == nil {
		for _, peer := range mpcServer.StoreManGroup {
			if peer == mpcServer.Self.ID {
				mpcServer.getOwnerP2pMessage(&mpcServer.Self.ID, code, msg)
			} else {
				err := mpcServer.P2pMessager.SendToPeer(&peer, code, msg)
				if err != nil {
					log.SyslogErr("BroadcastMessage fail", "peer", peer.String(), "err", err.Error())
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
					log.SyslogErr("BroadcastMessage fail", "peer", peerID.String(), "err", err.Error())
				}
			}
		}
	}

	return nil
}

func (mpcServer *MpcDistributor) newStoremanKeyStore(pKey *ecdsa.PublicKey,
	pShare *big.Int,
	seeds []uint64,
	passphrase string,
	accType string) (accounts.Account, error) {

	ks := mpcServer.AccountManager.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	account, err := ks.NewStoremanAccount(pKey, pShare, seeds, passphrase, accType)
	if err != nil {
		log.SyslogErr("NewStoremanKeyStore fail", "err", err.Error())
	} else {
		log.SyslogInfo("newStoremanKeyStore success", "addr", account.Address.String())
	}

	return account, err
}

func (mpcServer *MpcDistributor) CreateKeystore(result mpcprotocol.MpcResultInterface,
	peers *[]mpcprotocol.PeerInfo,
	accType string) error {

	log.SyslogInfo("MpcDistributor.CreateKeystore begin")
	point, err := result.GetValue(mpcprotocol.PublicKeyResult)
	if err != nil {
		log.SyslogErr("CreateKeystore fail. get PublicKeyResult fail")
		return err
	}

	private, err := result.GetValue(mpcprotocol.MpcPrivateShare)
	if err != nil {
		log.SyslogErr("CreateKeystore fail. get MpcPrivateShare fail")
		return err
	}

	result1 := new(ecdsa.PublicKey)
	result1.Curve = crypto.S256()
	//result1.X = &point[0]
	result1.X = big.NewInt(0).SetBytes(point[0].Bytes())
	result1.Y = big.NewInt(0).SetBytes(point[1].Bytes())
	seed := make([]uint64, len(*peers))

	for i, item := range *peers {
		seed[i] = item.Seed
	}

	_, err = mpcServer.newStoremanKeyStore(result1, &private[0], seed, mpcServer.password, accType)
	if err != nil {
		return err
	}

	result.SetByteValue(mpcprotocol.MpcContextResult, crypto.FromECDSAPub(result1))
	log.Info("=====Jacob CreateKeystore ",
		"gpk address", crypto.PubkeyToAddress(*result1),
		"gpk byte", crypto.FromECDSAPub(result1),
		"gpk hexutil.Encode", hexutil.Encode(crypto.FromECDSAPub(result1)),
		"gpk string", string(crypto.FromECDSAPub(result1)))

	return nil
}
