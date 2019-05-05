package randombeacon

import (
	"crypto/rand"
	"errors"
	"github.com/wanchain/go-wanchain/pos/rbselection"
	"sync"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"

	"math/big"

	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"
)

type RbEnsDataCollector struct {
	ens []*bn256.G1
	pk  *bn256.G1
}

type RbSIGDataCollector struct {
	data *vm.RbSIGTxPayload
	pk   *bn256.G1
}

type GetRBProposerGroupFunc func(epochId uint64) []bn256.G1
type GetCji func(db vm.StateDB, epochId uint64, proposerId uint32) ([]*bn256.G2, error)
type GetEnsFunc func(db vm.StateDB, epochId uint64, proposerId uint32) ([]*bn256.G1, error)
type GetRBMFunc func(db vm.StateDB, epochId uint64) ([]byte, error)

type LoopEvent struct {
	statedb vm.StateDB
	rc      *rpc.Client
	eid     uint64
	sid     uint64
}

type PolyInfo struct {
	poly rbselection.Polynomial
	s    *big.Int
}

type PolyMap map[uint32]PolyInfo
type TaskTags []bool

type RandomBeacon struct {
	loopEvents chan *LoopEvent

	epochStage   int
	epochId      uint64
	polys        PolyMap
	taskTags     TaskTags
	proposerPks  []bn256.G1
	myPropserIds []uint32

	statedb   vm.StateDB
	epocher   *epochLeader.Epocher
	rpcClient *rpc.Client

	wg sync.WaitGroup
	mutex sync.Mutex

	// based function
	getRBProposerGroupF GetRBProposerGroupFunc
	getCji              GetCji
	getEns              GetEnsFunc
	getRBM              GetRBMFunc
}

var (
	maxUint64      = uint64(1<<64 - 1)
	loopEventCount = 1000
	randomBeacon   RandomBeacon
)

var (
	errInvalidInParam  = errors.New("invalid input param")
	errEpochIdRollback = errors.New("epoch id rollback")
	errNoDKG1Data      = errors.New("no dkg1 data")
	errNoDKG1Poly      = errors.New("no dkg1 random polynomial")
	errInsufficient    = errors.New("insufficient proposer")
	errUninitialized   = errors.New("random beacon uninitialized")
)

func GetRandonBeaconInst() *RandomBeacon {
	return &randomBeacon
}

func (rb *RandomBeacon) Init(epocher *epochLeader.Epocher) {
	defer func() {
		rb.mutex.Unlock()
	}()

	rb.mutex.Lock()
	if rb.loopEvents != nil {
		return
	}

	rb.epochStage = vm.RbDkg1Stage
	rb.epochId = maxUint64
	rb.polys = make(PolyMap)
	rb.rpcClient = nil

	rb.epocher = epocher

	// function
	rb.getRBProposerGroupF = epochLeader.GetEpocher().GetRBProposerG1
	rb.getCji = vm.GetCji
	rb.getEns = vm.GetEncryptShare
	rb.getRBM = vm.GetRBM

	rb.loopEvents = make(chan *LoopEvent, loopEventCount)

	rb.wg.Add(1)
	go rb.LoopRoutine()
}

func (rb *RandomBeacon) Stop() {
	defer func() {
		rb.mutex.Unlock()
	}()

	rb.mutex.Lock()
	if rb.loopEvents == nil {
		return
	}

	close(rb.loopEvents)
	rb.wg.Wait()
	rb.loopEvents = nil
}

func (rb *RandomBeacon) Loop(statedb vm.StateDB, rc *rpc.Client, eid uint64, sid uint64) (err error) {
	defer func() {
		rb.mutex.Unlock()
		if e := recover(); e != nil {
			err = e.(error)
			log.SyslogErr("RB loop panic", "err", err.Error())
		}
	}()

	rb.mutex.Lock()
	if rb.loopEvents == nil {
		return errUninitialized
	}

	if statedb == nil || rc == nil {
		log.SyslogErr("invalid RB loop input param")
		return errInvalidInParam
	}

	rb.loopEvents <- &LoopEvent{statedb, rc, eid, sid}
	return
}

func (rb *RandomBeacon) LoopRoutine() {
	defer rb.wg.Done()

	for {
		event, ok := <-rb.loopEvents
		if !ok {
			break
		}

		rb.doLoop(event.statedb, event.rc, event.eid, event.sid)
	}
}

func (rb *RandomBeacon) updateEpochId(epochId uint64) {
	log.SyslogInfo("rb update epochId", "epochId", epochId)
	rb.epochId = epochId
	rb.myPropserIds = rb.getMyRBProposerId(epochId)

	// reset state
	rb.epochStage = vm.RbDkg1Stage
	rb.polys = make(PolyMap)
	rb.taskTags = nil
}

