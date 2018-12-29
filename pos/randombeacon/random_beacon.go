package randombeacon

import (
	"errors"
	"math/big"
	"strings"
	"crypto/rand"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/wanpos_crypto"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/rpc"
)

const (
	_ int = iota
	EPOCH_DKG		// 4K
	EPOCH_SIG		// 8K
	EPOCH_CMP		// compute random
	EPOCH_TAIL
)


var (
	maxUint64 		= uint64(1<<64 - 1)
	slot4kEndId 	= uint64(4*pos.Cfg().K - 1)
	slot4kConfirmId = uint64((4+1)*pos.Cfg().K - 1)
	slot8kEndId 	= uint64(8*pos.Cfg().K - 1)
	slot8kConfirmId = uint64((8+1)*pos.Cfg().K - 1)
)

type RbDKGDataCollector struct {
	data *vm.RbDKGTxPayload
	pk *bn256.G1
}

type RbSIGDataCollector struct {
	data *vm.RbSIGTxPayload
	pk *bn256.G1
}

type RandomBeacon struct {
	epochStage int
	epochId uint64

	statedb vm.StateDB
	key *keystore.Key
	epocher * epochLeader.Epocher
	rpcClient *rpc.Client
}

var (
	randomBeacon RandomBeacon
)

func init() {
	randomBeacon.Init(nil)
}

func GetRandonBeaconInst() *RandomBeacon {
	return &randomBeacon
}


func (rb *RandomBeacon) Init(epocher * epochLeader.Epocher) {
	rb.epochStage = EPOCH_DKG
	rb.epochId = maxUint64
	rb.rpcClient = nil

	rb.epocher = epocher
}



func (rb *RandomBeacon) Loop(statedb vm.StateDB, key *keystore.Key, epocher * epochLeader.Epocher, rc *rpc.Client) error {
	if statedb == nil || key == nil || epocher == nil || rc == nil {
		log.Error("invalid random beacon loop param")
		return errors.New("invalid random beacon loop param")
	}

	log.Info("RB Loop begin", "statedb", statedb, "key", key, "epocher", epocher)
	rb.statedb = statedb
	rb.key = key
	rb.epocher = epocher
	rb.rpcClient = rc

	// set local proposer info
	pos.Cfg().SelfPuK = new(bn256.G1)
	pos.Cfg().SelfPrK = new(big.Int)
	pos.Cfg().SelfPuK.Set(key.PrivateKey3.PublicKeyBn256.G1)
	pos.Cfg().SelfPrK.Set(key.PrivateKey3.D)

	log.Info("set miner account", "puk", pos.Cfg().SelfPuK, "prk", pos.Cfg().SelfPrK)

	// get epoch id, slot id
	epochId, slotId, err := slotleader.GetEpochSlotID()
	if err != nil {
		log.Error("get epoch slot id fail", "err", err)
		return nil
	}

	log.Info("get epoch slot id", "epochId", epochId, "slotId", slotId)
	if rb.epochId != maxUint64 && rb.epochId > epochId {
		log.Error("blockchain rollback")
		return errors.New("blockchain rollback")
	}

	log.Info("rb", "epochId", rb.epochId)
	if rb.epochId == maxUint64 {
		log.Info("rb epochId is original")
		err := rb.computeRandoms(0, epochId)
		if err != nil {
			log.Error("compute randoms fail", "err", err)
			return err
		}

		rb.epochId = epochId
		rb.epochStage = EPOCH_DKG
	}

	if rb.epochId < epochId {
		if !(rb.epochId == epochId-1 && rb.epochStage == EPOCH_TAIL) {
			err := rb.computeRandoms(rb.epochId, epochId)
			if err != nil {
				return err
			}
		}

		rb.epochId = epochId
		rb.epochStage = EPOCH_DKG
	}

	// rb.epochId == epochId
	myProposerIds := rb.getMyRBProposerId(epochId)
	log.Info("get my RB proposer id", "id", myProposerIds)
	if len(myProposerIds) == 0 {
		log.Info("my proposer len is zero")
		// not belong to RB proposer group
		// wait 8K point to compute random
		if rb.epochStage == EPOCH_TAIL {
			// computed random already
			return nil
		} else if slotId >= slot8kConfirmId {
			err := rb.computeRandoms(epochId, epochId+1)
			if err != nil {
				return err
			}

			rb.epochStage = EPOCH_TAIL
			return nil
		}
	} else {
		// belong to RB proposer group
		for {
			log.Info("do as proposer", "epoch stage", rb.epochStage)
			if rb.epochStage == EPOCH_DKG {
				if slotId < slot4kEndId {
					log.Info("do epoch dkg")
					err := rb.doDKGs(epochId, myProposerIds)
					if err != nil {
						return err
					}
				}

				rb.epochStage = EPOCH_SIG

			} else if rb.epochStage == EPOCH_SIG {
				if slotId < slot4kConfirmId {
					break
				} else if slotId < slot8kEndId {
					err := rb.doSIGs(epochId, myProposerIds)
					if err != nil {
						return err
					}

				}

				rb.epochStage = EPOCH_CMP

			} else if rb.epochStage == EPOCH_CMP {
				if slotId < slot8kConfirmId {
					break
				} else {
					err := rb.computeRandoms(epochId, epochId+1)
					if err != nil {
						return err
					}

					rb.epochStage = EPOCH_TAIL
				}
			} else {
				// EPOCH_TAIL
				break
			}
		}
	}

	return nil
}

