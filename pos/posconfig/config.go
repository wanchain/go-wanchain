package posconfig

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	bn256 "github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"

	"github.com/wanchain/go-wanchain/node"
)

var (
	// EpochBaseTime is the pos start time such as: 2018-12-12 00:00:00 == 1544544000
	//EpochBaseTime = uint64(0)
	FirstEpochId              = uint64(0)
	CurrentEpochId            = uint64(0)
	Pow2PosUpgradeBlockNumber = uint64(0)
	// SelfTestMode config whether it is in a simlate tese mode
	SelfTestMode = false
	IsDev        = false
	MineEnabled  = false
)

const (
	RbLocalDB        = "rblocaldb"
	EpLocalDB        = "eplocaldb"
	StakerLocalDB    = "stlocaldb"
	PosLocalDB       = "pos"
	IncentiveLocalDB = "incentive"
	ReorgLocalDB     = "forkdb"
	ApolloEpochID     = 18104
	AugustEpochID     = 18116  //TODO change it as mainnet 8.8

	TestnetAdditionalBlock = 6661460
)

var EpochLeadersHold [][]byte
var TestnetAdditionalValue = new(big.Int).Mul(big.NewInt(210000000),big.NewInt(1e18))

const (
	// EpochLeaderCount is count of pk in epoch leader group which is select by stake
	EpochLeaderCount = 50
	// RandomProperCount is count of pk in random leader group which is select by stake
	RandomProperCount = 25
	PosUpgradeEpochID = 2 // must send tx 2 epoch before.
	MaxEpHold         = 30
	MinEpHold         = 0
	Key3Suffix        = "bn256KeySuffix"
	StakeOutEpochKey  = "StakeOutEpochKey"
)
const (
	// SlotTime is the time span of a slot in second, So it's 1 hours for a epoch
	SlotTime = 5

	//Incentive should perform delay some epochs.
	IncentiveDelayEpochs = 1
	IncentiveStartStage  = Stage2K

	// K count of each epoch
	KCount = 12
	K      = 1440

	// SlotCount is slot count in an epoch
	SlotCount = K * KCount

	// Stage1K is divde a epoch into 10 pieces
	Stage1K  = uint64(K)
	Stage2K  = Stage1K * 2
	Stage3K  = Stage1K * 3
	Stage4K  = Stage1K * 4
	Stage5K  = Stage1K * 5
	Stage6K  = Stage1K * 6
	Stage7K  = Stage1K * 7
	Stage8K  = Stage1K * 8
	Stage9K  = Stage1K * 9
	Stage10K = Stage1K * 10
	Stage11K = Stage1K * 11
	Stage12K = Stage1K * 12

	Sma1Start = Stage2K
	Sma1End   = Stage4K
	Sma2Start = Stage6K
	Sma2End   = Stage8K
	Sma3Start = Stage10K
	Sma3End   = Stage12K

	// parameters for security and chain quality
	BlockSecurityParam = K
	SlotSecurityParam  = 2 * K

	MinimumChainQuality     = 0.5 //BlockSecurityParam / SlotSecurityParam
	CriticalReorgThreshold  = 3
	CriticalChainQuality    = 0.618
	NonCriticalChainQuality = 0.8

	MainnetMercuryEpochId = 18250 //2019.12.20
	TestnetMercuryEpochId = 18246 //2019.12.16

	MainnetVenusEpochId = 11112222
	TestnetVenusEpochId = 18369
)

var TxDelay = K

var GenesisPK string

//var GenesisPK = "04dc40d03866f7335e40084e39c3446fe676b021d1fcead11f2e2715e10a399b498e8875d348ee40358545e262994318e4dcadbc865bcf9aac1fc330f22ae2c786"
//var GenesisPKInit = "04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70"
//var GenesisPK = "046a5e1d2b8ca62accede9b8c7995dbd428ddbaf6a7f85673d426038b05bfdb428681046930a27b849a8f3541e71e8779948df95c78b2b303380769d0f4e8a753e"
var GenesisPKInit = ""
var PosOwnerAddr common.Address
var WhiteList [210]string

