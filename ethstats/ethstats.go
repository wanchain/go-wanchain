// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package ethstats implements the network stats reporting service.
package ethstats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/crypto/sha3"

	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posapi"
	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/mclock"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/eth"
	"github.com/wanchain/go-wanchain/event"
	"github.com/wanchain/go-wanchain/les"
	"github.com/wanchain/go-wanchain/p2p"
	"github.com/wanchain/go-wanchain/rpc"
	"golang.org/x/net/websocket"
)

const (
	// historyUpdateRange is the number of blocks a node should report upon login or
	// history request.
	historyUpdateRange = 50

	// txChanSize is the size of channel listening to TxPreEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
	// alarmLogChanSize is the size of channel listening to AlarmLogEvent
	alarmLogChanSize = 1024
	// reorgChanSize is the size of channel listening to ReorgEvent
	reorgChanSize = 1024
)

var (
	maxUint64 = uint64(1<<64 - 1)
)

type txPool interface {
	// SubscribeTxPreEvent should return an event subscription of
	// TxPreEvent and send events to the given channel.
	SubscribeTxPreEvent(chan<- core.TxPreEvent) event.Subscription
}

type blockChain interface {
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	SubscribeReorgEvent(ch chan<- core.ReorgEvent) event.Subscription
}

// Service implements an Ethereum netstats reporting daemon that pushes local
// chain statistics up to a monitoring server.
type Service struct {
	server *p2p.Server        // Peer-to-peer server to retrieve networking infos
	eth    *eth.Ethereum      // Full Ethereum service if monitoring a full node
	les    *les.LightEthereum // Light Ethereum service if monitoring a light node
	engine consensus.Engine   // Consensus engine to retrieve variadic block fields

	node string // Name of the node to display on the monitoring page
	pass string // Password to authorize access to the monitoring page
	host string // Remote address of the monitoring service

	pongCh chan struct{} // Pong notifications are fed into this channel
	histCh chan []uint64 // History request block numbers are fed into this channel

	epochId uint64
	api     *posapi.PosApi
}

// New returns a monitoring service ready for stats reporting.
func New(url string, ethServ *eth.Ethereum, lesServ *les.LightEthereum) (*Service, error) {
	// Parse the netstats connection url
	re := regexp.MustCompile("([^:@]*)(:([^@]*))?@(.+)")
	parts := re.FindStringSubmatch(url)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid netstats url: \"%s\", should be nodename:secret@host:port", url)
	}
	// Assemble and return the stats service
	var engine consensus.Engine
	if ethServ != nil {
		engine = ethServ.Engine()
	} else {
		engine = lesServ.Engine()
	}

	svr := &Service{
		eth:     ethServ,
		les:     lesServ,
		engine:  engine,
		node:    parts[1],
		pass:    parts[3],
		host:    parts[4],
		pongCh:  make(chan struct{}),
		histCh:  make(chan []uint64, 1),
		epochId: maxUint64,
	}

	svr.initApi()
	return svr, nil
}

func (s *Service) initApi() {
	if s.eth == nil || s.api != nil {
		return
	}

	apis := posapi.APIs(s.eth.BlockChain(), s.eth.ApiBackend)
	api, ok := apis[0].Service.(*posapi.PosApi)
	if !ok {
		log.Error("create posapi instance fail")
		return
	}

	s.api = api
}

// Protocols implements node.Service, returning the P2P network protocols used
// by the stats service (nil as it doesn't use the devp2p overlay network).
func (s *Service) Protocols() []p2p.Protocol { return nil }

// APIs implements node.Service, returning the RPC API endpoints provided by the
// stats service (nil as it doesn't provide any user callable APIs).
func (s *Service) APIs() []rpc.API { return nil }

// Start implements node.Service, starting up the monitoring and reporting daemon.
func (s *Service) Start(server *p2p.Server) error {
	s.server = server
	go s.loop()

	log.Info("Stats daemon started")
	return nil
}

// Stop implements node.Service, terminating the monitoring and reporting daemon.
func (s *Service) Stop() error {
	log.Info("Stats daemon stopped")
	return nil
}

