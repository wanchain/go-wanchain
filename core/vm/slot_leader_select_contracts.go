package vm

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/uleaderselection"

	pos "github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/postools"
	"github.com/wanchain/go-wanchain/pos/slotleader/slottools"

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

func init() {
	if errSlotLeaderSCInit != nil {
		panic("err in slot leader sc initialize :" + errSlotLeaderSCInit.Error())
	}

	stgOneIdArr, _ = slottools.GetStage1FunctionID(slotLeaderSCDef)
	stgTwoIdArr, _ = slottools.GetStage2FunctionID(slotLeaderSCDef)
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
		err := c.ValidTxStg1ByData(from, in[:])
		if err != nil {
			log.Error("slotLeaderSC:Run:ValidTxStg1ByData", "from", from)
			return nil, err
		}
		return c.handleStgOne(in[:], contract, evm) //Do not use [4:] because it has do it in function
	} else if methodId == stgTwoIdArr {
		err := c.ValidTxStg2ByData(from, in[:])
		if err != nil {
			log.Error("slotLeaderSC:Run:ValidTxStg2ByData", "from", from)
			return nil, err
		}
		return c.handleStgTwo(in[:], contract, evm)
	}

	functrace.Exit()
	return nil, errMethodId
}

func (c *slotLeaderSC) handleStgOne(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	log.Debug("slotLeaderSC handleStgOne is called")

	epochIDBuf, selfIndexBuf, err := slottools.RlpGetStage1IDFromTx(in, slotLeaderSCDef)
	if err != nil {
		return nil, err
	}

	if !isInValidStage(posdb.BytesToUint64(epochIDBuf), evm, pos.Sma1Start, pos.Sma1End) {
		log.Warn("Not in range handleStgOne", "hash", crypto.Keccak256Hash(in).Hex())
		return nil, slottools.ErrInvalidTx1Range
	}

	keyHash := GetSlotLeaderStage1KeyHash(epochIDBuf, selfIndexBuf)

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, in)

	err = updateSlotLeaderStageIndex(evm, epochIDBuf, SlotLeaderStag1Indexes, posdb.BytesToUint64(selfIndexBuf))

	if err != nil {
		return nil, err
	}

	addSlotScCallTimes(posdb.BytesToUint64(epochIDBuf))

	log.Debug(fmt.Sprintf("-----------------------------------------handleStgOne save data addr:%s, key:%s, data len:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), len(in)))
	log.Debug("handleStgOne save", "epochID", postools.BytesToUint64(epochIDBuf), "selfIndex", postools.BytesToUint64(selfIndexBuf))

	return nil, nil
}

func (c *slotLeaderSC) handleStgTwo(in []byte, contract *Contract, evm *EVM) ([]byte, error) {

	epochIDBuf, selfIndexBuf, err := slottools.RlpGetStage2IDFromTx(in, slotLeaderSCDef)
	if err != nil {
		return nil, err
	}

	if !isInValidStage(posdb.BytesToUint64(epochIDBuf), evm, pos.Sma2Start, pos.Sma2End) {
		log.Warn("Not in range handleStgTwo", "hash", crypto.Keccak256Hash(in).Hex())
		return nil, slottools.ErrInvalidTx2Range
	}

	keyHash := GetSlotLeaderStage2KeyHash(epochIDBuf, selfIndexBuf)

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, in)

	err = updateSlotLeaderStageIndex(evm, epochIDBuf, SlotLeaderStag2Indexes, posdb.BytesToUint64(selfIndexBuf))

	if err != nil {
		return nil, err
	}
	addSlotScCallTimes(posdb.BytesToUint64(epochIDBuf))

	log.Debug(fmt.Sprintf("-----------------------------------------handleStgTwo save data addr:%s, key:%s, data len:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), len(in)))
	log.Debug("handleStgTwo save", "epochID", postools.BytesToUint64(epochIDBuf), "selfIndex", postools.BytesToUint64(selfIndexBuf))

	functrace.Exit()
	return nil, nil
}

func (c *slotLeaderSC) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	// 0. verify pk and whether in Epoch group list.
	// 1. get transaction data
	// 2. parse data to get the Pie[i] and A[i]
	// 3. verify A[i]
	// 4. verify Pie[i]
	// 5. epochID verify
	var methodId [4]byte
	copy(methodId[:], tx.Data()[:4])

	if methodId == stgOneIdArr {
		return c.ValidTxStg1(signer, tx)
	} else if methodId == stgTwoIdArr {
		return c.ValidTxStg2(signer, tx)
	}
	return nil
}

func (c *slotLeaderSC) ValidTxStg1(signer types.Signer, tx *types.Transaction) error {
	sender, err := signer.Sender(tx)
	if err != nil {
		return err
	}

	return c.ValidTxStg1ByData(sender, tx.Data())
}

func (c *slotLeaderSC) ValidTxStg1ByData(from common.Address, payload []byte) error {

	epochIDBuf, _, err := slottools.RlpGetStage1IDFromTx(payload[:], slotLeaderSCDef)
	if err != nil {
		log.Error("ValidTxStg1 failed")
		return err
	}

	if !slottools.InEpochLeadersOrNotByAddress(postools.BytesToUint64(epochIDBuf), from) {
		log.Error("ValidTxStg1 failed")
		return slottools.ErrIllegalSender
	}

	//log.Info("ValidTxStg1 success")
	return nil
}

func (c *slotLeaderSC) ValidTxStg2ByData(from common.Address, payload []byte) error {
	epochID, selfIndex, _, alphaPkis, proofs, err := slottools.RlpUnpackStage2DataForTx(payload[:], slotLeaderSCDef)
	if err != nil {
		log.Error("ValidTxStg2:RlpUnpackStage2DataForTx failed")
		return err
	}

	if !slottools.InEpochLeadersOrNotByAddress(epochID, from) {
		log.Error("ValidTxStg2:InEpochLeadersOrNotByAddress failed")
		return slottools.ErrIllegalSender
	}

	//log.Info("ValidTxStg2 success")
	type slot interface {
		GetStg1StateDbInfo(epochID uint64, index uint64) (mi []byte, err error)
		GetStage2TxAlphaPki(epochID uint64, selfIndex uint64) (alphaPkis []*ecdsa.PublicKey, proofs []*big.Int, err error)
		GetEpochLeadersPK(epochID uint64) []*ecdsa.PublicKey
	}

	selector := slottools.GetSlotLeaderInst()

	if selector == nil {
		return nil
	}
	mi, err := selector.(slot).GetStg1StateDbInfo(epochID, selfIndex)
	if err != nil {
		log.Error("ValidTxStg2", "GetStg1StateDbInfo error", err.Error())
		return err
	}

	//mi
	if len(mi) == 0 || len(alphaPkis) != pos.EpochLeaderCount {
		log.Error("ValidTxStg2", "len(mi)==0 or len(alphaPkis) not equal", len(alphaPkis))
		return slottools.ErrInvalidTxLen
	}
	if !postools.PkEqual(crypto.ToECDSAPub(mi), alphaPkis[selfIndex]) {
		log.Error("ValidTxStg2", "mi is not equal alphaPkis[index]", selfIndex)
		return slottools.ErrTx1AndTx2NotConsistent
	}
	//Dleq
	epochLeaders := selector.(slot).GetEpochLeadersPK(epochID)
	if !(uleaderselection.VerifyDleqProof(epochLeaders, alphaPkis, proofs)) {
		log.Error("ValidTxStg2", "VerifyDleqProof false self Index", selfIndex)
		return slottools.ErrDleqProof
	}
	return nil
}

func (c *slotLeaderSC) ValidTxStg2(signer types.Signer, tx *types.Transaction) error {
	sender, err := signer.Sender(tx)
	if err != nil {
		return err
	}
	return c.ValidTxStg2ByData(sender, tx.Data())
}

// GetSlotLeaderSCAddress can get the precompile contract address
func GetSlotLeaderSCAddress() common.Address {
	return slotLeaderPrecompileAddr
}

// GetSlotLeaderScAbiString can get the precompile contract Define string
func GetSlotLeaderScAbiString() string {
	return slotLeaderSCDef
}

func addSlotScCallTimes(epochID uint64) error {
	buf, err := posdb.GetDb().Get(epochID, scCallTimes)
	times := uint64(0)
	if err != nil {
		if err.Error() != "leveldb: not found" {
			return err
		}
	} else {
		times = posdb.BytesToUint64(buf)
	}

	times++

	posdb.GetDb().Put(epochID, scCallTimes, posdb.Uint64ToBytes(times))
	return nil
}

// GetSlotScCallTimes can get this precompile contract called times
func GetSlotScCallTimes(epochID uint64) uint64 {
	buf, err := posdb.GetDb().Get(epochID, scCallTimes)
	if err != nil {
		return 0
	} else {
		return posdb.BytesToUint64(buf)
	}
}

func isInValidStage(epochID uint64, evm *EVM, kStart uint64, kEnd uint64) bool {
	eid, sid := postools.CalEpochSlotID(evm.Time.Uint64())
	if epochID != eid {
		log.Warn("Tx epochID is not current epoch", "epochID", eid, "slotID", sid, "currentEpochID", epochID)

		return false
	}

	if sid > kEnd || sid < kStart {
		log.Warn("Tx is out of valid stage range", "epochID", eid, "slotID", sid, "rangeStart", kStart, "rangeEnd", kEnd)

		return false
	}

	return true
}

// GetSlotLeaderStage1KeyHash use to get SlotLeader Stage 1 KeyHash by epochid and selfindex
func GetSlotLeaderStage1KeyHash(epochID, selfIndex []byte) common.Hash {
	return GetSlotLeaderStageKeyHash(epochID, selfIndex, SlotLeaderStag1)
}

// GetSlotLeaderStage2KeyHash use to get SlotLeader Stage 1 KeyHash by epochid and selfindex
func GetSlotLeaderStage2KeyHash(epochID, selfIndex []byte) common.Hash {
	return GetSlotLeaderStageKeyHash(epochID, selfIndex, SlotLeaderStag2)
}

func GetSlotLeaderStageKeyHash(epochID, selfIndex []byte, slotLeaderStage string) common.Hash {
	var keyBuf bytes.Buffer
	keyBuf.Write(epochID)
	keyBuf.Write(selfIndex)
	keyBuf.Write([]byte(slotLeaderStage))
	return crypto.Keccak256Hash(keyBuf.Bytes())
}

func GetSlotLeaderStage2IndexesKeyHash(epochID []byte) common.Hash {
	return GetSlotLeaderStageIndexesKeyHash(epochID, SlotLeaderStag2Indexes)
}

func GetSlotLeaderStageIndexesKeyHash(epochID []byte, slotLeaderStageIndexes string) common.Hash {
	var keyBuf bytes.Buffer
	keyBuf.Write(epochID)
	keyBuf.Write([]byte(slotLeaderStageIndexes))
	return crypto.Keccak256Hash(keyBuf.Bytes())
}

func updateSlotLeaderStageIndex(evm *EVM, epochID []byte, slotLeaderStageIndexes string, index uint64) error {
	var sendtrans [pos.EpochLeaderCount]bool
	var sendtransGet [pos.EpochLeaderCount]bool

	key := GetSlotLeaderStageIndexesKeyHash(epochID, slotLeaderStageIndexes)
	bytes := evm.StateDB.GetStateByteArray(slotLeaderPrecompileAddr, key)

	if len(bytes) == 0 {
		sendtrans[index] = true
		value, err := rlp.EncodeToBytes(sendtrans)
		if err != nil {
			return err
		}
		evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, key, value)

		log.Debug("updateSlotLeaderStageIndex", "key", key, "value", sendtrans)
	} else {
		err := rlp.DecodeBytes(bytes, &sendtransGet)
		if err != nil {
			return err
		}

		sendtransGet[index] = true
		value, err := rlp.EncodeToBytes(sendtransGet)
		if err != nil {
			return err
		}
		evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, key, value)
		log.Debug("updateSlotLeaderStageIndex", "key", key, "value", sendtransGet)
	}
	return nil
}
