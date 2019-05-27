package util

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"math/big"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	bn256 "github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/pos/posconfig"
)

func CalEpochSlotID(time uint64) (epochId, slotId uint64) {
	if posconfig.EpochBaseTime == 0 || time < posconfig.EpochBaseTime {
		return
	}
	//timeUnix := uint64(time.Now().Unix())
	timeUnix := time
	epochTimespan := uint64(posconfig.SlotTime * posconfig.SlotCount)
	epochId = uint64((timeUnix - posconfig.EpochBaseTime - posconfig.EpochOffsetTime) / epochTimespan)
	slotId = uint64((timeUnix - posconfig.EpochBaseTime  - posconfig.EpochOffsetTime) / posconfig.SlotTime % posconfig.SlotCount)
	//fmt.Println("CalEpochSlotID:", epochId, slotId)
	return epochId, slotId
}




func IncreaseOffsetTime(header *types.Header,uselocalTime  bool) {

	headerTime := header.Time.Uint64()

	if uselocalTime {
		headerTime = uint64(time.Now().Unix())
	}

	epid, slid := CalEpSlbyTd(header.Difficulty.Uint64())

	if epid == 0 && slid == 0 || posconfig.EpochBaseTime == 0{
		return
	}

	slotTime := (epid*posconfig.SlotCount+slid)*posconfig.SlotTime + posconfig.EpochBaseTime

	offset := ((headerTime - slotTime)/posconfig.SlotTime - 1)*posconfig.SlotTime

	posconfig.EpochOffsetTime = posconfig.EpochOffsetTime +  offset
	posconfig.EpochOTLatestBlk = header.Number.Uint64()

	SaveTimeOffset(posconfig.EpochOffsetTime,header.Number.Uint64())

}

func DecreaseOffsetTimeByBlock(chain []*types.Block) {

	for i := len(chain) - 1; i > 0 ; i-- {
		header := chain[i].Header()
		decreaseOffsetTime(header)
	}
}

func DecreaseOffsetTimeByHeader(chain []*types.Header) {

	for i := len(chain) - 1; i > 0 ; i-- {
		decreaseOffsetTime(chain[i])
	}
}

func decreaseOffsetTime(header *types.Header) {

	if header.Number.Uint64() <= posconfig.EpochOTLatestBlk {
		headerTime := header.Time.Uint64()
		epid, slid := CalEpSlbyTd(header.Difficulty.Uint64())
		slotTime := (epid*posconfig.SlotCount+slid)*posconfig.SlotTime + posconfig.EpochBaseTime

		offset := headerTime - slotTime

		posconfig.EpochOffsetTime = posconfig.EpochBaseTime - offset
		posconfig.EpochOTLatestBlk = header.Number.Uint64()

		RemoveTimeOffset(header.Number.Uint64())

	}
}


func SaveTimeOffset(offsetTime uint64, blknr uint64) {

	TmOffsetDb := posdb.GetDbByName(posconfig.EpochOffsetDB)
	if TmOffsetDb == nil {
		TmOffsetDb = posdb.NewDb(posconfig.EpochOffsetDB)
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, offsetTime)

	TmOffsetDb.Put(blknr, "offsetTime", b)
}

func ReadTimeOffset(blknr uint64) uint64 {

	TmOffsetDb := posdb.GetDbByName(posconfig.EpochOffsetDB)
	if TmOffsetDb == nil {
		TmOffsetDb = posdb.NewDb(posconfig.EpochOffsetDB)
		return 0
	}

	numberBytes,err:=TmOffsetDb.Get(blknr, "offsetTime")
	if err != nil || numberBytes == nil {
		return 0
	}

	offset := binary.BigEndian.Uint64(numberBytes)

	return offset
}

func RemoveTimeOffset(blknr uint64){
	TmOffsetDb := posdb.GetDbByName(posconfig.EpochOffsetDB)
	if TmOffsetDb == nil {
		return
	}
	b := make([]byte,8)
	binary.BigEndian.PutUint64(b, 0)
	TmOffsetDb.Put(blknr, "offsetTime", b)
}


var (
	curEpochId = uint64(0)
	curSlotId  = uint64(0)
)

