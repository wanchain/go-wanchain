package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/p2p/discover"
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
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
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

	initDelays()
	buildMapNodeId()
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


//

var (
	mapNodeID map[string]int     // string : nodeID, int: the index of the nodeID
	delays [50][50]float32
	localNodeStr string
	nodePrivate  [50]string
	)

func initDelays(){
	delays = [50][50]float32{
		{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{25.005,25.005,25.005,25.005,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{25.005,25.005,25.005,25.005,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{25.005,25.005,25.005,25.005,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{25.005,25.005,25.005,25.005,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{27.203,27.203,27.203,27.203,52.573,52.573,52.573,52.573,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{27.203,27.203,27.203,27.203,52.573,52.573,52.573,52.573,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{17.479,17.479,17.479,17.479,13.352,13.352,13.352,13.352,38.99,38.99,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{17.479,17.479,17.479,17.479,13.352,13.352,13.352,13.352,38.99,38.99,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{49.27,49.27,49.27,49.27,48.131,48.131,48.131,48.131,74.62,74.62,35.859,35.859,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{49.27,49.27,49.27,49.27,48.131,48.131,48.131,48.131,74.62,74.62,35.859,35.859,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{87.097,87.097,87.097,87.097,102.838,102.838,102.838,102.838,86.4,86.4,108.349,108.349,137.012,137.012,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{87.097,87.097,87.097,87.097,102.838,102.838,102.838,102.838,86.4,86.4,108.349,108.349,137.012,137.012,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{93.481,93.481,93.481,93.481,121.541,121.541,121.541,121.541,79.693,79.693,110.262,110.262,149.099,149.099,5.942,5.942,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{93.481,93.481,93.481,93.481,121.541,121.541,121.541,121.541,79.693,79.693,110.262,110.262,149.099,149.099,5.942,5.942,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{128.625,128.625,128.625,128.625,116.33,116.33,116.33,116.33,71.522,71.522,120.391,120.391,168.058,168.058,309.623,309.623,252.56,252.56,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{128.625,128.625,128.625,128.625,116.33,116.33,116.33,116.33,71.522,71.522,120.391,120.391,168.058,168.058,309.623,309.623,252.56,252.56,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{128.625,128.625,128.625,128.625,116.33,116.33,116.33,116.33,71.522,71.522,120.391,120.391,168.058,168.058,309.623,309.623,252.56,252.56,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{128.625,128.625,128.625,128.625,116.33,116.33,116.33,116.33,71.522,71.522,120.391,120.391,168.058,168.058,309.623,309.623,252.56,252.56,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{128.625,128.625,128.625,128.625,254.154,254.154,254.154,254.154,232.074,232.074,280.043,280.043,167.517,167.517,24.941,24.941,257.742,257.742,258.959,258.959,258.959,258.959,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{128.625,128.625,128.625,128.625,254.154,254.154,254.154,254.154,232.074,232.074,280.043,280.043,167.517,167.517,24.941,24.941,257.742,257.742,258.959,258.959,258.959,258.959,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{154.02,154.02,154.02,154.02,146.753,146.753,146.753,146.753,189.244,189.244,140.445,140.445,119.057,119.057,214.837,214.837,238.728,238.728,82.116,82.116,82.116,82.116,228.571,228.571,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{154.02,154.02,154.02,154.02,146.753,146.753,146.753,146.753,189.244,189.244,140.445,140.445,119.057,119.057,214.837,214.837,238.728,238.728,82.116,82.116,82.116,82.116,228.571,228.571,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{154.02,154.02,154.02,154.02,146.753,146.753,146.753,146.753,189.244,189.244,140.445,140.445,119.057,119.057,214.837,214.837,238.728,238.728,82.116,82.116,82.116,82.116,228.571,228.571,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{154.02,154.02,154.02,154.02,146.753,146.753,146.753,146.753,189.244,189.244,140.445,140.445,119.057,119.057,214.837,214.837,238.728,238.728,82.116,82.116,82.116,82.116,228.571,228.571,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{89.666,89.666,89.666,89.666,116.33,116.33,116.33,116.33,71.522,71.522,109.459,109.459,140.499,140.499,9.341,9.341,13.716,13.716,290.273,290.273,290.273,290.273,223.511,223.511,239.906,239.906,239.906,239.906,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{89.666,89.666,89.666,89.666,116.33,116.33,116.33,116.33,71.522,71.522,109.459,109.459,140.499,140.499,9.341,9.341,13.716,13.716,290.273,290.273,290.273,290.273,223.511,223.511,239.906,239.906,239.906,239.906,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{35.213,35.213,35.213,35.213,46.884,46.884,46.884,46.884,19.7,19.7,37.821,37.821,69.499,69.499,89.909,89.909,98.105,98.105,214.772,214.772,214.772,214.772,287.23,287.23,155.625,155.625,155.625,155.625,92.978,92.978,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{35.213,35.213,35.213,35.213,46.884,46.884,46.884,46.884,19.7,19.7,37.821,37.821,69.499,69.499,89.909,89.909,98.105,98.105,214.772,214.772,214.772,214.772,287.23,287.23,155.625,155.625,155.625,155.625,92.978,92.978,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{35.213,35.213,35.213,35.213,46.884,46.884,46.884,46.884,19.7,19.7,37.821,37.821,69.499,69.499,89.909,89.909,98.105,98.105,214.772,214.772,214.772,214.772,287.23,287.23,155.625,155.625,155.625,155.625,92.978,92.978,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{35.213,35.213,35.213,35.213,46.884,46.884,46.884,46.884,19.7,19.7,37.821,37.821,69.499,69.499,89.909,89.909,98.105,98.105,214.772,214.772,214.772,214.772,287.23,287.23,155.625,155.625,155.625,155.625,92.978,92.978,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{157.373,157.373,157.373,157.373,175.231,175.231,175.231,175.231,124.654,124.654,153.495,153.495,190.292,190.292,48.474,48.474,56.17,56.17,284.806,284.806,284.806,284.806,248.019,248.019,210.573,210.573,210.573,210.573,50.744,50.744,146.997,146.997,146.997,146.997,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{157.373,157.373,157.373,157.373,175.231,175.231,175.231,175.231,124.654,124.654,153.495,153.495,190.292,190.292,48.474,48.474,56.17,56.17,284.806,284.806,284.806,284.806,248.019,248.019,210.573,210.573,210.573,210.573,50.744,50.744,146.997,146.997,146.997,146.997,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{105.902,105.902,105.902,105.902,132.887,132.887,132.887,132.887,96.83,96.83,135.763,135.763,156.765,156.765,29.919,29.919,29.319,29.319,303.288,303.288,303.288,303.288,250.732,250.732,252.614,252.614,252.614,252.614,17.954,17.954,113.145,113.145,113.145,113.145,76.244,76.244,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{105.902,105.902,105.902,105.902,132.887,132.887,132.887,132.887,96.83,96.83,135.763,135.763,156.765,156.765,29.919,29.919,29.319,29.319,303.288,303.288,303.288,303.288,250.732,250.732,252.614,252.614,252.614,252.614,17.954,17.954,113.145,113.145,113.145,113.145,76.244,76.244,0,0,0,0,0,0,0,0,0,0,0,0,0,0},
		{329.447,329.447,329.447,329.447,178.032,178.032,178.032,178.032,220.14,220.14,174.227,174.227,138.732,138.732,260.28,260.28,286.912,286.912,308.499,308.499,308.499,308.499,237.804,237.804,114.11,114.11,114.11,114.11,278.547,278.547,206.074,206.074,206.074,206.074,337.606,337.606,291.397,291.397,0,0,0,0,0,0,0,0,0,0,0,0},
		{329.447,329.447,329.447,329.447,178.032,178.032,178.032,178.032,220.14,220.14,174.227,174.227,138.732,138.732,260.28,260.28,286.912,286.912,308.499,308.499,308.499,308.499,237.804,237.804,114.11,114.11,114.11,114.11,278.547,278.547,206.074,206.074,206.074,206.074,337.606,337.606,291.397,291.397,0,0,0,0,0,0,0,0,0,0,0,0},
		{101.491,101.491,101.491,101.491,110.044,110.044,110.044,110.044,81.837,81.837,115.474,115.474,146.002,146.002,12.013,12.013,12.024,12.024,227.359,227.359,227.359,227.359,282.781,282.781,254.811,254.811,254.811,254.811,10.847,10.847,95.11,95.11,95.11,95.11,51.363,51.363,28.594,28.594,288.496,288.496,0,0,0,0,0,0,0,0,0,0},
		{101.491,101.491,101.491,101.491,110.044,110.044,110.044,110.044,81.837,81.837,115.474,115.474,146.002,146.002,12.013,12.013,12.024,12.024,227.359,227.359,227.359,227.359,282.781,282.781,254.811,254.811,254.811,254.811,10.847,10.847,95.11,95.11,95.11,95.11,51.363,51.363,28.594,28.594,288.496,288.496,0,0,0,0,0,0,0,0,0,0},
		{253.423,253.423,253.423,253.423,254.87,254.87,254.87,254.87,213.892,213.892,232.803,232.803,217.568,217.568,174.865,174.865,140.487,140.487,222.023,222.023,222.023,222.023,229.972,229.972,115.368,115.368,115.368,115.368,215.415,215.415,225.992,225.992,225.992,225.992,177.067,177.067,127.642,127.642,132.016,132.016,134.452,134.452,0,0,0,0,0,0,0,0},
		{253.423,253.423,253.423,253.423,254.87,254.87,254.87,254.87,213.892,213.892,232.803,232.803,217.568,217.568,174.865,174.865,140.487,140.487,222.023,222.023,222.023,222.023,229.972,229.972,115.368,115.368,115.368,115.368,215.415,215.415,225.992,225.992,225.992,225.992,177.067,177.067,127.642,127.642,132.016,132.016,134.452,134.452,0,0,0,0,0,0,0,0},
		{108.988,108.988,108.988,108.988,133.65,133.65,133.65,133.65,91.794,91.794,120.838,120.838,157.993,157.993,27.136,27.136,32.847,32.847,251.223,251.223,251.223,251.223,330.442,330.442,269.058,269.058,269.058,269.058,20.214,20.214,109.154,109.154,109.154,109.154,50.76,50.76,25.543,25.543,302.561,302.561,23.896,23.896,127.26,127.26,0,0,0,0,0,0},
		{108.988,108.988,108.988,108.988,133.65,133.65,133.65,133.65,91.794,91.794,120.838,120.838,157.993,157.993,27.136,27.136,32.847,32.847,251.223,251.223,251.223,251.223,330.442,330.442,269.058,269.058,269.058,269.058,20.214,20.214,109.154,109.154,109.154,109.154,50.76,50.76,25.543,25.543,302.561,302.561,23.896,23.896,127.26,127.26,0,0,0,0,0,0},
		{128.625,128.625,128.625,128.625,126.685,126.685,126.685,126.685,94.887,94.887,120.391,120.391,165.265,165.265,24.941,24.941,40.681,40.681,317.107,317.107,317.107,317.107,258.959,258.959,273.601,273.601,273.601,273.601,16.937,16.937,108.525,108.525,108.525,108.525,56.891,56.891,21.587,21.587,309.598,309.598,22.944,22.944,158.536,158.536,7.979,7.979,0,0,0,0},
		{128.625,128.625,128.625,128.625,126.685,126.685,126.685,126.685,94.887,94.887,120.391,120.391,165.265,165.265,24.941,24.941,40.681,40.681,317.107,317.107,317.107,317.107,258.959,258.959,273.601,273.601,273.601,273.601,16.937,16.937,108.525,108.525,108.525,108.525,56.891,56.891,21.587,21.587,309.598,309.598,22.944,22.944,158.536,158.536,7.979,7.979,0,0,0,0},
		{231.899,231.899,231.899,231.899,251.033,251.033,251.033,251.033,87.548,87.548,241.526,241.526,273.799,273.799,142.59,142.59,149.645,149.645,400.171,400.171,400.171,400.171,365.793,365.793,360.747,360.747,360.747,360.747,150.558,150.558,225.663,225.663,225.663,225.663,194.239,194.239,171.903,171.903,398.483,398.483,150.365,150.365,284.76,284.76,164.325,164.325,160.668,160.668,0,0},
		{231.899,231.899,231.899,231.899,251.033,251.033,251.033,251.033,87.548,87.548,241.526,241.526,273.799,273.799,142.59,142.59,149.645,149.645,400.171,400.171,400.171,400.171,365.793,365.793,360.747,360.747,360.747,360.747,150.558,150.558,225.663,225.663,225.663,225.663,194.239,194.239,171.903,171.903,398.483,398.483,150.365,150.365,284.76,284.76,164.325,164.325,160.668,160.668,0,0}}


	nodePrivate = [...]string{
		"0405aa01a8c2008ad6ee334a686a56b24bb931f4846da500614ead1954185365",
		"8745704296945cf9aa271ed3dd162dc3b1a924be109787bf6093e728d7841714",
		"1ff912a0e784134ba7188ff947e3823580b3a61791471999b9c0a75ccc9d37f2",
		"2b68562e4b71f482f9606f312d946f868b0380618e357f17b56821bb36d51627",
		"7a01e8ae02634b3da2777146b3fa8d877ebc9e2c006d9bcb93b44a96253b0e45",
		"da06d94fde0a0fdff66ebd26460effa7fcf6e52d749bf37771bb32c0f8fd4142",
		"f022fcb4f379e4924d7d81cbc12cdd6dba259fee068af703cf73860c5e2710d4",
		"a4614d2841ac54d988c684b20b08730d41cb4709eb9b2691a552fac6dca58a2e",
		"09e852699fceaea6f7ebd3c7d811e92f834aa836fc3daa3acec1c6b6374cf857",
		"ac137f155433bac8f0ad520d47040e8c12402d6959939a0df36ebf234112d147",
		"851488144ad2b8817b1a874932aa452a3b2df1120d3c89a738a63af3f3140b25",
		"c5b40ce8be43d30f4fd402ecf35d789d8b7210b7125c495a3ec7f93ce4bafe64",
		"e048fbd171d280c78554516dd1e3b99d6af795015db468a359f9ea6207867f4f",
		"5a86f5a8b660a56d0291a903ceb8da4257721b878d1187716556e7d809909f22",
		"33437a14355a845580d11374965496ff570888e11ff3a5dbcfca0d993c33e4fe",
		"e850f2690c7f135384a506f5853a3fbc2e8cdfcdcb5848b4a1950483f0b74870",
		"38d7a8ab54ef27c03aad555ff3537a70c1cef8a6f2e2e4365cec3d6c888ab7c4",
		"d184b3849ae7787b24363112da8e420a2b3278095c0a73a7cfb7322d0d10f47e",
		"fc50a87b69f6634ccffb31d22aac1e888635553b9e4c54f315e4d60a898b8b52",
		"2c503fbf007fb92f488ba48d09b6e4a4a7257167284aac8335a3eadcc1e2a00b",
		"208d65e4c1da5d88432ad1001a7099b47bb0dddd0cd5d68ec69121cfd32b8ada",
		"d6f1139ea15359bff8d9d6328e151628b88205a2846deb230413630bbbae8ee6",
		"bd14182b4dacd8324a0a45dea3c8356f37d652d506d18357f82f00a1093c88ea",
		"2ce0ec65746fe0065b669d6865fae6e43c1f0eb892051083d895ca1e76881b88",
		"b11b88e19aced2030c0ca6ed051d8d43e2b149777c90469029ca8fd001d0974e",
		"62f629264a01dd49bea97f6768b333cec66233105b560e9c20ad48c6b5cd4a37",
		"10ebf43efa8f9bca50e529a9400b0cc56d79d58ef056ba951a4b1972b1656555",
		"12fff497805054a6694d7f2b691ee6c56b6962d399f686b5d0a229f93f50af1a",
		"7eb731ca581f160edb783000fb8f3cbb0cef395a685af058ab68d540beadef51",
		"d681686c5b4030b2029af8c79b9c485aaed30767578a8b981c7b85f65f27005b",
		"0f96b8234de1a8ec866dbc923bcd0dd6438777b4c903abd5fe6868ae26060aeb",
		"f95bc5e942ede472448c6e8e53f68f30fd9f1c1b274d229577e3a3030ac02123",
		"78a6f45fbd05b0ec45ac658c66ac573ce56bab72de0abd6fffccce435eb6d809",
		"d3ec2440e547d0a70f82625e78205cef83b0a46fd8729a564803f960bf3cfdb0",
		"05a999dd124fbd7613d80e76122387584c4a9ba5c35d53c13bb2322ae439cb96",
		"8d35c4eef6f97b6cab059145374c1623538f89322b52b9256cc3ab3dba459e82",
		"c9d086a40f766e6ed4ab012b9d673eed0b7c860a0af4aeaa7072bde182cf7f82",
		"978106c4c2869a971e37f3008292e0e85f054b8482770e7056254b75cd03f79e",
		"d5fadb11d9ee67ea432c258f5f468af688396b3715f7f51ba4310b1e60fc6b52",
		"bd5eece6c9ee72af45f103ed60c63379464ccbe6b8881c0f5d15851a389430ee",
		"58e645e349cfaa9a60a06ec9f9ab49033fe422644adb464436423c3da43fdf75",
		"fe7544e6a7d29931d0fae33f0f42cd14f6bc9e8b66f582603e33d4a4e8b0e618",
		"657692f73421f1fdec6bfea92bf57914250c549c7226b33aaf0b3221c3faf27c",
		"cd1e5e91875ec3ba43f2bf1f274f3fba198e4d8831e4a1056e85f11bfa28e59d",
		"7553710c899243841a601ad0a9621f6e3ac89322fbe07fc36b6f6f08e88b8b9e",
		"cb29622a9e45c9e13d1f12d1c343420e6c33205443504c42403fab51d359d911",
		"43e861b75db6b9fe79aca9bc77c27d3152b5b71b012c816aea70b70832d393b0",
		"b2076cc94003f47a590a967f8969401b00db01d8dd0439cdb268738660b836f0",
		"32ea346e9015a998dd44f27a3de14d9f79e6eccfcd3edbef9c83fc24b1aeb64d",
		"5a4a03bebfada82e828473fc90fa90352cc9bac53f20e2bbc54d26a849a4a35e"}

	mapNodeID = make(map[string]int, 50)
}

// unit: ms
func GetDelay(from, to string) float32 {
	return getDelayByIndex(getIndexByNodeString(from),getIndexByNodeString(to))
}

func GetLocalNodeString() string{
	return localNodeStr
}

func SetLocalNodeString(nodestr string){
	localNodeStr = nodestr
}

func getDelayByIndex(from int, to int) float32 {
	if from == to {
		return 0
	}
	if int(delays[from][to]) == 0 {
		return delays[to][from]
	}
	return delays[from][to]
}

func getIndexByNodeString(nodeString string) int {
	return mapNodeID[nodeString]
}

func buildMapNodeId(){
	//
	for index, value := range nodePrivate{
		mapNodeID[getStrByNodePrivate(value)] = index
	}

	for key, value := range mapNodeID {
		fmt.Printf("key:%s, value:%d\n",key,value)
	}
}

func getStrByNodePrivate(nodePrivate string) string{
	key, _ := hex.DecodeString(nodePrivate)
	privateKey,_:= crypto.ToECDSA(key)
	pub := privateKey.PublicKey
	var id discover.NodeID
	pbytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	if len(pbytes)-1 != len(id) {
		panic(fmt.Errorf("need %d bit pubkey, got %d bits", (len(id)+1)*8, len(pbytes)))
	}
	copy(id[:], pbytes[1:])
	return id.String()
}

func createNodekey(count int) []string{
	ret := make([]string ,0)
	for i:=0; i<count; i++ {
		key,_ := crypto.GenerateKey()
		ret = append(ret, hex.EncodeToString(crypto.FromECDSA(key)))
	}
	return ret
}
