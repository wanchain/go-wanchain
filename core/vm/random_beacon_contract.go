package vm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/pos/rbselection"
	"github.com/wanchain/go-wanchain/pos/util"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	posutil "github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
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
	// random beacon smart contract abi definition
	rbSCDefinition = `[
  {
    "constant": false,
    "inputs": [
      {
        "name": "info",
        "type": "string"
      }
    ],
    "name": "dkg1",
    "outputs": [
      
    ],
    "payable": false,
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "info",
        "type": "string"
      }
    ],
    "name": "dkg2",
    "outputs": [
      
    ],
    "payable": false,
    "type": "function"
  },
  {
    "constant": false,
    "inputs": [
      {
        "name": "info",
        "type": "string"
      }
    ],
    "name": "sigShare",
    "outputs": [
      
    ],
    "payable": false,
    "type": "function"
  },
	{
		"constant": true,
		"inputs": [
			{
				"name": "timestamp",
				"type": "uint256"
			}
		],
		"name": "getEpochId",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "timestamp",
				"type": "uint256"
			}
		],
		"name": "getRandomNumberByTimestamp",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "epochId",
				"type": "uint256"
			}
		],
		"name": "getRandomNumberByEpochId",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	}
]`
	// random beacon smart contract abi object
	rbSCAbi, errRbSCInit = abi.JSON(strings.NewReader(rbSCDefinition))

	// function "dkg1" "dkg2" "sigShare" 's solidity binary id
	dkg1Id     [4]byte
	dkg2Id     [4]byte
	sigShareId [4]byte
	getEpochIdId [4]byte
	getRandomNumberByEpochIdId [4]byte
	getRandomNumberByTimestampId [4]byte

	// prefix for the key hash
	kindCij = []byte{100}
	kindEns = []byte{101}
	kindR   = []byte{102}

	// bn256 curve's hBase
	hBase = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))

	// errors
	errDkg1Parse           = errors.New("dkg1 payload parse failed")
	errDkg2Parse           = errors.New("dkg2 payload parse failed")
	errSigParse            = errors.New("sig payload parse failed")
	errRSCode              = errors.New("rs code verify failed")
	errDiscreteLogarithmsEQ = errors.New("the equality of discrete logarithms verify failed")
	errRlpCij              = errors.New("rlp encode cij failed")
	errRlpEncryptShare     = errors.New("rlp encode encrypt share failed")
	errUnRlpCij            = errors.New("rlp decode cij failed")
	errUnRlpEncryptShare   = errors.New("rlp decode encrypt share failed")
	errInvalidCommitBytes  = errors.New("invalid dkg commit bytes")
	errInvalidEncryptShareBytes = errors.New("invalid dkg encrypt share bytes")
)

// return:
// 	current stage index;
//	elapsed slots number of current stage;
//	left slots number of current stage;
func GetRBStage(slotId uint64) (int, int, int) {
	if slotId <= posconfig.Cfg().Dkg1End {
		return RbDkg1Stage, int(slotId), int(posconfig.Cfg().Dkg1End - slotId)
	} else if slotId < posconfig.Cfg().Dkg2Begin {
		return RbDkg1ConfirmStage, int(slotId - posconfig.Cfg().Dkg1End - 1), int(posconfig.Cfg().Dkg2Begin - slotId - 1)
	} else if slotId <= posconfig.Cfg().Dkg2End {
		return RbDkg2Stage, int(slotId - posconfig.Cfg().Dkg2Begin), int(posconfig.Cfg().Dkg2End - slotId)
	} else if slotId < posconfig.Cfg().SignBegin {
		return RbDkg2ConfirmStage, int(slotId - posconfig.Cfg().Dkg2End - 1), int(posconfig.Cfg().SignBegin - slotId - 1)
	} else if slotId <= posconfig.Cfg().SignEnd {
		return RbSignStage, int(slotId - posconfig.Cfg().SignBegin), int(posconfig.Cfg().SignEnd - slotId)
	} else {
		return RbSignConfirmStage, int(slotId - posconfig.Cfg().SignEnd - 1), int(posconfig.SlotCount - slotId - 1)
	}
}

// One Epoch has 12k slots, random beacon protocol is used to generate a random r,
// which will be used to generate next epoch leaders and slot leaders.
// Random beacon protocol has 3 stages --- dkg1 (in 1k,2k slots), dkg2 (in 4k,5k slots), sigShare (in 8k, 9k slots)
type RandomBeaconContract struct {
}