func (rb *RandomBeacon) updateStage(stage int) {
	rb.epochStage = stage
	rb.taskTags = nil
}

func (rb *RandomBeacon) doLoop(statedb vm.StateDB, rc *rpc.Client, epochId uint64, slotId uint64) error {
	log.SyslogInfo("rb doLoop begin", "epochId", epochId, "slotId", slotId, "self epochId", rb.epochId)
	rb.statedb = statedb
	rb.rpcClient = rc

	if rb.epochId != maxUint64 && rb.epochId > epochId {
		log.SyslogErr("RB doloop fail", "err", errEpochIdRollback.Error())
		return errEpochIdRollback
	}

	if rb.epochId == maxUint64 || rb.epochId < epochId {
		rb.updateEpochId(epochId)
	}

	// rb.epochId == epochId
	if len(rb.myPropserIds) == 0 {
		return nil
	}

	rbStage, elapsedNum, leftNum := vm.GetRBStage(slotId)

	log.SyslogInfo("get my RB proposer id", "ids", rb.myPropserIds)
	log.SyslogInfo("get RB stage", "rbStage", rbStage, "elapsedNum", elapsedNum, "leftNum", leftNum)

	// belong to RB proposer group
	for {
		switch rb.epochStage {
		case vm.RbDkg1Stage:
			if rbStage == vm.RbDkg1Stage {
				// don't send tx in first and last slot
				if elapsedNum == 0 || leftNum == 0 {
					return nil
				}

				if rb.taskTags == nil {
					rb.taskTags = make(TaskTags, len(rb.myPropserIds))
				}

				err := rb.doDKG1s()
				if err != nil {
					return err
				}
			}

			rb.updateStage(vm.RbDkg2Stage)
		case vm.RbDkg2Stage:
			if rbStage < vm.RbDkg2Stage {
				return nil
			} else if rbStage == vm.RbDkg2Stage {
				// don't send tx in first and last slot
				if elapsedNum == 0 || leftNum == 0 {
					return nil
				}

				if rb.taskTags == nil {
					rb.taskTags = make(TaskTags, len(rb.myPropserIds))
				}

				err := rb.doDKG2s()
				// while errNoDKG1Poly, noneed to retry, stop dkg2 work
				if err != nil && err != errNoDKG1Poly {
					return err
				}
			}

			rb.updateStage(vm.RbSignStage)
		case vm.RbSignStage:
			if rbStage < vm.RbSignStage {
				return nil
			} else if rbStage == vm.RbSignStage {
				// don't send tx in first and last slot
				if elapsedNum == 0 || leftNum == 0 {
					return nil
				}

				if rb.taskTags == nil {
					rb.taskTags = make(TaskTags, len(rb.myPropserIds))
				}

				err := rb.doSIGs()

				// while errInsufficient, noneed to retry, stop SIG work
				if err != nil && err != errInsufficient {
					return err
				}
			}

			rb.updateStage(vm.RbSignConfirmStage)
		default:
			// RbSignConfirmStage
			return nil
		}
	}

	return nil
}

func (rb *RandomBeacon) getMyRBProposerId(epochId uint64) []uint32 {
	pks := rb.getRBProposerGroupF(epochId)
	rb.proposerPks = pks
	if len(pks) == 0 {
		return nil
	}

	selfPk := posconfig.Cfg().GetMinerBn256PK()
	if selfPk == nil {
		return nil
	}

	ids := make([]uint32, 0)
	for i, pk := range pks {
		if pk.String() == selfPk.String() {
			ids = append(ids, uint32(i))
		}
	}

	return ids
}

