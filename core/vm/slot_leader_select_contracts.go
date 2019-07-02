package vm

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/pos/uleaderselection"
	"github.com/wanchain/go-wanchain/rlp"

	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/pos/util/convert"

	"github.com/wanchain/go-wanchain/functrace"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/core/types"
)

const (
	SlotLeaderStag1 = "slotLeaderStag1"
	SlotLeaderStag2 = "slotLeaderStag2"

	SlotLeaderStag1Indexes = "slotLeaderStag1Indexes"
	SlotLeaderStag2Indexes = "slotLeaderStag2Indexes"
)

var (
	slotLeaderSCDef = `[
			{
				"constant": false,
				"type": "function",
				"inputs": [
					{
						"name": "data",
						"type": "string"
					}
				],
				"name": "slotLeaderStage1MiSave",
				"outputs": [
					{
						"name": "data",
						"type": "string"
					}
				]
			},
			{
				"constant": false,
				"type": "function",
				"inputs": [
					{
						"name": "data",
						"type": "string"
					}
				],
				"name": "slotLeaderStage2InfoSave",
				"outputs": [
					{
						"name": "data",	
						"type": "string"
					}
				]
			}
		]`
	slotLeaderAbi, errSlotLeaderSCInit = abi.JSON(strings.NewReader(slotLeaderSCDef))
	stgOneIdArr, stgTwoIdArr           [4]byte

	scCallTimes = "SLOT_LEADER_SC_CALL_TIMES"
)

var (
	ErrEpochID                         = errors.New("EpochID is not valid")
	ErrIllegalSender                   = errors.New("sender is not in epoch leaders ")
	ErrInvalidLocalPublicKey           = errors.New("getLocalPublicKey error, do not found unlock address")
	ErrInvalidPreEpochLeaders          = errors.New("can not found pre epoch leaders return epoch 0")
	ErrInvalidGenesisPk                = errors.New("invalid GenesisPK hex string")
	ErrSlotLeaderGroupNotReady         = errors.New("slot leaders group not ready")
	ErrSlotIDOutOfRange                = errors.New("slot id index out of range")
	ErrPkNotInCurrentEpochLeadersGroup = errors.New("local public key is not in current Epoch leaders")
	ErrInvalidRandom                   = errors.New("get random message error")
	ErrNotOnCurve                      = errors.New("not on curve")
	ErrTx1AndTx2NotConsistent          = errors.New("stageOneMi is not equal sageTwoAlphaPki")
	ErrEpochLeaderNotReady             = errors.New("epoch leaders are not ready")
	ErrNoTx2TransInDB                  = errors.New("tx2 is not in db")
	ErrCollectTxData                   = errors.New("collect tx data error")
	ErrRlpUnpackErr                    = errors.New("RlpUnpackDataForTx error")
	ErrNoTx1TransInDB                  = errors.New("GetStg1StateDbInfo: Found not data of key")
	ErrVerifyStg1Data                  = errors.New("stg1 data get from StateDb verified failed")
	ErrDleqProof                       = errors.New("VerifyDleqProof false")
	ErrInvalidTxLen                    = errors.New("len(mi)==0 or len(alphaPkis) is not right")
	ErrInvalidTx1Range                 = errors.New("slot leader tx1 is not in invalid range")
	ErrInvalidTx2Range                 = errors.New("slot leader tx2 is not in invalid range")
	ErrInvalidProof                    = errors.New("proof is bigZero")
	ErrPowRcvPosTrans                  = errors.New("pow phase receive pos protocol trans")
	ErrDuplicateStg1                   = errors.New("stage one transaction exists")
	ErrDuplicateStg2                   = errors.New("stage two transaction exists")
)

func init() {
	if errSlotLeaderSCInit != nil {
		panic("err in slot leader sc initialize :" + errSlotLeaderSCInit.Error())
	}

	stgOneIdArr, _ = GetStage1FunctionID(slotLeaderSCDef)
	stgTwoIdArr, _ = GetStage2FunctionID(slotLeaderSCDef)
}

