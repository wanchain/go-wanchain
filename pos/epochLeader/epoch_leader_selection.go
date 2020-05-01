package epochLeader

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/incentive"

)

var (
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

type RefundInfo struct {
	Addr common.Address
	Amount *big.Int
}

type EpochInfo struct {
	EpochID uint64
	BlockNumber uint64
}


var epocherInst *Epocher = nil

func NewEpocher(blc *core.BlockChain) *Epocher {

	if blc == nil {
		return nil
	}

	if epocherInst == nil {
		epocherInst = NewEpocherWithLBN(blc, posconfig.RbLocalDB, posconfig.EpLocalDB)
	}

	return epocherInst
}

func GetEpocher() *Epocher {
	return epocherInst
}

func NewEpocherWithLBN(blc *core.BlockChain, rbn string, epdbn string) *Epocher {

	rbdb := posdb.NewDb(rbn)
	epdb := posdb.NewDb(epdbn)
	inst := &Epocher{rbdb, epdb, blc}

	util.SetEpocherInst(inst)
	return inst
}

func (e *Epocher) GetBlkChain() *core.BlockChain {
	return e.blkChain
}

func (e *Epocher) GetCurrentHeader() *types.Header {

	inst := e.blkChain.GetHc()
	if inst == nil {
		return nil
	}

	return inst.CurrentHeader()
}


func (e *Epocher) GetTargetBlkNumber(epochId uint64) uint64 {
	if epochId < 2 {
		return uint64(0)
	}

	targetEpochId := epochId - 2

	//return e.GetEpochLastBlkNumber(targetEpochId)
	return util.GetEpochBlock(targetEpochId)
}

/*
NOTE: if the targetEpochId is future, will return current blockNumber.
*/
func (e *Epocher) GetEpochLastBlkNumber(targetEpochId uint64) uint64 {

	var curBlockHeader *types.Header

	curNum := e.blkChain.CurrentBlock().NumberU64()
	for {
		curBlockHeader := e.blkChain.GetHeaderByNumber(curNum)
		curEpochId, _ := util.GetEpochSlotIDFromDifficulty(curBlockHeader.Difficulty)
		if curEpochId <= targetEpochId {
			break
		}
		curNum--
	}

	targetBlkNum := curNum
	epochid, _ := util.CalEpochSlotID(uint64(time.Now().Unix()))
	if targetEpochId < epochid && targetEpochId >= posconfig.FirstEpochId {
		util.SetEpochBlock(targetEpochId, targetBlkNum, curBlockHeader.Hash())
	}

	return targetBlkNum
}

func (e *Epocher) SelectLeadersLoop(epochId uint64) error {

	targetBlkNum := e.GetTargetBlkNumber(epochId)

	//stateDb, err := e.blkChain.StateAt(e.blkChain.GetBlockByNumber(targetBlkNum).Root())
	stateDb, err := e.blkChain.StateAt(e.blkChain.GetHeaderByNumber(targetBlkNum).Root)
	if err != nil {
		return err
	}

	epochIdIn := epochId
	if epochIdIn > 0 {
		epochIdIn--
	}
	rb := vm.GetR(stateDb, epochIdIn)
	if rb == nil {
		log.Error(fmt.Sprintln("vm.GetR return nil at epochId:", epochId))
		rb = new(big.Int).SetBytes(crypto.Keccak256(big.NewInt(1).Bytes()))
	}

	r := rb.Bytes()
	err = e.selectLeaders(r, stateDb, epochId)
	if err != nil {
		return err
	}

	return nil
}

func (e *Epocher) reportSelectELFailed(epochId uint64) {
	if epochId == 0 {
		return
	}
	
	failedTimes := 1
	if epochId > 0 {
		if !e.IsGenerateELSuc(epochId - 1) {
			failedTimes = 2
		}
	}

	// print the failed log
	if failedTimes == 1 {
		log.SyslogCrit("select EL failed", "epochid", epochId)
	} else if failedTimes == 2 {
		log.SyslogAlert("select EL failed in two consecutive epoch", "epochid", epochId)
	}
}

func (e *Epocher) reportSelectRBPFailed(epochId uint64) {
	if epochId == 0 {
		return
	}

	failedTimes := 1
	if epochId > 0 {
		if !e.IsGenerateRBPSuc(epochId - 1) {
			failedTimes = 2
		}
	}

	// print the failed log
	if failedTimes == 1 {
		log.SyslogCrit("select RNP failed", "epochid", epochId)
	} else if failedTimes == 2 {
		log.SyslogAlert("select RNP failed in two consecutive epoch", "epochid", epochId)
	}
}

func (e *Epocher) selectLeaders(r []byte, statedb *state.StateDB, epochId uint64) error {
	log.Debug("select randoms", "epochId", epochId, "r", common.ToHex(r))

	pa, err := e.createStakerProbabilityArray(statedb, epochId)
	if pa == nil || err != nil {
		e.reportSelectELFailed(epochId)
		e.reportSelectRBPFailed(epochId)
		return err
	}

	err = e.epochLeaderSelection(r, pa, epochId)
	if err != nil {
		e.reportSelectELFailed(epochId)
	}

	err = e.randomProposerSelection(r, pa, epochId)
	if err != nil {
		e.reportSelectRBPFailed(epochId)
	}

	return nil
}

type Proposer struct {
	PubSec256     []byte
	PubBn256      []byte
	Probabilities *big.Int
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
	return s[i].Probabilities.Cmp(s[j].Probabilities) < 0
}

//Swap()
func (s ProposerSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (e *Epocher) CalProbability(amountWin *big.Int, lockTime uint64) *big.Int {
	//amount := big.NewInt(0).Div(amountWin, big.NewInt(params.Wan))
	pb := big.NewInt(0)

	lockWeight := vm.CalLocktimeWeight(lockTime)
	timeBig := big.NewInt(int64(lockWeight))

	pb.Mul(amountWin, timeBig)

	log.Debug("CalProbability ", "pb: ", pb)

	return pb
}

func (e *Epocher) createStakerProbabilityArray(statedb *state.StateDB, epochID uint64) (ProposerSorter, error) {
	if statedb == nil {
		return nil, vm.ErrUnknown
	}

	listAddr := vm.StakersInfoAddr
	ps := newProposerSorter()

	statedb.ForEachStorageByteArray(listAddr, func(key common.Hash, value []byte) bool {
		staker := vm.StakerInfo{}
		err := rlp.DecodeBytes(value, &staker)
		if err != nil {
			log.Error(err.Error())
			return true
		}
		_, p, err := CalEpochProbabilityStaker(&staker, epochID)
		if err != nil || p == nil {
			// this validator has no enough
			return true
		}
		item := Proposer{
			PubSec256:     staker.PubSec256,
			PubBn256:      staker.PubBn256,
			Probabilities: p,
		}
		ps = append(ps, item)
		log.Debug(common.ToHex(item.Probabilities.Bytes()))
		return true
	})

	sort.Stable(ProposerSorter(ps))

	for idx, _ := range ps {
		if idx == 0 {
			continue
		}

		ps[idx].Probabilities = big.NewInt(0).Add(ps[idx].Probabilities, ps[idx-1].Probabilities)
	}

	log.Debug("get createStakerProbabilityArray", "len", len(ps))

	return ps, nil
}

//select epoch leader from PublicKeys based on proportion of Probabilities
func (e *Epocher) epochLeaderSelection(r []byte, ps ProposerSorter, epochId uint64) error {
	if r == nil || len(ps) == 0 {
		return ErrInvalidRandomProposerSelection
	}

	//the last one is total properties
	tp := ps[len(ps)-1].Probabilities

	var Byte0 = []byte{byte(0)}
	var buffer bytes.Buffer
	buffer.Write(Byte0)
	buffer.Write(r)
	r0 := buffer.Bytes()       //r0 = 0||r
	cr := crypto.Keccak256(r0) //cr = hash(r0)

	//randomProposerPublicKeys := make([]*ecdsa.PublicKey, 0)  //store the selected publickeys
	log.Debug("epochLeaderSelection selecting")
	selectionCount := posconfig.EpochLeaderCount
	info, err := e.GetWhiteInfo(epochId)
	if err == nil {
		selectionCount = posconfig.EpochLeaderCount - int(info.WlCount.Uint64())
	}
	for i := 0; i < selectionCount; i++ {

		crBig := new(big.Int).SetBytes(cr)
		crBig = crBig.Mod(crBig, tp) //cr_big = cr mod tp

		//fmt.Println("epoch leader mod tp=" + common.ToHex(crBig.Bytes()))
		//select pki whose probability bigger than cr_big left
		idx := sort.Search(len(ps), func(i int) bool { return ps[i].Probabilities.Cmp(crBig) > 0 })

		log.Debug("select epoch leader", "epochid=", epochId, "idx=", i, "pub=", ps[idx].PubSec256)
		//randomProposerPublicKeys = append(randomProposerPublicKeys, ps[idx].PubSec256)
		val, err := rlp.EncodeToBytes(&ps[idx])
		if err != nil {
			continue
		}
		e.epochLeadersDb.PutWithIndex(epochId, uint64(i), "", val)

		cr = crypto.Keccak256(cr)
	}

	return nil
}

func (e *Epocher) GetWhiteInfo(epochId uint64) (*vm.UpgradeWhiteEpochLeaderParam, error) {
	targetBlkNum := e.GetTargetBlkNumber(epochId)

	block := e.GetBlkChain().GetHeaderByNumber(targetBlkNum)
	//block := e.GetBlkChain().GetBlockByNumber(targetBlkNum)
	if block == nil {
		return nil, errors.New("Unkown block")
	}
	stateDb, err := e.GetBlkChain().StateAt(block.Root)
	if err != nil {
		return nil, err
	}
	info := vm.GetEpochWLInfo(stateDb, epochId)
	return info, nil

}

func (e *Epocher) GetWhiteByEpochId(epochId uint64) ([]string, error) {
	info, err := e.GetWhiteInfo(epochId)
	if err != nil {
		return nil, err
	}
	return posconfig.WhiteList[info.WlIndex.Uint64() : info.WlIndex.Uint64()+info.WlCount.Uint64()], nil
}

func (e *Epocher) GetWhiteArrayByEpochId(epochId uint64) ([][]byte, error) {
	info, err := e.GetWhiteInfo(epochId)
	if err != nil {
		return nil, err
	}
	return posconfig.EpochLeadersHold[info.WlIndex.Uint64() : info.WlIndex.Uint64()+info.WlCount.Uint64()], nil

}

//*bn256.G1
//samples ne epoch leaders by random number r from PublicKeys based on proportion of Probabilities
func (e *Epocher) randomProposerSelection(r []byte, ps ProposerSorter, epochId uint64) error {
	if r == nil || len(ps) == 0 {
		return ErrInvalidEpochProposerSelection
	}

	//the last one is total properties
	tp := ps[len(ps)-1].Probabilities

	var Byte1 = []byte{byte(1)}
	var buffer bytes.Buffer
	buffer.Write(Byte1)
	buffer.Write(r)
	r1 := buffer.Bytes()       //r1 = 1||r
	cr := crypto.Keccak256(r1) //cr = hash(r1)

	log.Info("random proposer selecting...\n")
	for i := 0; i < posconfig.RandomProperCount; i++ {

		crBig := new(big.Int).SetBytes(cr)
		crBig = crBig.Mod(crBig, tp) //cr_big = cr mod tp

		//select pki whose probability bigger than cr_big left
		idx := sort.Search(len(ps), func(i int) bool { return ps[i].Probabilities.Cmp(crBig) > 0 })

		val, err := rlp.EncodeToBytes(ps[idx])

		if err != nil {
			continue
		}

		e.rbLeadersDb.PutWithIndex(epochId, uint64(i), "", val)

		cr = crypto.Keccak256(cr)
	}

	return nil
}

func (e *Epocher) IsGenerateELSuc(epochID uint64) bool {
	epArray := posdb.GetEpochLeaderGroup(epochID)
	return len(epArray) != 0
}

func (e *Epocher) IsGenerateRBPSuc(epochID uint64) bool {
	rbArray := posdb.GetRBProposerGroup(epochID)
	return len(rbArray) != 0
}

//get epochLeaders of epochID in localdb
func (e *Epocher) GetEpochLeaders(epochID uint64) [][]byte {

	// TODO: how to cache these
	epArray := posdb.GetEpochLeaderGroup(epochID)
	wa, err := e.GetWhiteArrayByEpochId(epochID)
	if err == nil {
		if len(epArray) == posconfig.EpochLeaderCount-len(wa) {
			epArray = append(epArray, wa...)
		}
	}
	return epArray

}
func (e *Epocher) GetRBProposer(epochID uint64) [][]byte {
	// TODO: how to cache these
	rbArray := posdb.GetRBProposerGroup(epochID)
	return rbArray

}
func (e *Epocher) GetRBProposerG1(epochID uint64) []bn256.G1 {

	rbArray := e.GetRBProposer(epochID)
	length := len(rbArray)

	g1s := make([]bn256.G1, length, length)

	for i := 0; i < length; i++ {
		g1s[i] = *new(bn256.G1)
		_, err := g1s[i].Unmarshal(rbArray[i])
		if err != nil {
			log.Error("G1 unmarshal failed: ", "err", err)
		}
	}

	return g1s

}

//get rbLeaders of epochID in localdb only for incentive. for incentive.
func (e *Epocher) GetRBProposerGroup(epochID uint64) []vm.Leader {
	proposersArray := e.rbLeadersDb.GetStorageByteArray(epochID)
	length := len(proposersArray)
	g1s := make([]vm.Leader, length)

	for i := 0; i < length; i++ {
		proposer := Proposer{}
		err := rlp.DecodeBytes(proposersArray[i], &proposer)
		if err != nil {
			log.Error("can't rlp decode:", "err", err)
		}
		g1s[i].Type = 1
		g1s[i].PubSec256 = proposer.PubSec256
		g1s[i].PubBn256 = proposer.PubBn256
		pub := crypto.ToECDSAPub(proposer.PubSec256)
		if nil == pub {
			continue
		}

		g1s[i].SecAddr = crypto.PubkeyToAddress(*pub)
	}

	return g1s
}

func (e *Epocher) GetEpLeaderGroup(epochID uint64) []vm.Leader {
	epLeaderArray := e.epochLeadersDb.GetStorageByteArray(epochID)
	length := len(epLeaderArray)
	if length == 0 {
		return make([]vm.Leader, 0)
	}
	g1s := make([]vm.Leader, posconfig.EpochLeaderCount)

	for i := 0; i < length; i++ {
		proposer := Proposer{}
		err := rlp.DecodeBytes(epLeaderArray[i], &proposer)
		if err != nil {
			log.Error("can't rlp decode:", "err", err)
		}
		g1s[i].Type = 0
		g1s[i].PubSec256 = proposer.PubSec256
		g1s[i].PubBn256 = proposer.PubBn256
		pub := crypto.ToECDSAPub(proposer.PubSec256)
		if nil == pub {
			continue
		}

		g1s[i].SecAddr = crypto.PubkeyToAddress(*pub)
	}
	wa, err := e.GetWhiteArrayByEpochId(epochID)
	if err == nil && length < posconfig.EpochLeaderCount {
		for i := 0; i < posconfig.EpochLeaderCount-length; i++ {
			g1s[i+length].Type = 0
			g1s[i+length].PubSec256 = wa[i]
			g1s[i+length].PubBn256 = ([]byte)("")
			pub := crypto.ToECDSAPub(wa[i])
			if nil == pub {
				continue
			}
			g1s[i+length].SecAddr = crypto.PubkeyToAddress(*pub)
		}
	}
	return g1s
}
func (e *Epocher) GetLeaderGroup(epochID uint64) []vm.Leader {
	eps := e.GetEpLeaderGroup(epochID)
	rbs := e.GetRBProposerGroup(epochID)
	leaders := make([]vm.Leader, 0)
	leaders = append(leaders, eps...)
	leaders = append(leaders, rbs...)
	return leaders
}
func (e *Epocher) GetProposerBn256PK(epochID uint64, idx uint64, addr common.Address) []byte {
	valSet := e.rbLeadersDb.GetStorageByteArray(epochID)

	if valSet == nil || len(valSet) == 0 {
		return nil
	}

	psValue := valSet[idx]

	proposer := Proposer{}
	err := rlp.DecodeBytes(psValue, &proposer)
	if err != nil {
		log.Error("can't rlp decode:", "err", err)

		// todo : return ??
	}

	pub := crypto.ToECDSAPub(proposer.PubSec256)

	if pub == nil {
		return nil
	}

	bingoAddr := crypto.PubkeyToAddress(*pub)

	if bingoAddr == addr {
		return proposer.PubBn256
	} else {
		return nil
	}
}

// TODO Is this  right?
func CalEpochProbabilityStaker(staker *vm.StakerInfo, epochID uint64) (infors []vm.ClientProbability, totalProbability *big.Int, err error) {
	if staker.StakingEpoch == 0 && staker.LockEpochs != 0 {
		staker.StakingEpoch = posconfig.FirstEpochId + 2
		for j := 0; j < len(staker.Partners); j++ {
			staker.Partners[j].StakingEpoch = posconfig.FirstEpochId + 2
		}
	}
	// check validator is exiting.
	if staker.LockEpochs != 0 && epochID >= staker.StakingEpoch+staker.LockEpochs-1 { // the last epoch only miner, don't send tx.
		return nil, nil, errors.New("Validator is exiting")
	}

	// check if the validator's amount(include partner) is not enough, can't delegatein
	totalAmount := big.NewInt(0).Set(staker.Amount)
	for i := 0; i < len(staker.Partners); i++ {
		if epochID < staker.Partners[i].StakingEpoch+staker.Partners[i].LockEpochs-1 { // the last epoch only miner, don't send tx.
			totalAmount.Add(totalAmount, staker.Partners[i].Amount)
		}
	}
	if staker.FeeRate != vm.PSNodeleFeeRate && totalAmount.Cmp(vm.MinValidatorStake) < 0 {
		return nil, nil, errors.New("Validator don't have enough amount.")
	}

	totalPartnerProbability := big.NewInt(0).Set(staker.StakeAmount)
	for i := 0; i < len(staker.Partners); i++ {
		if epochID < staker.Partners[i].StakingEpoch+staker.Partners[i].LockEpochs-1 { // the last epoch only miner, don't send tx.
			totalPartnerProbability.Add(totalPartnerProbability, staker.Partners[i].StakeAmount)
		}
	}

	infors = make([]vm.ClientProbability, 1)
	infors[0].ValidatorAddr = staker.Address
	infors[0].WalletAddr = staker.From
	infors[0].Probability = big.NewInt(0).Set(totalPartnerProbability)

	totalProbability = big.NewInt(0).Set(totalPartnerProbability)
	for i := 0; i < len(staker.Clients); i++ {
		if staker.Clients[i].QuitEpoch == 0 ||
			epochID < staker.Clients[i].QuitEpoch-1 {
			c := staker.Clients[i]
			info := vm.ClientProbability{}
			info.ValidatorAddr = staker.Address
			info.WalletAddr = c.Address
			info.Probability = c.StakeAmount
			totalProbability.Add(totalProbability, info.Probability)
			infors = append(infors, info)
		}
	}
	// if totalProbability > (localAmount+partners)*5, use (localAmount+partners)*5
	probabilityMax := big.NewInt(0).Set(totalPartnerProbability)
	probabilityMax.Mul(probabilityMax, big.NewInt(vm.MaxTimeDelegate+1))
	if totalProbability.Cmp(probabilityMax) > 0 {
		totalProbability = probabilityMax
	}
	return infors, totalProbability, nil
}

// incentive  use it.
func (e *Epocher) GetEpochProbability(epochId uint64, addr common.Address) (*vm.ValidatorInfo, error) {

	targetBlkNum := e.GetTargetBlkNumber(epochId)

	//stateDb, err := e.blkChain.StateAt(e.blkChain.GetBlockByNumber(targetBlkNum).Root())
	stateDb, err := e.blkChain.StateAt(e.blkChain.GetHeaderByNumber(targetBlkNum).Root)
	if err != nil {
		return nil, err
	}

	addrHash := common.BytesToHash(addr[:])
	stakerBytes := stateDb.GetStateByteArray(vm.StakersInfoAddr, addrHash)

	staker := vm.StakerInfo{}
	err = rlp.DecodeBytes(stakerBytes, &staker)
	if nil != err {
		log.Error("GetEpochProbability DecodeBytes failed", "addr", addr)
		return nil, err
	}

	infors, totalProbability, err := CalEpochProbabilityStaker(&staker, epochId)
	if err != nil {
		return nil, err
	}

	// try to get current feeRate
	feeRate := staker.FeeRate
	curStateDb, err := e.blkChain.StateAt(e.blkChain.CurrentBlock().Root())
	if err != nil {
		return nil, err
	} else {
		stakeBytesNew := curStateDb.GetStateByteArray(vm.StakersInfoAddr, addrHash)
		stakeNew := vm.StakerInfo{}
		err = rlp.DecodeBytes(stakeBytesNew, &stakeNew)
		if nil == err {
			feeRate = stakeNew.FeeRate
		}
	}

	validator := &vm.ValidatorInfo{
		FeeRate:          feeRate,
		ValidatorAddr:    staker.Address,
		WalletAddr:       staker.From,
		TotalProbability: totalProbability,
		Infos:            infors,
	}
	return validator, nil
}

func (e *Epocher) SetEpochIncentive(epochId uint64, infors [][]vm.ClientIncentive) (err error) {
	return nil
}

func recordStakeOut(infos []RefundInfo, addr common.Address, amount *big.Int)([]RefundInfo) {
	record :=  RefundInfo{
		Addr: addr,
		Amount: amount,
	}
	infos = append(infos, record)
	return infos
}
func saveStakeOut(stakeOutInfo []RefundInfo, epochID uint64) error {
	stakeByte, err := rlp.EncodeToBytes(stakeOutInfo)
	if err != nil {
		return err
	}
	_, err = posdb.GetDb().Put(epochID, posconfig.StakeOutEpochKey,stakeByte)
	if err != nil {
		log.Error("saveStakeOut Failed:", "error", err)
		return err
	}
	log.Info("Save refund information done.","epochID",epochID)
	return nil
}
func coreTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	if core.CanTransfer(db, sender, amount) {
		core.Transfer(db, sender, recipient,amount)
	}
}
func isInactiveValidator(state *state.StateDB, addr common.Address, baseEpochId uint64) bool{
	checkCount := (uint64)(64)
	for i:=(uint64)(1); i<=checkCount; i++ {
		addrs,incents := incentive.GetEpochRBLeaderActivity(state, baseEpochId-i)
		for k:=0; k<len(addrs); k++ {
			if addrs[k] == addr && incents[k]==1{
				return false
			}
		}
	}
	return true
}
func ListValidator(stateDb *state.StateDB) {
	stakers := vm.GetStakersSnap(stateDb)
	for i := 0; i < len(stakers); i++ {
		log.Info("ListValidator", "i",i,"address",stakers[i].Address)
	}
}
func CleanInactiveValidator(stateDb *state.StateDB, epochID uint64){
	stakers := vm.GetStakersSnap(stateDb)
	for i := 0; i < len(stakers); i++ {
		staker := stakers[i]
		if isInactiveValidator(stateDb, staker.Address, epochID) {
			log.Info("CleanInactiveValidator", "address",staker.Address)
			for j := 0; j < len(staker.Clients); j++ {
				coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.Clients[j].Address, staker.Clients[j].Amount)
			}
			for j := 0; j < len(staker.Partners); j++ {
				coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.Partners[j].Address, staker.Partners[j].Amount)
			}
			coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.From, staker.Amount)
			key := vm.GetStakeInKeyHash(staker.Address)

			vm.UpdateInfo(stateDb, vm.StakersInfoAddr, key, nil)
		}
	}
}
func StakeOutRun(stateDb *state.StateDB, epochID uint64) bool {
	if vm.StakeoutIsFinished(stateDb, epochID) {
		return true
	}
	vm.StakeoutSetEpoch(stateDb, epochID)
	stakeOutInfo := make([]RefundInfo,0)
	stakers := vm.GetStakersSnap(stateDb)
	for i := 0; i < len(stakers); i++ {
		// stakeout delegated client. client will expire at the same time with delegate node
		staker := stakers[i]
		var changed = false
		// LockEpochs==0 means NO expire
		if staker.LockEpochs == 0 {
			continue
		}

		// handle the staker registed in pow phase. only once
		if staker.StakingEpoch == 0 && staker.LockEpochs != 0 {
			staker.StakingEpoch = posconfig.FirstEpochId + 2
			for j := 0; j < len(staker.Partners); j++ {
				staker.Partners[j].StakingEpoch = posconfig.FirstEpochId + 2
			}
			changed = true
		}

		// update new fee rate
		//key := vm.GetStakeInKeyHash(staker.Address)
		//newFeeBytes, err := vm.GetInfo(stateDb, vm.StakersFeeAddr, key)
		//if err == nil && newFeeBytes != nil {
		//	var newFee vm.UpdateFeeRate
		//	err = rlp.DecodeBytes(newFeeBytes, &newFee)
		//	if err == nil {
		//		if newFee.EffectiveEpoch == 0 {
		//			newFee.EffectiveEpoch = staker.StakingEpoch + staker.LockEpochs
		//		}
		//
		//		if newFee.EffectiveEpoch <= epochID && staker.FeeRate != newFee.FeeRate {
		//			staker.FeeRate = newFee.FeeRate
		//			changed = true
		//		}
		//	}
		//}
		//if err != nil {
		//	log.SyslogErr("update new fee rate Failed: ", "err", err)
		//	return false
		//}

		// check if delegator want to quit.
		newClients := make([]vm.ClientInfo, 0)
		clientChanged := false
		for j := 0; j < len(staker.Clients); j++ {
			// edit the validator Amount
			if epochID >= staker.Clients[j].QuitEpoch && staker.Clients[j].QuitEpoch != 0 {
				coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.Clients[j].Address, staker.Clients[j].Amount)
				stakeOutInfo = recordStakeOut(stakeOutInfo, staker.Clients[j].Address, staker.Clients[j].Amount)
				clientChanged = true
			} else {
				newClients = append(newClients, staker.Clients[j])
			}
		}
		if clientChanged {
			staker.Clients = newClients
		}

		// check if partner want to quit.
		newPartners := make([]vm.PartnerInfo, 0)
		partnerchanged := false
		for j := 0; j < len(staker.Partners); j++ {
			// edit the validator Amount
			if epochID >= staker.Partners[j].StakingEpoch+staker.Partners[j].LockEpochs {
				coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.Partners[j].Address, staker.Partners[j].Amount)
				stakeOutInfo = recordStakeOut(stakeOutInfo, staker.Partners[j].Address, staker.Partners[j].Amount)
				partnerchanged = true
			} else {
				newPartners = append(newPartners, staker.Partners[j])
			}
		}
		if partnerchanged {
			staker.Partners = newPartners
		}

		if epochID >= staker.StakingEpoch+staker.LockEpochs {
			for j := 0; j < len(staker.Clients); j++ {
				coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.Clients[j].Address, staker.Clients[j].Amount)
				stakeOutInfo = recordStakeOut(stakeOutInfo,staker.Clients[j].Address, staker.Clients[j].Amount)
			}
			for j := 0; j < len(staker.Partners); j++ {
				coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.Partners[j].Address, staker.Partners[j].Amount)
				stakeOutInfo = recordStakeOut(stakeOutInfo, staker.Partners[j].Address, staker.Partners[j].Amount)
			}
			key := vm.GetStakeInKeyHash(staker.Address)
			// quit the validator
			coreTransfer(stateDb, vm.WanCscPrecompileAddr, staker.From, staker.Amount)
			stakeOutInfo = recordStakeOut(stakeOutInfo, staker.From, staker.Amount)
			vm.UpdateInfo(stateDb, vm.StakersInfoAddr, key, nil)
			newFeeBytes, err := vm.GetInfo(stateDb, vm.StakersFeeAddr, key)
			if err == nil && newFeeBytes != nil {
				vm.UpdateInfo(stateDb, vm.StakersFeeAddr, key, nil)
			}
			continue
		}

		// check the renew
		if epochID+vm.QuitDelay >= staker.StakingEpoch+staker.LockEpochs {
			// TODO: how to apply changed FeeRate
			if staker.NextLockEpochs != 0 {
				staker.LockEpochs = staker.NextLockEpochs
				//staker.FeeRate = staker.NextFeeRate
				// recalculate the staker.
				weight := vm.CalLocktimeWeight(staker.NextLockEpochs)
				staker.StakeAmount = big.NewInt(0)
				staker.StakeAmount.Mul(staker.Amount, big.NewInt(int64(weight)))
				staker.StakingEpoch = epochID + vm.JoinDelay
				changed = true
			}
			for j := 0; j < len(staker.Partners); j++ {
				if staker.Partners[j].Renewal {
					staker.Partners[j].LockEpochs = staker.NextLockEpochs
					weight := vm.CalLocktimeWeight(staker.NextLockEpochs)
					staker.Partners[j].StakeAmount = big.NewInt(0)
					staker.Partners[j].StakeAmount.Mul(staker.Partners[j].Amount, big.NewInt(int64(weight)))
					staker.Partners[j].StakingEpoch = epochID + vm.JoinDelay
					changed = true
				}
			}
		}
		if changed || partnerchanged || clientChanged {
			stakerBytes, err := rlp.EncodeToBytes(staker)
			if err != nil {
				// this will rollback. next slot will retry.
				log.SyslogErr("StakeOutRun Failed: ", "err", err)
				return false
			}
			vm.UpdateInfo(stateDb, vm.StakersInfoAddr, vm.GetStakeInKeyHash(staker.Address), stakerBytes)
		}
	}
	saveStakeOut(stakeOutInfo, epochID)
	return true
}
