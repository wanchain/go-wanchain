package epochLeader

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	bn256 "github.com/wanchain/pos/cloudflare"
)

var (
	safeK = uint64(1)
	Nr    = pos.RandomProperCount //num of random proposers
	Ne    = pos.EpochLeaderCount  //num of epoch leaders, limited <= 256 now

	Big1                                   = big.NewInt(1)
	Big0                                   = big.NewInt(0)
	ErrInvalidRandomProposerSelection      = errors.New("Invalid Random Proposer Selection")                  //Invalid Random Proposer Selection
	ErrInvalidEpochProposerSelection       = errors.New("Invalid Epoch Proposer Selection")                   //Invalid Random Proposer Selection
	ErrInvalidProbabilityfloat2big         = errors.New("Invalid Transform Probability From Float To Bigint") //Invalid Transform Probability From Float To Bigint
	ErrInvalidGenerateCommitment           = errors.New("Invalid Commitment Generation")                      //Invalid Commitment Generation
	ErrInvalidArrayPieceGeneration         = errors.New("Invalid ArrayPiece Generation")                      //Invalid ArrayPiece Generation
	ErrInvalidDleqProofGeneration          = errors.New("Invalid DLEQ Proof Generation")                      //Invalid DLEQ Proof Generation
	ErrInvalidSecretMessageArrayGeneration = errors.New("Invalid Secret Message Array Generation")            //Invalid Secret Message Array Generation
	ErrInvalidSortPublicKeys               = errors.New("Invalid PublicKeys Sort Operation")                  //Invalid PublicKeys Sort Operation
	ErrInvalidSlotLeaderSequenceGeneration = errors.New("Invalid Slot Leader Sequence Generation")            //Invalid Slot Leader Sequence Generation
	ErrInvalidSlotLeaderLocation           = errors.New("Invalid Slot Leader Location")                       //Invalid Slot Leader Location
	ErrInvalidSlotLeaderProofGeneration    = errors.New("Invalid Slot Leader Proof Generation")               //Invalid Slot Leader Proof Generation
)

type Epocher struct {
	rbLeadersDb    *posdb.Db
	epochLeadersDb *posdb.Db
	blkChain       *core.BlockChain
}

func NewEpocher(blc *core.BlockChain) *Epocher {

	rbdb := posdb.NewDb("rblocaldb")

	epdb := posdb.NewDb("eplocaldb")

	inst := &Epocher{rbdb, epdb, blc}

	posdb.SetEpocherInst(inst)

	return inst
}


func (e *Epocher) getTargetBlkNumber(epochId uint64) uint64 {
	// TODO how to get thee target blockNumber
	if epochId < 2 {
		return uint64(0)
	}
	targetEpochId := epochId-2
	targetBlkNum := posdb.GetEpochBlock(targetEpochId)
	if targetBlkNum == 0 {
		curNum := e.blkChain.CurrentBlock().NumberU64()
		for {
			curBlock := e.blkChain.GetBlockByNumber(curNum)
			curEpochId := curBlock.Header().Difficulty.Uint64()>>32
			if curEpochId <= targetEpochId {
				break
			}
			curNum--
		}
		targetBlkNum = curNum
		posdb.UpdateEpochBlock(targetEpochId, targetBlkNum)
	}
	return targetBlkNum
	//targetBlkNum := e.blkChain.CurrentBlock().NumberU64()
	//if targetBlkNum >= safeK {
	//	return (targetBlkNum - safeK)
	//} else {
	//	return 0
	//}
}
func (e *Epocher) SelectLeadersLoop(epochId uint64) error {

	targetBlkNum := e.getTargetBlkNumber(epochId)

	stateDb, err := e.blkChain.StateAt(e.blkChain.GetBlockByNumber(targetBlkNum).Root())
	if err != nil {
		return err
	}

	epochIdIn := epochId
	if epochIdIn > 0 {
		epochIdIn --
	}
	rb := vm.GetR(stateDb, epochIdIn)

	if rb == nil {
		log.Error(fmt.Sprintln("vm.GetR return nil at epochId:", epochId))
		rb = big.NewInt(1)
	}

	r := rb.Bytes()

	err = e.SelectLeaders(r, Ne, Nr, stateDb, epochId)
	if err != nil {
		return err
	}

	return nil
}

