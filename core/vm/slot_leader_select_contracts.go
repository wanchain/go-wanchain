package vm

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/pos/postools"
	"github.com/wanchain/go-wanchain/pos/postools/slottools"

	"github.com/wanchain/go-wanchain/functrace"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/core/types"
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
	errIllegalSender                   = errors.New("sender is not in epoch leaders ")
	scCallTimes                        = "SLOT_LEADER_SC_CALL_TIMES"
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

	if methodId == stgOneIdArr {
		return c.handleStgOne(in[:], contract, evm) //Do not use [4:] because it has do it in function
	} else if methodId == stgTwoIdArr {
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

	if !isInValidStage(posdb.BytesToUint64(epochIDBuf), evm, 0, pos.Stage1K) {
		log.Warn("Not in range handleStgOne", "hash", crypto.Keccak256Hash(in).Hex())
		return nil, errors.New("Not in range handleStgOne hash:" + crypto.Keccak256Hash(in).Hex())
	}

	keyHash := GetSlotLeaderStage1KeyHash(epochIDBuf, selfIndexBuf)

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, in)

	addSlotScCallTimes(posdb.BytesToUint64(epochIDBuf))

	log.Debug(fmt.Sprintf("-----------------------------------------handleStgOne save data addr:%s, key:%s, data len:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), len(in)))
	log.Debug("handleStgTwo save", "epochID", postools.BytesToUint64(epochIDBuf), "selfIndex", postools.BytesToUint64(selfIndexBuf))

	return nil, nil
}

func (c *slotLeaderSC) handleStgTwo(in []byte, contract *Contract, evm *EVM) ([]byte, error) {

	epochIDBuf, selfIndexBuf, err := slottools.RlpGetStage2IDFromTx(in, slotLeaderSCDef)
	if err != nil {
		return nil, err
	}

	if !isInValidStage(posdb.BytesToUint64(epochIDBuf), evm, pos.Stage2K, pos.Stage4K) {
		log.Warn("Not in range handleStgTwo", "hash", crypto.Keccak256Hash(in).Hex())
		return nil, errors.New("Not in range handleStgTwo hash:" + crypto.Keccak256Hash(in).Hex())
	}

	keyHash := GetSlotLeaderStage2KeyHash(epochIDBuf, selfIndexBuf)

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, in)

	addSlotScCallTimes(posdb.BytesToUint64(epochIDBuf))

	log.Debug(fmt.Sprintf("-----------------------------------------handleStgTwo save data addr:%s, key:%s, data len:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), len(in)))
	log.Debug("handleStgTwo save", "epochID", postools.BytesToUint64(epochIDBuf), "selfIndex", postools.BytesToUint64(selfIndexBuf))

	functrace.Exit()
	return nil, nil
}

func (c *slotLeaderSC) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	//TODO
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

	epochIDBuf, _, err := slottools.RlpGetStage1IDFromTx(tx.Data(), slotLeaderSCDef)
	if err != nil {
		log.Error("ValidTxStg1 failed")
		return err
	}

	if !slottools.InEpochLeadersOrNotByAddress(postools.BytesToUint64(epochIDBuf), sender) {
		log.Error("ValidTxStg1 failed")
		return errIllegalSender
	}

	log.Info("ValidTxStg1 success")
	return nil
}

func (c *slotLeaderSC) ValidTxStg2(signer types.Signer, tx *types.Transaction) error {
	sender, err := signer.Sender(tx)
	if err != nil {
		return err
	}

	epochIDBuf, _, err := slottools.RlpGetStage2IDFromTx(tx.Data(), slotLeaderSCDef)
	if err != nil {
		log.Error("ValidTxStg2 failed")
		return err
	}

	if !slottools.InEpochLeadersOrNotByAddress(postools.BytesToUint64(epochIDBuf), sender) {
		log.Error("ValidTxStg2 failed")
		return errIllegalSender
	}

	//log.Info("ValidTxStg2 success")
	return nil
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
	var keyBuf bytes.Buffer
	keyBuf.Write(epochID)
	keyBuf.Write(selfIndex)
	keyBuf.Write([]byte("slotLeaderStag1"))
	return crypto.Keccak256Hash(keyBuf.Bytes())
}

// GetSlotLeaderStage2KeyHash use to get SlotLeader Stage 1 KeyHash by epochid and selfindex
func GetSlotLeaderStage2KeyHash(epochID, selfIndex []byte) common.Hash {
	var keyBuf bytes.Buffer
	keyBuf.Write(epochID)
	keyBuf.Write(selfIndex)
	keyBuf.Write([]byte("slotLeaderStag2"))
	return crypto.Keccak256Hash(keyBuf.Bytes())
}
