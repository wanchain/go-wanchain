package randombeacon

import (
	"crypto/rand"
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/pos/wanpos_crypto"
	"math/big"
	"time"
)

var (
	maxUint64 = uint64(1<<64 - 1)
)

type RbDKGDataCollector struct {
	data *vm.RbDKGTxPayload
	pk   *bn256.G1
}

type RbSIGDataCollector struct {
	data *vm.RbSIGTxPayload
	pk   *bn256.G1
}

type GetRBProposerGroupFunc func(epochId uint64) []bn256.G1

type LoopEvent struct {
	statedb vm.StateDB
	rc      *rpc.Client
	eid     uint64
	sid     uint64
}

type RandomBeacon struct {
	loopEvents chan *LoopEvent

	epochStage int
	epochId    uint64

	statedb   vm.StateDB
	epocher   *epochLeader.Epocher
	rpcClient *rpc.Client

	// based function
	getRBProposerGroupF GetRBProposerGroupFunc
}

var (
	randomBeacon RandomBeacon
)

func init() {
	//randomBeacon.Init(nil)
}

func GetRandonBeaconInst() *RandomBeacon {
	return &randomBeacon
}

func (rb *RandomBeacon) Init(epocher *epochLeader.Epocher) {
	rb.epochStage = vm.RB_DKG_STAGE
	rb.epochId = maxUint64
	rb.rpcClient = nil

	rb.epocher = epocher

	// function
	rb.getRBProposerGroupF = posdb.GetRBProposerGroup

	rb.loopEvents = make(chan *LoopEvent, 1000)

	go rb.LoopRoutine()
}

