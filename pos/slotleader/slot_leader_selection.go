package slotleader

import (
	"bytes"
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
	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"

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
const LengthPublicKeyBytes = 65

const (
	// EpochGenesisTime is the pos start time such as: 2018-12-12 00:00:00 == 1544544000
	EpochGenesisTime = uint64(1544544000)

	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 10

	// SlotCount is slot count in an epoch
	SlotCount = 180

	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 1

	// SlotStage1 is 40% of slot count
	SlotStage1 = int(SlotCount * 0.4)
	// SlotStage2 is 80% of slot count
	SlotStage2 = int(SlotCount * 0.8)
	EpochLeaders 			= "epochLeaders"
	SecurityMsg  			= "securityMsg"
	RandFromProposer 		= "randFromProposer"
	RandomSeqs				= "randomSeqs"
	SlotLeader				= "slotLeader"
)

const (
	//Ready to start slot leader selection stage1
	slotLeaderSelectionStage1 = iota + 1 //1

	//Slot leader selection stage1 finish
	slotLeaderSelectionStage2 = iota + 1 //2
)

var (
	wanCscPrecompileAddr   = common.BytesToAddress(big.NewInt(210).Bytes())
	ErrEpochID = errors.New("EpochID is not valid")
)
//SlotLeaderSelection use to select unique slot leader
type SlotLeaderSelection struct {
	workingEpochID uint64
	workStage      int
	rc             *rpc.Client
	epochLeadersArray 	[]string      			// len(pki)=65
	epochLeadersMap		map[string] []uint64   	//
}

var slotLeaderSelection *SlotLeaderSelection

func init() {
	slotLeaderSelection = &SlotLeaderSelection{}
	slotLeaderSelection.epochLeadersMap 	= make(map[string] []uint64)
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
	functrace.Enter("SlotLeaderSelection Loop")
	s.rc = rc

	epochID, slotID, err := GetEpochSlotID()
	s.log("Now epchoID:" + posdb.Uint64ToString(epochID) + " slotID:" + posdb.Uint64ToString(slotID))

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
	functrace.Exit()
}

// startStage1Work start the stage 1 work and send tx
func (s *SlotLeaderSelection) startStage1Work(epochLeaders []*ecdsa.PublicKey) error {
	functrace.Enter("startStage1Work")
	selfPublicKey, _ := s.getLocalPublicKey()

	for i := 0; i < len(epochLeaders); i++ {
		if posdb.PkEqual(selfPublicKey, epochLeaders[i]) {
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
				return err
			}
		}
	}

	functrace.Exit()
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
	selfIndexBuf := posdb.Uint64ToBytes(selfIndexInEpochLeader)

	log.Debug("epochIDBuf(hex): " + hex.EncodeToString(epochIDBuf))
	log.Debug("selfIndexBuf: " + hex.EncodeToString(selfIndexBuf))
	log.Debug("pkCompress: " + hex.EncodeToString(pkCompress))
	log.Debug("miCompress: " + hex.EncodeToString(miCompress))

	buffer, err := s.RlpPackCompressedPK(epochIDBuf, selfIndexBuf, pkCompress, miCompress)

	posdb.GetDb().PutWithIndex(epochID, selfIndexInEpochLeader, "alpha", alpha.Bytes())

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
	buf, err := posdb.GetDb().GetWithIndex(epochID, selfIndex, "alpha")
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
	ret, err := posdb.GetDb().Get(0, "testSelfPK")
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
	posdb.GetDb().Put(0, "testSelfPK", crypto.FromECDSAPub(epochLeaders[3]))

	return epochLeaders
}

//getWorkStage get work stage of epochID from levelDB
func (s *SlotLeaderSelection) getWorkStage(epochID uint64) (int, error) {
	ret, err := posdb.GetDb().Get(epochID, "slotLeaderWorkStage")
	workStageUint64 := posdb.BytesToUint64(ret)
	return int(workStageUint64), err
}

//saveWorkStage save the work stage of epochID in levelDB
func (s *SlotLeaderSelection) setWorkStage(epochID uint64, workStage int) error {
	workStageBig := big.NewInt(int64(workStage))
	_, err := posdb.GetDb().Put(epochID, "slotLeaderWorkStage", workStageBig.Bytes())
	return err
}