func (rb *RandomBeacon) getMyRBProposerId(epochId uint64) []uint32 {
	pks := rb.getRBProposerGroup(epochId)
	if len(pks) == 0 {
		return nil
	}

	selfPk := pos.Cfg().SelfPuK
	if selfPk == nil {
		return nil
	}

	ids := make([]uint32, 0)
	for i, pk := range pks {
		if pk.String() == selfPk.String() {
		//if true || pk.String() != "" {
			ids = append(ids, uint32(i))
		}
	}

	return ids
}

func (rb *RandomBeacon) doDKGs(epochId uint64, proposerIds []uint32) error {
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

	pks := rb.getRBProposerGroup(epochId)
	nr := len(pks)
	if nr == 0 {
		log.Error("can't find random beacon proposer group")
		return errors.New("can't find random beacon proposer group")
	}

	//thres := pos.Cfg().PolymDegree+1
	//pubkey := Cfg().SelfPuK
	//prikey := Cfg().SelfPrK

	// Fix the evaluation point: Hash(Pub[1]+1), Hash(Pub[2]+2), ..., Hash(Pub[Nr]+Nr)
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(vm.GetPolynomialX(&pks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	s, err := rand.Int(rand.Reader, bn256.Order)
	if err != nil {
		log.Error("get rand fail", "err", err)
		return err
	}

	sshare := make([]big.Int, nr, nr)
	poly := wanpos.RandPoly(int(pos.Cfg().PolymDegree), *s)	// fi(x), set si as its constant term
	for i := 0; i < nr; i++ {
		sshare[i] = wanpos.EvaluatePoly(poly, &x[i], int(pos.Cfg().PolymDegree)) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
	}

	// Encrypt the secret share, i.e. mutiply with the receiver's public key
	enshare := make([]*bn256.G1, nr, nr)
	for i := 0; i < nr; i++ { // enshare[j] = sshare[j]*Pub[j], it is a point on ECC
		enshare[i] = new(bn256.G1).ScalarMult(&pks[i], &sshare[i])
	}

	// Make commitment for the secret share, i.e. mutiply with the generator of G2
	commit := make([]*bn256.G2, nr, nr)
	for i := 0; i < nr; i++ { // commit[j] = sshare[j] * G2
		commit[i] = new(bn256.G2).ScalarBaseMult(&sshare[i])
	}

	// generate DLEQ proof
	proof := make([]wanpos.DLEQproof, nr, nr)
	for i := 0; i < nr; i++ { // proof = (a1, a2, z)
		proof[i] = wanpos.DLEQ(pks[i], *wanpos.Hbase, &sshare[i])
	}

	txPayload := vm.RbDKGTxPayload{epochId, proposerId, enshare[:], commit[:], proof[:]}
	//log.Info("do dkg", "txPayload", txPayload)
	return rb.sendDKG(&txPayload)
}

func (rb *RandomBeacon) doSIGs(epochId uint64, proposerIds []uint32) error {
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

	prikey := pos.Cfg().SelfPrK
	datas := make([]RbDKGDataCollector, 0)
	for id, pk := range pks {
		data, err := vm.GetDkg(rb.statedb, epochId, uint32(id))
		if err == nil && data != nil {
			datas = append(datas, RbDKGDataCollector{data, &pk})
		}
	}

	dkgCount := len(datas)
	log.Info("collecte dkg", "count", dkgCount)
	if uint(dkgCount) < pos.Cfg().MinRBProposerCnt {
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
	mBuf, err := vm.GetRBM(epochId)
	if err != nil {
		return err
	}

	m := new(big.Int).SetBytes(mBuf)

	// Compute signature share
	gsigshare := new(bn256.G1).ScalarMult(gskshare, m)
	return rb.sendSIG(&vm.RbSIGTxPayload{epochId, proposerId, gsigshare})
}

func (rb *RandomBeacon) computeRandoms(bgEpochId uint64, endEpochId uint64) error {
	log.Info("RB compute randoms", "beEpochId", bgEpochId, "endEpochId", endEpochId)
	for i := bgEpochId; i < endEpochId; i++ {
		err := rb.DoComputeRandom(i)
		if err != nil {
			log.Error("do compute random fail", "err", err)
			return err
		}

	}

	return nil
}

func (rb *RandomBeacon) DoComputeRandom(epochId uint64) error {
	log.Info("RB do compute random", "epochId", epochId)
	randomInt, err := posdb.GetRandom(epochId+1)
	if err == nil && randomInt != nil && randomInt.Cmp(big.NewInt(0)) != 0 {
		// exist already
		log.Info("random exist already", "epochId", epochId+1, "random", randomInt.String())
		return nil
	}

	pks := rb.getRBProposerGroup(epochId)
	if len(pks) == 0 {
		log.Error("can't find random beacon proposer group")
		return errors.New("can't find random beacon proposer group")
	}

	// collact gsigshare
	// collect DKG data
	dkgDatas := make([]RbDKGDataCollector, 0)
	sigDatas := make([]RbSIGDataCollector, 0)
	for id, pk := range pks {
		dkgData, err := vm.GetDkg(rb.statedb, epochId, uint32(id))
		if err == nil && dkgData != nil {
			dkgDatas = append(dkgDatas, RbDKGDataCollector{dkgData, &pk})
		}

		sigData, err := vm.GetSig(rb.statedb, epochId, uint32(id))
		if err == nil && sigData != nil {
			sigDatas = append(sigDatas, RbSIGDataCollector{sigData, &pk})
		}
	}

	if uint(len(sigDatas)) < pos.Cfg().MinRBProposerCnt {
		log.Error("compute random fail, insufficient proposer", "epochId", epochId, "min", pos.Cfg().MinRBProposerCnt, "acture", len(sigDatas))
		// return errors.New("insufficient proposer")

		randomInt, err := posdb.GetRandom(epochId)
		if err != nil {
			log.Error("get random fail", "epochId", epochId, "err", err)
			return err
		}

		newRandom := crypto.Keccak256(randomInt.Bytes())
		err = rb.saveRandom(epochId+1, new(big.Int).SetBytes(newRandom))
		if err != nil {
			log.Error("set random fail", "err", err)
		} else {
			log.Info("set random success", "epochId", epochId+1, "random", common.Bytes2Hex(newRandom))
		}

		return err
	}

	gsigshare := make([]bn256.G1, len(sigDatas))
	xSig := make([]big.Int, len(sigDatas))
	for i, data := range sigDatas {
		gsigshare[i] = *data.data.Gsigshare
		xSig[i].SetBytes(vm.GetPolynomialX(data.pk, data.data.ProposerId))
	}

	// Compute the Output of Random Beacon
	gsig := wanpos.LagrangeSig(gsigshare, xSig, int(pos.Cfg().PolymDegree))
	random := crypto.Keccak256(gsig.Marshal())
	log.Info("sig lagrange", "gsig", gsig, "gsigshare", gsigshare)

	// Verification Logic for the Output of Random Beacon
	// Computation of group public key
	nr := len(pks)
	c := make([]bn256.G2, nr)
	for i := 0; i < nr; i++ {
		c[i].ScalarBaseMult(big.NewInt(int64(0)))
		for j := 0; j < len(dkgDatas); j++ {
			c[i].Add(&c[i], dkgDatas[j].data.Commit[i])
		}
	}

	xAll := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		xAll[i].SetBytes(vm.GetPolynomialX(&pks[i], uint32(i)))
		xAll[i].Mod(&xAll[i], bn256.Order)
	}
	gPub := wanpos.LagrangePub(c, xAll, int(pos.Cfg().PolymDegree))

	// mG
	mBuf, err := vm.GetRBM(epochId)
	if err != nil {
		log.Error("get M fail", "err", err)
		return err
	}

	m := new(big.Int).SetBytes(mBuf)
	mG := new(bn256.G1).ScalarBaseMult(m)

	// Verify using pairing
	pair1 := bn256.Pair(&gsig, wanpos.Hbase)
	pair2 := bn256.Pair(mG, &gPub)
	log.Info("verify random", "pair1", pair1.String(), "pair2", pair2.String())
	if pair1.String() != pair2.String() {
		return errors.New("Final Pairing Check Failed")
	}

	err = rb.saveRandom(epochId+1, new(big.Int).SetBytes(random))
	if err != nil {
		log.Error("set random fail", "err", err)
	} else {
		log.Info("set random success", "epochId", epochId+1, "random", common.Bytes2Hex(random))
	}

	return err
}

func (rb *RandomBeacon) saveRandom(epochId uint64, random *big.Int) error {
	if random == nil {
		log.Error("invalid random")
		return errors.New("invalid random")
	}

	err := posdb.SetRandom(epochId, random)
	if err != nil {
		return err
	}

	return rb.sendRandom(epochId, random)
}

func (rb *RandomBeacon) sendDKG(payloadObj *vm.RbDKGTxPayload) error {
	log.Info("begin send dkg")
	payload, err := getRBDKGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	//log.Info("send dkg", "payload", common.Bytes2Hex(payload))
	return rb.doSendRBTx(payload)
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

func (rb *RandomBeacon) sendRandom(epochId uint64, random *big.Int) error {
	log.Info("begin send random")
	payload, err := getGenRTxPayloadBytes(epochId, random)
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
	arg["gas"] = (*hexutil.Big)(big.NewInt(1500000))
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)

	log.Info("do send rb tx", "payload len", len(payload))
	_, err := pos.SendTx(rb.rpcClient, arg)
	return err
}

func (rb *RandomBeacon) getTxFrom() common.Address {
	return rb.key.Address
}

func (rb *RandomBeacon) getRBProposerGroup(epochId uint64) []bn256.G1 {
	pks := rb.epocher.GetRBProposerGroup(epochId)
	log.Info("get rb proposer group", "proposer", pks)
	return pks
}


func getRBDKGTxPayloadBytes(payload * vm.RbDKGTxPayload) ([]byte, error) {
	if payload == nil {
		log.Error("get dkg tx payload fail, invalid DKG payload object")
		return nil, errors.New("invalid DKG payload object")
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		log.Error("rlp encode fail", "err", err)
		return nil, err
	}

	payloadStr := common.Bytes2Hex(payloadBytes)
	//log.Info("dkg payload hex string", "playload", payloadStr)
	rbAbi, err := abi.JSON(strings.NewReader(vm.GetRBAbiDefinition()))
	if err != nil {
		log.Error("create abi instance fail", "err", err)
		return nil, err
	}


	ret, err := rbAbi.Pack("dkg", &payloadStr)
	if err != nil {
		log.Error("abi pack fail", "err", err)
		return nil, err
	}

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

	payloadStr := common.Bytes2Hex(payloadBytes)
	rbAbi, err := abi.JSON(strings.NewReader(vm.GetRBAbiDefinition()))
	if err != nil {
		return nil, err
	}

	ret, err := rbAbi.Pack("sigshare", payloadStr)
	if err != nil {
		log.Error("abi pack payload", "err", err)
		return nil, err
	}

	return ret, nil
}

func getGenRTxPayloadBytes(epochId uint64, random *big.Int) ([]byte, error) {
	log.Info("get GenR tx payload begin")
	if random == nil {
		return nil, errors.New("invalid random")
	}

	rbAbi, err := abi.JSON(strings.NewReader(vm.GetRBAbiDefinition()))
	if err != nil {
		return nil, err
	}

	ret, err := rbAbi.Pack("genR", big.NewInt(int64(epochId)), random)
	if err != nil {
		log.Error("abi pack payload", "err", err)
		return nil, err
	}

	return ret, nil
}