type slotLeaderSC struct {
}

func (c *slotLeaderSC) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *slotLeaderSC) Run(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	functrace.Enter()
	log.Debug("slotLeaderSC run is called")

	if len(in) < 4 {
		return nil, errParameters
	}

	var methodId [4]byte
	copy(methodId[:], in[:4])
	var from common.Address
	from = contract.CallerAddress

	if methodId == stgOneIdArr {
		vldReset := validStg1Reset(evm.StateDB, from, in[:], evm.Time.Uint64())
		vldService := validStg1Service(from, in[:])

		if !(vldReset && vldService) {
			return nil, errors.New("ValidTx stg1")
		}
		return handleStgOne(in[:], contract, evm) //Do not use [4:] because it has do it in function
	} else if methodId == stgTwoIdArr {
		vldReset := validStg2Reset(evm.StateDB, from, in[:], evm.Time.Uint64())
		vldService := validStg2Service(evm.StateDB, from, in[:])

		if !(vldReset && vldService) {
			return nil, errors.New("ValidTx stg2")
		}
		return handleStgTwo(in[:], contract, evm) //Do not use [4:] because it has do it in function
	}

	functrace.Exit()
	log.SyslogErr("slotLeaderSC:Run", "", errMethodId.Error())
	return nil, errMethodId
}

func (c *slotLeaderSC) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {

	if posconfig.FirstEpochId == 0 {
		log.SyslogErr("slotLeaderSC:ValidTx", "", ErrPowRcvPosTrans.Error())
		return ErrPowRcvPosTrans
	}

	from, err := types.Sender(signer, tx)
	if err != nil {
		return err
	}

	payload := tx.Data()
	if len(payload) < 4 {
		return errParameters
	}

	var methodId [4]byte
	copy(methodId[:], payload[:4])

	if methodId == stgOneIdArr {
		vldReset := validStg1Reset(stateDB, from, payload, uint64(time.Now().Unix()))
		vldService := validStg1Service(from, payload)

		if vldReset && vldService {
			return nil
		} else {
			return errors.New("ValidTx stg1")
		}
	} else if methodId == stgTwoIdArr {
		vldReset := validStg2Reset(stateDB, from, payload, uint64(time.Now().Unix()))
		vldService := validStg2Service(stateDB, from, payload)

		if vldReset && vldService {
			return nil
		} else {
			return errors.New("ValidTx stg2")
		}
	} else {
		log.SyslogErr("slotLeaderSC:ValidTx", "", errMethodId.Error())
		return errMethodId
	}
}

// valid common data
func ValidPosELTx(stateDB StateDB, from common.Address, payload []byte) error {

	if len(payload) < 4 {
		return errParameters
	}

	var methodId [4]byte
	copy(methodId[:], payload[:4])

	if methodId == stgOneIdArr {
		if validStg1Reset(stateDB, from, payload, uint64(time.Now().Unix())) {
			return nil
		} else {
			return errors.New("ValidPosELTx stage1 error")
		}
	} else if methodId == stgTwoIdArr {
		if validStg2Reset(stateDB, from, payload, uint64(time.Now().Unix())) {
			return nil
		} else {
			return errors.New("ValidPosELTx stage2 error")
		}
	} else {
		log.SyslogErr("slotLeaderSC:ValidTx", "", errMethodId.Error())
		return errMethodId
	}
}

func validStg1Reset(stateDB StateDB, from common.Address, payload []byte, time uint64) bool {
	epochIDBuf, selfIndexBuf, err := RlpGetStage1IDFromTx(payload)
	if err != nil {
		log.Error("validStg1Reset failed")
		return false
	}

	// epoch stage
	invalidStage := isInValidStage(convert.BytesToUint64(epochIDBuf), time, posconfig.Sma1Start, posconfig.Sma1End)
	if !invalidStage {
		return false
	}
	// duplicated
	dup := isDuplicateTrans(stateDB, convert.BytesToUint64(epochIDBuf), convert.BytesToUint64(selfIndexBuf), SlotLeaderStag1)
	if dup {
		return false
	}

	return true
}