func (e *Epocher) SelectLeaders(r []byte, ne int, nr int, statedb *state.StateDB, epochId uint64) error {

	fmt.Println("\n\nselect randoms", epochId, common.ToHex(r))

	pa, err := e.createStakerProbabilityArray(statedb, epochId)
	if pa == nil || err != nil {
		return err
	}

	epl := e.GetEpochLeaders(epochId)
	if len(epl) != ne {
		e.epochLeaderSelection(r, ne, pa, epochId)
	}

	rbl := e.GetRBProposerGroup(epochId)
	if len(rbl) != nr {
		e.randomProposerSelection(r, nr, pa, epochId)
	}

	return nil

}

type Proposer struct {
	pubSec256     *ecdsa.PublicKey //staker’s ethereum public key
	pubBn256      *bn256.G1        //staker’s bn256 public key
	probabilities *big.Int
}

type ProposerSorter []Proposer

func newProposerSorter() ProposerSorter {
	ps := make(ProposerSorter, 0)
	return ps
}

//Len()
func (s ProposerSorter) Len() int {
	return len(s)
}

func (s ProposerSorter) Less(i, j int) bool {
	return s[i].probabilities.Cmp(s[j].probabilities) < 0
}

//Swap()
func (s ProposerSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

const Accuracy float64 = 1024.0 //accuracy to magnificate
//wanhumber*locktime*(exp-(t) ),t=(locktime - passedtime/locktime)
func (e *Epocher) generateProblility(pstaker *vm.StakerInfo, epochId uint64, blkTime uint64) (*Proposer, error) {

	amount := big.NewInt(0).Div(pstaker.Amount, big.NewInt(params.Wan)).Int64()
	lockTime := pstaker.LockTime

	var leftTimePercent float64
	if epochId < 2 {

		leftTimePercent = 1

	} else {

		leftTimePercent = (float64(int64(lockTime)-(int64(blkTime)-pstaker.StakingTime)) / float64(lockTime))
	}

	pb := float64(amount) * float64(lockTime) * math.Exp(-leftTimePercent) * Accuracy

	//if pb == 0 {
	fmt.Println(epochId,amount,lockTime,leftTimePercent,pb,pstaker.StakingTime)
	//}

	gb := new(bn256.G1)
	_, err := gb.Unmarshal(pstaker.PubBn256)
	if err != nil {
		return nil, err
	}

	p := &Proposer{
		pubSec256:     crypto.ToECDSAPub(pstaker.PubSec256),
		pubBn256:      gb,
		probabilities: big.NewInt(int64(pb)),
	}

	return p, nil

}

func (e *Epocher) createStakerProbabilityArray(statedb *state.StateDB, epochId uint64) (ProposerSorter, error) {

	if statedb == nil {
		return nil, vm.ErrUnknown
	}

	listAddr := vm.StakersInfoAddr
	ps := newProposerSorter()

	blkTime := epochId*(pos.SlotTime*pos.SlotCount) + pos.EpochBaseTime

	statedb.ForEachStorageByteArray(listAddr, func(key common.Hash, value []byte) bool {

		//fmt.Println("for each get data",value)
		staker := vm.StakerInfo{}
		err := json.Unmarshal(value, &staker)
		if err != nil {
			return false
		}

		pitem, err := e.generateProblility(&staker, epochId, blkTime)
		if err != nil {
			return false
		}

		if staker.Amount.Cmp(Big0) > 0 {
			ps = append(ps, *pitem)
			fmt.Println(common.ToHex((*pitem).probabilities.Bytes()))
		}

		return true
	})

	for idx, _ := range ps {
		if idx == 0 {
			continue
		}

		fmt.Println("epochid=" +  strconv.Itoa(int(epochId)) + "   probility" + strconv.Itoa(idx) + "  	" + common.ToHex(ps[idx].probabilities.Bytes()))
	}


	sort.Sort(ProposerSorter(ps))


	fmt.Println("------------------------------------------------------------------------------------------")
	for idx, _ := range ps {
		if idx == 0 {
			continue
		}

		ps[idx].probabilities = big.NewInt(0).Add(ps[idx].probabilities, ps[idx-1].probabilities)

		fmt.Println("after sum probility" + strconv.Itoa(idx) + "  	" + common.ToHex(ps[idx].probabilities.Bytes()))
	}

	return ps, nil
}

//samples nr random proposers by random number r（Random Beacon) from PublicKeys based on proportion of Probabilities
func (e *Epocher) epochLeaderSelection(r []byte, nr int, ps ProposerSorter, epochId uint64) error {
	if r == nil || nr <= 0 || len(ps) == 0 {
		return ErrInvalidRandomProposerSelection
	}

	//the last one is total properties
	tp := ps[len(ps)-1].probabilities

	fmt.Println("tp=" + common.ToHex(tp.Bytes()))

	var Byte0 = []byte{byte(0)}
	var buffer bytes.Buffer
	buffer.Write(Byte0)
	buffer.Write(r)
	r0 := buffer.Bytes()       //r0 = 0||r
	cr := crypto.Keccak256(r0) //cr = hash(r0)

	//randomProposerPublicKeys := make([]*ecdsa.PublicKey, 0)  //store the selected publickeys
	fmt.Println("\n\n\n")
	for i := 0; i < nr; i++ {

		crBig := new(big.Int).SetBytes(cr)
		crBig = crBig.Mod(crBig, tp) //cr_big = cr mod tp

		//fmt.Println("epoch leader mod tp=" + common.ToHex(crBig.Bytes()))
		//select pki whose probability bigger than cr_big left
		idx := sort.Search(len(ps), func(i int) bool { return ps[i].probabilities.Cmp(crBig) > 0 })

		if idx == len(ps) {
			idx = len(ps) - 1
		}

		fmt.Println("select epoch leader epochid,idx,pubs", epochId, i, common.ToHex(crypto.FromECDSAPub(ps[idx].pubSec256)))
		//randomProposerPublicKeys = append(randomProposerPublicKeys, ps[idx + 1].pubSec256)
		e.epochLeadersDb.PutWithIndex(epochId, uint64(i), "", crypto.FromECDSAPub(ps[idx].pubSec256))

		cr = crypto.Keccak256(cr)
	}

	fmt.Println("\n\n\n")

	return nil
}

//*bn256.G1
//samples ne epoch leaders by random number r from PublicKeys based on proportion of Probabilities
func (e *Epocher) randomProposerSelection(r []byte, nr int, ps ProposerSorter, epochId uint64) error {
	if r == nil || nr <= 0 || len(ps) == 0 {
		return ErrInvalidEpochProposerSelection
	}

	//the last one is total properties
	tp := ps[len(ps)-1].probabilities

	var Byte1 = []byte{byte(1)}
	var buffer bytes.Buffer
	buffer.Write(Byte1)
	buffer.Write(r)
	r1 := buffer.Bytes()       //r1 = 1||r
	cr := crypto.Keccak256(r1) //cr = hash(r1)

	fmt.Println("\n\n\n")
	for i := 0; i < nr; i++ {

		crBig := new(big.Int).SetBytes(cr)
		crBig = crBig.Mod(crBig, tp) //cr_big = cr mod tp

		//fmt.Println("mod tp=" + common.ToHex(crBig.Bytes()))

		//select pki whose probability bigger than cr_big left
		idx := sort.Search(len(ps), func(i int) bool { return ps[i].probabilities.Cmp(crBig) > 0 })

		if idx == len(ps) {
			idx = len(ps) - 1
		}

		fmt.Println("select random leader epochid,idx,pubs", epochId, i, common.ToHex(ps[idx].pubBn256.Marshal()))

		e.rbLeadersDb.PutWithIndex(epochId, uint64(i), "", ps[idx].pubBn256.Marshal())
		//EpochLeaderBn256G1s = append(EpochLeaderBn256G1s, ps[idx + 1].pubBn256)

		cr = crypto.Keccak256(cr)
	}

	fmt.Println("\n\n\n")

	return nil
}

//get epochLeaders of epochID in localdb
func (e *Epocher) GetEpochLeaders(epochID uint64) [][]byte {

	// TODO: how to cache these
	//e.SelectLeadersLoop(epochID)
	ksarray := e.epochLeadersDb.GetStorageByteArray(epochID)

	return ksarray

}

//get rbLeaders of epochID in localdb
func (e *Epocher) GetRBProposerGroup(epochID uint64) []bn256.G1 {

	ksarray := e.rbLeadersDb.GetStorageByteArray(epochID)
	g1ksArray := make([]bn256.G1, 0)

	for _, ks := range ksarray {

		gb := new(bn256.G1)
		_, err := gb.Unmarshal(ks)
		if err != nil {
			return nil
		}

		g1ksArray = append(g1ksArray, *gb)
	}

	return g1ksArray
}
