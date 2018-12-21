package vm

import (
	"encoding/hex"
	"errors"
	"strings"

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
		}
	]`
	slotLeaderAbi, errSlotLeaderSCInit = abi.JSON(strings.NewReader(slotLeaderSCDef))
	stgOneIdArr, stgTwoIdArr           [4]byte

	//StampValueSet   = make(map[string]string, 5)
	//WanCoinValueSet = make(map[string]string, 10)
	errIllegalSender = errors.New("sender is not in epoch leaders ")
)

func init() {
	if errSlotLeaderSCInit != nil {
		panic("err in slot leader sc initialize ")
	}

	s := slotleader.GetSlotLeaderSelection()
	stgOneIdArr, _ = s.GetStage1FunctionID()
}

type slotLeaderSC struct {
}

func (c *slotLeaderSC) RequiredGas(input []byte) uint64 {

	// A_i=α_i*PKi i = {1,2,....n}. size = sizeof(ecdsa.PublicKey)*N
	// π_i							size = sizeof(uint64)x2 w[0]=e w[1]=z
	//return params.SlsStgTwoPerByteGas * uint64(len(input))
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

	hashEpochID := crypto.Keccak256Hash(epochID)

	// StateDB useage
	// Level 1 : epochID's Hash  -> common.Address
	// Level 2 : string joined Hash -> common.hash
	// Level 3 : data -> byte[]

	var level1 common.Address
	var level2 common.Hash

	level1 = common.BytesToAddress(hashEpochID.Bytes())

	keyValue := make([]byte, 0)
	keyValue = append(keyValue, stgOneIdArr[0], stgOneIdArr[1], stgOneIdArr[2], stgOneIdArr[3])
	keyValue = append(keyValue, selfIndex...)

	level2 = crypto.Keccak256Hash(keyValue)

	evm.StateDB.SetStateByteArray(level1, level2, data)

	// Read and Verify
	readBuf := evm.StateDB.GetStateByteArray(level1, level2)

	epID, index, pk, pkMi, err := s.RlpUnpackWithCompressedPK(readBuf)

	if hex.EncodeToString(epID) == hex.EncodeToString(epochID) &&
		hex.EncodeToString(index) == hex.EncodeToString(selfIndex) &&
		hex.EncodeToString(pk) == hex.EncodeToString(pkSelf) &&
		hex.EncodeToString(pkMi) == hex.EncodeToString(miGen) &&
		err == nil {
		log.Debug("Data save to StateDb and verified success")
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
	s := slotleader.GetSlotLeaderSelection()
	data, err := s.UnpackStage1Data(tx.Data())
	if err != nil {
		return err
	}
	_, _, pkSelf, _, err := s.RlpUnpackAndWithUncompressPK(data) // use this function to unpack rlp []byte
	if err != nil {
		return err
	}
	if !s.InEpochLeadersOrNotByPk(pkSelf) {
		return errIllegalSender
	}
	return nil
}