// loop keeps trying to connect to the netstats server, reporting chain events
// until termination.
func (s *Service) loop() {
	// Subscribe to chain events to execute updates on
	var blockchain blockChain
	var txpool txPool
	if s.eth != nil {
		blockchain = s.eth.BlockChain()
		txpool = s.eth.TxPool()
	} else {
		blockchain = s.les.BlockChain()
		txpool = s.les.TxPool()
	}

	chainHeadCh := make(chan core.ChainHeadEvent, chainHeadChanSize)
	headSub := blockchain.SubscribeChainHeadEvent(chainHeadCh)
	defer headSub.Unsubscribe()

	txEventCh := make(chan core.TxPreEvent, txChanSize)
	txSub := txpool.SubscribeTxPreEvent(txEventCh)
	defer txSub.Unsubscribe()

	alarmEventCh := make(chan log.LogInfo, alarmLogChanSize)
	alarmSub := log.SubscribeAlarm(alarmEventCh)
	defer alarmSub.Unsubscribe()

	reorgEventCh := make(chan core.ReorgEvent, reorgChanSize)
	reorgSub := blockchain.SubscribeReorgEvent(reorgEventCh)
	defer reorgSub.Unsubscribe()

	// Start a goroutine that exhausts the subsciptions to avoid events piling up
	var (
		quitCh  = make(chan struct{})
		headCh  = make(chan *types.Block, 1)
		txCh    = make(chan struct{}, 1)
		alarmCh = make(chan log.LogInfo, 10)
		reorgCh = make(chan core.ReorgEvent, 10)
	)
	go func() {
		var lastTx mclock.AbsTime

	HandleLoop:
		for {
			log.Debug("wanstats handle loop begin..")
			select {
			// Notify of chain head events, but drop if too frequent
			case head := <-chainHeadCh:
				select {
				case headCh <- head.Block:
				default:
				}

			// Notify of new transaction events, but drop if too frequent
			case <-txEventCh:
				if time.Duration(mclock.Now()-lastTx) < time.Second {
					continue
				}
				lastTx = mclock.Now()

				select {
				case txCh <- struct{}{}:
				default:
				}

			// Notify of new alarm
			case alarm := <-alarmEventCh:
				select {
				case alarmCh <- alarm:
				default:
				}

			// Notify of new reorg
			case len := <-reorgEventCh:
				select {
				case reorgCh <- len:
				default:
				}

			// node stopped
			case <-txSub.Err():
				break HandleLoop
			case <-headSub.Err():
				break HandleLoop
			case <-alarmSub.Err():
				break HandleLoop
			case <-reorgSub.Err():
				break HandleLoop
			}
		}
		close(quitCh)
		return
	}()
	// Loop reporting until termination
	for {
		log.Info("wanstats report big loop begin..")
		// Resolve the URL, defaulting to TLS, but falling back to none too
		path := fmt.Sprintf("%s/api", s.host)
		urls := []string{path}

		if !strings.Contains(path, "://") { // url.Parse and url.IsAbs is unsuitable (https://github.com/golang/go/issues/19779)
			urls = []string{"wss://" + path, "ws://" + path}
		}
		// Establish a websocket connection to the server on any supported URL
		var (
			conf *websocket.Config
			conn *websocket.Conn
			err  error
		)
		for _, url := range urls {
			if conf, err = websocket.NewConfig(url, "http://localhost/"); err != nil {
				continue
			}
			conf.Dialer = &net.Dialer{Timeout: 5 * time.Second}
			if conn, err = websocket.DialConfig(conf); err == nil {
				log.Info("connect wanstats server successful", "url", url)
				break
			}
		}
		if err != nil {
			log.Warn("Stats server unreachable", "err", err)
			time.Sleep(10 * time.Second)
			continue
		}
		// Authenticate the client with the server
		if err = s.login(conn); err != nil {
			log.Warn("Stats login failed", "err", err)
			conn.Close()
			time.Sleep(10 * time.Second)
			continue
		}
		go s.readLoop(conn)

		if !s.isPos() {
			// Send the initial stats so our node looks decent from the get go
			if err = s.report(conn); err != nil {
				log.Warn("Initial stats report failed", "err", err)
				conn.Close()
				continue
			}
		} else {
			// Send the initial stats so our node looks decent from the get go
			if err = s.reportPos(conn); err != nil {
				log.Warn("Initial stats reportPos failed", "err", err)
				conn.Close()
				continue
			}

			// send the initial leader info
			if err = s.reportLeader(conn); err != nil {
				log.Warn("Initial leader report failed", "err", err)
				conn.Close()
				continue
			}
		}

		// Keep sending status updates until the connection breaks
		fullReport := time.NewTicker(posconfig.SlotTime * time.Second)

		for err == nil {
			log.Debug("wanstats report small loop begin..")

			s.initApi()

			select {
			case <-quitCh:
				conn.Close()
				return

			case <-fullReport.C:
				if !s.isPos() {
					if err = s.report(conn); err != nil {
						log.Warn("Full stats report failed", "err", err)
					}
				} else {
					if err = s.reportPos(conn); err != nil {
						log.Warn("Full pos-stats report failed", "err", err)
					}
				}

			case list := <-s.histCh:
				if !s.isPos() {
					if err = s.reportHistory(conn, list); err != nil {
						log.Warn("Requested history report failed", "err", err)
					}
				} else {
					if err = s.reportPosHistory(conn, list); err != nil {
						log.Warn("Requested pos history report failed", "err", err)
					}
				}
			case head := <-headCh:
				if !s.isPos() {
					if err = s.reportBlock(conn, head); err != nil {
						log.Warn("Block stats report failed", "err", err)
					}
				} else {
					if err = s.reportPosBlock(conn, head); err != nil {
						log.Warn("Pos block stats report failed", "err", err)
					}
				}

				if err = s.reportPending(conn); err != nil {
					log.Warn("Post-block transaction stats report failed", "err", err)
				}
			case <-txCh:
				if err = s.reportPending(conn); err != nil {
					log.Warn("Transaction stats report failed", "err", err)
				}
			case alarm := <-alarmCh:
				if err = s.reportPosAlarm(conn, &alarm); err != nil {
					log.Warn("pos alarm report failed", "err", err)
				}
			case reorg := <-reorgCh:
				if !s.isPos() {
					continue
				}

				posReorg := pos_reorg{reorg.EpochId, reorg.SlotId, reorg.Len}
				if err = s.reportPosReorg(conn, &posReorg); err != nil {
					log.Warn("reorg length report failed", "err", err)
				}
			}
		}
		// Make sure the connection is closed
		conn.Close()
	}
}