func validStg2Reset(stateDB StateDB, from common.Address, payload []byte, time uint64) bool {
	epochIDBuf, selfIndexBuf, err := RlpGetStage2IDFromTx(payload)
	if err != nil {
		log.Error("validStg1Reset failed")
		return false
	}

	// epoch stage
	invalidStage := isInValidStage(convert.BytesToUint64(epochIDBuf), time, posconfig.Sma2Start, posconfig.Sma2End)
	if !invalidStage {
		return false
	}

	//duplicated
	dup := isDuplicateTrans(stateDB, convert.BytesToUint64(epochIDBuf), convert.BytesToUint64(selfIndexBuf), SlotLeaderStag2)
	if dup {
		return false
	}

	return true
}

func validStg1Service(from common.Address, payload []byte) bool {
	epochIDBuf, selfIndexBuf, err := RlpGetStage1IDFromTx(payload[:])
	if err != nil {
		log.Error("validStg1Service failed")
		return false
	}

	if !InEpochLeadersOrNotByAddress(convert.BytesToUint64(epochIDBuf), convert.BytesToUint64(selfIndexBuf), from) {
		log.SyslogErr(ErrIllegalSender.Error())
		return false
	}

	_, _, _, err = RlpUnpackStage1DataForTx(payload[:])
	if err != nil {
		log.Error(err.Error())
		return false
	}
	return true
}

func validStg2Service(stateDB StateDB, from common.Address, payload []byte) bool {
	epochID, selfIndex, _, alphaPkis, proofs, err := RlpUnpackStage2DataForTx(payload[:])
	if err != nil {
		log.Error("validTxStg2:RlpUnpackStage2DataForTx failed")
		return false
	}

	if !InEpochLeadersOrNotByAddress(epochID, selfIndex, from) {
		log.SyslogErr("validTxStg2:InEpochLeadersOrNotByAddress failed")
		return false
	}

	for _, proof := range proofs {
		if proof.Cmp(bigZero) == 0 {
			log.SyslogErr("validTxStg2ByData", "proofs", ErrInvalidProof.Error())
			return false
		}
	}

	//log.Info("validTxStg2 success")

	mi, err := GetStg1StateDbInfo(stateDB, epochID, selfIndex)
	if err != nil {
		log.Error("validTxStg2", "GetStg1StateDbInfo error", err.Error())
		return false
	}

	//mi
	if len(mi) == 0 || len(alphaPkis) != posconfig.EpochLeaderCount {
		log.SyslogErr("validTxStg2", "len(mi)==0 or len(alphaPkis) not equal", len(alphaPkis))
		return false
	}
	if !util.PkEqual(crypto.ToECDSAPub(mi), alphaPkis[selfIndex]) {
		log.SyslogErr("validTxStg2", "mi is not equal alphaPkis[index]", selfIndex)
		return false
	}
	//Dleq

	ep := util.GetEpocherInst()
	if ep == nil {
		log.Error(ErrEpochID.Error())
		return false
	}
	buff := ep.GetEpochLeaders(epochID)
	if buff == nil || len(buff) == 0 {
		log.SyslogWarning("epoch leader is not ready  at validStg2Service", "epochID", epochID)
		return false
	}
	epochLeaders := make([]*ecdsa.PublicKey, len(buff))
	for i := 0; i < len(buff); i++ {
		epochLeaders[i] = crypto.ToECDSAPub(buff[i])
	}

	if !(uleaderselection.VerifyDleqProof(epochLeaders, alphaPkis, proofs)) {
		log.SyslogErr("validTxStg2", "VerifyDleqProof false self Index", selfIndex)
		return false
	}
	return true
}