func (rb *RandomBeacon) doDKG1s() error {
	for i, id := range rb.myPropserIds {
		if rb.taskTags[i] {
			continue
		}

		err := rb.doDKG1(id)
		if err == nil {
			rb.taskTags[i] = true
		} else {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) doDKG1(proposerId uint32) error {
	log.SyslogInfo("begin do dkg1", "proposerId", proposerId)
	txPayload, err := rb.generateDKG1(proposerId)
	if err != nil {
		return err
	}

	return rb.sendDKG1(txPayload)
}

func (rb *RandomBeacon) generateDKG1(proposerId uint32) (*vm.RbDKG1FlatTxPayload, error) {
	nr := len(rb.proposerPks)

	// fix the evaluation point: Hash(Pub[1]+1), Hash(Pub[2]+2), ..., Hash(Pub[Nr]+Nr)
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&rb.proposerPks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	sshare := make([]big.Int, nr)

	// fi(x)
	s, err := rand.Int(rand.Reader, bn256.Order)
	if err != nil {
		log.SyslogErr("dkg1, get rand fail", "err", err.Error())
		return nil, err
	}

	poly := rbselection.RandPoly(int(posconfig.Cfg().PolymDegree), *s)
	rb.polys[proposerId] = PolyInfo{poly, s}
	for i := 0; i < nr; i++ {
		// share for i is fi(x) evaluation result on x[i]
		sshare[i], _ = rbselection.EvaluatePoly(poly, &x[i], int(posconfig.Cfg().PolymDegree))
	}

	// make commitment for the secret share, i.e. multiply with the generator of G2
	commit := make([]*bn256.G2, nr)
	for i := 0; i < nr; i++ {
		// commit[i] = sshare[i] * G2
		commit[i] = new(bn256.G2).ScalarBaseMult(&sshare[i])
	}

	commitBytes := make([][]byte, nr)
	for i := 0; i < nr; i++ {
		commitBytes[i] = commit[i].Marshal()
	}

	txPayload := vm.RbDKG1FlatTxPayload{rb.epochId, proposerId, commitBytes}

	return &txPayload, nil
}

func (rb *RandomBeacon) doDKG2s() error {
	// random proposer group
	for i, id := range rb.myPropserIds {
		if rb.taskTags[i] {
			continue
		}

		err := rb.doDKG2(id)
		if err == nil || err == errNoDKG1Data {
			rb.taskTags[i] = true
		} else {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) doDKG2(proposerId uint32) error {
	log.SyslogInfo("begin do dkg2", "proposerId", proposerId)
	txPayload, err := rb.generateDKG2(proposerId)
	if err != nil {
		return err
	}

	return rb.sendDKG2(txPayload)
}

func (rb *RandomBeacon) generateDKG2(proposerId uint32) (*vm.RbDKG2FlatTxPayload, error) {
	nr := len(rb.proposerPks)

	// check dkg1
	commit, err := rb.getCji(rb.statedb, rb.epochId, proposerId)
	if err != nil || len(commit) == 0 {
		log.SyslogErr("generate DKG2 payload fail", "err", errNoDKG1Data.Error())
		return nil, errNoDKG1Data
	}

	// fix the evaluation point: Hash(Pub[1]+1), Hash(Pub[2]+2), ..., Hash(Pub[Nr]+Nr)
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&rb.proposerPks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	sshare := make([]big.Int, nr)

	// fi(x)
	if rb.polys[proposerId].s == nil || rb.polys[proposerId].poly == nil {
		log.SyslogErr("generate DKG2 payload fail", "err", errNoDKG1Poly.Error())
		return nil, errNoDKG1Poly
	}

	poly := rb.polys[proposerId].poly
	for i := 0; i < nr; i++ {
		// share for i is fi(x) evaluation result on x[i]
		sshare[i], _ = rbselection.EvaluatePoly(poly, &x[i], int(posconfig.Cfg().PolymDegree))
	}

	// encrypt the secret share, i.e. multiply with the receiver's public key
	enshare := make([]*bn256.G1, nr)
	for i := 0; i < nr; i++ {
		// enshare[i] = sshare[i]*Pub[i], it is a point on ECC
		enshare[i] = new(bn256.G1).ScalarMult(&rb.proposerPks[i], &sshare[i])
	}

	// generate DLEQ proof
	proof := make([]rbselection.DLEQproof, nr)
	for i := 0; i < nr; i++ {
		// proof = (a1, a2, z)
		proof[i] = rbselection.DLEQ(rb.proposerPks[i], *rbselection.Hbase, &sshare[i])
	}

	enshareBytes := make([][]byte, nr)
	proofBytes := make([]rbselection.DLEQproofFlat, nr)
	for i := 0; i < nr; i++ {
		enshareBytes[i] = enshare[i].Marshal()
		proofBytes[i] = rbselection.ProofToProofFlat(&proof[i])
	}

	txPayload := vm.RbDKG2FlatTxPayload{rb.epochId, proposerId, enshareBytes, proofBytes}

	return &txPayload, nil
}

func (rb *RandomBeacon) doSIGs() error {
	for i, id := range rb.myPropserIds {
		if rb.taskTags[i] {
			continue
		}

		// try the best to send every SIG tx, prevent that one error stop all left task
		err := rb.doSIG(id)
		if err == nil {
			rb.taskTags[i] = true
		} else {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) doSIG(proposerId uint32) error {
	log.SyslogInfo("do sig begin", "proposerId", proposerId)
	sig, err := rb.generateSIG(proposerId)
	if err != nil {
		return err
	}

	return rb.sendSIG(sig)
}

func (rb *RandomBeacon) generateSIG(proposerId uint32) (*vm.RbSIGTxPayload, error) {
	prikey := posconfig.Cfg().GetMinerBn256SK()
	datas := make([]RbEnsDataCollector, 0)

	for id, pk := range rb.proposerPks {
		data, err := rb.getEns(rb.statedb, rb.epochId, uint32(id))
		if err == nil && data != nil {
			datas = append(datas, RbEnsDataCollector{data, &pk})
		} else {
			// do nothing
		}
	}

	dkgCount := len(datas)
	log.SyslogInfo("collecte dkg", "count", dkgCount)

	if uint(dkgCount) < posconfig.Cfg().RBThres {
		log.SyslogErr("generate sig fail", "err", errInsufficient.Error())
		return nil, errInsufficient
	}

	// Compute Group Secret Key Share
	// Random proposers get information from the blockchain and compute its group secret share.

	//set zero
	gskshare := new(bn256.G1).ScalarBaseMult(big.NewInt(int64(0)))

	// sk^-1
	skinver := new(big.Int).ModInverse(prikey, bn256.Order)
	for i := 0; i < dkgCount; i++ {
		temp := new(bn256.G1).ScalarMult(datas[i].ens[proposerId], skinver)

		// gskshare[i] = (sk^-1)*(enshare[1][i]+...+enshare[Nr][i])
		gskshare.Add(gskshare, temp)
	}

	// Signing Stage
	// In this stage, each random proposer computes its signature share and sends it on chain.
	mBuf, err := rb.getRBM(rb.statedb, rb.epochId)
	if err != nil {
		return nil, err
	}

	m := new(big.Int).SetBytes(mBuf)

	// Compute signature share
	gsigshare := new(bn256.G1).ScalarMult(gskshare, m)
	return &vm.RbSIGTxPayload{rb.epochId, proposerId, gsigshare}, nil
}

func (rb *RandomBeacon) sendDKG1(payloadObj *vm.RbDKG1FlatTxPayload) error {
	log.SyslogInfo("begin send dkg1")
	payload, err := getRBDKG1TxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	return rb.doSendRBTx(payload)
}

func (rb *RandomBeacon) sendDKG2(payloadObj *vm.RbDKG2FlatTxPayload) error {
	log.SyslogInfo("begin send dkg2")
	payload, err := getRBDKG2TxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	return rb.doSendRBTx(payload)
}

func (rb *RandomBeacon) sendSIG(payloadObj *vm.RbSIGTxPayload) error {
	log.SyslogInfo("begin send sig")
	payload, err := getRBSIGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	return rb.doSendRBTx(payload)
}

func (rb *RandomBeacon) doSendRBTx(payload []byte) error {
	to := vm.GetRBAddress()
	data := hexutil.Bytes(payload)
	gas := core.IntrinsicGas(data, &to, true)

	arg := map[string]interface{}{}
	arg["from"] = rb.getTxFrom()
	arg["to"] = to
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(gas)
	arg["txType"] = types.POS_TX
	arg["data"] = data


	log.SyslogInfo("do send rb tx", "payload len", len(payload))
	_, err := util.SendTx(rb.rpcClient, arg)
	return err
}

func (rb *RandomBeacon) getTxFrom() common.Address {
	return posconfig.Cfg().GetMinerAddr()
}

func getRBDKG1TxPayloadBytes(payload *vm.RbDKG1FlatTxPayload) ([]byte, error) {
	if payload == nil {
		err := errors.New("invalid dkg1 payload object")
		log.SyslogErr(err.Error())
		return nil, err
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.SyslogErr("rlp encode dkg1 fail", "err", err.Error())
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetDkg1Id())
	copy(ret[4:], payloadBytes)

	return ret, nil
}

func getRBDKG2TxPayloadBytes(payload *vm.RbDKG2FlatTxPayload) ([]byte, error) {
	if payload == nil {
		err := errors.New("invalid dkg2 payload object")
		log.SyslogErr(err.Error())
		return nil, err
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.SyslogErr("rlp encode dkg2 fail", "err", err.Error())
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetDkg2Id())
	copy(ret[4:], payloadBytes)

	return ret, nil
}

func getRBSIGTxPayloadBytes(payload *vm.RbSIGTxPayload) ([]byte, error) {
	if payload == nil {
		err := errors.New("invalid sig payload object")
		log.SyslogErr(err.Error())
		return nil, err
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.SyslogErr("rlp encode sig payload", "err", err.Error())
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetSigShareId())
	copy(ret[4:], payloadBytes)

	return ret, nil
}