func (s *Service) reportPosReorg(conn *websocket.Conn, reorg *pos_reorg) error {
	log.Trace("Sending reorg statistics to ethstats", "len", reorg.Len)
	stats := map[string]interface{}{
		"id":        s.node,
		"pos-reorg": reorg,
	}
	report := map[string][]interface{}{
		"emit": {"pos-reorg", stats},
	}
	return s.doSendReportData(conn, report)
}

func (s *Service) reportPosLog(conn *websocket.Conn) error {
	warn, wrong := log.GetWarnAndWrongLogCount()
	logCount := pos_log{warn, wrong}
	log.Trace("Sending log statistics to ethstats", "warn", warn, "wrong", wrong)

	stats := map[string]interface{}{
		"id":      s.node,
		"pos-log": logCount,
	}
	report := map[string][]interface{}{
		"emit": {"pos-log", stats},
	}
	return s.doSendReportData(conn, report)
}

func (s *Service) reportPosAlarm(conn *websocket.Conn, alarm *log.LogInfo) error {
	log.Trace("Sending alarm to ethstats", "level", alarm.Lvl, "msg", alarm.Msg)

	posAlarm := log2PosAlarm(alarm)
	stats := map[string]interface{}{
		"id":        s.node,
		"pos-alarm": posAlarm,
	}
	report := map[string][]interface{}{
		"emit": {"pos-alarm", stats},
	}
	return s.doSendReportData(conn, report)
}

func (s *Service) isPos() bool {
	if s.eth == nil {
		return false
	}

	bc := s.eth.BlockChain()
	if bc == nil {
		return false
	}

	block := bc.CurrentBlock()
	if block == nil {
		return false
	}

	return block.Number().Cmp(bc.Config().PosFirstBlock) >= 0
}

// readLoop loops as long as the connection is alive and retrieves data packets
// from the network socket. If any of them match an active request, it forwards
// it, if they themselves are requests it initiates a reply, and lastly it drops
// unknown packets.
func (s *Service) readLoop(conn *websocket.Conn) {
	log.Info("wanstats readloop begin")
	// If the read loop exists, close the connection
	defer conn.Close()

	for {
		// Retrieve the next generic network packet and bail out on error
		var msg map[string][]interface{}
		if err := websocket.JSON.Receive(conn, &msg); err != nil {
			log.Warn("Failed to decode stats server message", "err", err)
			return
		}
		log.Trace("Received message from stats server", "msg", msg)
		if len(msg["emit"]) == 0 {
			log.Warn("Stats server sent non-broadcast", "msg", msg)
			return
		}
		command, ok := msg["emit"][0].(string)
		if !ok {
			log.Warn("Invalid stats server message type", "type", msg["emit"][0])
			return
		}
		// If the message is a ping reply, deliver (someone must be listening!)
		if len(msg["emit"]) == 2 && command == "node-pong" {
			select {
			case s.pongCh <- struct{}{}:
				// Pong delivered, continue listening
				continue
			default:
				// Ping routine dead, abort
				log.Warn("Stats server pinger seems to have died")
				return
			}
		}
		// If the message is a history request, forward to the event processor
		if len(msg["emit"]) == 2 && command == "history" {
			// Make sure the request is valid and doesn't crash us
			request, ok := msg["emit"][1].(map[string]interface{})
			if !ok {
				log.Warn("Invalid stats history request", "msg", msg["emit"][1])
				s.histCh <- nil
				continue // Ethstats sometime sends invalid history requests, ignore those
			}
			list, ok := request["list"].([]interface{})
			if !ok {
				log.Warn("Invalid stats history block list", "list", request["list"])
				return
			}
			// Convert the block number list to an integer list
			numbers := make([]uint64, len(list))
			for i, num := range list {
				n, ok := num.(float64)
				if !ok {
					log.Warn("Invalid stats history block number", "number", num)
					return
				}
				numbers[i] = uint64(n)
			}
			select {
			case s.histCh <- numbers:
				continue
			default:
			}
		}
		// Report anything else and continue
		log.Info("Unknown stats message", "msg", msg)
	}
}

// nodeInfo is the collection of metainformation about a node that is displayed
// on the monitoring page.
type nodeInfo struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Network  string `json:"net"`
	Protocol string `json:"protocol"`
	API      string `json:"api"`
	Os       string `json:"os"`
	OsVer    string `json:"os_v"`
	Client   string `json:"client"`
	History  bool   `json:"canUpdateHistory"`
}

// authMsg is the authentication infos needed to login to a monitoring server.
type authMsg struct {
	Id            string   `json:"id"`
	Info          nodeInfo `json:"info"`
	Secret        string   `json:"secret"`
	ClientTime    int64    `json:"clientTime"`
	NodeId        string   `json:"nodeId"`
	ValidatorAddr string   `json:"validatorAddr"`
	Signature     string   `json:"signature"`
	GenesisHash   string   `json:"genesisHash"`
}