func handleStgOne(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	log.Debug("slotLeaderSC handleStgOne is called")

	epochIDBuf, selfIndexBuf, err := RlpGetStage1IDFromTx(in)
	if err != nil {
		return nil, err
	}

	keyHash := GetSlotLeaderStage1KeyHash(epochIDBuf, selfIndexBuf)

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, in)

	err = updateSlotLeaderStageIndex(evm, epochIDBuf, SlotLeaderStag1Indexes, convert.BytesToUint64(selfIndexBuf))

	if err != nil {
		return nil, err
	}

	addSlotScCallTimes(convert.BytesToUint64(epochIDBuf))

	log.Debug(fmt.Sprintf("handleStgOne save data addr:%s, key:%s, data len:%d", slotLeaderPrecompileAddr.Hex(),
		keyHash.Hex(), len(in)))
	log.Debug("handleStgOne save", "epochID", convert.BytesToUint64(epochIDBuf), "selfIndex",
		convert.BytesToUint64(selfIndexBuf))

	return nil, nil
}

func handleStgTwo(in []byte, contract *Contract, evm *EVM) ([]byte, error) {

	epochIDBuf, selfIndexBuf, err := RlpGetStage2IDFromTx(in)
	if err != nil {
		return nil, err
	}

	keyHash := GetSlotLeaderStage2KeyHash(epochIDBuf, selfIndexBuf)

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, in)

	err = updateSlotLeaderStageIndex(evm, epochIDBuf, SlotLeaderStag2Indexes, convert.BytesToUint64(selfIndexBuf))

	if err != nil {
		return nil, err
	}
	addSlotScCallTimes(convert.BytesToUint64(epochIDBuf))

	log.Debug(fmt.Sprintf("handleStgTwo save data addr:%s, key:%s, data len:%d", slotLeaderPrecompileAddr.Hex(),
		keyHash.Hex(), len(in)))
	log.Debug("handleStgTwo save", "epochID", convert.BytesToUint64(epochIDBuf), "selfIndex",
		convert.BytesToUint64(selfIndexBuf))

	functrace.Exit()
	return nil, nil
}

// GetSlotLeaderStage2KeyHash use to get SlotLeader Stage 1 KeyHash by epochid and selfindex
func GetSlotLeaderStage2KeyHash(epochID, selfIndex []byte) common.Hash {
	return getSlotLeaderStageKeyHash(epochID, selfIndex, SlotLeaderStag2)
}

func GetSlotLeaderStage2IndexesKeyHash(epochID []byte) common.Hash {
	return getSlotLeaderStageIndexesKeyHash(epochID, SlotLeaderStag2Indexes)
}

// GetSlotLeaderSCAddress can get the precompile contract address
func GetSlotLeaderSCAddress() common.Address {
	return slotLeaderPrecompileAddr
}

// GetSlotLeaderScAbiString can get the precompile contract Define string
func GetSlotLeaderScAbiString() string {
	return slotLeaderSCDef
}

// GetSlotScCallTimes can get this precompile contract called times
func GetSlotScCallTimes(epochID uint64) uint64 {
	buf, err := posdb.GetDb().Get(epochID, scCallTimes)
	if err != nil {
		return 0
	} else {
		return convert.BytesToUint64(buf)
	}
}

// GetSlotLeaderStage1KeyHash use to get SlotLeader Stage 1 KeyHash by epoch id and self index

func GetSlotLeaderStage1KeyHash(epochID, selfIndex []byte) common.Hash {
	return getSlotLeaderStageKeyHash(epochID, selfIndex, SlotLeaderStag1)
}

func GetStage1FunctionID(abiString string) ([4]byte, error) {
	var slotStage1ID [4]byte

	abi, err := util.GetAbi(abiString)
	if err != nil {
		log.SyslogErr("slotLeaderSC", "GetStage1FunctionID:GetAbi", err.Error())
		return slotStage1ID, err
	}

	copy(slotStage1ID[:], abi.Methods["slotLeaderStage1MiSave"].Id())

	return slotStage1ID, nil
}

