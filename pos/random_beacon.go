package pos

import (
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	"github.com/wanchain/pos/cloudflare"
	"errors"
	"math/big"
	"github.com/wanchain/go-wanchain/crypto"
	"crypto/rand"
	"github.com/wanchain/pos/wanpos_crypto"
	"fmt"
	"encoding/hex"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/log"
	"strings"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/rlp"
	"strconv"
	"github.com/wanchain/go-wanchain/pos/posdb"
)

const (
	_ int = iota
	EPOCH_DKG		// 4K
	EPOCH_SIG		// 8K
	EPOCH_CMP		// compute random
	EPOCH_TAIL
)


var maxUint64 uint64 = uint64(-1)
//var maxUint32 uint32 = uint32(-1)
var slot4kEndId uint64 = uint64(4*Cfg().K - 1)
var slot4kConfirmId uint64 = uint64((4+1)*Cfg().K - 1)
var slot8kEndId uint64 = uint64(8*Cfg().K - 1)
var slot8kConfirmId uint64 = uint64((8+1)*Cfg().K - 1)


type RandomBeacon struct {
	epochStage int
	epochId uint64
}


func (rb *RandomBeacon) Init() {
	rb.epochStage = EPOCH_DKG
	rb.epochId = maxUint64
}


func (rb *RandomBeacon) RBLoop() error {
	// get epoch id, slot id
	epochId, slotId, err := slotleader.GetEpochSlotID()
	if err != nil {
		return nil
	}

	if rb.epochId != maxUint64 && rb.epochId > epochId {
		return errors.New("blockchain rollback")
	}

	if rb.epochId == maxUint64 {
		err := ComputeRandom(0, epochId)
		if err != nil {
			return err
		}

		rb.epochId = epochId
		rb.epochStage = EPOCH_DKG
	}

	if rb.epochId < epochId {
		if !(rb.epochId == epochId-1 && rb.epochStage == EPOCH_TAIL) {
			err := ComputeRandom(rb.epochId, epochId)
			if err != nil {
				return err
			}
		}

		rb.epochId = epochId
		rb.epochStage = EPOCH_DKG
	}

	// rb.epochId == epochId
	proposerIds := GetRBProposerId(epochId)
	if len(proposerIds) == 0 {
		// not belong to RB proposer group
		// wait 8K point to compute random
		if rb.epochStage == EPOCH_TAIL {
			// computed random already
			return nil
		} else if slotId >= slot8kConfirmId {
			err := ComputeRandom(epochId, epochId+1)
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
					err := DoDKGs(epochId, proposerIds)
					if err != nil {
						return err
					}

					rb.epochStage = EPOCH_SIG
				} else {
					rb.epochStage = EPOCH_SIG
				}
			} else if rb.epochStage == EPOCH_SIG {
				if slotId < slot4kConfirmId {
					break
				} else if rb.epochId < slot8kEndId {
					err := DoSIGs(epochId, proposerIds)
					if err != nil {
						return err
					}

					rb.epochStage = EPOCH_CMP
				} else {
					rb.epochStage = EPOCH_CMP
				}
			} else if rb.epochStage == EPOCH_CMP {
				if slotId < slot8kConfirmId {
					break
				} else {
					err := ComputeRandom(epochId, epochId+1)
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

func GetRBProposerId(epochId uint64) []uint32 {
	pks := GetRBProposerGroup(epochId)
	if len(pks) == 0 {
		return nil
	}

	selfPk := Cfg().SelfPuK
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

func DoDKGs(epochId uint64, proposerIds []uint32) error {
	for _, id := range proposerIds {
		err := DoDKG(epochId, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func DoDKG(epochId uint64, proposerId uint32) error {
	pks := GetRBProposerGroup(epochId)
	nr := len(pks)
	if nr == 0 {
		return errors.New("can't find random beacon proposer group")
	}

	thres := Cfg().PolymDegree+1
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

	var sshare [nr] big.Int

	poly := wanpos.RandPoly(int(thres-1), *s)	// fi(x), set si as its constant term
	for i := 0; i < nr; i++ {
		sshare[i] = wanpos.EvaluatePoly(poly, &x[i]) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
	}

	// Encrypt the secret share, i.e. mutiply with the receiver's public key
	var enshare [nr] *bn256.G1
	for i := 0; i < nr; i++ { // enshare[j] = sshare[j]*Pub[j], it is a point on ECC
		enshare[i] = new(bn256.G1).ScalarMult(&pks[i], &sshare[i])
	}

	// Make commitment for the secret share, i.e. mutiply with the generator of G2
	var commit [nr] *bn256.G2
	for i := 0; i < nr; i++ { // commit[j] = sshare[j] * G2
		commit[i] = new(bn256.G2).ScalarBaseMult(&sshare[i])
	}

	// generate DLEQ proof
	var proof [nr] wanpos.DLEQproof
	for i := 0; i < nr; i++ { // proof = (a1, a2, z)
		proof[i] = wanpos.DLEQ(pks[i], *wanpos.Hbase, &sshare[i])
	}

	txPayload := RbDKGTxPayload{epochId, proposerId, enshare[:], commit[:], proof[:]}
	return SendDKG(&txPayload)
}

func DoSIGs(epochId uint64, proposerIds []uint32) error {
	for _, id := range proposerIds {
		err := DoSIG(epochId, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func DoSIG(epochId uint64, proposerId uint32) error {
	pks := GetRBProposerGroup(epochId)
	nr := len(pks)
	if nr == 0 {
		return errors.New("can't find random beacon proposer group")
	}

	//thres := Cfg().PolymDegree+1
	//pubkey := Cfg().SelfPuK
	prikey := Cfg().SelfPrK
	datas := make([]RbDKGDataCollector, 0)
	for id, pk := range pks {
		data, err := vm.GetDKGData(epochId, uint(id))
		if err == nil && data != nil {
			datas = append(datas, RbDKGDataCollector{nil, &pk})
		}
	}

	if uint(len(datas)) < Cfg().MinRBProposerCnt {
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
	mBuf, err := GetRBM(epochId)
	if err != nil {
		return err
	}

	m := new(big.Int).SetBytes(mBuf)

	// Compute signature share
	gsigshare := new(bn256.G1).ScalarMult(gskshare, m)
	return SendSIG(&RbSIGTxPayload{epochId, proposerId, gsigshare})
}

func ComputeRandom(bgEpochId uint64, endEpochId uint64) error {
	for i := bgEpochId; i < endEpochId; i++ {
		err := DoComputeRandom(i)
		if err != nil {
			return err
		}

	}

	return nil
}

func DoComputeRandom(epochId uint64) error {
	randomInt, err := GetRandom(epochId)
	if err == nil && randomInt != nil && randomInt.Cmp(big.NewInt(0)) != 0 {
		// exist already
		return nil
	}

	pks := GetRBProposerGroup(epochId)
	nr := len(pks)
	if nr == 0 {
		return errors.New("can't find random beacon proposer group")
	}

	// collact gsigshare
	// collect DKG data
	dkgDatas := make([]RbDKGDataCollector, 0)
	sigDatas := make([]RbSIGDataCollector, 0)
	for id, pk := range pks {
		dkgData, err := vm.GetDKGData(epochId, uint(id))
		if err == nil && dkgData != nil {
			dkgDatas = append(dkgDatas, RbDKGDataCollector{nil, &pk})
		}

		sigData, err := vm.GetSIGData(epochId, uint(id))
		if err == nil && sigData != nil {
			sigDatas = append(sigDatas, RbSIGDataCollector{nil, &pk})
		}
	}

	if uint(len(sigDatas)) < Cfg().MinRBProposerCnt {
		return errors.New("insufficient proposer")
	}

	gsigshare := make([]bn256.G1, len(sigDatas))
	x := make([]big.Int, len(sigDatas))
	for i, data := range sigDatas {
		gsigshare[i] = *data.data.Gsigshare
		x[i].SetBytes(crypto.Keccak256(data.pk.Marshal()))
	}

	// Compute the Output of Random Beacon
	gsig := wanpos.LagrangeSig(gsigshare, x)
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

	gPub := wanpos.LagrangePub(c, x)

	// mG
	mBuf, err := GetRBM(epochId)
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

	return SetRandom(epochId, new(big.Int).SetBytes(random))
}

func SendDKG(payloadObj *RbDKGTxPayload) error {
	payload, err := GetRBDKGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	log.Debug("ready to write data of payload: " + "0x" + hexutil.Encode(payload))
	return SoSendRBTx(payload)
}


func SendSIG(payloadObj *RbSIGTxPayload) error {
	payload, err := GetRBSIGTxPayloadBytes(payloadObj)
	if err != nil {
		return err
	}

	log.Debug("ready to write data of payload: " + "0x" + hexutil.Encode(payload))
	return SoSendRBTx(payload)
}

func SoSendRBTx(payload []byte) error {
	arg := map[string]interface{}{}
	arg["from"] = GetTxFrom()
	arg["to"] = GetRBPrecompileAddr()
	arg["value"] = (*hexutil.Big)(big.NewInt(0))
	arg["txType"] = 1
	arg["data"] = hexutil.Bytes(payload)

	return sendTx(arg)
}


var RANDOMBEACON_DB_KEY = "PosRandomBeacon"

func GetRandom(epochId uint64) (*big.Int, error) {
	bt, err := posdb.GetDb().Get(epochId, RANDOMBEACON_DB_KEY)
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(bt), nil
}

func SetRandom(epochId uint64, random *big.Int) error {
	_, err := posdb.GetDb().Put(epochId, RANDOMBEACON_DB_KEY, random.Bytes())
	return err
}

//>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>test>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

var rbAbiStr =
`[
  {
	"constant": false,
	"inputs": [
	  {
		"name": "info",
		"type": "string"
	  }
	],
	"name": "dkg",
	"outputs": [],
	"payable": false,
	"type": "function",
	"stateMutability": "nonpayable"
  },
  {
	"constant": false,
	"inputs": [
	  {
		"name": "info",
		"type": "string"
	  }
	],
	"name": "sigshare",
	"outputs": [],
	"payable": false,
	"type": "function",
	"stateMutability": "nonpayable"
  },
  {
	"constant": false,
	"inputs": [
	  {
		"name": "info",
		"type": "string"
	  }
	],
	"name": "genR",
	"outputs": [],
	"payable": false,
	"type": "function",
	"stateMutability": "nonpayable"
  }
]`


func GetRBDKGTxPayloadBytes(payload * RbDKGTxPayload) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("invalid DKG payload object")
	}

	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := common.Bytes2Hex(payloadBytes)
	rbAbi, err := abi.JSON(strings.NewReader(rbAbiStr))
	if err != nil {
		return nil, err
	}

	return rbAbi.Pack("dkg", payloadStr)

}

func GetRBSIGTxPayloadBytes(payload *RbSIGTxPayload) ([]byte, error) {
	if payload == nil {
		return nil, errors.New("invalid DKG payload object")
	}



	payloadBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := common.Bytes2Hex(payloadBytes)
	rbAbi, err := abi.JSON(strings.NewReader(rbAbiStr))
	if err != nil {
		return nil, err
	}

	return rbAbi.Pack("sigshare", payloadStr)

}


func GetRBM(epochId uint64) ([]byte, error) {
	epochIdBigInt := big.NewInt(int64(epochId))
	preRandom, err := GetRandom(epochId - 1)
	if err != nil {
		return nil, err
	}

	buf := epochIdBigInt.Bytes()
	buf = append(buf, preRandom.Bytes()...)
	return crypto.Keccak256(buf), nil
}

func Set()  {
	
}

//<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<test<<<<<<<<<<<<<<<<<<<<<<<<<<<<<