func GetEpochSlotID() (uint64, uint64) {
	return curEpochId, curSlotId
}
func CalEpochSlotIDByNow() {
	//if posconfig.EpochBaseTime == 0 {
	//	return
	//}
	//timeUnix := uint64(time.Now().Unix())
	//epochTimeSpan := uint64(posconfig.SlotTime * posconfig.SlotCount)
	//curEpochId = uint64((timeUnix - posconfig.EpochBaseTime) / epochTimeSpan)
	//curSlotId = uint64((timeUnix - posconfig.EpochBaseTime) / posconfig.SlotTime % posconfig.SlotCount)
	//fmt.Println("CalEpochSlotID:", curEpochId, curSlotId)

	curEpochId,curSlotId = CalEpochSlotID(uint64(time.Now().Unix()))
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

type SelectLead interface {
	SelectLeadersLoop(epochId uint64) error
	GetProposerBn256PK(epochID uint64, idx uint64, addr common.Address) []byte
	GetRBProposerG1(epochID uint64) []bn256.G1
	GetEpochLeaders(epochID uint64) [][]byte
	GetEpochLastBlkNumber(targetEpochId uint64) uint64
	//TryGetAndSaveAllStakerInfoBytes(epochId uint64) (*[][]byte, error)
}

var (
	lastBlockEpoch     = make(map[uint64]uint64)
	lastBlockHashEpoch = make(map[uint64]common.Hash)
	lbe                = sync.Mutex{}
	selecter           SelectLead
	lastEpochId        = uint64(0)
	selectedEpochId    = uint64(0)
)

func SetEpocherInst(sor SelectLead) {
	selecter = sor
}

func GetEpocherInst() SelectLead {
	// TODO: can't be nil
	if selecter == nil {
		panic("GetEpocherInst")
	}
	return selecter
}

func CalEpSlbyTd(blkTd uint64) (epochID uint64, slotID uint64) {
	epochID = (blkTd >> 32)
	slotID = ((blkTd & 0xffffffff) >> 8)
	return epochID, slotID
}
func UpdateEpochBlock(block *types.Block) {
	blkTd := block.Difficulty().Uint64()
	epochID, slotID := CalEpSlbyTd(blkTd)
	updateEpochBlock(epochID, slotID, block.Header().Number.Uint64(), block.Header().Hash())
}
func updateEpochBlock(epochID uint64, slotID uint64, blockNumber uint64, hash common.Hash) {
	if epochID != lastEpochId {
		lastEpochId = epochID
	}
	// there is 2K slot, so need not think about reorg
	if slotID >= 2*posconfig.K+1 && selectedEpochId != epochID+1 {
		go GetEpocherInst().SelectLeadersLoop(epochID + 1)
		selectedEpochId = epochID + 1
	}

	SetEpochBlock(epochID, blockNumber, hash)
}

func SetEpochBlock(epochID uint64, blockNumber uint64, hash common.Hash) {
	lbe.Lock()
	lastBlockEpoch[epochID] = blockNumber
	lastBlockHashEpoch[epochID] = hash
	lbe.Unlock()
}

//this function only can return
func GetEpochBlock(epochID uint64) uint64 {

	lbe.Lock()
	b := lastBlockEpoch[epochID]
	lbe.Unlock()
	
	if b == 0 {
		b = selecter.GetEpochLastBlkNumber(epochID)
	}
	
	return b
}


func GetEpochBlockHash(epochID uint64) common.Hash {
	lbe.Lock()
	bh := lastBlockHashEpoch[epochID]
	lbe.Unlock()
	return bh
}
func GetProposerBn256PK(epochID uint64, idx uint64, addr common.Address) []byte {
	return GetEpocherInst().GetProposerBn256PK(epochID, idx, addr)
}

func TryGetAndSaveAllStakerInfoBytes(epochId uint64) (*[][]byte, error) {
	//return GetEpocherInst().TryGetAndSaveAllStakerInfoBytes(epochId)
	return nil, nil
}

// CompressPk
func CompressPk(pk *ecdsa.PublicKey) ([]byte, error) {
	if !crypto.S256().IsOnCurve(pk.X, pk.Y) {
		return nil, errors.New("Pk point is not on S256 curve")
	}
	pkBtc := btcec.PublicKey(*pk)
	return pkBtc.SerializeCompressed(), nil
}

// UncompressPk
func UncompressPk(buf []byte) (*ecdsa.PublicKey, error) {
	key, err := btcec.ParsePubKey(buf, btcec.S256())
	if err != nil {
		return nil, err
	}
	return (*ecdsa.PublicKey)(key), nil
}

func GetAbi(abiString string) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(abiString))
}

// GetEpochSlotIDFromDifficulty can get epochID and slotID from difficulty.
func GetEpochSlotIDFromDifficulty(difficulty *big.Int) (epochID, slotID uint64) {
	if difficulty == nil {
		return 0, 0
	}

	epochID = difficulty.Uint64() >> 32
	slotID = (difficulty.Uint64() >> 8) & 0x00ffffff
	return
}

// FromWin use to calc win to wan
func FromWin(win *big.Int) float64 {
	winStr := win.String()
	wan, err := strconv.ParseFloat(winStr, 64)
	if err != nil {
		return 0
	}
	return wan
}