func GetStage2FunctionID(abiString string) ([4]byte, error) {
	var slotStage2ID [4]byte

	abi, err := util.GetAbi(abiString)
	if err != nil {
		log.SyslogErr("slotLeaderSC", "GetStage2FunctionID:GetAbi", err.Error())
		return slotStage2ID, err
	}

	copy(slotStage2ID[:], abi.Methods["slotLeaderStage2InfoSave"].Id())

	return slotStage2ID, nil
}

// PackStage1Data can pack stage1 data into abi []byte for tx payload
func PackStage1Data(input []byte, abiString string) ([]byte, error) {
	id, err := GetStage1FunctionID(abiString)
	outBuf := make([]byte, len(id)+len(input))
	copy(outBuf[:4], id[:])
	copy(outBuf[4:], input[:])
	return outBuf, err
}

func InEpochLeadersOrNotByAddress(epochID uint64, selfIndex uint64, senderAddress common.Address) bool {
	ep := util.GetEpocherInst()
	if ep == nil {
		return false
	}
	epochLeaders := ep.GetEpochLeaders(epochID)
	if len(epochLeaders) != posconfig.EpochLeaderCount {
		log.SyslogWarning("epoch leader is not ready  at InEpochLeadersOrNotByAddress", "epochID", epochID)
		return false
	}

	if int64(selfIndex) < 0 || int64(selfIndex) >= posconfig.EpochLeaderCount {
		log.SyslogErr("InEpochLeadersOrNotByAddress", "selfIndex out of range", int64(selfIndex))
		return false
	}

	if crypto.PubkeyToAddress(*crypto.ToECDSAPub(epochLeaders[selfIndex])).Hex() == senderAddress.Hex() {
		return true
	}

	addr1 := crypto.PubkeyToAddress(*crypto.ToECDSAPub(epochLeaders[selfIndex])).Hex()
	addr2 := senderAddress.Hex()

	log.Info("epochleader not match", "epochleader array address", addr1, "sender", addr2)
	return false
}

type stage1Data struct {
	EpochID    uint64
	SelfIndex  uint64
	MiCompress []byte
}

// RlpPackStage1DataForTx
func RlpPackStage1DataForTx(epochID uint64, selfIndex uint64, mi *ecdsa.PublicKey, abiString string) ([]byte, error) {
	pkBuf, err := util.CompressPk(mi)
	if err != nil {
		return nil, err
	}
	data := &stage1Data{
		EpochID:    epochID,
		SelfIndex:  selfIndex,
		MiCompress: pkBuf,
	}

	buf, err := rlp.EncodeToBytes(data)
	if err != nil {
		log.SyslogErr("RlpPackStage1DataForTx", "rlp.EncodeToBytes", err.Error())
		return nil, err
	}

	return PackStage1Data(buf, abiString)
}

// RlpUnpackStage1DataForTx
func RlpUnpackStage1DataForTx(input []byte) (epochID uint64, selfIndex uint64, mi *ecdsa.PublicKey, err error) {
	var data *stage1Data

	buf := input[4:]

	err = rlp.DecodeBytes(buf, &data)
	if err != nil {
		log.SyslogErr("RlpUnpackStage1DataForTx", "rlp.DecodeBytes", err.Error())
		return
	}

	epochID = data.EpochID
	selfIndex = data.SelfIndex
	// UncompressPK has verified the point on curve or not.
	mi, err = util.UncompressPk(data.MiCompress)
	if err != nil {
		log.SyslogErr("RlpUnpackStage1DataForTx", "util.UncompressPk", err.Error())
	}
	return
}

// RlpGetStage1IDFromTx
func RlpGetStage1IDFromTx(input []byte) (epochIDBuf []byte, selfIndexBuf []byte, err error) {
	var data *stage1Data

	buf := input[4:]

	err = rlp.DecodeBytes(buf, &data)
	if err != nil {
		log.SyslogErr("RlpGetStage1IDFromTx", "rlp.DecodeBytes", err.Error())
		return
	}
	epochIDBuf = convert.Uint64ToBytes(data.EpochID)
	selfIndexBuf = convert.Uint64ToBytes(data.SelfIndex)
	return
}

