package randombeacon

import (
	"crypto/rand"
	"errors"
	"github.com/wanchain/go-wanchain/pos/rbselection"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"

	"math/big"

	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
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

type RandomBeacon struct {
	loopEvents chan *LoopEvent

	epochStage int
	epochId    uint64
	polys      PolyMap

	statedb   vm.StateDB
	epocher   *epochLeader.Epocher
	rpcClient *rpc.Client

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

func GetRandonBeaconInst() *RandomBeacon {
	return &randomBeacon
}

func (rb *RandomBeacon) Init(epocher *epochLeader.Epocher) {
	rb.epochStage = vm.RbDkg1Stage
	rb.epochId = maxUint64
	rb.polys = make(PolyMap)
	rb.rpcClient = nil

	rb.epocher = epocher

	// function
	rb.getRBProposerGroupF = posdb.GetRBProposerGroup
	rb.getCji = vm.GetCji
	rb.getEns = vm.GetEncryptShare
	rb.getRBM = vm.GetRBM

	rb.loopEvents = make(chan *LoopEvent, loopEventCount)

	go rb.LoopRoutine()
}

func (rb *RandomBeacon) Stop() {
	close(rb.loopEvents)
}

func (rb *RandomBeacon) Loop(statedb vm.StateDB, rc *rpc.Client, eid uint64, sid uint64) {
	rb.loopEvents <- &LoopEvent{statedb, rc, eid, sid}
}

func (rb *RandomBeacon) LoopRoutine() {
	for {
		event, ok := <-rb.loopEvents
		if !ok {
			break
		}

		rb.doLoop(event.statedb, event.rc, event.eid, event.sid)
	}
}

func (rb *RandomBeacon) doLoop(statedb vm.StateDB, rc *rpc.Client, epochId uint64, slotId uint64) error {
	if statedb == nil || rc == nil {
		err := errors.New("invalid random beacon loop param")
		log.Error(err.Error())
		return err
	}

	log.Info("rb doLoop begin", "epochId", epochId, "slotId", slotId, "self epochId", rb.epochId)
	rb.statedb = statedb
	rb.rpcClient = rc

	if rb.epochId != maxUint64 && rb.epochId > epochId {
		err := errors.New("blockchain rollback")
		log.Error(err.Error())
		return err
	}

	if rb.epochId == maxUint64 || rb.epochId < epochId {
		log.Info("rb epochId is original")

		rb.epochId = epochId
		rb.epochStage = vm.RbDkg1Stage
		rb.polys = make(PolyMap)
	}

	// rb.epochId == epochId
	myProposerIds := rb.getMyRBProposerId(epochId)
	if len(myProposerIds) == 0 {
		return nil
	}

	rbStage, elapsedNum, leftNum := vm.GetRBStage(slotId)

	log.Info("get my RB proposer id", "id", myProposerIds)
	log.Info("get RB stage", "rbStage", rbStage, "elapsedNum", elapsedNum, "leftNum", leftNum)

	// belong to RB proposer group
	for {
		switch rb.epochStage {
		case vm.RbDkg1Stage:
			if rbStage == vm.RbDkg1Stage {
				// don't send tx in first and last slot
				if elapsedNum == 0 || leftNum == 0 {
					return nil
				}

				err := rb.doDKG1s(epochId, myProposerIds)
				if err != nil {
					return err
				}
			}

			rb.epochStage = vm.RbDkg2Stage
		case vm.RbDkg2Stage:
			if rbStage < vm.RbDkg2Stage {
				return nil
			} else if rbStage == vm.RbDkg2Stage {
				// don't send tx in first and last slot
				if elapsedNum == 0 || leftNum == 0 {
					return nil
				}

				err := rb.doDKG2s(epochId, myProposerIds)
				if err != nil {
					return err
				}
			}

			rb.epochStage = vm.RbSignStage
		case vm.RbSignStage:
			if rbStage < vm.RbSignStage {
				return nil
			} else if rbStage == vm.RbSignStage {
				// don't send tx in first and last slot
				if elapsedNum == 0 || leftNum == 0 {
					return nil
				}

				err := rb.doSIGs(epochId, myProposerIds)
				if err != nil {
					return err
				}
			}

			rb.epochStage = vm.RbSignConfirmStage
		default:
			// RbSignConfirmStage
			return nil
		}
	}

	return nil
}

func (rb *RandomBeacon) getMyRBProposerId(epochId uint64) []uint32 {
	pks := rb.getRBProposerGroup(epochId)
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

func (rb *RandomBeacon) doDKG1s(epochId uint64, proposerIds []uint32) error {
	pks := rb.getRBProposerGroup(epochId)
	nr := len(pks)
	if nr == 0 {
		err := errors.New("can't find random beacon proposer group")
		log.Error(err.Error())
		return err
	}

	for _, id := range proposerIds {
		err := rb.doDKG1(epochId, id, pks)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) doDKG1(epochId uint64, proposerId uint32, groupPks []bn256.G1) error {
	log.Info("begin do dkg1", "epochId", epochId, "proposerId", proposerId)
	txPayload, err := rb.generateDKG1(epochId, proposerId, groupPks)
	if err != nil {
		return err
	}

	return rb.sendDKG1(txPayload)
}

func (rb *RandomBeacon) generateDKG1(epochId uint64, proposerId uint32, groupPks []bn256.G1) (*vm.RbDKG1FlatTxPayload, error) {
	nr := len(groupPks)

	// fix the evaluation point: Hash(Pub[1]+1), Hash(Pub[2]+2), ..., Hash(Pub[Nr]+Nr)
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&groupPks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	sshare := make([]big.Int, nr)

	// fi(x)
	s, err := rand.Int(rand.Reader, bn256.Order)
	if err != nil {
		log.Error("get rand fail", "err", err)
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

	txPayload := vm.RbDKG1FlatTxPayload{epochId, proposerId, commitBytes}

	return &txPayload, nil
}

func (rb *RandomBeacon) doDKG2s(epochId uint64, proposerIds []uint32) error {
	// random proposer group
	pks := rb.getRBProposerGroup(epochId)
	nr := len(pks)
	if nr == 0 {
		err := errors.New("can't find random beacon proposer group")
		log.Error(err.Error())
		return err
	}

	for _, id := range proposerIds {
		rb.doDKG2(epochId, id, pks)
	}

	return nil
}

func (rb *RandomBeacon) doDKG2(epochId uint64, proposerId uint32, groupPks []bn256.G1) error {
	log.Info("begin do dkg2", "epochId", epochId, "proposerId", proposerId)
	txPayload, err := rb.generateDKG2(epochId, proposerId, groupPks)
	if err != nil {
		return err
	}

	if txPayload != nil {
		return rb.sendDKG2(txPayload)
	} else {
		return nil
	}
}

func (rb *RandomBeacon) generateDKG2(epochId uint64, proposerId uint32, groupPks []bn256.G1) (*vm.RbDKG2FlatTxPayload, error) {
	nr := len(groupPks)

	// check dkg1
	commit, err := rb.getCji(rb.statedb, epochId, proposerId)
	if err != nil || len(commit) == 0 {
		err := errors.New("no dkg1 data in chain")
		log.Error(err.Error())
		return nil, err
	}

	// fix the evaluation point: Hash(Pub[1]+1), Hash(Pub[2]+2), ..., Hash(Pub[Nr]+Nr)
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&groupPks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	sshare := make([]big.Int, nr)

	// fi(x)
	if rb.polys[proposerId].s == nil || rb.polys[proposerId].poly == nil {
		err := errors.New("no DKG1's random, can't generate DKG2 tx")
		log.Error(err.Error())
		return nil, err
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
		enshare[i] = new(bn256.G1).ScalarMult(&groupPks[i], &sshare[i])
	}

	// generate DLEQ proof
	proof := make([]rbselection.DLEQproof, nr)
	for i := 0; i < nr; i++ {
		// proof = (a1, a2, z)
		proof[i] = rbselection.DLEQ(groupPks[i], *rbselection.Hbase, &sshare[i])
	}

	enshareBytes := make([][]byte, nr)
	proofBytes := make([]rbselection.DLEQproofFlat, nr)
	for i := 0; i < nr; i++ {
		enshareBytes[i] = enshare[i].Marshal()
		proofBytes[i] = rbselection.ProofToProofFlat(&proof[i])
	}

	txPayload := vm.RbDKG2FlatTxPayload{epochId, proposerId, enshareBytes, proofBytes}

	return &txPayload, nil
}

func (rb *RandomBeacon) doSIGs(epochId uint64, proposerIds []uint32) error {
	pks := rb.getRBProposerGroup(epochId)
	if len(pks) == 0 {
		err := errors.New("can't find random beacon proposer group")
		log.Error(err.Error())
		return err
	}

	for _, id := range proposerIds {
		// try the best to send every SIG tx, prevent that one error stop all left task
		rb.doSIG(epochId, id, pks)
	}

	return nil
}

func (rb *RandomBeacon) doSIG(epochId uint64, proposerId uint32, groupPks []bn256.G1) error {
	log.Info("do sig begin", "epochId", epochId, "proposerId", proposerId)
	sig, err := rb.generateSIG(epochId, proposerId, groupPks)
	if err != nil {
		return err
	}

	if sig != nil {
		return rb.sendSIG(sig)
	} else {
		return nil
	}
}

func (rb *RandomBeacon) generateSIG(epochId uint64, proposerId uint32, groupPks []bn256.G1) (*vm.RbSIGTxPayload, error) {
	prikey := posconfig.Cfg().GetMinerBn256SK()
	datas := make([]RbEnsDataCollector, 0)

	for id, pk := range groupPks {
		data, err := rb.getEns(rb.statedb, epochId, uint32(id))
		if err == nil && data != nil {
			datas = append(datas, RbEnsDataCollector{data, &pk})
		} else {
			// do nothing
		}
	}

	dkgCount := len(datas)
	log.Info("collecte dkg", "count", dkgCount)
	if uint(dkgCount) < posconfig.Cfg().RBThres {
		err := errors.New("insufficient proposer")
		log.Error(err.Error())
		return nil, err
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
	mBuf, err := rb.getRBM(rb.statedb, epochId)
	if err != nil {
		return nil, err
	}

	m := new(big.Int).SetBytes(mBuf)

	// Compute signature share
	gsigshare := new(bn256.G1).ScalarMult(gskshare, m)
	return &vm.RbSIGTxPayload{epochId, proposerId, gsigshare}, nil
}

func (rb *RandomBeacon) sendDKG1(payloadObj *vm.RbDKG1FlatTxPayload) error {
	log.Info("begin send dkg1")
	payload, err := getRBDKG1TxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	return rb.doSendRBTx(payload)
}

func (rb *RandomBeacon) sendDKG2(payloadObj *vm.RbDKG2FlatTxPayload) error {
	log.Info("begin send dkg2")
	payload, err := getRBDKG2TxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	return rb.doSendRBTx(payload)
}

func (rb *RandomBeacon) sendSIG(payloadObj *vm.RbSIGTxPayload) error {
	log.Info("begin send sig")
	payload, err := getRBSIGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	return rb.doSendRBTx(payload)
}

func (rb *RandomBeacon) doSendRBTx(payload []byte) error {
	arg := map[string]interface{}{}
	arg["from"] = rb.getTxFrom()
	arg["to"] = vm.GetRBAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(4500000)) // todo: should optimize
	arg["txType"] = types.POS_TX
	arg["data"] = hexutil.Bytes(payload)

	log.Info("do send rb tx", "payload len", len(payload))
	_, err := util.SendTx(rb.rpcClient, arg)
	return err
}

func (rb *RandomBeacon) getTxFrom() common.Address {
	return posconfig.Cfg().GetMinerAddr()
}

func (rb *RandomBeacon) getRBProposerGroup(epochId uint64) []bn256.G1 {
	pks := rb.getRBProposerGroupF(epochId)

	pksStr := ""
	for _, pk := range pks {
		pksStr += common.ToHex(pk.Marshal()) + ", "
	}

	return pks
}

func getRBDKG1TxPayloadBytes(payload *vm.RbDKG1FlatTxPayload) ([]byte, error) {
	if payload == nil {
		err := errors.New("invalid DKG payload object")
		log.Error(err.Error())
		return nil, err
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.Error("rlp encode fail", "err", err)
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetDkg1Id())
	copy(ret[4:], payloadBytes)

	return ret, nil
}

func getRBDKG2TxPayloadBytes(payload *vm.RbDKG2FlatTxPayload) ([]byte, error) {
	if payload == nil {
		err := errors.New("invalid DKG payload object")
		log.Error(err.Error())
		return nil, err
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.Error("rlp encode fail", "err", err)
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetDkg2Id())
	copy(ret[4:], payloadBytes)

	return ret, nil
}

func getRBSIGTxPayloadBytes(payload *vm.RbSIGTxPayload) ([]byte, error) {
	if payload == nil {
		err := errors.New("invalid DKG payload object")
		log.Error(err.Error())
		return nil, err
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.Error("rlp encode sig payload", "err", err)
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetSigShareId())
	copy(ret[4:], payloadBytes)

	return ret, nil
}