// login tries to authorize the client at the remote server.
func (s *Service) login(conn *websocket.Conn) error {
	// Construct and send the login authentication
	infos := s.server.NodeInfo()
	clientTime := time.Now().Unix()
	validatorAddr := ""
	signature := ""
	genesisHash := ""

	if s.eth != nil {
		coinBase, err := s.eth.Etherbase()
		if err != nil {
			log.Info("wanstats cant get coinbase", "err", err)
		} else {
			validatorAddr = coinBase.String()
			account := accounts.Account{Address: coinBase}
			wallet, err := s.eth.AccountManager().Find(account)
			if err != nil {
				log.Info("wanstats cant find wallet from account", "err", err)
			} else {
				signContent := fmt.Sprintf("%d%s%s", clientTime, infos.ID, validatorAddr)
				hasher := sha3.NewKeccak256()
				hasher.Write([]byte(signContent))
				hash := common.Hash{}
				hasher.Sum(hash[:0])
				signed, err := wallet.SignHash(account, hash[:])
				if err != nil {
					log.Info("wanstats sign the hello msg fail", "err", err)
				} else {
					signature = common.ToHex(signed)
				}
			}
		}

		if len(signature) == 0 {
			validatorAddr = ""
		}

		gBlk := s.eth.BlockChain().GetBlockByNumber(0)
		if gBlk != nil {
			genesisHash = gBlk.Hash().String()
		}
	}

	var network, protocol string
	if info := infos.Protocols["wan"]; info != nil {
		network = fmt.Sprintf("%d", info.(*eth.EthNodeInfo).Network)
		protocol = fmt.Sprintf("eth/%d", eth.ProtocolVersions[0])
	} else {
		network = fmt.Sprintf("%d", infos.Protocols["les"].(*eth.EthNodeInfo).Network)
		protocol = fmt.Sprintf("les/%d", les.ProtocolVersions[0])
	}
	auth := &authMsg{
		Id: s.node,
		Info: nodeInfo{
			Name:     s.node,
			Node:     infos.Name,
			Port:     infos.Ports.Listener,
			Network:  network,
			Protocol: protocol,
			API:      "No",
			Os:       runtime.GOOS,
			OsVer:    runtime.GOARCH,
			Client:   "0.1.1",
			History:  true,
		},
		Secret:        s.pass,
		ClientTime:    clientTime,
		NodeId:        infos.ID,
		ValidatorAddr: validatorAddr,
		Signature:     signature,
		GenesisHash:   genesisHash,
	}
	login := map[string][]interface{}{
		"emit": {"hello", auth},
	}
	if err := s.doSendReportData(conn, login); err != nil {
		return err
	}
	// Retrieve the remote ack or connection termination
	var ack map[string][]string
	if err := websocket.JSON.Receive(conn, &ack); err != nil || len(ack["emit"]) != 1 || ack["emit"][0] != "ready" {
		return errors.New("unauthorized")
	}
	return nil
}