type stage2Data struct {
	EpochID   uint64
	SelfIndex uint64
	SelfPk    []byte
	AlphaPki  [][]byte
	Proof     []*big.Int
}

func RlpPackStage2DataForTx(epochID uint64, selfIndex uint64, selfPK *ecdsa.PublicKey, alphaPki []*ecdsa.PublicKey,
	proof []*big.Int, abiString string) ([]byte, error) {
	pk, err := util.CompressPk(selfPK)
	if err != nil {
		log.SyslogErr("RlpPackStage2DataForTx", "util.CompressPk", err.Error())
		return nil, err
	}

	pks := make([][]byte, len(alphaPki))
	for i := 0; i < len(alphaPki); i++ {
		pks[i], err = util.CompressPk(alphaPki[i])
		if err != nil {
			log.SyslogErr("RlpPackStage2DataForTx", "util.CompressPk", err.Error())
			return nil, err
		}
	}

	data := &stage2Data{
		EpochID:   epochID,
		SelfIndex: selfIndex,
		SelfPk:    pk,
		AlphaPki:  pks,
		Proof:     proof,
	}

	buf, err := rlp.EncodeToBytes(data)
	if err != nil {
		log.SyslogErr("RlpPackStage2DataForTx", "rlp.EncodeToBytes", err.Error())
		return nil, err
	}

	id, err := GetStage2FunctionID(abiString)
	if err != nil {
		return nil, err
	}

	outBuf := make([]byte, len(id)+len(buf))
	copy(outBuf[:4], id[:])
	copy(outBuf[4:], buf[:])

	return outBuf, nil
}

func RlpUnpackStage2DataForTx(input []byte) (epochID uint64, selfIndex uint64, selfPK *ecdsa.PublicKey,
	alphaPki []*ecdsa.PublicKey, proof []*big.Int, err error) {
	inputBuf := input[4:]

	var data stage2Data
	err = rlp.DecodeBytes(inputBuf, &data)
	if err != nil {
		log.SyslogErr("RlpUnpackStage2DataForTx", "rlp.DecodeBytes", err.Error())
		return
	}

	epochID = data.EpochID
	selfIndex = data.SelfIndex
	selfPK, err = util.UncompressPk(data.SelfPk)
	if err != nil {
		log.SyslogErr("RlpUnpackStage2DataForTx", "util.UncompressPk", err.Error())
		return
	}

	alphaPki = make([]*ecdsa.PublicKey, len(data.AlphaPki))
	for i := 0; i < len(data.AlphaPki); i++ {
		alphaPki[i], err = util.UncompressPk(data.AlphaPki[i])
		if err != nil {
			log.SyslogErr("RlpUnpackStage2DataForTx", "util.UncompressPk", err.Error())
			return
		}
	}

	proof = data.Proof
	return
}

func RlpGetStage2IDFromTx(input []byte) (epochIDBuf []byte, selfIndexBuf []byte, err error) {
	inputBuf := input[4:]

	var data stage2Data
	err = rlp.DecodeBytes(inputBuf, &data)
	if err != nil {
		log.SyslogErr("RlpGetStage2IDFromTx", "rlp.DecodeBytes", err.Error())
		return
	}

	epochIDBuf = convert.Uint64ToBytes(data.EpochID)
	selfIndexBuf = convert.Uint64ToBytes(data.SelfIndex)
	return
}

