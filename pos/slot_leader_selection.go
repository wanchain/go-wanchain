package pos

import (
	"context"
	"crypto/ecdsa"
	Rand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/accounts/abi"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"

	"github.com/btcsuite/btcd/btcec"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/uleaderselection"
)

//CompressedPubKeyLen means a compressed public key byte len.
const CompressedPubKeyLen = 33

const (
	// EpochGenesisTime is the pos start time such as: 2018-12-12 00:00:00 == 1544544000
	EpochGenesisTime = uint64(1544544000)

	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 10

	// SlotCount is slot count in an epoch
	SlotCount = 180

	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 10

	// SlotStage1 is 40% of slot count
	SlotStage1 = int(SlotCount * 0.4)
	// SlotStage2 is 80% of slot count
	SlotStage2 = int(SlotCount * 0.8)
)

const (
	//Ready to start slot leader selection stage1
	slotLeaderSelectionStage1 = iota + 1 //1

	//Slot leader selection stage1 finish
	slotLeaderSelectionStage2 = iota + 1 //2
)

//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	workingEpochID uint64
	workStage      int
	rc             *rpc.Client
}

var slotLeaderSelection *SlotLeaderSelection

func init() {
	slotLeaderSelection = &SlotLeaderSelection{}
}

//GetSlotLeaderSelection get the SlotLeaderSelection's object
func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}

//--------------Workflow functions-------------------------------------------------------------

//Loop check work every 10 second. Called by backend loop
//It's all slotLeaderSelection's main workflow loop
//It's not loop at all, it is loop called by backend
func (s *SlotLeaderSelection) Loop(rc *rpc.Client) {
	s.rc = rc

	epochID, slotID, err := GetEpochSlotID()
	s.log("Now epchoID:" + Uint64ToString(epochID) + " slotID:" + Uint64ToString(slotID))

	workStage, err := s.getWorkStage(epochID)

	if err != nil {
		if err.Error() == "leveldb: not found" {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
			workStage = slotLeaderSelectionStage1
		} else {
			s.log("getWorkStage error: " + err.Error())
		}
	}

	switch workStage {
	case slotLeaderSelectionStage1:
		epochLeaders := s.getEpochLeaders(epochID)
		if epochLeaders != nil {
			s.setWorkingEpochID(epochID)
			err := s.startStage1Work(epochLeaders)
			if err != nil {
				s.log(err.Error())
			} else {
				s.setWorkStage(epochID, slotLeaderSelectionStage2)
			}
		}

	case slotLeaderSelectionStage2:
		//If New epoch start
		s.workingEpochID, err = s.getWorkingEpochID()
		if epochID > s.workingEpochID {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
		}

	default:
	}

}

// startStage1Work start the stage 1 work and send tx
func (s *SlotLeaderSelection) startStage1Work(epochLeaders []*ecdsa.PublicKey) error {
	selfPublicKey, _ := s.getLocalPublicKey()

	for i := 0; i < len(epochLeaders); i++ {
		if PkEqual(selfPublicKey, epochLeaders[i]) {
			workingEpochID, err := s.getWorkingEpochID()
			if err != nil {
				return err
			}
			data, err := s.GenerateCommitment(selfPublicKey, workingEpochID, uint64(i))
			if err != nil {
				return err
			}

			err = s.sendTx(data)
			if err != nil {
				s.log(err.Error())
			}
		}
	}

	return nil
}

//GenerateCommitment generate a commitment and send it by tx message
//Returns the commitment buffer []byte which is publicKey and alpha * publicKey
//payload should be send with tx.
func (s *SlotLeaderSelection) GenerateCommitment(publicKey *ecdsa.PublicKey,
	epochID uint64, selfIndexInEpochLeader uint64) ([]byte, error) {

	if publicKey == nil || publicKey.X == nil || publicKey.Y == nil {
		return nil, errors.New("Invalid input parameters")
	}

	if !crypto.S256().IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, errors.New("Public key point is not on S256 curve")
	}

	alpha, err := uleaderselection.RandFieldElement(Rand.Reader)
	if err != nil {
		return nil, err
	}
	fmt.Println("alpha:", hex.EncodeToString(alpha.Bytes()))

	commitment, err := uleaderselection.GenerateCommitment(publicKey, alpha)
	if err != nil {
		return nil, err
	}

	pk := btcec.PublicKey(*commitment[0])
	mi := btcec.PublicKey(*commitment[1])

	pkCompress := pk.SerializeCompressed()
	miCompress := mi.SerializeCompressed()
	epochIDBuf := big.NewInt(0).SetUint64(epochID).Bytes()
	selfIndexBuf := Uint64ToBytes(selfIndexInEpochLeader)

	buffer, err := s.RlpPackCompressedPK(epochIDBuf, selfIndexBuf, pkCompress, miCompress)

	GetDb().PutWithIndex(epochID, selfIndexInEpochLeader, "alpha", alpha.Bytes())

	return buffer, err
}

