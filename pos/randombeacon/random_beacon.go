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
}

var randomBeacon RandomBeacon

func init() {
	randomBeacon.Init()
}

func GetRandonBeaconInst() *RandomBeacon {
	return &randomBeacon
}


func (rb *RandomBeacon) Init() {
	rb.epochStage = EPOCH_DKG
	rb.epochId = maxUint64
}


func (rb *RandomBeacon) Loop(statedb vm.StateDB, key *keystore.Key, epocher * epochLeader.Epocher) error {
	if statedb == nil || key == nil || epocher == nil {
		return errors.New("invalid random beacon loop param")
	}

	rb.statedb = statedb
	rb.key = key
	rb.epocher = epocher

	// set local proposer info
	pos.Cfg().SelfPuK = key.PrivateKey3.PublicKeyBn256.G1
	pos.Cfg().SelfPrK = key.PrivateKey3.D

	// get epoch id, slot id
	epochId, slotId, err := slotleader.GetEpochSlotID()
	if err != nil {
		return nil
	}

	if rb.epochId != maxUint64 && rb.epochId > epochId {
		return errors.New("blockchain rollback")
	}

	if rb.epochId == maxUint64 {
		err := rb.ComputeRandoms(0, epochId)
		if err != nil {
			return err
		}

		rb.epochId = epochId
		rb.epochStage = EPOCH_DKG
	}

	if rb.epochId < epochId {
		if !(rb.epochId == epochId-1 && rb.epochStage == EPOCH_TAIL) {
			err := rb.ComputeRandoms(rb.epochId, epochId)
			if err != nil {
				return err
			}
		}

		rb.epochId = epochId
		rb.epochStage = EPOCH_DKG
	}

	// rb.epochId == epochId
	myProposerIds := rb.GetMyRBProposerId(epochId)
	if len(myProposerIds) == 0 {
		// not belong to RB proposer group
		// wait 8K point to compute random
		if rb.epochStage == EPOCH_TAIL {
			// computed random already
			return nil
		} else if slotId >= slot8kConfirmId {
			err := rb.ComputeRandoms(epochId, epochId+1)
			if err != nil {
				return err
			}

			rb.epochStage = EPOCH_TAIL
			return nil
		}
	} else {
		// belong to RB proposer group
		for {
			if rb.epochStage == EPOCH_DKG {
				if slotId < slot4kEndId {
					err := rb.DoDKGs(epochId, myProposerIds, epocher)
					if err != nil {
						return err
					}
				}

				rb.epochStage = EPOCH_SIG

			} else if rb.epochStage == EPOCH_SIG {
				if slotId < slot4kConfirmId {
					break
				} else if slotId < slot8kEndId {
					err := rb.DoSIGs(epochId, myProposerIds)
					if err != nil {
						return err
					}

				}

				rb.epochStage = EPOCH_CMP

			} else if rb.epochStage == EPOCH_CMP {
				if slotId < slot8kConfirmId {
					break
				} else {
					err := rb.ComputeRandoms(epochId, epochId+1)
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

func (rb *RandomBeacon) GetMyRBProposerId(epochId uint64) []uint32 {
	pks := rb.epocher.GetRBProposerGroup(epochId + 1)
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
			ids = append(ids, uint32(i))
		}
	}

	return ids
}

func (rb *RandomBeacon) DoDKGs(epochId uint64, proposerIds []uint32, epocher * epochLeader.Epocher) error {
	for _, id := range proposerIds {
		err := rb.DoDKG(epochId, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) DoDKG(epochId uint64, proposerId uint32) error {
	pks := rb.epocher.GetRBProposerGroup(epochId + 1)
	nr := len(pks)
	if nr == 0 {
		return errors.New("can't find random beacon proposer group")
	}

	thres := pos.Cfg().PolymDegree+1
	//pubkey := Cfg().SelfPuK
	//prikey := Cfg().SelfPrK

	// Fix the evaluation point: Hash(Pub[1]), Hash(Pub[2]), ..., Hash(Pub[Nr])
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(crypto.Keccak256(pks[i].Marshal()))
		x[i].Mod(&x[i], bn256.Order)
	}

	s, err := rand.Int(rand.Reader, bn256.Order)
	if err != nil {
		return err
	}

	sshare := make([]big.Int, nr, nr)
	poly := wanpos.RandPoly(int(thres-1), *s)	// fi(x), set si as its constant term
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
	return rb.SendDKG(&txPayload)
}

func (rb *RandomBeacon) DoSIGs(epochId uint64, proposerIds []uint32) error {
	for _, id := range proposerIds {
		err := rb.DoSIG(epochId, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rb *RandomBeacon) DoSIG(epochId uint64, proposerId uint32) error {
	pks := rb.epocher.GetRBProposerGroup(epochId + 1)
	nr := len(pks)
	if nr == 0 {
		return errors.New("can't find random beacon proposer group")
	}

	//thres := Cfg().PolymDegree+1
	//pubkey := Cfg().SelfPuK
	prikey := pos.Cfg().SelfPrK
	datas := make([]RbDKGDataCollector, 0)
	for id, pk := range pks {
		data, err := vm.GetDkg(rb.statedb, epochId, uint32(id))
		if err == nil && data != nil {
			datas = append(datas, RbDKGDataCollector{data, &pk})
		}
	}

	if uint(len(datas)) < pos.Cfg().MinRBProposerCnt {
		return errors.New("insufficient proposer")
	}

	// Compute Group Secret Key Share
	// Random proposers get information from the blockchain and compute its group secret share.
	gskshare := new(bn256.G1).ScalarBaseMult(big.NewInt(int64(0))) //set zero
	skinver := new(big.Int).ModInverse(prikey, bn256.Order) // sk^-1
	for i := 0; i < nr; i++ {
		temp := new(bn256.G1).ScalarMult(datas[i].data.Enshare[proposerId], skinver)
		gskshare.Add(gskshare, temp) // gskshare[i] = (sk^-1)*(enshare[1][i]+...+enshare[Nr][i])
	}

	// Signing Stage
	// In this stage, each random proposer computes its signature share and sends it on chain.
	mBuf, err := GetRBM(epochId + 1)
	if err != nil {
		return err
	}

	m := new(big.Int).SetBytes(mBuf)

	// Compute signature share
	gsigshare := new(bn256.G1).ScalarMult(gskshare, m)
	return rb.SendSIG(&vm.RbSIGTxPayload{epochId, proposerId, gsigshare})
}

func (rb *RandomBeacon) ComputeRandoms(bgEpochId uint64, endEpochId uint64) error {
	for i := bgEpochId; i < endEpochId; i++ {
		err := rb.DoComputeRandom(i)
		if err != nil {
			return err
		}

	}

	return nil
}

func (rb *RandomBeacon) DoComputeRandom(epochId uint64) error {
	randomInt, err := vm.GetRandom(epochId+1)
	if epochId == 0 && err != nil {
		return errors.New("invalid genesis epoch random")
	}

	if err == nil && randomInt != nil && randomInt.Cmp(big.NewInt(0)) != 0 {
		// exist already
		return nil
	}

	pks := rb.epocher.GetRBProposerGroup(epochId + 1)
	nr := len(pks)
	if nr == 0 {
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
		return errors.New("insufficient proposer")
	}

	gsigshare := make([]bn256.G1, len(sigDatas))
	x := make([]big.Int, len(sigDatas))
	for i, data := range sigDatas {
		gsigshare[i] = *data.data.Gsigshare
		x[i].SetBytes(crypto.Keccak256(data.pk.Marshal()))
	}

	// Compute the Output of Random Beacon
	gsig := wanpos.LagrangeSig(gsigshare, x, int(pos.Cfg().PolymDegree))
	random := crypto.Keccak256(gsig.Marshal())

	// Verification Logic for the Output of Random Beacon
	// Computation of group public key
	c := make([]bn256.G2, len(dkgDatas))
	for i := 0; i < nr; i++ {
		c[i].ScalarBaseMult(big.NewInt(int64(0)))
		for j := 0; j < len(dkgDatas); j++ {
			c[i].Add(&c[i], dkgDatas[j].data.Commit[i])
		}
	}

	gPub := wanpos.LagrangePub(c, x, int(pos.Cfg().PolymDegree))

	// mG
	mBuf, err := GetRBM(epochId + 1)
	if err != nil {
		return err
	}

	m := new(big.Int).SetBytes(mBuf)
	mG := new(bn256.G1).ScalarBaseMult(m)

	// Verify using pairing
	pair1 := bn256.Pair(&gsig, wanpos.Hbase)
	pair2 := bn256.Pair(mG, &gPub)
	if pair1.String() != pair2.String() {
		return errors.New("Final Pairing Check Failed")
	}

	return SetRandom(epochId+1, new(big.Int).SetBytes(random))
}

func (rb *RandomBeacon) SendDKG(payloadObj *vm.RbDKGTxPayload) error {
	payload, err := GetRBDKGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	log.Debug("ready to write data of payload: " + "0x" + hexutil.Encode(payload))
	return rb.DoSendRBTx(payload)
}

func (rb *RandomBeacon) SendSIG(payloadObj *vm.RbSIGTxPayload) error {
	payload, err := GetRBSIGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	log.Debug("ready to write data of payload: " + "0x" + hexutil.Encode(payload))
	return rb.DoSendRBTx(payload)
}

func (rb *RandomBeacon) DoSendRBTx(payload []byte) error {
	arg := map[string]interface{}{}
	arg["from"] = rb.GetTxFrom()
	arg["to"] = vm.GetRBAddress()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["gas"] = (*hexutil.Big)(big.NewInt(1000000))
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)

	_, err := pos.SendTx(arg)
	return err
}

func (rb *RandomBeacon) GetTxFrom() common.Address {
	return rb.key.Address
}


func GetRBDKGTxPayloadBytes(payload * vm.RbDKGTxPayload) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("invalid DKG payload object")
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := common.Bytes2Hex(payloadBytes)
	rbAbi, err := abi.JSON(strings.NewReader(vm.GetRBAbiDefinition()))
	if err != nil {
		return nil, err
	}

	return rbAbi.Pack("dkg", payloadStr)

}

func GetRBSIGTxPayloadBytes(payload *vm.RbSIGTxPayload) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("invalid DKG payload object")
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := common.Bytes2Hex(payloadBytes)
	rbAbi, err := abi.JSON(strings.NewReader(vm.GetRBAbiDefinition()))
	if err != nil {
		return nil, err
	}

	return rbAbi.Pack("sigshare", payloadStr)

}


func GetRBM(epochId uint64) ([]byte, error) {
	if epochId < 1 {
		return nil, errors.New("epoch id too low")
	}

	epochIdBigInt := big.NewInt(int64(epochId))
	preRandom, err := vm.GetRandom(epochId-1)
	if err != nil {
		return nil, err
	}

	buf := epochIdBigInt.Bytes()
	buf = append(buf, preRandom.Bytes()...)
	return crypto.Keccak256(buf), nil
}


func SetRandom(epochId uint64, random *big.Int) error {
	_, err := posdb.GetDb().Put(epochId, vm.RANDOMBEACON_DB_KEY, random.Bytes())
	return err
}