//
// package init
//
func init() {
	if errRbSCInit != nil {
		panic("err in rb smart contract abi initialize")
	}

	copy(dkg1Id[:], rbSCAbi.Methods["dkg1"].Id())
	copy(dkg2Id[:], rbSCAbi.Methods["dkg2"].Id())
	copy(sigShareId[:], rbSCAbi.Methods["sigShare"].Id())
	copy(getEpochIdId[:], rbSCAbi.Methods["getEpochId"].Id())
	copy(getRandomNumberByEpochIdId[:], rbSCAbi.Methods["getRandomNumberByEpochId"].Id())
	copy(getRandomNumberByTimestampId[:], rbSCAbi.Methods["getRandomNumberByTimestamp"].Id())
}

//
// contract interface
//
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

	if methodId == dkg1Id {
		return c.dkg1(input[4:], contract, evm)
	} else if methodId == dkg2Id {
		return c.dkg2(input[4:], contract, evm)
	} else if methodId == sigShareId {
		return c.sigShare(input[4:], contract, evm)
	} else {
		epochId,_ :=util.CalEpochSlotID(evm.Time.Uint64())
		if epochId >= posconfig.Cfg().MercuryEpochId {
			if methodId == getEpochIdId {
				return c.getEpochId(input[4:], contract, evm)
			} else if methodId == getRandomNumberByEpochIdId {
				return c.getRandomNumberByEpochId(input[4:], contract, evm)
			} else if methodId == getRandomNumberByTimestampId {
				return c.getRandomNumberByTimestamp(input[4:], contract, evm)
			}
		}
		log.SyslogErr("random beacon contract no match id found")
		return nil, errors.New("no function")
	}
}

func (c *RandomBeaconContract) getEpochId(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	timestamp := new(big.Int).SetBytes(getData(payload, 0, 32)).Uint64()

	epochId,_ := posutil.CalEpochSlotID(timestamp)

	eBig := new(big.Int).SetUint64(epochId)
	return common.LeftPadBytes(eBig.Bytes(), 32), nil
}

func (c *RandomBeaconContract) getRandomNumberByEpochId(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	epochId := new(big.Int).SetBytes(getData(payload, 0, 32)).Uint64()

	r := GetStateR(evm.StateDB, epochId)

	if r == nil {
		r = big.NewInt(0)
	}

	return common.LeftPadBytes(r.Bytes(), 32), nil
}

func (c *RandomBeaconContract) getRandomNumberByTimestamp(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	timestamp := new(big.Int).SetBytes(getData(payload, 0, 32)).Uint64()

	epochId,_ := posutil.CalEpochSlotID(timestamp)

	r := GetStateR(evm.StateDB, epochId)

	if r == nil {
		r = big.NewInt(0)
	}

	return common.LeftPadBytes(r.Bytes(), 32), nil
}

func (c *RandomBeaconContract) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	if posconfig.FirstEpochId == 0 {
		return  errParameters
	}
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

	return ValidPosRBTx(stateDB, from, payload)
}
func getRBProposerGroup(eid uint64)([]bn256.G1,error){
	ep := util.GetEpocherInst()
	if ep == nil {
		return nil,  errors.New("GetEpocherInst() == nil")
	}
		pks := ep.GetRBProposerG1(eid)
	if len(pks) == 0 {
		return nil, errors.New("len(pks) == 0")
	}
	return pks,nil
}

