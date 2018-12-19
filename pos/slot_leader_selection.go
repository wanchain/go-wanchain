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
	workingEpochID uint64
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

//Loop check work every 10 second. Called by backend loop
//It's all slotLeaderSelection's main workflow loop
//It's not loop at all, it is loop called by backend
func (s *SlotLeaderSelection) Loop() {
	epochID := s.getEpochID()
	s.log("Now epchoID: " + Uint64ToString(epochID))

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

//GenerateCommitment generate a commitment and send it by tx message
//Returns the commitment buffer []byte which is publicKey and alpha * publicKey
//payload should be send with tx.
func (s *SlotLeaderSelection) GenerateCommitment(publicKey *ecdsa.PublicKey, epochID uint64) ([]byte, error) {
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
	epochIDBuf := big.NewInt(int64(epochID)).Bytes()

	buffer, err := rlp.EncodeToBytes([][]byte{epochIDBuf, pkCompress, miCompress})

	GetDb().Put(epochID, "alpha", alpha.Bytes())

	return buffer, err
}

//GetAlpha get alpha of epochID
func (s *SlotLeaderSelection) GetAlpha(epochID uint64) (*big.Int, error) {
	buf, err := GetDb().Get(epochID, "alpha")
	if err != nil {
		return nil, err
	}

	var alpha = big.NewInt(0).SetBytes(buf)
	return alpha, nil
}

//getLocalPublicKey get local public key from memory keystore
func (s *SlotLeaderSelection) getLocalPublicKey() (*ecdsa.PublicKey, error) {
	return nil, nil
}

//getEpochID get epochID by local time
func (s *SlotLeaderSelection) getEpochID() uint64 {
	epochTimespan := int64(SlotTime * SlotCount)
	timeUnix := time.Now().Unix()

	epochID := uint64((timeUnix - EpochGenesisTime) / epochTimespan)
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
func (s *SlotLeaderSelection) getEpochLeaders(epochID uint64) []*ecdsa.PublicKey {

	//generate test publicKey
	epochLeaders := make([]*ecdsa.PublicKey, EpochLeaderCount)
	for i := 0; i < EpochLeaderCount; i++ {
		key, _ := crypto.GenerateKey()
		epochLeaders = append(epochLeaders, &key.PublicKey)
	}

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

func (s *SlotLeaderSelection) startStage1Work(epochLeaders []*ecdsa.PublicKey) error {
	// selfPublicKey, _ := s.getLocalPublicKey()

	// for i := 0; i < len(epochLeaders); i++ {
	// 	if PkEqual(selfPublicKey, epochLeaders[i]) {
	// 		workingEpochID, err := s.getWorkingEpochID()
	// 		if err != nil {
	// 			return err
	// 		}
	// 		//data, err := s.GenerateCommitment(selfPublicKey, workingEpochID)
	// 	}
	// }

	return nil
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

func (s *SlotLeaderSelection) sendTx(data []byte) {

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