func (rb *RandomBeacon) Uninit() {
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
		log.Error("invalid random beacon loop param")
		return errors.New("invalid random beacon loop param")
	}

	log.Info("rb doLoop begin", "epochId", epochId, "slotId", slotId)
	rb.statedb = statedb
	rb.rpcClient = rc

	if rb.epochId != maxUint64 && rb.epochId > epochId {
		log.Error("blockchain rollback")
		return errors.New("blockchain rollback")
	}

	log.Info("rb", "epochId", rb.epochId)
	if rb.epochId == maxUint64 || rb.epochId < epochId {
		log.Info("rb epochId is original")

		rb.epochId = epochId
		rb.epochStage = vm.RB_DKG_STAGE
	}

	// rb.epochId == epochId
	myProposerIds := rb.getMyRBProposerId(epochId)
	log.Info("get my RB proposer id", "id", myProposerIds)
	if len(myProposerIds) == 0 {
		return nil
	}

	rbStage := vm.GetRBStage(slotId)
	log.Info("get current rb statge", "statge", rbStage)

	// belong to RB proposer group
	for {
		log.Info("do as proposer", "epoch stage", rb.epochStage)
		switch rb.epochStage {
		case vm.RB_DKG_STAGE:
			if rbStage == vm.RB_DKG_STAGE {
				err := rb.doDKGs(epochId, myProposerIds)
				if err != nil {
					return err
				}
			}
			rb.epochStage = vm.RB_SIGN_STAGE
		case vm.RB_SIGN_STAGE:
			if rbStage < vm.RB_SIGN_STAGE {
				return nil
			} else if rbStage == vm.RB_SIGN_STAGE {
				err := rb.doSIGs(epochId, myProposerIds)
				if err != nil {
					return err
				}
			}

			rb.epochStage = vm.RB_AFTER_SIGH_STAGE
		default:
			// RB_AFTER_SIGH_STAGE
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

	selfPk := pos.Cfg().GetMinerBn256PK()
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

func (rb *RandomBeacon) doDKGs(epochId uint64, proposerIds []uint32) error {
	log.Info("do dkgs begin")
	for _, id := range proposerIds {
		err := rb.doDKG(epochId, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) doDKG(epochId uint64, proposerId uint32) error {
	log.Info("begin do dkg", "epochId", epochId, "proposerId", proposerId)
	txPayload, err := rb.generateDKG(epochId, proposerId)
	if err != nil {
		return err
	}

	return rb.sendDKG(txPayload)
}

func (rb *RandomBeacon) generateDKG(epochId uint64, proposerId uint32) (*vm.RbDKGTxPayload, error) {
	//log.Info("time", "1", time.Now().Unix())
	start := time.Now()

	pks := rb.getRBProposerGroup(epochId)
	nr := len(pks)
	if nr == 0 {
		err := errors.New("can't find random beacon proposer group")
		log.Error(err.Error())
		return nil, err
	}

	// fix the evaluation point: Hash(Pub[1]+1), Hash(Pub[2]+2), ..., Hash(Pub[Nr]+Nr)
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&pks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	sshare := make([]big.Int, nr)

	// fi(x)
	s, err := rand.Int(rand.Reader, bn256.Order)
	if err != nil {
		log.Error("get rand fail", "err", err)
		return nil, err
	}

	poly := wanpos.RandPoly(int(pos.Cfg().PolymDegree), *s)
	for i := 0; i < nr; i++ {
		// share for i is fi(x) evaluation result on x[i]
		sshare[i], _ = wanpos.EvaluatePoly(poly, &x[i], int(pos.Cfg().PolymDegree))
	}

	// encrypt the secret share, i.e. multiply with the receiver's public key
	enshare := make([]*bn256.G1, nr)
	for i := 0; i < nr; i++ {
		// enshare[i] = sshare[i]*Pub[i], it is a point on ECC
		enshare[i] = new(bn256.G1).ScalarMult(&pks[i], &sshare[i])
	}

	// make commitment for the secret share, i.e. multiply with the generator of G2
	commit := make([]*bn256.G2, nr)
	for i := 0; i < nr; i++ {
		// commit[i] = sshare[i] * G2
		commit[i] = new(bn256.G2).ScalarBaseMult(&sshare[i])
	}

	// generate DLEQ proof
	proof := make([]wanpos.DLEQproof, nr)
	for i := 0; i < nr; i++ {
		// proof = (a1, a2, z)
		proof[i] = wanpos.DLEQ(pks[i], *wanpos.Hbase, &sshare[i])
	}

	txPayload := vm.RbDKGTxPayload{epochId, proposerId, enshare[:], commit[:], proof[:]}

	log.Info("generateDKG used time", "time", time.Since(start))
	return &txPayload, nil
}

func (rb *RandomBeacon) doSIGs(epochId uint64, proposerIds []uint32) error {
	log.Info("do sigs begin")
	for _, id := range proposerIds {
		err := rb.doSIG(epochId, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) doSIG(epochId uint64, proposerId uint32) error {
	log.Info("do sig begin", "epochId", epochId, "proposerId", proposerId)
	pks := rb.getRBProposerGroup(epochId)
	if len(pks) == 0 {
		log.Error("can't find random beacon proposer group")
		return errors.New("can't find random beacon proposer group")
	}

	prikey := pos.Cfg().GetMinerBn256SK()
	datas := make([]RbDKGDataCollector, 0)
	for id, pk := range pks {
		data, err := vm.GetDkg(rb.statedb, epochId, uint32(id))
		if err == nil && data != nil {
			datas = append(datas, RbDKGDataCollector{data, &pk})
		} else {
			//log.Debug("vm.GetDkg failed", "err", err)
		}
	}

	dkgCount := len(datas)
	log.Info("collecte dkg", "count", dkgCount)
	if uint(dkgCount) < pos.Cfg().RBThres {
		return errors.New("insufficient proposer")
	}

	// Compute Group Secret Key Share
	// Random proposers get information from the blockchain and compute its group secret share.

	//set zero
	gskshare := new(bn256.G1).ScalarBaseMult(big.NewInt(int64(0)))

	// sk^-1
	skinver := new(big.Int).ModInverse(prikey, bn256.Order)
	for i := 0; i < dkgCount; i++ {
		log.Info("compute gskshare", "i", i, "enshare len", len(datas[i].data.Enshare))
		temp := new(bn256.G1).ScalarMult(datas[i].data.Enshare[proposerId], skinver)

		// gskshare[i] = (sk^-1)*(enshare[1][i]+...+enshare[Nr][i])
		gskshare.Add(gskshare, temp)
	}

	// Signing Stage
	// In this stage, each random proposer computes its signature share and sends it on chain.
	mBuf, err := vm.GetRBM(rb.statedb, epochId)
	if err != nil {
		return err
	}

	m := new(big.Int).SetBytes(mBuf)

	// Compute signature share
	gsigshare := new(bn256.G1).ScalarMult(gskshare, m)
	return rb.sendSIG(&vm.RbSIGTxPayload{epochId, proposerId, gsigshare})
}

func (rb *RandomBeacon) sendDKG(payloadObj *vm.RbDKGTxPayload) error {
	log.Info("begin send dkg")
	start := time.Now()
	payload, err := getRBDKGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	//log.Info("send dkg", "payload", common.Bytes2Hex(payload))
	err = rb.doSendRBTx(payload)

	log.Info("sendDKG used time", "time", time.Since(start))
	return err
}

func (rb *RandomBeacon) sendSIG(payloadObj *vm.RbSIGTxPayload) error {
	log.Info("begin send sig")
	payload, err := getRBSIGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	//log.Info("send sig tx", "payload", common.Bytes2Hex(payload))
	return rb.doSendRBTx(payload)
}

func (rb *RandomBeacon) doSendRBTx(payload []byte) error {
	arg := map[string]interface{}{}
	arg["from"] = rb.getTxFrom()
	arg["to"] = vm.GetRBAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(4500000))
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)

	log.Info("do send rb tx", "payload len", len(payload))
	_, err := pos.SendTx(rb.rpcClient, arg)
	return err
}

func (rb *RandomBeacon) getTxFrom() common.Address {
	return pos.Cfg().GetMinerAddr()
}

func (rb *RandomBeacon) getRBProposerGroup(epochId uint64) []bn256.G1 {
	pks := rb.getRBProposerGroupF(epochId)

	pksStr := ""
	for _, pk := range pks {
		pksStr += common.ToHex(pk.Marshal()) + ", "
	}

	log.Debug("get rb proposer group", "proposer", pksStr)
	return pks
}

func getRBDKGTxPayloadBytes(payload *vm.RbDKGTxPayload) ([]byte, error) {
	if payload == nil {
		log.Error("get dkg tx payload fail, invalid DKG payload object")
		return nil, errors.New("invalid DKG payload object")
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.Error("rlp encode fail", "err", err)
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetDkgId())
	copy(ret[4:], payloadBytes)

	//log.Info("dkg abi packed payload", "payload", common.Bytes2Hex(ret))
	return ret, nil
}

func getRBSIGTxPayloadBytes(payload *vm.RbSIGTxPayload) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("invalid DKG payload object")
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.Error("rlp encode sig payload", "err", err)
		return nil, err
	}

	ret := make([]byte, 4+len(payloadBytes))
	copy(ret, vm.GetSigshareId())
	copy(ret[4:], payloadBytes)

	return ret, nil
}
