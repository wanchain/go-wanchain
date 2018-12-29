package vm

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/wanchain/go-wanchain/pos/posdb"

	"github.com/wanchain/go-wanchain/functrace"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/slotleader"

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
)

func init() {
	if errSlotLeaderSCInit != nil {
		panic("err in slot leader sc initialize :" + errSlotLeaderSCInit.Error())
	}

	s := slotleader.GetSlotLeaderSelection()
	stgOneIdArr, _ = s.GetStage1FunctionID()
	copy(stgTwoIdArr[:], slotLeaderAbi.Methods["slotLeaderStage2InfoSave"].Id())
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
		return c.handleStgTwo(in[4:], contract, evm)
	}

	functrace.Exit()
	return nil, errMethodId
}

func (c *slotLeaderSC) handleStgOne(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	functrace.Enter()
	log.Debug("slotLeaderSC handleStgOne is called")
	if evm == nil {
		return nil, errors.New("state db is not ready")
	}
	s := slotleader.GetSlotLeaderSelection()
	data, err := s.UnpackStage1Data(in)
	if err != nil {
		return nil, err
	}

	epochID, selfIndex, pkSelf, miGen, err := s.RlpUnpackWithCompressedPK(data) // use this function to unpack rlp []byte
	if err != nil {
		return nil, err
	}

	// address : sc slotLeaderPrecompileAddr
	// key:      hash(epochID,selfIndex,"slotLeaderStag2")
	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())

	var keyBuf bytes.Buffer
	keyBuf.Write(epochID)
	keyBuf.Write(selfIndex)
	keyBuf.Write([]byte("slotLeaderStag1"))
	keyHash := crypto.Keccak256Hash(keyBuf.Bytes())

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, data)

	// Read and Verify
	readBuf := evm.StateDB.GetStateByteArray(slotLeaderPrecompileAddr, keyHash)

	epID, index, pk, pkMi, err := s.RlpUnpackWithCompressedPK(readBuf)

	if hex.EncodeToString(epID) == hex.EncodeToString(epochID) &&
		hex.EncodeToString(index) == hex.EncodeToString(selfIndex) &&
		hex.EncodeToString(pk) == hex.EncodeToString(pkSelf) &&
		hex.EncodeToString(pkMi) == hex.EncodeToString(miGen) &&
		err == nil {
		log.Debug("--------------------------------------------------handleStgOne Data save to StateDb and verified success")
		log.Debug("epID:" + hex.EncodeToString(epID))
		log.Debug("index:" + hex.EncodeToString(index))
		log.Debug("pk:" + hex.EncodeToString(pk))
		log.Debug("pkMi:" + hex.EncodeToString(pkMi))

	} else {
		log.Debug("Data save to StateDb and verified failed")
		return nil, errors.New("Data save to StateDb and verified failed")
	}

	functrace.Exit()
	return nil, nil
}

func (c *slotLeaderSC) handleStgTwo(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	functrace.Enter()
	log.Debug("slotLeaderSC handleStgTwo is called")
	if evm == nil {
		return nil, errors.New("state db is not ready")
	}

	s := slotleader.GetSlotLeaderSelection()
	data, err := s.UnpackStage2Data(in)
	if err != nil {
		return nil, err
	}
	//epochIDBuf,selfIndexBuf,_,alphaPki,proof,err := s.RlpUnpackStage2Data(data)
	epochIDBuf, selfIndexBuf, _, _, _, err := s.RlpUnpackStage2Data(data)
	if err != nil {
		return nil, err
	}
	// address : sc slotLeaderPrecompileAddr
	// key:      hash(epochID,selfIndex,"slotLeaderStag2")
	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())

	var keyBuf bytes.Buffer
	//keyBuf.Write([]byte(epochIDBuf))
	//keyBuf.Write([]byte(selfIndexBuf))

	// epochIDBufDec, err := hex.DecodeString(epochIDBuf)
	// if err != nil {
	// 	return nil, err
	// }

	epochIDBufDec := posdb.Uint64StringToByte(epochIDBuf)

	keyBuf.Write(epochIDBufDec)

	// selfIndexBufDec, err := hex.DecodeString(selfIndexBuf)
	// if err != nil {
	// 	return nil, err
	// }
	selfIndexBufDec := posdb.Uint64StringToByte(selfIndexBuf)

	keyBuf.Write(selfIndexBufDec)

	keyBuf.Write([]byte("slotLeaderStag2"))
	keyHash := crypto.Keccak256Hash(keyBuf.Bytes())

	evm.StateDB.SetStateByteArray(slotLeaderPrecompileAddr, keyHash, data)
	log.Debug(fmt.Sprintf("-----------------------------------------handleStgTwo save data addr:%s, key:%s, data len:%d", slotLeaderPrecompileAddr.Hex(), keyHash.Hex(), len(data)))
	log.Debug("handleStgTwo save", "epochID", epochIDBuf, "selfIndex", selfIndexBuf)

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
		return c.ValidTxStg1(&stateDB, signer, tx)
	} else if methodId == stgTwoIdArr {
		return c.ValidTxStg2(&stateDB, signer, tx)
	}
	return nil
}

func (c *slotLeaderSC) ValidTxStg1(stateDB *StateDB, signer types.Signer, tx *types.Transaction) error {
	s := slotleader.GetSlotLeaderSelection()
	data, err := s.UnpackStage1Data(tx.Data())
	if err != nil {
		return err
	}
	epochIDBuf, _, pkSelf, _, err := s.RlpUnpackAndWithUncompressPK(data) // use this function to unpack rlp []byte
	if err != nil {
		return err
	}
	if !s.InEpochLeadersOrNotByPk(posdb.BytesToUint64(epochIDBuf), pkSelf) {
		return errIllegalSender
	}
	return nil
}

func (c *slotLeaderSC) ValidTxStg2(stateDB *StateDB, signer types.Signer, tx *types.Transaction) error {
	s := slotleader.GetSlotLeaderSelection()
	data, err := s.UnpackStage2Data(tx.Data()[4:])
	if err != nil {
		return err
	}
	epochIDString, _, pk, _, _, err := s.RlpUnpackStage2Data(data)
	if err != nil {
		return err
	}

	pkiDec, err := hex.DecodeString(pk)
	if err != nil {
		return err
	}
	if !s.InEpochLeadersOrNotByPk(posdb.StringToUint64(epochIDString), pkiDec) {
		return errIllegalSender
	}
	return nil
}