func GetStage2TxAlphaPki(stateDb StateDB, epochID uint64, selfIndex uint64) (alphaPkis []*ecdsa.PublicKey,
	proofs []*big.Int, err error) {

	slotLeaderPrecompileAddr := GetSlotLeaderSCAddress()

	keyHash := GetSlotLeaderStage2KeyHash(convert.Uint64ToBytes(epochID), convert.Uint64ToBytes(selfIndex))

	data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if data == nil {
		log.Debug(fmt.Sprintf("try to get stateDB addr:%s, key:%s", slotLeaderPrecompileAddr.Hex(), keyHash.Hex()))
		log.SyslogErr(fmt.Sprintf("try to get stateDB addr:%s, key:%s", slotLeaderPrecompileAddr.Hex(), keyHash.Hex()))
		return nil, nil, ErrNoTx2TransInDB
	}

	epID, slfIndex, _, alphaPki, proof, err := RlpUnpackStage2DataForTx(data)
	if err != nil {
		return nil, nil, err
	}

	if epID != epochID || slfIndex != selfIndex {
		log.SyslogErr("GetStage2TxAlphaPki", "error", ErrRlpUnpackErr.Error())
		return nil, nil, ErrRlpUnpackErr
	}

	return alphaPki, proof, nil
}

func GetStg1StateDbInfo(stateDb StateDB, epochID uint64, index uint64) (mi []byte, err error) {
	slotLeaderPrecompileAddr := GetSlotLeaderSCAddress()
	keyHash := GetSlotLeaderStage1KeyHash(convert.Uint64ToBytes(epochID), convert.Uint64ToBytes(index))

	// Read and Verify
	readBuf := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
	if readBuf == nil {
		log.SyslogErr("GetStg1StateDbInfo", "error", ErrNoTx1TransInDB.Error())
		return nil, ErrNoTx1TransInDB
	}

	epID, idxID, miPoint, err := RlpUnpackStage1DataForTx(readBuf)
	if err != nil {
		return nil, ErrRlpUnpackErr
	}
	mi = crypto.FromECDSAPub(miPoint)
	//pk and mi is 65 bytes length

	if epID == epochID &&
		idxID == index &&
		err == nil {
		return
	}

	return nil, ErrVerifyStg1Data
}

func getSlotLeaderStageIndexesKeyHash(epochID []byte, slotLeaderStageIndexes string) common.Hash {
	var keyBuf bytes.Buffer
	keyBuf.Write(epochID)
	keyBuf.Write([]byte(slotLeaderStageIndexes))
	return crypto.Keccak256Hash(keyBuf.Bytes())
}

func getSlotLeaderStageKeyHash(epochID, selfIndex []byte, slotLeaderStage string) common.Hash {
	var keyBuf bytes.Buffer
	keyBuf.Write(epochID)
	keyBuf.Write(selfIndex)
	keyBuf.Write([]byte(slotLeaderStage))
	return crypto.Keccak256Hash(keyBuf.Bytes())
}

func addSlotScCallTimes(epochID uint64) error {
	buf, err := posdb.GetDb().Get(epochID, scCallTimes)
	times := uint64(0)
	if err != nil {
		if err.Error() != "leveldb: not found" {
			log.SyslogErr("addSlotScCallTimes", "error", err.Error())
			return err
		}
	} else {
		times = convert.BytesToUint64(buf)
	}

	times++

	posdb.GetDb().Put(epochID, scCallTimes, convert.Uint64ToBytes(times))
	return nil
}

func isInValidStage(epochID uint64, time uint64, kStart uint64, kEnd uint64) bool {
	//eid, sid := util.CalEpochSlotID(evm.Time.Uint64())
	eid, sid := util.CalEpochSlotID(time)
	if epochID != eid {
		log.SyslogWarning("Tx epochID is not current epoch", "epochID", eid, "slotID", sid, "currentEpochID", epochID)

		return false
	}

	if sid > kEnd || sid < kStart {
		log.SyslogWarning("Tx is out of valid stage range", "epochID", eid, "slotID", sid, "rangeStart", kStart,
			"rangeEnd", kEnd)

		return false
	}

	return true
}