//
// params or gas check functions
//
func ValidPosRBTx(stateDB StateDB, from common.Address, payload []byte) error {
	log.Debug("ValidPosRBTx")
	var methodId [4]byte
	copy(methodId[:], payload[:4])

	if methodId == dkg1Id {
		_, err := validDkg1(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else if methodId == dkg2Id {
		_, err := validDkg2(stateDB, uint64(time.Now().Unix()), from, payload[4:])
		return err
	} else if methodId == sigShareId {
		_, _, _, err := validSigShare(stateDB, uint64(time.Now().Unix()), from, payload[4:])
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
// 'time' should be current system time when the method is called by tx pool.
//
// 'caller' is the caller of DKG1. It should be set as Contract.CallerAddress
// when called by precompiled contract. And should be set as tx's sender when
// called by tx pool.
func validDkg1(stateDB StateDB, time uint64, caller common.Address,
	payload []byte) (*RbDKG1FlatTxPayload, error) {

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

	pks, err := getRBProposerGroupVar(eid)
	if err != nil {
		return nil, err
	}
	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RbDkg1Stage, time) {
		return nil, logError(errors.New("invalid rb stage, expect RbDkg1Stage. epochId " + strconv.FormatUint(eid, 10)))
	}

	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(pks, eid, pid, caller) {
		return nil, logError(errors.New("invalid proposer, proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// 3. prevent reset
	existC, err := GetCji(stateDB, eid, pid)
	if err == nil && len(existC) != 0 {
		return nil, logError(errors.New("dkg1 commit exist already, proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// 4.Commit has the same size
	// check same size
	nr := len(pks)
	if nr != len(dkg1Param.Commit) {
		return nil, logError(buildError("error in dkg1 params have invalid commits length", eid, pid))
	}

	// 5. Reed-Solomon code verification
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(GetPolynomialX(&pks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	temp := make([]bn256.G2, nr)
	for j := 0; j < nr; j++ {
		temp[j] = *dkg1Param.Commit[j]
	}

	if !rbselection.RScodeVerify(temp, x, int(posconfig.Cfg().PolymDegree)) {
		return nil, logError(errRSCode)
	}

	return &dkg1FlatParam, nil
}

func validDkg2(stateDB StateDB, time uint64, caller common.Address,
	payload []byte) (*RbDKG2FlatTxPayload, error) {

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

	pks, err := getRBProposerGroupVar(eid)
	if err != nil {
		return nil, err
	}

	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RbDkg2Stage, time) {
		return nil, logError(errors.New("invalid rb stage, expect RbDkg2Stage. error epochId " + strconv.FormatUint(eid, 10)))
	}

	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(pks, eid, pid, caller) {
		return nil, logError(errors.New("error proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// prevent reset
	existE, err := GetEncryptShare(stateDB, eid, pid)
	if err == nil && len(existE) != 0 {
		return nil, logError(errors.New("dkg2 encrypt share exist already, proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	commit, err := GetCji(stateDB, eid, pid)
	if err != nil || len(commit) == 0 {
		return nil, logError(buildError("error in dkg2 can't get commit data", eid, pid))
	}

	// 3. EncryptShare, Commit, Proof has the same size
	// check same size
	nr := len(pks)
	if nr != len(dkg2Param.EnShare) || nr != len(dkg2Param.Proof) || nr != len(commit) {
		return nil, logError(buildError("error in dkg2 params have different length", eid, pid))
	}

	// 4. proof verification
	for j := 0; j < nr; j++ {
		// get send public Key
		if !rbselection.VerifyDLEQ(dkg2Param.Proof[j], pks[j], *hBase, *dkg2Param.EnShare[j], *commit[j]) {
			return nil, logError(errDiscreteLogarithmsEQ)
		}
	}

	return &dkg2FlatParam, nil
}

func validSigShare(stateDB StateDB, time uint64, caller common.Address,
	payload []byte) (*RbSIGTxPayload, []bn256.G1, []RbCijDataCollector, error) {

	var sigShareParam RbSIGTxPayload
	err := rlp.DecodeBytes(payload, &sigShareParam)
	if err != nil {
		return nil, nil, nil, logError(errors.New("error in dkg param has a wrong struct"))
	}

	eid := sigShareParam.EpochId
	pid := sigShareParam.ProposerId

	pks, err := getRBProposerGroupVar(eid)
	if err != nil {
		return nil, nil, nil, err
	}

	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RbSignStage, time) {
		return nil, nil, nil, logError(errors.New("invalid rb stage, expect RbSignStage. error epochId " + strconv.FormatUint(eid, 10)))
	}

	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(pks, eid, pid, caller) {
		return nil, nil, nil, logError(errors.New(" error proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// 3. Verification
	M, err := getRBMVar(stateDB, eid)
	if err != nil {
		return nil, nil, nil, logError(buildError("getRBM error", eid, pid))
	}
	m := new(big.Int).SetBytes(M)

	var gPKShare bn256.G2

	dkgData := make([]RbCijDataCollector, 0)
	for id := range pks {
		if IsJoinDKG2(stateDB, eid, uint32(id)) {
			dkgDatum, err := GetCji(stateDB, eid, uint32(id))
			if err == nil && dkgDatum != nil {
				dkgData = append(dkgData, RbCijDataCollector{dkgDatum, &pks[id]})
				gPKShare.Add(&gPKShare, dkgDatum[pid])
			}
		}
	}

	if uint(len(dkgData)) < posconfig.Cfg().RBThres {
		return nil, nil, nil, logError(buildError("insufficient proposer", eid, pid))
	}

	mG := new(bn256.G1).ScalarBaseMult(m)
	pair1 := bn256.Pair(sigShareParam.GSignShare, hBase)
	pair2 := bn256.Pair(mG, &gPKShare)
	if pair1.String() != pair2.String() {
		return nil, nil, nil, logError(buildError("unequal si gi", eid, pid))
	}

	return &sigShareParam, pks, dkgData, nil
}

//
// public get methods
//
func GetDkg1Id() []byte {
	return dkg1Id[:]
}

func GetDkg2Id() []byte {
	return dkg2Id[:]
}

func GetSigShareId() []byte {
	return sigShareId[:]
}

// get key hash
func GetRBKeyHash(kind []byte, epochId uint64, proposerId uint32) *common.Hash {
	keyBytes := make([]byte, 16)
	copy(keyBytes, kind)
	copy(keyBytes[4:], UIntToByteSlice(epochId))
	copy(keyBytes[12:], UInt32ToByteSlice(proposerId))
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	return &hash
}

// get r's key hash
func GetRBRKeyHash(epochId uint64) *common.Hash {
	keyBytes := make([]byte, 12)
	copy(keyBytes, kindR[:])
	copy(keyBytes[4:], UIntToByteSlice(epochId))
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	return &hash
}

// get r of one epoch, if not exist return r in epoch 0
func GetR(db StateDB, epochId uint64) *big.Int {
	if epochId == posconfig.FirstEpochId {
		return GetStateR(db, posconfig.FirstEpochId)
	}
	r := GetStateR(db, epochId)
	if r == nil {
		if epochId > posconfig.FirstEpochId+2 {
			log.SyslogWarning("***Can not found random r just use the first epoch R", "epochId", epochId)
		}
		r = GetStateR(db, posconfig.FirstEpochId)
	}
	return r
}

// get r of one epoch
func GetStateR(db StateDB, epochId uint64) *big.Int {
	if epochId == posconfig.FirstEpochId {
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

// get sig of one epoch, stored in "sigShare" function
func GetSig(db StateDB, epochId uint64, proposerId uint32) (*RbSIGTxPayload, error) {
	hash := GetRBKeyHash(sigShareId[:], epochId, proposerId)
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(payloadBytes) == 0 {
		return nil, nil
	}

	var sigParam RbSIGTxPayload
	err := rlp.DecodeBytes(payloadBytes, &sigParam)
	if err != nil {
		return nil, errSigParse
	}

	return &sigParam, nil
}

// get M
func GetRBM(db StateDB, epochId uint64) ([]byte, error) {
	epochIdBigInt := big.NewInt(int64(epochId + 1))
	preRandom := GetR(db, epochId)

	buf := epochIdBigInt.Bytes()
	buf = append(buf, preRandom.Bytes()...)
	return crypto.Keccak256(buf), nil
}

func GetRBAddress() common.Address {
	return randomBeaconPrecompileAddr
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

// get encrypt share
func GetEncryptShare(db StateDB, epochId uint64, proposerId uint32) ([]*bn256.G1, error) {
	hash := GetRBKeyHash(kindEns, epochId, proposerId)
	dkgBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(dkgBytes) == 0 {
		return nil, nil
	}

	enShare := make([][]byte, 0)
	err := rlp.DecodeBytes(dkgBytes, &enShare)
	if err != nil {
		return nil, errUnRlpEncryptShare
	}

	return BytesToEncryptShare(&enShare)
}

func IsJoinDKG2(db StateDB, epochId uint64, proposerId uint32) bool {
	hash := GetRBKeyHash(kindEns, epochId, proposerId)
	dkgBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(dkgBytes) == 0 {
		return false
	}

	return true
}

//
// more public functions
//
// active： participate in all stages --- dkg1 dkg2 sign
func IsRBActive(db StateDB, epochId uint64, proposerId uint32) bool {
	hash := GetRBKeyHash(sigShareId[:], epochId, proposerId)
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if payloadBytes == nil || len(payloadBytes) == 0 {
		return false
	}
	if IsJoinDKG2(db, epochId, proposerId) {
		return true
	}

	return false
}

//
// help function for serial
//
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

//
// structures for params
//
type RbDKG1FlatTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	// 128 --
	Commit [][]byte
}

type RbDKG2FlatTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	// encrypt share
	EnShare [][]byte
	Proof   []rbselection.DLEQproofFlat
}

type RbSIGTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	GSignShare  *bn256.G1
}

//
// params bytes fields to object
//
type RbDKG1TxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Commit     []*bn256.G2
}

type RbDKG2TxPayload struct {
	EpochId    uint64
	ProposerId uint32
	// encrypt share
	EnShare    []*bn256.G1
	Proof      []rbselection.DLEQproof
}

//
// storage structures
//
type RbDKGTxPayloadPure struct {
	EpochId    uint64
	ProposerId uint32
	EnShare    []*bn256.G1
	Commit     []*bn256.G2
}

type RbCijDataCollector struct {
	cij []*bn256.G2
	pk  *bn256.G1
}

type RbSIGDataCollector struct {
	data *RbSIGTxPayload
	pk   *bn256.G1
}

//
// storage parse functions
//
func BytesToCij(d *[][]byte) ([]*bn256.G2, error) {
	l := len(*d)
	cij := make([]*bn256.G2, l, l)
	g2s := make([]bn256.G2, l, l)
	for i := 0; i < l; i++ {
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

func BytesToEncryptShare(d *[][]byte) ([]*bn256.G1, error) {
	l := len(*d)
	ens := make([]*bn256.G1, l, l)
	g1s := make([]bn256.G1, l, l)
	for i := 0; i < l; i++ {
		left, err := g1s[i].UnmarshalPure((*d)[i])
		if err != nil {
			return nil, err
		}

		if len(left) != 0 {
			return nil, errInvalidEncryptShareBytes
		}

		ens[i] = &g1s[i]
	}

	return ens, nil
}

//
// params bytes fields convert functions
//
func Dkg1FlatToDkg1(d *RbDKG1FlatTxPayload) (*RbDKG1TxPayload, error) {
	var dkgParam RbDKG1TxPayload
	dkgParam.EpochId = d.EpochId
	dkgParam.ProposerId = d.ProposerId

	l := len(d.Commit)
	dkgParam.Commit = make([]*bn256.G2, l, l)
	g2s := make([]bn256.G2, l, l)
	for i := 0; i < l; i++ {
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

func Dkg1ToDkg1Flat(d *RbDKG1TxPayload) *RbDKG1FlatTxPayload {
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

func Dkg2FlatToDkg2(d *RbDKG2FlatTxPayload) (*RbDKG2TxPayload, error) {
	var dkg2Param RbDKG2TxPayload
	dkg2Param.EpochId = d.EpochId
	dkg2Param.ProposerId = d.ProposerId

	l := len(d.EnShare)
	dkg2Param.EnShare = make([]*bn256.G1, l, l)
	g1s := make([]bn256.G1, l, l)
	for i := 0; i < l; i++ {
		left, err := g1s[i].Unmarshal(d.EnShare[i])
		if err != nil {
			return nil, err
		}

		if len(left) != 0 {
			return nil, errInvalidEncryptShareBytes
		}

		dkg2Param.EnShare[i] = &g1s[i]
	}

	l = len(d.Proof)
	dkg2Param.Proof = make([]rbselection.DLEQproof, l, l)
	for i := 0; i < l; i++ {
		(&dkg2Param.Proof[i]).ProofFlatToProof(&d.Proof[i])
	}

	return &dkg2Param, nil
}

func Dkg2ToDkg2Flat(d *RbDKG2TxPayload) *RbDKG2FlatTxPayload {
	var df RbDKG2FlatTxPayload
	df.EpochId = d.EpochId
	df.ProposerId = d.ProposerId

	l := len(d.EnShare)
	df.EnShare = make([][]byte, l)
	for i := 0; i < l; i++ {
		df.EnShare[i] = d.EnShare[i].Marshal()
	}

	l = len(d.Proof)
	df.Proof = make([]rbselection.DLEQproofFlat, l)
	for i := 0; i < l; i++ {
		df.Proof[i] = rbselection.ProofToProofFlat(&d.Proof[i])
	}

	return &df
}

//
// in file help functions
//
// check time in the right stage, dkg1 --- 1k,2k slot, dkg2 --- 5k,6k slot, sig --- 8k,9k slot
func isValidEpochStage(epochId uint64, stage int, time uint64) bool {
	eid, sid := util.CalEpochSlotID(time)
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
	if len(pks) <= int(proposerId) || int(proposerId)<0 {
		return false
	}
	ep := util.GetEpocherInst()
	if ep == nil {
		return false
	}
	pk1 := ep.GetProposerBn256PK(epochId, uint64(proposerId), address)
	if pk1 != nil {
		return bytes.Equal(pk1, pks[proposerId].Marshal())
	}
	return false
}

func buildError(err string, epochId uint64, proposerId uint32) error {
	return errors.New(fmt.Sprintf("%v epochId = %v, proposerId = %v ", err, epochId, proposerId))
}

func logError(err error) error {
	log.SyslogErr(err.Error())
	return err
}

func getSignorsNum(epochId uint64, evm *EVM) uint32 {
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

func setSignorsNum(epochId uint64, num uint32, evm *EVM) {
	tmpKey := common.Hash{0}
	dataBytes := make([]byte, 12)
	copy(dataBytes[0:], UIntToByteSlice(epochId))
	copy(dataBytes[8:], UInt32ToByteSlice(num))
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, tmpKey, dataBytes)
}

//
// variables for mock
//
var getRBMVar = GetRBM
var isValidEpochStageVar = isValidEpochStage
var isInRandomGroupVar = isInRandomGroup
var getRBProposerGroupVar = getRBProposerGroup

//
// contract abi methods
//
// dkg1: happens in 0~2k-1 slots, send the commits to chain
func (c *RandomBeaconContract) dkg1(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	log.Debug("dkg1")
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

	log.Debug("vm.dkg1", "dkg1Id", dkg1Id, "epochID", eid, "proposerId", pid, "hash", hash.Hex())
	return nil, nil
}

// dkg2: happens in 5k~7k-1 slots, send the proof, enShare to chain
func (c *RandomBeaconContract) dkg2(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	log.Debug("dkg2")
	dkg2FlatParam, err := validDkg2(evm.StateDB, evm.Time.Uint64(), contract.CallerAddress, payload)
	if err != nil {
		return nil, err
	}

	eid := dkg2FlatParam.EpochId
	pid := dkg2FlatParam.ProposerId

	// save encrypt share
	hash := GetRBKeyHash(kindEns, eid, pid)
	encryptShareBytes, err := rlp.EncodeToBytes(dkg2FlatParam.EnShare)
	if err != nil {
		return nil, logError(errRlpEncryptShare)
	}
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, encryptShareBytes)

	log.Debug("vm.dkg2", "dkgId", dkg2Id, "epochID", eid, "proposerId", pid, "hash", hash.Hex())
	return nil, nil
}

// sigShare: sign, happens in 8k~10k-1 slots, generate R if enough signers
func (c *RandomBeaconContract) sigShare(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	log.Debug("sigShare")
	sigShareParam, pks, dkgData, err := validSigShare(evm.StateDB, evm.Time.Uint64(), contract.CallerAddress, payload)
	if err != nil {
		return nil, err
	}

	eid := sigShareParam.EpochId
	pid := sigShareParam.ProposerId

	// save
	hash := GetRBKeyHash(sigShareId[:], eid, pid)
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, payload)

	/////////////////
	// calc r if not exist
	sigNum := getSignorsNum(eid, evm) + 1
	setSignorsNum(eid, sigNum, evm)
	if uint(sigNum) >= posconfig.Cfg().RBThres {
		r, err := computeRandom(evm.StateDB, eid, dkgData, pks)
		if r != nil && err == nil {
			hashR := GetRBRKeyHash(eid + 1)
			evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hashR, r.Bytes())
			evm.StateDB.AddLog(&types.Log{
				Address: contract.Address(),
				Topics:  []common.Hash{common.BigToHash(new(big.Int).SetUint64(eid)), common.BytesToHash(r.Bytes())},
				// This is a non-consensus field, but assigned here because
				// core/state doesn't know the current block number.
				BlockNumber: evm.BlockNumber.Uint64(),
			})
		}
	}

	log.Debug("contract do sig end", "epochId", eid, "proposerId", pid)

	return nil, nil
}

//
// calc random
//
// compute random[epochId+1] by data of epoch[epochId]
func computeRandom(stateDB StateDB, epochId uint64, dkgData []RbCijDataCollector, pks []bn256.G1) (*big.Int, error) {
	randomInt := GetStateR(stateDB, epochId+1)
	if randomInt != nil && randomInt.Cmp(big.NewInt(0)) != 0 {
		return randomInt, errors.New("random exist already")
	}

	if len(pks) == 0 {
		return nil, logError(errors.New("can't find random beacon proposer group"))
	}

	// collect DKG SIG
	sigData := make([]RbSIGDataCollector, 0)
	for id := range pks {
		sigDatum, err := GetSig(stateDB, epochId, uint32(id))
		if err == nil && sigDatum != nil {
			sigData = append(sigData, RbSIGDataCollector{sigDatum, &pks[id]})
		}
	}

	if uint(len(sigData)) < posconfig.Cfg().RBThres {
		return nil, logError(errors.New("insufficient sign proposer"))
	}

	gSignatureShare := make([]bn256.G1, len(sigData))
	xSig := make([]big.Int, len(sigData))
	for i, data := range sigData {
		gSignatureShare[i] = *data.data.GSignShare
		xSig[i].SetBytes(GetPolynomialX(data.pk, data.data.ProposerId))
	}

	// Compute the Output of Random Beacon
	gSignature := rbselection.LagrangeSig(gSignatureShare, xSig, int(posconfig.Cfg().PolymDegree))
	random := crypto.Keccak256(gSignature.Marshal())

	// Verification Logic for the Output of Random Beacon
	// Computation of group public key
	nr := len(pks)
	c := make([]bn256.G2, nr)
	for i := 0; i < nr; i++ {
		c[i].ScalarBaseMult(big.NewInt(int64(0)))
		for j := 0; j < len(dkgData); j++ {
			c[i].Add(&c[i], dkgData[j].cij[i])
		}
	}

	xAll := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		xAll[i].SetBytes(GetPolynomialX(&pks[i], uint32(i)))
		xAll[i].Mod(&xAll[i], bn256.Order)
	}
	gPub := rbselection.LagrangePub(c, xAll, int(posconfig.Cfg().PolymDegree))

	// mG
	mBuf, err := getRBMVar(stateDB, epochId)
	if err != nil {
		return nil, logError(err)
	}

	m := new(big.Int).SetBytes(mBuf)
	mG := new(bn256.G1).ScalarBaseMult(m)

	// Verify using pairing
	pair1 := bn256.Pair(&gSignature, rbselection.Hbase)
	pair2 := bn256.Pair(mG, &gPub)
	if pair1.String() != pair2.String() {
		return nil, logError(errors.New("final pairing check failed"))
	}

	log.SyslogInfo("compute random success", "epochId", epochId+1, "random", common.ToHex(random))
	return big.NewInt(0).SetBytes(random), nil
}


func GetValidDkg1Cnt(db StateDB, epochId uint64) uint64 {
	if db == nil {
		return 0
	}

	count := uint64(0)
	for i := 0; i < posconfig.RandomProperCount; i++ {
		c, err := GetCji(db, epochId, uint32(i))
		if err == nil && len(c) != 0 {
			count++
		}
	}

	return count
}

func GetValidDkg2Cnt(db StateDB, epochId uint64) uint64 {
	if db == nil {
		return 0
	}

	count := uint64(0)
	for i := 0; i < posconfig.RandomProperCount; i++ {
		c, err := GetEncryptShare(db, epochId, uint32(i))
		if err == nil && len(c) != 0 {
			count++
		}
	}

	return count
}

func GetValidSigCnt(db StateDB, epochId uint64) uint64 {
	if db == nil {
		return 0
	}

	count := uint64(0)
	for i := 0; i < posconfig.RandomProperCount; i++ {
		c, err := GetSig(db, epochId, uint32(i))
		if err == nil && c != nil {
			count++
		}
	}

	return count
}