// report collects all possible data to report and send it to the stats server.
// This should only be used on reconnects or rarely to avoid overloading the
// server. Use the individual methods for reporting subscribed events.
func (s *Service) report(conn *websocket.Conn) error {
	var err error
	if err = s.reportLatency(conn); err != nil {
		return err
	}

	err = s.reportBlock(conn, nil)
	if err != nil {
		return err
	}

	if err = s.reportPending(conn); err != nil {
		return err
	}

	err = s.reportStats(conn)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) reportPos(conn *websocket.Conn) error {
	var err error
	if err = s.reportLatency(conn); err != nil {
		return err
	}

	err = s.reportPosBlock(conn, nil)
	if err != nil {
		return err
	}

	if err = s.reportPending(conn); err != nil {
		return err
	}

	err = s.reportPosStats(conn)
	if err != nil {
		return err
	}

	if err := s.reportPosLog(conn); err != nil {
		return err
	}

	oldEpochId := s.updateEpochId()
	if oldEpochId != s.epochId {
		if err := s.reportLeader(conn); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) reportLeader(conn *websocket.Conn) error {
	// Gather the block details from the header or block chain
	if s.api == nil {
		return nil
	}

	el, err := s.api.GetEpochLeadersAddrByEpochID(s.epochId)
	if err != nil {
		return err
	}

	rnpl, err := s.api.GetRandomProposersAddrByEpochID(s.epochId)
	if err != nil {
		return err
	}

	preEpBlkCnt := uint64(0)
	if s.epochId > 0 {
		preEpBlkCnt, _ = s.api.GetEpochBlkCnt(s.epochId - 1)
	}

	posL := pos_leader{s.epochId, el, rnpl, preEpBlkCnt}
	stats := map[string]interface{}{
		"id":         s.node,
		"pos-leader": posL,
	}
	report := map[string][]interface{}{
		"emit": {"pos-leader", stats},
	}
	return s.doSendReportData(conn, report)
}

// reportLatency sends a ping request to the server, measures the RTT time and
// finally sends a latency update.
func (s *Service) reportLatency(conn *websocket.Conn) error {
	// Send the current time to the ethstats server
	start := time.Now()

	ping := map[string][]interface{}{
		"emit": {"node-ping", map[string]string{
			"id":         s.node,
			"clientTime": start.String(),
		}},
	}
	if err := s.doSendReportData(conn, ping); err != nil {
		return err
	}
	// Wait for the pong request to arrive back
	select {
	case <-s.pongCh:
		// Pong delivered, report the latency
	case <-time.After(5 * time.Second):
		// Ping timeout, abort
		return errors.New("ping timed out")
	}
	latency := strconv.Itoa(int((time.Since(start) / time.Duration(2)).Nanoseconds() / 1000000))

	// Send back the measured latency
	log.Trace("Sending measured latency to ethstats", "latency", latency)

	stats := map[string][]interface{}{
		"emit": {"latency", map[string]string{
			"id":      s.node,
			"latency": latency,
		}},
	}
	return s.doSendReportData(conn, stats)
}

// blockStats is the information to report about individual blocks.
type blockStats struct {
	Number     *big.Int       `json:"number"`
	Hash       common.Hash    `json:"hash"`
	ParentHash common.Hash    `json:"parentHash"`
	Timestamp  *big.Int       `json:"timestamp"`
	Miner      common.Address `json:"miner"`
	GasUsed    *big.Int       `json:"gasUsed"`
	GasLimit   *big.Int       `json:"gasLimit"`
	Diff       string         `json:"difficulty"`
	TotalDiff  string         `json:"totalDifficulty"`
	Txs        []txStats      `json:"transactions"`
	TxHash     common.Hash    `json:"transactionsRoot"`
	Root       common.Hash    `json:"stateRoot"`
	Uncles     uncleStats     `json:"uncles"`
}

type pos_blockStats struct {
	Number     *big.Int       `json:"number"`
	Hash       common.Hash    `json:"hash"`
	ParentHash common.Hash    `json:"parentHash"`
	Timestamp  *big.Int       `json:"timestamp"`
	Miner      common.Address `json:"miner"`
	GasUsed    *big.Int       `json:"gasUsed"`
	GasLimit   *big.Int       `json:"gasLimit"`
	Txs        []txStats      `json:"transactions"`
	TxHash     common.Hash    `json:"transactionsRoot"`
	Root       common.Hash    `json:"stateRoot"`
}

type pos_leader struct {
	EpochId        uint64           `json:"epochId"`
	ELList         []common.Address `json:"elList"`
	RNPList        []common.Address `json:"rnpList"`
	PreEpochBlkCnt uint64           `json:"preEpochBlkCnt"`
}

type pos_reorg struct {
	EpochId uint64 `json:"epochId"`
	SlotId  uint64 `json:"slotId"`
	Len     uint64 `json:"len"`
}

type pos_log struct {
	WarnCnt  uint64 `json:"warnCnt"`
	ErrorCnt uint64 `json:"errorCnt"`
}

type pos_alarm struct {
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

// txStats is the information to report about individual transactions.
type txStats struct {
	Hash common.Hash `json:"hash"`
}

// uncleStats is a custom wrapper around an uncle array to force serializing
// empty arrays instead of returning null for them.
type uncleStats []*types.Header

func (s uncleStats) MarshalJSON() ([]byte, error) {
	if uncles := ([]*types.Header)(s); len(uncles) > 0 {
		return json.Marshal(uncles)
	}
	return []byte("[]"), nil
}

// reportBlock retrieves the current chain head and repors it to the stats server.
func (s *Service) reportBlock(conn *websocket.Conn, block *types.Block) error {
	// Gather the block details from the header or block chain
	details := s.assembleBlockStats(block)
	if details.Number.Uint64() == 0 {
		return nil
	}

	// Assemble the block report and send it to the server
	log.Trace("Sending new block to ethstats", "number", details.Number, "hash", details.Hash)

	stats := map[string]interface{}{
		"id":    s.node,
		"block": details,
	}
	report := map[string][]interface{}{
		"emit": {"block", stats},
	}
	return s.doSendReportData(conn, report)
}

// reportBlock retrieves the current chain head and repors it to the stats server.
func (s *Service) reportPosBlock(conn *websocket.Conn, block *types.Block) error {
	// Gather the block details from the header or block chain
	details := s.assemblePosBlockStats(block)
	if details.Number.Uint64() == 0 {
		return nil
	}

	// Assemble the block report and send it to the server
	log.Trace("Sending new block to ethstats", "number", details.Number, "hash", details.Hash)

	stats := map[string]interface{}{
		"id":        s.node,
		"pos-block": details,
	}
	report := map[string][]interface{}{
		"emit": {"pos-block", stats},
	}
	return s.doSendReportData(conn, report)
}

// assembleBlockStats retrieves any required metadata to report a single block
// and assembles the block stats. If block is nil, the current head is processed.
func (s *Service) assembleBlockStats(block *types.Block) *blockStats {
	// Gather the block infos from the local blockchain
	var (
		header *types.Header
		td     *big.Int
		txs    []txStats
		uncles []*types.Header
	)
	if s.eth != nil {
		// Full nodes have all needed information available
		if block == nil {
			block = s.eth.BlockChain().CurrentBlock()
		}
		header = block.Header()
		td = s.eth.BlockChain().GetTd(header.Hash(), header.Number.Uint64())

		txs = make([]txStats, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			txs[i].Hash = tx.Hash()
		}
		uncles = block.Uncles()
	} else {
		// Light nodes would need on-demand lookups for transactions/uncles, skip
		if block != nil {
			header = block.Header()
		} else {
			header = s.les.BlockChain().CurrentHeader()
		}
		td = s.les.BlockChain().GetTd(header.Hash(), header.Number.Uint64())
		txs = []txStats{}
	}
	// Assemble and return the block stats
	author, _ := s.engine.Author(header)

	return &blockStats{
		Number:     header.Number,
		Hash:       header.Hash(),
		ParentHash: header.ParentHash,
		Timestamp:  header.Time,
		Miner:      author,
		GasUsed:    new(big.Int).Set(header.GasUsed),
		GasLimit:   new(big.Int).Set(header.GasLimit),
		Diff:       header.Difficulty.String(),
		TotalDiff:  td.String(),
		Txs:        txs,
		TxHash:     header.TxHash,
		Root:       header.Root,
		Uncles:     uncles,
	}
}

func (s *Service) assemblePosBlockStats(block *types.Block) *pos_blockStats {
	// Gather the block infos from the local blockchain
	var (
		header *types.Header
		txs    []txStats
	)
	if s.eth != nil {
		// Full nodes have all needed information available
		if block == nil {
			block = s.eth.BlockChain().CurrentBlock()
		}
		header = block.Header()

		txs = make([]txStats, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			txs[i].Hash = tx.Hash()
		}
	} else {
		// Light nodes would need on-demand lookups for transactions/uncles, skip
		if block != nil {
			header = block.Header()
		} else {
			header = s.les.BlockChain().CurrentHeader()
		}
		txs = []txStats{}
	}
	// Assemble and return the block stats
	author, _ := s.engine.Author(header)

	return &pos_blockStats{
		Number:     header.Number,
		Hash:       header.Hash(),
		ParentHash: header.ParentHash,
		Timestamp:  header.Time,
		Miner:      author,
		GasUsed:    new(big.Int).Set(header.GasUsed),
		GasLimit:   new(big.Int).Set(header.GasLimit),
		Txs:        txs,
		TxHash:     header.TxHash,
		Root:       header.Root,
	}
}

// reportHistory retrieves the most recent batch of blocks and reports it to the
// stats server.
func (s *Service) reportHistory(conn *websocket.Conn, list []uint64) error {
	// Figure out the indexes that need reporting
	indexes := make([]uint64, 0, historyUpdateRange)
	if len(list) > 0 {
		// Specific indexes requested, send them back in particular
		indexes = append(indexes, list...)
	} else {
		// No indexes requested, send back the top ones
		var head int64
		if s.eth != nil {
			head = s.eth.BlockChain().CurrentHeader().Number.Int64()
		} else {
			head = s.les.BlockChain().CurrentHeader().Number.Int64()
		}
		start := head - historyUpdateRange + 1
		if start < 0 {
			start = 0
		}
		for i := uint64(start); i <= uint64(head); i++ {
			indexes = append(indexes, i)
		}
	}
	// Gather the batch of blocks to report
	history := make([]*blockStats, len(indexes))
	for i, number := range indexes {
		// Retrieve the next block if it's known to us
		var block *types.Block
		if s.eth != nil {
			block = s.eth.BlockChain().GetBlockByNumber(number)
		} else {
			if header := s.les.BlockChain().GetHeaderByNumber(number); header != nil {
				block = types.NewBlockWithHeader(header)
			}
		}
		// If we do have the block, add to the history and continue
		if block != nil {
			history[len(history)-1-i] = s.assembleBlockStats(block)
			continue
		}
		// Ran out of blocks, cut the report short and send
		history = history[len(history)-i:]
		break
	}
	// Assemble the history report and send it to the server
	if len(history) > 0 {
		log.Trace("Sending historical blocks to ethstats", "first", history[0].Number, "last", history[len(history)-1].Number)
	} else {
		log.Trace("No history to send to stats server")
	}
	stats := map[string]interface{}{
		"id":      s.node,
		"history": history,
	}
	report := map[string][]interface{}{
		"emit": {"history", stats},
	}
	return s.doSendReportData(conn, report)
}

func (s *Service) reportPosHistory(conn *websocket.Conn, list []uint64) error {
	// Figure out the indexes that need reporting
	indexes := make([]uint64, 0, historyUpdateRange)
	if len(list) > 0 {
		// Specific indexes requested, send them back in particular
		indexes = append(indexes, list...)
	} else {
		// No indexes requested, send back the top ones
		var head int64
		if s.eth != nil {
			head = s.eth.BlockChain().CurrentHeader().Number.Int64()
		} else {
			head = s.les.BlockChain().CurrentHeader().Number.Int64()
		}
		start := head - historyUpdateRange + 1
		if start < 0 {
			start = 0
		}
		for i := uint64(start); i <= uint64(head); i++ {
			indexes = append(indexes, i)
		}
	}
	// Gather the batch of blocks to report
	history := make([]*pos_blockStats, len(indexes))
	for i, number := range indexes {
		// Retrieve the next block if it's known to us
		var block *types.Block
		if s.eth != nil {
			block = s.eth.BlockChain().GetBlockByNumber(number)
		} else {
			if header := s.les.BlockChain().GetHeaderByNumber(number); header != nil {
				block = types.NewBlockWithHeader(header)
			}
		}
		// If we do have the block, add to the history and continue
		if block != nil {
			history[len(history)-1-i] = s.assemblePosBlockStats(block)
			continue
		}
		// Ran out of blocks, cut the report short and send
		history = history[len(history)-i:]
		break
	}
	// Assemble the history report and send it to the server
	if len(history) > 0 {
		log.Trace("Sending historical blocks to ethstats", "first", history[0].Number, "last", history[len(history)-1].Number)
	} else {
		log.Trace("No history to send to stats server")
	}
	stats := map[string]interface{}{
		"id":          s.node,
		"pos-history": history,
	}
	report := map[string][]interface{}{
		"emit": {"pos-history", stats},
	}
	return s.doSendReportData(conn, report)
}

// pendStats is the information to report about pending transactions.
type pendStats struct {
	Pending int `json:"pending"`
}

// reportPending retrieves the current number of pending transactions and reports
// it to the stats server.
func (s *Service) reportPending(conn *websocket.Conn) error {
	// Retrieve the pending count from the local blockchain
	var pending int
	if s.eth != nil {
		pending, _ = s.eth.TxPool().Stats()
	} else {
		pending = s.les.TxPool().Stats()
	}
	// Assemble the transaction stats and send it to the server
	log.Trace("Sending pending transactions to ethstats", "count", pending)

	stats := map[string]interface{}{
		"id": s.node,
		"stats": &pendStats{
			Pending: pending,
		},
	}
	report := map[string][]interface{}{
		"emit": {"pending", stats},
	}
	return s.doSendReportData(conn, report)
}

// nodeStats is the information to report about the local node.
type nodeStats struct {
	Active   bool `json:"active"`
	Syncing  bool `json:"syncing"`
	Mining   bool `json:"mining"`
	Hashrate int  `json:"hashrate"`
	Peers    int  `json:"peers"`
	GasPrice int  `json:"gasPrice"`
	Uptime   int  `json:"uptime"`
}

type pos_nodeStats struct {
	Active          bool   `json:"active"`
	Syncing         bool   `json:"syncing"`
	Mining          bool   `json:"mining"`
	Peers           int    `json:"peers"`
	GasPrice        int    `json:"gasPrice"`
	Uptime          int    `json:"uptime"`
	EpochId         uint64 `json:"epochId"`
	SlotId          uint64 `json:"slotId"`
	ChainQuality    string `json:"chainQuality"`
	EpBlockCount    uint64 `json:"epBlockCount"`
	CurRandom       string `json:"curRandom"`
	NextRandom      string `json:"nextRandom"`
	CurRBStage      uint64 `json:"curRbStage"`
	ValidDKG1Cnt    uint64 `json:"validDkg1Cnt"`
	ValidDKG2Cnt    uint64 `json:"validDkg2Cnt"`
	ValidSIGCnt     uint64 `json:"validSigCnt"`
	CurSLStage      uint64 `json:"curSlStage"`
	ValidSMA1Cnt    uint64 `json:"validSma1Cnt"`
	ValidSMA2Cnt    uint64 `json:"validSma2Cnt"`
	SelfMinedBlks   uint64 `json:"selfMinedBlks"`
	SelfELActivity  uint64 `json:"selfElActivity"`
	SelfRNPActivity uint64 `json:"selfRnpActivity"`
}

// reportPending retrieves various stats about the node at the networking and
// mining layer and reports it to the stats server.
func (s *Service) reportStats(conn *websocket.Conn) error {
	// Gather the syncing and mining infos from the local miner instance
	var (
		mining   bool
		hashrate int
		syncing  bool
		gasprice int
	)
	if s.eth != nil {
		mining = s.eth.Miner().Mining()
		hashrate = int(s.eth.Miner().HashRate())

		sync := s.eth.Downloader().Progress()
		syncing = s.eth.BlockChain().CurrentHeader().Number.Uint64() >= sync.HighestBlock

		price, _ := s.eth.ApiBackend.SuggestPrice(context.Background())
		gasprice = int(price.Uint64())
	} else {
		sync := s.les.Downloader().Progress()
		syncing = s.les.BlockChain().CurrentHeader().Number.Uint64() >= sync.HighestBlock
	}
	// Assemble the node stats and send it to the server
	log.Trace("Sending node details to ethstats")

	stats := map[string]interface{}{
		"id": s.node,
		"stats": &nodeStats{
			Active:   true,
			Mining:   mining,
			Hashrate: hashrate,
			Peers:    s.server.PeerCount(),
			GasPrice: gasprice,
			Syncing:  syncing,
			Uptime:   100,
		},
	}
	report := map[string][]interface{}{
		"emit": {"stats", stats},
	}
	return s.doSendReportData(conn, report)
}

func (s *Service) reportPosStats(conn *websocket.Conn) error {
	// Gather the syncing and mining infos from the local miner instance
	var (
		err      error
		mining   bool
		syncing  bool
		gasprice int

		epochId         uint64
		slotId          uint64
		iChainQuality   uint64
		chainQuality    string
		epBlockCount    uint64
		curRandom       string
		nextRandom      string
		curRbStage      uint64
		validDkg1Cnt    uint64
		validDkg2Cnt    uint64
		validSigCnt     uint64
		curSlStage      uint64
		validSma1Cnt    uint64
		validSma2Cnt    uint64
		selfMinedBlks   uint64
		selfElActivity  uint64
		selfRnpActivity uint64
	)
	if s.eth != nil {
		mining = s.eth.Miner().Mining()

		sync := s.eth.Downloader().Progress()
		syncing = s.eth.BlockChain().CurrentHeader().Number.Uint64() >= sync.HighestBlock

		price, _ := s.eth.ApiBackend.SuggestPrice(context.Background())
		gasprice = int(price.Uint64())

		if s.api != nil {
			epochId = s.api.GetEpochID()
			slotId = s.api.GetSlotID()

			blockNum := s.eth.BlockChain().CurrentHeader().Number.Uint64()
			iChainQuality, err = s.api.GetChainQuality(epochId, slotId)
			if err != nil {
				log.Error("get chain quality fail", "blocknumber", blockNum, "err", err)
			} else {
				chainQuality = fmt.Sprintf("%.1f", float64(iChainQuality)/10.0)
			}

			epBlockCount, _ = s.api.GetEpochBlkCnt(epochId)
			curR, err := s.api.GetRandom(epochId, -1)
			if err == nil {
				curRandom = common.ToHex(curR.Bytes())
			}

			nextR, err := s.api.GetRandom(epochId+1, -1)
			if err == nil {
				nextRandom = common.ToHex(nextR.Bytes())
			}

			curRbStage = s.api.GetRbStage(slotId)
			validDkgCnts, _ := s.api.GetValidRBCnt(epochId)
			if len(validDkgCnts) == 3 {
				validDkg1Cnt = validDkgCnts[0]
				validDkg2Cnt = validDkgCnts[1]
				validSigCnt = validDkgCnts[2]
			}

			curSlStage = s.api.GetSlStage(slotId)
			validSmaCnts, _ := s.api.GetValidSMACnt(epochId)
			if len(validSmaCnts) == 2 {
				validSma1Cnt = validSmaCnts[0]
				validSma2Cnt = validSmaCnts[1]
			}
			selfMinedBlks, selfElActivity, selfRnpActivity = s.getSelfActivity(s.api, epochId)
		}

	} else {
		sync := s.les.Downloader().Progress()
		syncing = s.les.BlockChain().CurrentHeader().Number.Uint64() >= sync.HighestBlock
	}
	// Assemble the node stats and send it to the server
	log.Trace("Sending node details to ethstats")

	stats := map[string]interface{}{
		"id": s.node,
		"pos-stats": &pos_nodeStats{
			Active:          true,
			Mining:          mining,
			Peers:           s.server.PeerCount(),
			GasPrice:        gasprice,
			Syncing:         syncing,
			Uptime:          100,
			EpochId:         epochId,
			SlotId:          slotId,
			ChainQuality:    chainQuality,
			EpBlockCount:    epBlockCount,
			CurRandom:       curRandom,
			NextRandom:      nextRandom,
			CurRBStage:      curRbStage,
			ValidDKG1Cnt:    validDkg1Cnt,
			ValidDKG2Cnt:    validDkg2Cnt,
			ValidSIGCnt:     validSigCnt,
			CurSLStage:      curSlStage,
			ValidSMA1Cnt:    validSma1Cnt,
			ValidSMA2Cnt:    validSma2Cnt,
			SelfMinedBlks:   selfMinedBlks,
			SelfELActivity:  selfElActivity,
			SelfRNPActivity: selfRnpActivity,
		},
	}
	report := map[string][]interface{}{
		"emit": {"pos-stats", stats},
	}
	return s.doSendReportData(conn, report)
}

func (s *Service) getSelfActivity(api *posapi.PosApi, epochId uint64) (uint64, uint64, uint64) {
	minedBlks, elActivity, rnpActivity := uint64(0), uint64(0), uint64(0)
	if api == nil {
		return minedBlks, elActivity, rnpActivity
	}

	activity, err := api.GetActivity(epochId)
	if err != nil {
		log.Error("posapi get activity fail", "err", err)
		return minedBlks, elActivity, rnpActivity
	}

	selfAddr := posconfig.Cfg().GetMinerAddr()
	if (selfAddr == common.Address{}) {
		return minedBlks, elActivity, rnpActivity
	}

	for i := range activity.EpLeader {
		if activity.EpLeader[i] == selfAddr {
			elActivity += uint64(activity.EpActivity[i])
		}
	}

	for i := range activity.RpLeader {
		if activity.RpLeader[i] == selfAddr {
			rnpActivity += uint64(activity.RpActivity[i])
		}
	}

	for i := range activity.SltLeader {
		if activity.SltLeader[i] == selfAddr {
			minedBlks += uint64(activity.SlBlocks[i])
		}
	}

	return minedBlks, elActivity, rnpActivity
}

func (s *Service) doSendReportData(conn *websocket.Conn, v interface{}) error {
	log.Debug("wanstats send report data", "data", v)
	return websocket.JSON.Send(conn, v)
}

// return old epochId
func (s *Service) updateEpochId() uint64 {
	if s.api == nil {
		return s.epochId
	}

	epochId := s.api.GetEpochIDByTime(uint64(time.Now().Unix()))
	oldEpochId := s.epochId
	s.epochId = epochId
	return oldEpochId
}

func log2PosAlarm(plog *log.LogInfo) pos_alarm {
	if plog == nil {
		return pos_alarm{}
	}

	lvlStr := ""
	switch plog.Lvl {
	case log.LOG_EMERG:
		lvlStr = "EMERG"
	case log.LOG_ALERT:
		lvlStr = "ALERT"
	case log.LOG_CRIT:
		lvlStr = "CRIT"
	case log.LOG_ERR:
		lvlStr = "ERR"
	case log.LOG_WARNING:
		lvlStr = "WARNING"
	case log.LOG_NOTICE:
		lvlStr = "NOTICE"
	case log.LOG_INFO:
		lvlStr = "INFO"
	case log.LOG_DEBUG:
		lvlStr = "DEBUG"
	}

	return pos_alarm{lvlStr, plog.Msg}
}