func isDuplicateTrans(stateDb StateDB, epochID uint64, index uint64, stageName string) bool {
	slotLeaderPrecompileAddr := GetSlotLeaderSCAddress()
	var keyHash common.Hash
	if stageName == SlotLeaderStag1 {
		keyHash = GetSlotLeaderStage1KeyHash(convert.Uint64ToBytes(epochID), convert.Uint64ToBytes(index))
		data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
		if data == nil {
			return false
		}
		log.SyslogWarning("isDuplicateTrans", "epochID", epochID, "index", index, "stageName", stageName)
		return true
	}

	if stageName == SlotLeaderStag2 {
		keyHash = GetSlotLeaderStage2KeyHash(convert.Uint64ToBytes(epochID), convert.Uint64ToBytes(index))
		data := stateDb.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)
		if data == nil {
			return false
		}
		log.SyslogWarning("isDuplicateTrans", "epochID", epochID, "index", index, "stageName", stageName)
		return true
	}
	return false
}

func updateSlotLeaderStageIndex(evm *EVM, epochID []byte, slotLeaderStageIndexes string, index uint64) error {
	var sendtrans [posconfig.EpochLeaderCount]bool
	var sendtransGet [posconfig.EpochLeaderCount]bool

	key := getSlotLeaderStageIndexesKeyHash(epochID, slotLeaderStageIndexes)
	bytes := evm.StateDB.GetStateByteArray(slotLeaderPrecompileAddr, key)

	if len(bytes) == 0 {
		sendtrans[index] = true
		value, err := rlp.EncodeToBytes(sendtrans)
		if err != nil {
			log.SyslogErr("updateSlotLeaderStageIndex", "rlp.EncodeToBytes", err.Error())
			return err
		}
		evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, key, value)

		log.Debug("updateSlotLeaderStageIndex", "key", key, "value", sendtrans)
	} else {
		err := rlp.DecodeBytes(bytes, &sendtransGet)
		if err != nil {
			log.SyslogErr("updateSlotLeaderStageIndex", "rlp.DecodeBytes", err.Error())
			return err
		}

		sendtransGet[index] = true
		value, err := rlp.EncodeToBytes(sendtransGet)
		if err != nil {
			log.SyslogErr("updateSlotLeaderStageIndex", "rlp.EncodeToBytes", err.Error())
			return err
		}
		evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, key, value)
		log.Debug("updateSlotLeaderStageIndex", "key", key, "value", sendtransGet)
	}
	return nil
}

func GetValidSMA1Cnt(db StateDB, epochId uint64) uint64 {
	return getValidIndexCnt(db, epochId, SlotLeaderStag1Indexes)
}

func GetValidSMA2Cnt(db StateDB, epochId uint64) uint64 {
	return getValidIndexCnt(db, epochId, SlotLeaderStag2Indexes)
}

func getValidIndexCnt(db StateDB, epochId uint64, indexKey string) uint64 {
	var sendtransGet [posconfig.EpochLeaderCount]bool
	epochIDBuf := convert.Uint64ToBytes(epochId)

	key := getSlotLeaderStageIndexesKeyHash(epochIDBuf, indexKey)
	bytes := db.GetStateByteArray(slotLeaderPrecompileAddr, key)
	if len(bytes) == 0 {
		return 0
	}

	err := rlp.DecodeBytes(bytes, &sendtransGet)
	if err != nil {
		log.SyslogErr("GetValidSMA1Cnt, rlp decode fail", "err", err.Error())
		return 0
	}

	cnt := uint64(0)
	for i := range sendtransGet {
		if sendtransGet[i] {
			cnt++
		}
	}

	return cnt
}

func GetSlStage(slotId uint64) uint64 {
	if slotId <= posconfig.Sma1End {
		return 1
	} else if slotId < posconfig.Sma2Start {
		return 2
	} else if slotId <= posconfig.Sma2End {
		return 3
	} else if slotId < posconfig.Sma3Start {
		return 4
	} else if slotId <= posconfig.Sma3End {
		return 5
	} else {
		return 6
	}
}
