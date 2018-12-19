package pos

import (
	"crypto/ecdsa"
	Rand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/wanchain/go-wanchain/rlp"

	"github.com/btcsuite/btcd/btcec"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/uleaderselection"
)

//CompressedPubKeyLen means a compressed public key byte len.
const CompressedPubKeyLen = 33

const (
	// EpochGenesisTime is the pos start time such as: 2018-12-12 00:00:00
	EpochGenesisTime = 1544544000

	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 10

	// SlotCount is slot count in an epoch
	SlotCount = 180

	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 20

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
	workingEpochID *big.Int
	workStage      int
}

var slotLeaderSelection *SlotLeaderSelection

func init() {
	slotLeaderSelection = &SlotLeaderSelection{}
}

//GetSlotLeaderSelection get the SlotLeaderSelection's object
func GetSlotLeaderSelection() *SlotLeaderSelection {
	return slotLeaderSelection
}

//GenerateCommitment generate a commitment and send it by tx message
//Returns the commitment buffer []byte which is publicKey and alpha * publicKey
//payload should be send with tx.
func (s *SlotLeaderSelection) GenerateCommitment(publicKey *ecdsa.PublicKey, epochID *big.Int) ([]byte, error) {
	if publicKey == nil || epochID == nil || publicKey.X == nil || publicKey.Y == nil {
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
	epochIDBuf := epochID.Bytes()

	buffer, err := rlp.EncodeToBytes([][]byte{epochIDBuf, pkCompress, miCompress})

	GetDb().Put(epochID, "alpha", alpha.Bytes())

	return buffer, err
}

//GetAlpha get alpha of epochID
func (s *SlotLeaderSelection) GetAlpha(epochID *big.Int) (*big.Int, error) {
	buf, err := GetDb().Get(epochID, "alpha")
	if err != nil {
		return nil, err
	}

	var alpha = new(big.Int).SetBytes(buf)
	return alpha, nil
}

//Loop check work every 10 second. Called by backend loop
func (s *SlotLeaderSelection) Loop() {
	epochID := s.getEpochID()
	s.log("Now epchoID: " + epochID.String())

	workStage, err := s.getWorkStage(epochID)

	if err != nil {
		if err.Error() == "leveldb: not found" {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
			workStage = slotLeaderSelectionStage1
		} else {
			s.log("getWorkStage error: " + err.Error())
		}
	}

	s.workingEpochID, err = s.getWorkingEpochID()

	switch workStage {
	case slotLeaderSelectionStage1:
		epochLeaders := s.getEpochLeaders(epochID)
		if epochLeaders != nil {
			err := s.startStage1Work(epochLeaders)
			if err != nil {
				s.log(err.Error())
			} else {
				s.setWorkStage(epochID, slotLeaderSelectionStage2)
				s.setWorkingEpochID(epochID)
			}
		}

	case slotLeaderSelectionStage2:
		//If New epoch start
		if s.workingEpochID.Cmp(epochID) == -1 {
			s.setWorkStage(epochID, slotLeaderSelectionStage1)
		}
	default:
	}
}

//getLocalPublicKey get local public key from memory keystore
func (s *SlotLeaderSelection) getLocalPublicKey() (*ecdsa.PublicKey, error) {
	return nil, nil
}

//getEpochID get epochID by local time
func (s *SlotLeaderSelection) getEpochID() *big.Int {
	epochTimespan := int64(SlotTime * SlotCount)
	timeUnix := time.Now().Unix()

	epochID := big.NewInt((timeUnix - EpochGenesisTime) / epochTimespan)
	return epochID
}

//getSlotID get current slot by local time
func (s *SlotLeaderSelection) getSlotID() uint64 {
	epochTimespan := int64(SlotTime * SlotCount)
	timeUnix := time.Now().Unix()

	epochIndex := int64((timeUnix - EpochGenesisTime) / epochTimespan)

	epochStartTime := epochIndex * epochTimespan

	timeInEpoch := timeUnix - epochStartTime

	slotID := uint64(timeInEpoch / SlotTime)
	return slotID
}

//getEpochLeaders get epochLeaders of epochID in StateDB
func (s *SlotLeaderSelection) getEpochLeaders(epochID *big.Int) []*ecdsa.PublicKey {

	//generate test publicKey
	epochLeaders := make([]*ecdsa.PublicKey, EpochLeaderCount)
	for i := 0; i < EpochLeaderCount; i++ {
		key, _ := crypto.GenerateKey()
		epochLeaders = append(epochLeaders, &key.PublicKey)
	}

	return epochLeaders
}

//getWorkStage get work stage of epochID from levelDB
func (s *SlotLeaderSelection) getWorkStage(epochID *big.Int) (int, error) {
	ret, err := GetDb().Get(epochID, "slotLeaderWorkStage")
	workStageBig := big.NewInt(0).SetBytes(ret)
	return int(workStageBig.Int64()), err
}

//saveWorkStage save the work stage of epochID in levelDB
func (s *SlotLeaderSelection) setWorkStage(epochID *big.Int, workStage int) error {
	workStageBig := big.NewInt(int64(workStage))
	_, err := GetDb().Put(epochID, "slotLeaderWorkStage", workStageBig.Bytes())
	return err
}

func (s *SlotLeaderSelection) startStage1Work(epochLeaders []*ecdsa.PublicKey) error {
	return nil
}

func (s *SlotLeaderSelection) log(info string) {
	fmt.Println(info)
}

func (s *SlotLeaderSelection) getWorkingEpochID() (*big.Int, error) {
	ret, err := GetDb().Get(big.NewInt(0), "slotLeaderCurrentSlotID")
	retBig := big.NewInt(0).SetBytes(ret)
	return retBig, err
}

func (s *SlotLeaderSelection) setWorkingEpochID(workingEpochID *big.Int) error {
	_, err := GetDb().Put(big.NewInt(0), "slotLeaderCurrentSlotID", workingEpochID.Bytes())
	return err
}