type Config struct {
	PolymDegree   uint
	K             uint
	RBThres       uint
	EpochInterval uint64
	PosStartTime  int64
	MinerKey      *keystore.Key
	Dbpath        string
	NodeCfg       *node.Config
	Dkg1End       uint64
	Dkg2Begin     uint64
	Dkg2End       uint64
	SignBegin     uint64
	SignEnd       uint64

	MercuryEpochId uint64
	VenusEpochId uint64
	DefaultGasPrice	 *big.Int

	SyncTargetBlokcNum uint64
}

var DefaultConfig = Config{
	12,
	K,
	13,
	0,
	0,
	nil,
	"",
	nil,
	Stage2K - 1,
	Stage4K,
	Stage6K - 1,
	Stage8K,
	Stage10K - 1,
	0,
	0,

	nil,

	0,
}

func Cfg() *Config {
	return &DefaultConfig
}

func (c *Config) GetMinerAddr() common.Address {
	if c.MinerKey == nil {
		return common.Address{}
	}

	return c.MinerKey.Address
}

func (c *Config) GetMinerBn256PK() *bn256.G1 {
	D3 := GenerateD3byKey2(c.MinerKey.PrivateKey2)
	return new(bn256.G1).ScalarBaseMult(D3)
}

func GenerateD3byKey2(PrivateKey *ecdsa.PrivateKey) *big.Int {
	var one = new(big.Int).SetInt64(1)
	params := crypto.S256().Params()
	var ebuffer bytes.Buffer
	ebuffer.Write(PrivateKey.D.Bytes())
	ebuffer.Write(([]byte)(Key3Suffix))

	ebyte := crypto.Keccak256(ebuffer.Bytes())
	d3 := new(big.Int).SetBytes(ebyte)

	n := new(big.Int).Sub(params.N, one)
	d3.Mod(d3, n)
	d3.Add(d3, one)
	return d3
}

func (c *Config) GetMinerBn256SK() *big.Int {
	return GenerateD3byKey2(c.MinerKey.PrivateKey2)
}

func Init(nodeCfg *node.Config, networkId uint64) {
	if networkId == 1 {
		// this is mainnet. *****
		WhiteList = WhiteListMainnet
		PosOwnerAddr = PosOwnerAddrMainnet

		DefaultConfig.MercuryEpochId = MainnetMercuryEpochId
		DefaultConfig.VenusEpochId   = MainnetVenusEpochId

	} else if networkId == 6 {
		PosOwnerAddr = PosOwnerAddrInternal
		if IsDev { // --plutodev
			WhiteList = WhiteListDev // only one whiteAccount, used as single node.
		} else {
			WhiteList = WhiteListOrig
		}
		DefaultConfig.MercuryEpochId = TestnetMercuryEpochId
		DefaultConfig.VenusEpochId   = TestnetVenusEpochId
	} else if networkId == 4 {
		PosOwnerAddr = PosOwnerAddrInternal
		WhiteList = WhiteListOrig
		DefaultConfig.MercuryEpochId = TestnetMercuryEpochId
		DefaultConfig.VenusEpochId   = TestnetVenusEpochId
	} else { // testnet
		PosOwnerAddr = PosOwnerAddrTestnet
		WhiteList = WhiteListTestnet

		DefaultConfig.MercuryEpochId = TestnetMercuryEpochId
		DefaultConfig.VenusEpochId = TestnetVenusEpochId
	}

	EpochLeadersHold = make([][]byte, len(WhiteList))
	for i := 0; i < len(WhiteList); i++ {
		EpochLeadersHold[i] = hexutil.MustDecode(WhiteList[i])
	}
	DefaultConfig.NodeCfg = nodeCfg
}

func GetRandomGenesis() *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256(big.NewInt(1).Bytes()))
}
