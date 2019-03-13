package vm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/pos/wanpos_crypto"
)

const (
	_ int = iota
	RbDkg1Stage
	RbDkg1ConfirmStage
	RbDkg2Stage
	RbDkg2ConfirmStage
	RbSignStage
	RbSignConfirmStage
)

var (
	rbscDefinition       = `[{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg1","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg2","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"epochId","type":"uint256"},{"name":"r","type":"uint256"}],"name":"genR","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"sigshare","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	rbscAbi, errRbscInit = abi.JSON(strings.NewReader(rbscDefinition))

	dkg1Id     [4]byte
	dkg2Id     [4]byte
	sigshareId [4]byte
	genRId     [4]byte

	kindCij   = []byte{100}
	kindEns   = []byte{200}
	// Generator of G1
	//gbase = new(bn256.G1).ScalarBaseMult(big.NewInt(int64(1)))
	// Generator of G2
	hbase = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))

	//dkgEndId = pos.Cfg().DkgEnd
	//signBeginId = pos.Cfg().SignBegin
	//signEndId = pos.Cfg().SignEnd

	errDkg1Parse 			= errors.New("dkg1 payload parse failed")
	errDkg2Parse 			= errors.New("dkg2 payload parse failed")
	errSigParse 			= errors.New("sig payload parse failed")
	errRScode				= errors.New("rs code verify failed")
	errDleq					= errors.New("dleq verify failed")
	errRlpCij				= errors.New("rlp encode cij failed")
	errRlpEns				= errors.New("rlp encode ens failed")
	errUnRlpCij				= errors.New("rlp decode cij failed")
	errUnRlpEns				= errors.New("rlp decode ens failed")
	errInvalidCommitBytes	= errors.New("invalid dkg commit bytes")
	errInvalidEnshareBytes	= errors.New("invalid dkg enshare bytes")
)

// return:
// 	current stage index;
//	elapsed slots number of current stage;
//	left slots number of current stage;
func GetRBStage(slotId uint64) (int,int,int) {
	if slotId <= pos.Cfg().Dkg1End {
		return RbDkg1Stage, int(slotId), int(pos.Cfg().Dkg1End - slotId)
	} else if slotId < pos.Cfg().Dkg2Begin {
		return RbDkg1ConfirmStage, int(slotId - pos.Cfg().Dkg1End - 1), int(pos.Cfg().Dkg2Begin - slotId - 1)
	} else if slotId <= pos.Cfg().Dkg2End {
		return RbDkg2Stage, int(slotId - pos.Cfg().Dkg2Begin), int(pos.Cfg().Dkg2End - slotId)
	} else if slotId < pos.Cfg().SignBegin {
		return RbDkg2ConfirmStage, int(slotId - pos.Cfg().Dkg2End - 1), int(pos.Cfg().SignBegin - slotId - 1)
	} else if slotId <= pos.Cfg().SignEnd {
		return RbSignStage, int(slotId - pos.Cfg().SignBegin), int(pos.Cfg().SignEnd - slotId)
	} else {
		return RbSignConfirmStage, int(slotId - pos.Cfg().SignEnd - 1), int(pos.SlotCount - slotId - 1)
	}
}

type RandomBeaconContract struct {
}

func init() {
	if errRbscInit != nil {
		panic("err in rbsc abi initialize")
	}

	copy(dkg1Id[:], rbscAbi.Methods["dkg1"].Id())
	copy(dkg2Id[:], rbscAbi.Methods["dkg2"].Id())
	copy(sigshareId[:], rbscAbi.Methods["sigshare"].Id())
	copy(genRId[:], rbscAbi.Methods["genR"].Id())
}

func GetDkg1Id() []byte {
	return dkg1Id[:]
}

func GetDkg2Id() []byte {
	return dkg2Id[:]
}

func GetSigshareId() []byte {
	return sigshareId[:]
}

func (c *RandomBeaconContract) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *RandomBeaconContract) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	// check data
	if len(input) < 4 {
		return nil, errParameters
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	log.Debug("RandomBeaconContract is called", "inputLen", len(input), "methodId", methodId, "dkg1Id", dkg1Id, "dkg2Id", dkg2Id, "sigshareId", sigshareId, "genRId", genRId)

	if methodId == dkg1Id {
		return c.dkg1(input[4:], contract, evm)
	} else if methodId == dkg2Id {
		return c.dkg2(input[4:], contract, evm)
	} else if methodId == sigshareId {
		return c.sigshare(input[4:], contract, evm)
	} else {
		log.Debug("No match id found")
	}

	return nil, nil
}

func ValidPosTx(stateDB StateDB, from common.Address, payload []byte, gasPrice *big.Int,
	intrGas *big.Int, txValue *big.Int, gasLimit *big.Int) error {
	if intrGas == nil || intrGas.BitLen() > 64 || gasLimit == nil || intrGas.Cmp(gasLimit) > 0 {
		return ErrOutOfGas
	}

	if txValue.Sign() != 0 {
		return ErrInvalidPosValue
	}

	if gasPrice == nil || gasPrice.Sign() != 1 {
		return ErrInvalidGasPrice
	}

	totalCost := new(big.Int).Mul(gasPrice, gasLimit)
	totalCost.Add(totalCost, txValue)
	if stateDB.GetBalance(from).Cmp(totalCost) < 0 {
		return ErrOutOfGas
	}

	var methodId [4]byte
	copy(methodId[:], payload[:4])

	if methodId == dkg1Id {
		_, err := validDkg1(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else if methodId == dkg2Id {
		_, err := validDkg2(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else if methodId == sigshareId {
		_, _, _, err := validSigshare(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else {
		return errParameters
	}

}


func (c *RandomBeaconContract) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	if stateDB == nil || signer == nil || tx == nil {
		return errParameters
	}

	payload := tx.Data()
	if len(payload) < 4 {
		return errParameters
	}

	from, err := types.Sender(signer, tx)
	if err != nil {
		return err
	}

	var methodId [4]byte
	copy(methodId[:], payload[:4])

	if methodId == dkg1Id {
		_, err := validDkg1(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else if methodId == dkg2Id {
		_, err := validDkg2(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else if methodId == sigshareId {
		_, _, _, err := validSigshare(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else {
		return errParameters
	}
}

// ValidDkg1 verify DKG1 precompiled contract transaction
//
// 'time' is the time when tx is sealed into block,
// 'time' should be block timestamp when the method is called by sealing
// block and block verify.
// 'time' should be current system time when the method is called by txpool.
//
// 'caller' is the caller of DKG1. It should be set as Contract.CallerAddress
// when called by precompiled contract. And should be set as tx's sender when
// called by txpool.
func validDkg1(statedb StateDB, time uint64, caller common.Address,
	payload []byte) (*RbDKG1FlatTxPayload, error) {
	log.Info("valid dkg1 begin", "time", time, "calller", caller)

	var dkg1FlatParam RbDKG1FlatTxPayload
	err := rlp.DecodeBytes(payload, &dkg1FlatParam)
	if err != nil {
		return nil, logError(errDkg1Parse)
	}

	dkg1Param, err := Dkg1FlatToDkg1(&dkg1FlatParam)
	if err != nil {
		return nil, logError(err)
	}

	eid := dkg1Param.EpochId
	pid := dkg1Param.ProposerId
	log.Info("dkg1 transaction info", "epochId", eid, "proposerId", pid)

	pks := getRBProposerGroupVar(eid)

	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RbDkg1Stage, time) {
		return nil, logError(errors.New("invalid rb stage, expect RbDkg1Stage. epochId " + strconv.FormatUint(eid, 10)))
	}

	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(pks, eid, pid, caller) {
		return nil, logError(errors.New("invalid proposer, proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// prevent reset
	existC, err := GetCji(statedb, eid, pid)
	if err == nil && len(existC) != 0 {
		return nil, logError(errors.New("dkg1 commit exist already, proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// 3. Enshare, Commit, Proof has the same size
	// check same size
	nr := len(pks)
	if nr != len(dkg1Param.Commit) {
		return nil, logError(buildError("error in dkg1 params have invalid commits length", eid, pid))
	}

	// 4. Reed-Solomon code verification
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(GetPolynomialX(&pks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	temp := make([]bn256.G2, nr)
	for j := 0; j < nr; j++ {
		temp[j] = *dkg1Param.Commit[j]
	}

	if !wanpos.RScodeVerify(temp, x, int(pos.Cfg().PolymDegree)) {
		return nil, logError(errRScode)
	}

	return &dkg1FlatParam, nil
}

func validDkg2(statedb StateDB, time uint64, caller common.Address,
	payload []byte) (*RbDKG2FlatTxPayload, error) {
	log.Info("valid dkg2 begin", "time", time, "calller", caller)

	var dkg2FlatParam RbDKG2FlatTxPayload
	err := rlp.DecodeBytes(payload, &dkg2FlatParam)
	if err != nil {
		return nil, logError(errDkg2Parse)
	}

	dkg2Param, err := Dkg2FlatToDkg2(&dkg2FlatParam)
	if err != nil {
		return nil, logError(err)
	}

	eid := dkg2Param.EpochId
	pid := dkg2Param.ProposerId
	log.Info("dkg2 transaction info", "epochId", eid, "proposerId", pid)

	pks := getRBProposerGroupVar(eid)
	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RbDkg2Stage, time) {
		return nil, logError(errors.New("invalid rb stage, expect RbDkg2Stage. error epochId " + strconv.FormatUint(eid, 10)))
	}

	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(pks, eid, pid, caller) {
		return nil, logError(errors.New("error proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// prevent reset
	existE, err := GetEns(statedb, eid, pid)
	if err == nil && len(existE) != 0 {
		return nil, logError(errors.New("dkg2 enshare exist already, proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	commit, err := GetCji(statedb, eid, pid)
	if err != nil || len(commit) == 0 {
		log.Error("get commit for dkg2 fail", "eid", eid, "pid", pid, "err", err)
		return nil, logError(buildError("error in dkg2 can't get commit data", eid, pid))
	}

	// 3. Enshare, Commit, Proof has the same size
	// check same size
	nr := len(pks)
	if nr != len(dkg2Param.Enshare) || nr != len(commit) {
		return nil, logError(buildError("error in dkg2 params have different length", eid, pid))
	}

	// 4. proof verification
	for j := 0; j < nr; j++ {
		// get send public Key
		if !wanpos.VerifyDLEQ(dkg2Param.Proof[j], pks[j], *hbase, *dkg2Param.Enshare[j], *commit[j]) {
			return nil, logError(errDleq)
		}
	}

	return &dkg2FlatParam, nil
}


func validSigshare(statedb StateDB, time uint64, caller common.Address,
	payload []byte) (*RbSIGTxPayload, []bn256.G1, []RbCijDataCollector, error) {
	log.Info("valid sigshare begin", "time", time, "calller", caller)

	var sigshareParam RbSIGTxPayload
	err := rlp.DecodeBytes(payload, &sigshareParam)
	if err != nil {
		return nil, nil, nil, logError(errors.New("error in dkg param has a wrong struct"))
	}

	eid := sigshareParam.EpochId
	pid := sigshareParam.ProposerId
	log.Info("sigshare transaction param", "epochId", eid, "proposerId", pid)

	pks := getRBProposerGroupVar(eid)
	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RbSignStage, time) {
		return nil, nil, nil, logError(errors.New("invalid rb stage, expect RbSignStage. error epochId " + strconv.FormatUint(eid, 10)))
	}

	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(pks, eid, pid, caller) {
		return nil, nil, nil, logError(errors.New(" error proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// 3. Verification
	M, err := getRBMVar(statedb, eid)
	if err != nil {
		return nil, nil, nil, logError(buildError("getRBM error", eid, pid))
	}
	m := new(big.Int).SetBytes(M)

	var gpkshare bn256.G2

	dkgData := make([]RbCijDataCollector, 0)
	for id := range pks {
		if isCjiValid(statedb, eid, uint32(id)) {
			dkgDataOne, err := GetCji(statedb, eid, uint32(id))
			if err == nil && dkgDataOne != nil {
				dkgData = append(dkgData, RbCijDataCollector{dkgDataOne, &pks[id]})
				gpkshare.Add(&gpkshare, dkgDataOne[pid])
			}
		}
	}

	if uint(len(dkgData)) < pos.Cfg().RBThres {
		return nil, nil, nil, logError(buildError("insufficient proposer", eid, pid))
	}

	mG := new(bn256.G1).ScalarBaseMult(m)
	pair1 := bn256.Pair(sigshareParam.Gsigshare, hbase)
	pair2 := bn256.Pair(mG, &gpkshare)
	if pair1.String() != pair2.String() {
		return nil, nil, nil, logError(buildError("unequal si gi", eid, pid))
	}

	return &sigshareParam, pks, dkgData, nil
}


func GetRBKeyHash(kind []byte, epochId uint64, proposerId uint32) *common.Hash {
	keyBytes := make([]byte, 12 + len(kind))
	copy(keyBytes, kind)
	copy(keyBytes[4:], UIntToByteSlice(epochId))
	copy(keyBytes[12:], UInt32ToByteSlice(proposerId))
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	return &hash
}

func GetRBRKeyHash(epochId uint64) *common.Hash {
	keyBytes := make([]byte, 12)
	copy(keyBytes, genRId[:])
	copy(keyBytes[4:], UIntToByteSlice(epochId))
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	return &hash
}

func GetR(db StateDB, epochId uint64) *big.Int {
	r := GetStateR(db, epochId)
	if r == nil {
		log.Warn("***Can not found random r just use epoch 0 R", "epochID", epochId)
		r = GetStateR(db, 0)
	}
	return r
}

// get r in statedb
func GetStateR(db StateDB, epochId uint64) *big.Int {
	if epochId == 0 {
		return new(big.Int).SetBytes(crypto.Keccak256(big.NewInt(1).Bytes()))
	}
	hash := GetRBRKeyHash(epochId)
	rBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(rBytes) != 0 {
		r := new(big.Int).SetBytes(rBytes)
		return r
	}
	return nil
}

func IsRBActive(db StateDB, epochId uint64, proposerId uint32) bool {
	hash := GetRBKeyHash(sigshareId[:], epochId, proposerId)
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if payloadBytes == nil || len(payloadBytes) == 0 {
		return false
	}

	return true
}

func GetSig(db StateDB, epochId uint64, proposerId uint32) (*RbSIGTxPayload, error) {
	hash := GetRBKeyHash(sigshareId[:], epochId, proposerId)
	log.Debug("vm.GetSig", "len(sigshareId)", len(sigshareId), "epochID", epochId, "proposerId",
		proposerId, "hash", hash.Hex())
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	// if missing
	if len(payloadBytes) == 0 {
		return nil, nil
	}

	var sigParam RbSIGTxPayload
	err := rlp.DecodeBytes(payloadBytes, &sigParam)
	// if in wrong format
	if err != nil {
		return nil, errSigParse
	}

	return &sigParam, nil
}

func GetRBM(db StateDB, epochId uint64) ([]byte, error) {
	epochIdBigInt := big.NewInt(int64(epochId + 1))
	preRandom := GetR(db, epochId)

	buf := epochIdBigInt.Bytes()
	buf = append(buf, preRandom.Bytes()...)
	return crypto.Keccak256(buf), nil
}

func GetRBAbiDefinition() string {
	return rbscDefinition
}

func GetRBAddress() common.Address {
	return randomBeaconPrecompileAddr
}

func UIntToByteSlice(num uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, num)
	return b
}
func UInt32ToByteSlice(num uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, num)
	return b
}

func ByteSliceToUInt(bs []byte) uint64 {
	return binary.LittleEndian.Uint64(bs)
}
func ByteSliceToUInt32(bs []byte) uint32 {
	return binary.LittleEndian.Uint32(bs)
}

type RbDKG1TxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Commit     []*bn256.G2
}

type RbDKG2TxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Enshare    []*bn256.G1
	Proof      []wanpos.DLEQproof
}

type RbDKG1FlatTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	// 128 --
	Commit     [][]byte
}

type RbDKG2FlatTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	// 64
	Enshare    [][]byte
	Proof      []wanpos.DLEQproofFlat
}

type RbSIGTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Gsigshare  *bn256.G1
}

type RbDKGTxPayloadPure struct {
	EpochId    uint64
	ProposerId uint32
	Enshare    []*bn256.G1
	Commit     []*bn256.G2
}


func BytesToCij(d *[][]byte) ([]*bn256.G2, error) {
	l := len(*d)
	cij := make([]*bn256.G2, l, l)
	g2s := make([]bn256.G2, l, l)
	for i := 0; i<l; i++ {
		//var g2 bn256.G2
		left, err := g2s[i].UnmarshalPure((*d)[i])
		if err != nil {
			return nil, err
		}

		if len(left) != 0 {
			return nil, errInvalidCommitBytes
		}

		cij[i] = &g2s[i]
	}

	return cij, nil
}

func BytesToEns(d *[][]byte) ([]*bn256.G1, error) {
	l := len(*d)
	ens := make([]*bn256.G1, l, l)
	g1s := make([]bn256.G1, l, l)
	for i := 0; i<l; i++ {
		left, err := g1s[i].UnmarshalPure((*d)[i])
		if err != nil {
			return nil, err
		}

		if len(left) != 0 {
			return nil, errInvalidEnshareBytes
		}

		ens[i] = &g1s[i]
	}

	return ens, nil
}

func Dkg1FlatToDkg1(d * RbDKG1FlatTxPayload) (*RbDKG1TxPayload, error) {
	var dkgParam RbDKG1TxPayload
	dkgParam.EpochId = d.EpochId
	dkgParam.ProposerId = d.ProposerId

	l := len(d.Commit)
	dkgParam.Commit = make([]*bn256.G2, l, l)
	g2s := make([]bn256.G2, l, l)
	for i := 0; i<l; i++ {
		//var g2 bn256.G2
		left, err := g2s[i].Unmarshal(d.Commit[i])
		if err != nil {
			return nil, err
		}

		if len(left) != 0 {
			return nil, errInvalidCommitBytes
		}

		dkgParam.Commit[i] = &g2s[i]
	}

	return &dkgParam, nil
}

func Dkg1ToDkg1Flat(d * RbDKG1TxPayload) *RbDKG1FlatTxPayload {
	var df RbDKG1FlatTxPayload
	df.EpochId = d.EpochId
	df.ProposerId = d.ProposerId

	l := len(d.Commit)

	df.Commit = make([][]byte, l)
	for i := 0; i < l; i++ {
		df.Commit[i] = d.Commit[i].Marshal()
	}

	return &df
}

func Dkg2FlatToDkg2(d * RbDKG2FlatTxPayload) (*RbDKG2TxPayload, error) {
	var dkg2Param RbDKG2TxPayload
	dkg2Param.EpochId = d.EpochId
	dkg2Param.ProposerId = d.ProposerId

	l := len(d.Enshare)
	dkg2Param.Enshare = make([]*bn256.G1, l, l)
	g1s := make([]bn256.G1, l, l)
	for i := 0; i<l; i++ {
		left, err := g1s[i].Unmarshal(d.Enshare[i])
		if err != nil {
			return nil, err
		}

		if len(left) != 0 {
			return nil, errInvalidEnshareBytes
		}

		dkg2Param.Enshare[i] = &g1s[i]
	}

	l = len(d.Proof)
	dkg2Param.Proof = make([]wanpos.DLEQproof, l, l)
	for i := 0; i<l; i++ {
		(&dkg2Param.Proof[i]).ProofFlatToProof(&d.Proof[i])
	}
	return &dkg2Param, nil
}

func Dkg2ToDkg2Flat(d * RbDKG2TxPayload) *RbDKG2FlatTxPayload {
	var df RbDKG2FlatTxPayload
	df.EpochId = d.EpochId
	df.ProposerId = d.ProposerId

	l := len(d.Enshare)
	df.Enshare = make([][]byte, l)
	for i := 0; i < l; i++ {
		df.Enshare[i] = d.Enshare[i].Marshal()
	}

	l = len(d.Proof)
	df.Proof = make([]wanpos.DLEQproofFlat, l)
	for i := 0; i < l; i++ {
		df.Proof[i] = wanpos.ProofToProofFlat(&d.Proof[i])
	}

	return &df
}

// stage 0, 1 dkg sign
func isValidEpochStage(epochId uint64, stage int, time uint64) bool {
	eid, sid := postools.CalEpochSlotID(time)
	if epochId != eid {
		return false
	}

	ss, _, _ := GetRBStage(sid)
	if ss != stage {
		return false
	}

	return true
}

func isInRandomGroup(pks []bn256.G1, epochId uint64, proposerId uint32, address common.Address) bool {
	if len(pks) <= int(proposerId) {
		return false
	}
	pk1 := posdb.GetProposerBn256PK(epochId, uint64(proposerId), address)
	if pk1 != nil {
		return bytes.Equal(pk1, pks[proposerId].Marshal())
	}
	return false
}

var getRBProposerGroupVar = posdb.GetRBProposerGroup
var getRBMVar = GetRBM
var isValidEpochStageVar = isValidEpochStage
var isInRandomGroupVar = isInRandomGroup

func buildError(err string, epochId uint64, proposerId uint32) error {
	return errors.New(fmt.Sprintf("%v epochId = %v, proposerId = %v ", err, epochId, proposerId))
}

func logError(err error) error {
	log.Error(err.Error())
	return err
}

func GetPolynomialX(pk *bn256.G1, proposerId uint32) []byte {
	return crypto.Keccak256(pk.Marshal(), big.NewInt(int64(proposerId)).Bytes())
}

func GetCji(db StateDB, epochId uint64, proposerId uint32) ([]*bn256.G2, error) {
	hash := GetRBKeyHash(kindCij, epochId, proposerId)
	dkgBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(dkgBytes) == 0 {
		return nil, nil
	}
	cij := make([][]byte, 0)
	err := rlp.DecodeBytes(dkgBytes, &cij)
	if err != nil {
		return nil, errUnRlpCij
	}

	return BytesToCij(&cij)
}

func isCjiValid(db StateDB, epochId uint64, proposerId uint32) bool {
	hash := GetRBKeyHash(kindEns, epochId, proposerId)
	dkgBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if dkgBytes == nil || len(dkgBytes) == 0 {
		return false
	}
	return true
}

func GetEns(db StateDB, epochId uint64, proposerId uint32) ([]*bn256.G1, error) {
	hash := GetRBKeyHash(kindEns, epochId, proposerId)
	dkgBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(dkgBytes) == 0 {
		return nil, nil
	}
	ens := make([][]byte, 0)
	err := rlp.DecodeBytes(dkgBytes, &ens)
	if err != nil {
		return nil, errUnRlpEns
	}

	return BytesToEns(&ens)
}

func (c *RandomBeaconContract) dkg1(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	dkgStart := time.Now()

	log.Info("contract do dkg1 begin")
	dkg1FlatParam, err := validDkg1(evm.StateDB, evm.Time.Uint64(), contract.CallerAddress, payload)
	if err != nil {
		return nil, err
	}

	eid := dkg1FlatParam.EpochId
	pid := dkg1FlatParam.ProposerId

	// save cij
	hash := GetRBKeyHash(kindCij, eid, pid)
	cijBytes, err := rlp.EncodeToBytes(dkg1FlatParam.Commit)
	if err != nil {
		return nil, logError(errRlpCij)
	}
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, cijBytes)

	// TODO: add an dkg event
	// add event
	log.Debug("vm.dkg1", "dkg1Id", dkg1Id, "epochID", eid, "proposerId", pid, "hash", hash.Hex())
	dkgTime := time.Since(dkgStart)
	log.Info("dkg1 time", "dkgTime", dkgTime)
	return nil, nil
}

func (c *RandomBeaconContract) dkg2(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	dkgStart := time.Now()

	dkg2FlatParam, err := validDkg2(evm.StateDB, evm.Time.Uint64(), contract.CallerAddress, payload)
	if err != nil {
		return nil, err
	}

	eid := dkg2FlatParam.EpochId
	pid := dkg2FlatParam.ProposerId

	// save ens
	hash := GetRBKeyHash(kindEns, eid, pid)
	ensBytes, err := rlp.EncodeToBytes(dkg2FlatParam.Enshare)
	if err != nil {
		return nil, logError(errRlpEns)
	}
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, ensBytes)

	// TODO: add an dkg event
	// add event
	log.Debug("vm.dkg2", "dkgId", dkg2Id, "epochID", eid, "proposerId", pid, "hash", hash.Hex())
	dkgTime := time.Since(dkgStart)
	log.Info("dkg2 time:", "dkgTime", dkgTime)
	return nil, nil
}

func getSigsNum(epochId uint64, evm *EVM) uint32 {
	tmpKey := common.Hash{0}
	bs := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, tmpKey)
	if bs != nil {
		eid := ByteSliceToUInt(bs)
		if eid == epochId {
			num := ByteSliceToUInt32(bs[8:12])
			return num
		}
	}
	return 0
}

func setSigsNum(epochId uint64, num uint32, evm *EVM) {
	tmpKey := common.Hash{0}
	dataBytes := make([]byte, 12)
	copy(dataBytes[0:], UIntToByteSlice(epochId))
	copy(dataBytes[8:], UInt32ToByteSlice(num))
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, tmpKey, dataBytes)
}

func (c *RandomBeaconContract) sigshare(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	t1 := time.Now()

	sigshareParam, pks, dkgData, err := validSigshare(evm.StateDB, evm.Time.Uint64(), contract.CallerAddress, payload)
	if err != nil {
		return nil, err
	}

	eid := sigshareParam.EpochId
	pid := sigshareParam.ProposerId

	// save
	hash := GetRBKeyHash(sigshareId[:], eid, pid)
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, payload)

	/////////////////
	// calc r if not exist
	signum := getSigsNum(eid, evm) + 1
	setSigsNum(eid, signum, evm)
	if uint(signum) >= pos.Cfg().RBThres {
		r, err := computeRandom(evm.StateDB, eid, dkgData, pks)
		if r != nil && err == nil {
			hashR := GetRBRKeyHash(eid + 1)
			evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hashR, r.Bytes())
			log.Info("generate random", "epochId", eid+1, "r", r)
		}
	}

	log.Info("contract do sig end", "epochId", eid, "proposerId", pid)

	elapsed := time.Since(t1)
	fmt.Println("***App elapsed: ", elapsed)
	return nil, nil
}

type RbCijDataCollector struct {
	cij []*bn256.G2
	pk   *bn256.G1
}

type RbSIGDataCollector struct {
	data *RbSIGTxPayload
	pk   *bn256.G1
}

// compute random[epochid+1] by data of epoch[epochid]
func computeRandom(statedb StateDB, epochId uint64, dkgDatas []RbCijDataCollector, pks []bn256.G1) (*big.Int, error) {
	log.Info("do compute random", "epochId", epochId)
	randomInt := GetStateR(statedb, epochId+1)
	if randomInt != nil && randomInt.Cmp(big.NewInt(0)) != 0 {
		// exist already
		log.Info("random exist already", "epochId", epochId+1, "random", randomInt.String())
		return randomInt, errors.New("random exist already")
	}

	if len(pks) == 0 {
		log.Error("can't find random beacon proposer group")
		return nil, errors.New("can't find random beacon proposer group")
	}

	// collact DKG SIG
	sigDatas := make([]RbSIGDataCollector, 0)
	for id := range pks {
		sigData, err := GetSig(statedb, epochId, uint32(id))
		if err == nil && sigData != nil {
			sigDatas = append(sigDatas, RbSIGDataCollector{sigData, &pks[id]})
		}
	}

	log.Info("dkgDatas and sigDatas length", "len(dkgDatas)", len(dkgDatas), "len(sigDatas)", len(sigDatas))
	if uint(len(sigDatas)) < pos.Cfg().RBThres {
		log.Warn("compute random fail, insufficient proposer", "epochId", epochId, "min", pos.Cfg().RBThres, "acture", len(sigDatas))
		return nil, errors.New("insufficient proposer")
	}

	gsigshare := make([]bn256.G1, len(sigDatas))
	xSig := make([]big.Int, len(sigDatas))
	for i, data := range sigDatas {
		gsigshare[i] = *data.data.Gsigshare
		xSig[i].SetBytes(GetPolynomialX(data.pk, data.data.ProposerId))
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
			c[i].Add(&c[i], dkgDatas[j].cij[i])
		}
	}

	xAll := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		xAll[i].SetBytes(GetPolynomialX(&pks[i], uint32(i)))
		xAll[i].Mod(&xAll[i], bn256.Order)
	}
	gPub := wanpos.LagrangePub(c, xAll, int(pos.Cfg().PolymDegree))

	// mG
	mBuf, err := getRBMVar(statedb, epochId)
	if err != nil {
		log.Error("get m fail", "err", err)
		return nil, err
	}

	m := new(big.Int).SetBytes(mBuf)
	mG := new(bn256.G1).ScalarBaseMult(m)

	// Verify using pairing
	pair1 := bn256.Pair(&gsig, wanpos.Hbase)
	pair2 := bn256.Pair(mG, &gPub)
	log.Info("verify random", "pair1", pair1.String(), "pair2", pair2.String())
	if pair1.String() != pair2.String() {
		return nil, errors.New("final pairing check failed")
	}

	log.Info("compute random success", "epochId", epochId+1, "random", common.Bytes2Hex(random))
	return big.NewInt(0).SetBytes(random), nil
}