func (s *SlotLeaderSelection) buildEpochLeaderGroup(epochID uint64) error {
	// clear Array
	s.epochLeadersArray = make([]string,0)
	// clear map
	s.epochLeadersMap = make(map[string] []uint64)
	// build Array and map
	for index, value := range s.getEpochLeaders(epochID) {
		s.epochLeadersArray[index] = string(value)
		s.epochLeadersMap[string(value)] = append(s.epochLeadersMap[string(value)],uint64(index))
	}
	return nil
}
func (s *SlotLeaderSelection) generateSlotLeadsGroup(epochID uint64) error {
	// 1. get SMA[pre]
	piecesPtr := make([]*ecdsa.PublicKey,0)
	// pieces: alpha[1]*G, alpha[2]*G, .....
	pieces, err := posdb.GetDb().Get(epochID-1,SecurityMsg)
	if err != nil {
		return err
	}
	piecesCount := len(pieces)/LengthPublicKeyBytes
	var pubKeyByte []byte
	for i := 0; i < piecesCount; i++ {
		if i< piecesCount-2 {
			pubKeyByte = pieces[i*LengthPublicKeyBytes:(i+1)*LengthPublicKeyBytes]
		}else{
			pubKeyByte = pieces[i*LengthPublicKeyBytes:]
		}

		piecesPtr = append(piecesPtr, crypto.ToECDSAPub(pubKeyByte))
	}
	// 2. get random
	rb ,err := posdb.GetDb().Get(epochID-1,RandFromProposer)
	if err != nil {
		return err
	}
	// 3. get epochLeaders
	epochLeadersPtr := make([]*ecdsa.PublicKey,0)
	for _,epochLeaderPub := range s.epochLeadersArray {
		epochLeadersPtr = append(epochLeadersPtr, crypto.ToECDSAPub([]byte(epochLeaderPub)))
	}
	
	// 5. return slot leaders pointers.
	slotLeadersPtr := make([]*ecdsa.PublicKey,0)
	slotLeadersPtr, _,err = uleaderselection.GenerateSlotLeaderSeq(piecesPtr,epochLeadersPtr,rb,SlotCount)
	if err != nil {
		return err
	}
	// 6. insert slot address to local DB
	for index, val := range slotLeadersPtr {
		_, err = posdb.GetDb().PutWithIndex(epochID,uint64(index),SlotLeader,crypto.FromECDSAPub(val))
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *SlotLeaderSelection) generateSecurityPieces(epochID uint64,PrivateKey *ecdsa.PrivateKey,
	ArrayPiece []*ecdsa.PublicKey) ([]*ecdsa.PublicKey, error){
	return nil, nil
}
func (s *SlotLeaderSelection) generateSecurityMsg(epochID uint64,PrivateKey *ecdsa.PrivateKey,
	ArrayPiece []*ecdsa.PublicKey) error{
	smasPtr := make([]*ecdsa.PublicKey,0)
	var smasBytes bytes.Buffer
	var err 	error
	smasPtr, err = uleaderselection.GenerateSMA(PrivateKey,ArrayPiece)
	if err != nil {
		return err
	}
	for _, value := range smasPtr {
		smasBytes.Write(crypto.FromECDSAPub(value))
	}
	_, err = posdb.GetDb().Put(epochID,SecurityMsg,smasBytes.Bytes())
	if err != nil {
		return err
	}
	return nil
}
func (s *SlotLeaderSelection) setCurrentWorkStage(workStage int) {
	currentEpochID, _ := s.getWorkingEpochID()
	s.setWorkStage(currentEpochID, workStage)
}

func (s *SlotLeaderSelection) log(info string) {
	log.Debug(info)
	fmt.Println(info)
}

func (s *SlotLeaderSelection) getWorkingEpochID() (uint64, error) {
	ret, err := posdb.GetDb().Get(0, "slotLeaderCurrentSlotID")
	retUint64 := posdb.BytesToUint64(ret)
	return retUint64, err
}

func (s *SlotLeaderSelection) setWorkingEpochID(workingEpochID uint64) error {
	_, err := posdb.GetDb().Put(0, "slotLeaderCurrentSlotID", posdb.Uint64ToBytes(workingEpochID))
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

	log.Debug("ready to write data of payload: " + "0x" + hexutil.Encode(payload))

	arg["data"] = hexutil.Bytes(payload)

	log.Debug("finish to write data of payload")

	var txHash common.Hash
	callErr := rc.CallContext(ctx, &txHash, "eth_sendTransaction", arg)
	if nil != callErr {
		fmt.Println(callErr)
		log.Error("tx send failed")
		return errors.New("tx send failed")
	}
	fmt.Println(txHash)
	log.Debug("tx send success")
	return nil
}
