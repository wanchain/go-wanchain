package util

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/common/hexutil"

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
	//if posconfig.EpochBaseTime == 0 || time < posconfig.EpochBaseTime {
	//	return
	//}
	//timeUnix := uint64(time.Now().Unix())
	timeUnix := time
	epochTimespan := uint64(posconfig.SlotTime * posconfig.SlotCount)
	epochId = uint64(timeUnix / epochTimespan)
	slotId = uint64(timeUnix / posconfig.SlotTime % posconfig.SlotCount)
	//fmt.Println("CalEpochSlotID:", epochId, slotId)
	return epochId, slotId
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
	timeUnix := uint64(time.Now().Unix())
	epochTimeSpan := uint64(posconfig.SlotTime * posconfig.SlotCount)
	curEpochId = uint64((timeUnix) / epochTimeSpan)
	curSlotId = uint64((timeUnix) / posconfig.SlotTime % posconfig.SlotCount)
	//fmt.Println("CalEpochSlotID:", curEpochId, curSlotId)
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
	GetCurrentHeader() *types.Header
	//TryGetAndSaveAllStakerInfoBytes(epochId uint64) (*[][]byte, error)
}

func GetCurrentBlkEpochSlotID() (epochID, slotID uint64) {

	inst :=  GetEpocherInst()
	if inst == nil {
		return 0,0
	}

	curheader := inst.GetCurrentHeader()
	if curheader == nil {
		return 0,0
	}

	return GetEpochSlotIDFromDifficulty(curheader.Difficulty)
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
	// if haven't switch to pos, it could be nil
	if selecter == nil {
		return nil
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
	posconfig.CurrentEpochId = epochID
}
func updateEpochBlock(epochID uint64, slotID uint64, blockNumber uint64, hash common.Hash) {
	if epochID != lastEpochId {
		lastEpochId = epochID
	}
	// there is 2K slot, so need not think about reorg  // selec epoch leader from the whole epoch.
	if slotID >= 2*posconfig.K+1 && selectedEpochId != epochID+1 && epochID != posconfig.FirstEpochId {
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

func IsPosBlock(number uint64) bool {
	return number >= posconfig.Pow2PosUpgradeBlockNumber
}

func FirstPosBlockNumber() uint64 {
	return posconfig.Pow2PosUpgradeBlockNumber
}

var (
	whiteMap map[common.Address]bool
)

func init() {
	whiteMap = make(map[common.Address]bool)
	for i := 0; i < len(posconfig.WhiteListOrig); i++ {
		pk := crypto.ToECDSAPub(hexutil.MustDecode(posconfig.WhiteListOrig[i]))
		addr := crypto.PubkeyToAddress(*pk)
		whiteMap[addr] = true
	}
}

func IsWhiteAddr(addr *common.Address) bool {
	if addr == nil {
		return false
	}

	if _, ok := whiteMap[*addr]; ok {
		return true
	}

	return false
}

// Get app memory use
func MemStat() uint64 {
	memStat := new(runtime.MemStats)
	runtime.ReadMemStats(memStat)
	return memStat.Alloc
}