// RlpPackCompressedPK pack infomations into rlp []byte
func (s *SlotLeaderSelection) RlpPackCompressedPK(epochIDBuf []byte, selfIndexBuf []byte, pkCompress []byte, miCompress []byte) ([]byte, error) {
	return rlp.EncodeToBytes([][]byte{epochIDBuf, selfIndexBuf, pkCompress, miCompress})
}

func (s *SlotLeaderSelection) RlpUnpackCompressedPK(buf []byte) (epochIDBuf []byte, selfIndexBuf []byte, pkCompress []byte, miCompress []byte, err error) {
	var output [][]byte
	err = rlp.DecodeBytes(buf, &output)
	epochIDBuf = output[0]
	selfIndexBuf = output[1]
	pkCompress = output[2]
	miCompress = output[3]
	return
}

//GetAlpha get alpha of epochID
func (s *SlotLeaderSelection) GetAlpha(epochID uint64, selfIndex uint64) (*big.Int, error) {
	buf, err := GetDb().GetWithIndex(epochID, selfIndex, "alpha")
	if err != nil {
		return nil, err
	}

	var alpha = big.NewInt(0).SetBytes(buf)
	return alpha, nil
}

// GetStage1Abi can get a abi instance of slot leader selection stage1
func (s *SlotLeaderSelection) GetStage1Abi() (abi.ABI, error) {
	slotDefinition :=
		`[
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
	return abi.JSON(strings.NewReader(slotDefinition))
}

// PackStage1Data can pack stage1 data into abi []byte for tx payload
func (s *SlotLeaderSelection) PackStage1Data(input []byte) ([]byte, error) {
	abi, err := s.GetStage1Abi()
	if err != nil {
		return nil, err
	}
	data := hex.EncodeToString(input)
	return abi.Pack("slotLeaderStage1MiSave", data)
}

// UnpackStage1Data use to unpack payload
// NOTE: Must Use payload[:] for input
func (s *SlotLeaderSelection) UnpackStage1Data(input []byte) ([]byte, error) {
	abi, err := s.GetStage1Abi()
	if err != nil {
		return nil, err
	}
	var data string
	err = abi.Unpack(&data, "slotLeaderStage1MiSave", input[4:])
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(data)
}

// GetStage1FunctionID get function id of slot select stage1 from abi
func (s *SlotLeaderSelection) GetStage1FunctionID() ([4]byte, error) {
	var slotStage1ID [4]byte

	abi, err := s.GetStage1Abi()
	if err != nil {
		return slotStage1ID, err
	}

	copy(slotStage1ID[:], abi.Methods["slotLeaderStage1MiSave"].Id())

	return slotStage1ID, nil
}

// GetFuncIDFromPayload get function id from payload data
func (s *SlotLeaderSelection) GetFuncIDFromPayload(payload []byte) ([4]byte, error) {
	var methodID [4]byte
	if len(payload) < 4 {
		return methodID, errors.New("input is too short")
	}

	copy(methodID[:], payload[:4])

	return methodID, nil
}

//---------------Information get/set functions--------------------------------------------

//getLocalPublicKey get local public key from memory keystore
func (s *SlotLeaderSelection) getLocalPublicKey() (*ecdsa.PublicKey, error) {

	// test
	ret, err := GetDb().Get(0, "testSelfPK")
	pk := crypto.ToECDSAPub(ret)

	return pk, err
}

// GetEpochSlotID get current epochID and slotID in this epoch by local time
// returns epochID, slotID, error
func GetEpochSlotID() (uint64, uint64, error) {
	epochTimespan := uint64(SlotTime * SlotCount)
	timeUnix := uint64(time.Now().Unix())

	if EpochGenesisTime > timeUnix {
		return 0, 0, errors.New("Epoch genesis time is not arrive")
	}

	epochID := uint64((timeUnix - EpochGenesisTime) / epochTimespan)

	epochIndex := uint64((timeUnix - EpochGenesisTime) / epochTimespan)

	epochStartTime := epochIndex*epochTimespan + EpochGenesisTime

	timeInEpoch := timeUnix - epochStartTime

	slotID := uint64(timeInEpoch / SlotTime)

	return epochID, slotID, nil
}

//getEpochLeaders get epochLeaders of epochID in StateDB
func (s *SlotLeaderSelection) getEpochLeaders(epochID uint64) []*ecdsa.PublicKey {

	//test: generate test publicKey
	epochLeaders := make([]*ecdsa.PublicKey, EpochLeaderCount)
	for i := 0; i < EpochLeaderCount; i++ {
		key, _ := crypto.GenerateKey()
		epochLeaders[i] = &key.PublicKey
	}

	//test:
	GetDb().Put(0, "testSelfPK", crypto.FromECDSAPub(epochLeaders[3]))

	return epochLeaders
}

//getWorkStage get work stage of epochID from levelDB
func (s *SlotLeaderSelection) getWorkStage(epochID uint64) (int, error) {
	ret, err := GetDb().Get(epochID, "slotLeaderWorkStage")
	workStageUint64 := BytesToUint64(ret)
	return int(workStageUint64), err
}

//saveWorkStage save the work stage of epochID in levelDB
func (s *SlotLeaderSelection) setWorkStage(epochID uint64, workStage int) error {
	workStageBig := big.NewInt(int64(workStage))
	_, err := GetDb().Put(epochID, "slotLeaderWorkStage", workStageBig.Bytes())
	return err
}

func (s *SlotLeaderSelection) setCurrentWorkStage(workStage int) {
	currentEpochID, _ := s.getWorkingEpochID()
	s.setWorkStage(currentEpochID, workStage)
}

func (s *SlotLeaderSelection) log(info string) {
	fmt.Println(info)
}

func (s *SlotLeaderSelection) getWorkingEpochID() (uint64, error) {
	ret, err := GetDb().Get(0, "slotLeaderCurrentSlotID")
	retUint64 := BytesToUint64(ret)
	return retUint64, err
}

func (s *SlotLeaderSelection) setWorkingEpochID(workingEpochID uint64) error {
	_, err := GetDb().Put(0, "slotLeaderCurrentSlotID", Uint64ToBytes(workingEpochID))
	return err
}

//--------------Transacton create / send --------------------------------------------

func (s *SlotLeaderSelection) sendTx(data []byte) error {
	//test
	fmt.Println("Simulator send tx:", hex.EncodeToString(data))

	if s.rc == nil {
		return errors.New("rc is not ready")
	}

	ctx := context.Background()
	rc := s.rc

	slotLeaderPrecompileAddr := common.BytesToAddress(big.NewInt(600).Bytes())
	var to = slotLeaderPrecompileAddr
	amount := new(big.Int)
	amount.SetString("100", 10) // 100 tokens

	//type SendTxArgs struct {
	//	From     common.Address  `json:"from"`
	//	To       *common.Address `json:"to"`
	//	Gas      *hexutil.Big    `json:"gas"`
	//	GasPrice *hexutil.Big    `json:"gasPrice"`
	//	Value    *hexutil.Big    `json:"value"`
	//	Data     hexutil.Bytes   `json:"data"`
	//	Nonce    *hexutil.Uint64 `json:"nonce"`
	//}
	arg := map[string]interface{}{}
	arg["from"] = common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
	arg["to"] = &to
	arg["value"] = (*hexutil.Big)(amount)
	arg["txType"] = 1
	//Set payload infomation--------------

	payload, err := s.PackStage1Data(data)
	if err != nil {
		return err
	}
	arg["data"] = payload

	var txHash common.Hash
	callErr := rc.CallContext(ctx, &txHash, "eth_sendTransaction", arg)
	if nil != callErr {
		fmt.Println(callErr)
	}
	fmt.Println(txHash)
	return nil
}

//-------------common functions---------------------------------------

//PkEqual only can use in same curve. return whether the two points equal
func PkEqual(pk1, pk2 *ecdsa.PublicKey) bool {
	if pk1 == nil || pk2 == nil {
		return false
	}

	if hex.EncodeToString(pk1.X.Bytes()) == hex.EncodeToString(pk2.X.Bytes()) &&
		hex.EncodeToString(pk1.Y.Bytes()) == hex.EncodeToString(pk2.Y.Bytes()) {
		return true
	}
	return false
}

// Uint64ToBytes use a big.Int to transfer uint64 to bytes
// Must use big.Int to reverse
func Uint64ToBytes(input uint64) []byte {
	return big.NewInt(0).SetUint64(input).Bytes()
}

// BytesToUint64 use a big.Int to transfer uint64 to bytes
// Must input a big.Int bytes
func BytesToUint64(input []byte) uint64 {
	return big.NewInt(0).SetBytes(input).Uint64()
}

// Uint64ToString can change uint64 to string through a big.Int
func Uint64ToString(input uint64) string {
	return big.NewInt(0).SetUint64(input).String()
}

//-------------------------------------------------------------------
