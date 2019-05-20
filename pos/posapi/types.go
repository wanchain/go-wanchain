package posapi

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/core/vm"
)

type Activity struct {
	EpLeader    []common.Address
	EpActivity  []int
	RpLeader    []common.Address
	RpActivity  []int
	SltLeader   []common.Address
	SlBlocks    []int
	SlActivity  float64
	SlCtrlCount int
}

type ValidatorInfo struct {
	Addr       common.Address        `json:"addr"`
	Incentive  *math.HexOrDecimal256 `json:"incentive"`
	Type       string                `json:"type"`
	Delegators []DelegatorInfo       `json:"delegators"`
}
type DelegatorInfo struct {
	Addr      common.Address        `json:"addr"`
	Incentive *math.HexOrDecimal256 `json:"incentive"`
	Type      string                `json:"type"`
}

type ApiClientProbability struct {
	Addr        common.Address
	Probability *math.HexOrDecimal256
}

type ApiStakerInfo struct {
	Addr             common.Address
	Infors           []ApiClientProbability
	FeeRate          uint64
	TotalProbability *math.HexOrDecimal256
}

type ClientInfo struct {
	Address     common.Address
	Amount      *math.HexOrDecimal256
	StakeAmount *math.HexOrDecimal256
	QuitEpoch   uint64
}
type PartnerInfo struct {
	Address      common.Address
	Amount       *math.HexOrDecimal256
	StakeAmount  *math.HexOrDecimal256
	Renewal      bool
	LockEpochs   uint64
	StakingEpoch uint64
}
type StakerJson struct {
	Address   common.Address
	PubSec256 string //stakeholder’s wan public key
	PubBn256  string //stakeholder’s bn256 public key

	Amount         *math.HexOrDecimal256
	StakeAmount    *math.HexOrDecimal256
	LockEpochs     uint64 //lock time which is input by user. 0 means unexpired.
	NextLockEpochs uint64 //lock time which is input by user. 0 means unexpired.
	From           common.Address

	StakingEpoch uint64 //the user’s staking time
	FeeRate      uint64
	//NextFeeRate  uint64
	Clients  []ClientInfo
	Partners []PartnerInfo
}

func ToStakerJson(staker *vm.StakerInfo) *StakerJson {
	stakeJson := StakerJson{}
	stakeJson.Address = staker.Address
	stakeJson.Amount = (*math.HexOrDecimal256)(staker.Amount)
	stakeJson.StakeAmount = (*math.HexOrDecimal256)(staker.StakeAmount)
	stakeJson.LockEpochs = staker.LockEpochs
	stakeJson.NextLockEpochs = staker.NextLockEpochs
	stakeJson.From = staker.From
	stakeJson.StakingEpoch = staker.StakingEpoch
	stakeJson.FeeRate = staker.FeeRate
	stakeJson.Clients = make([]ClientInfo, 0)
	for i := 0; i < len(staker.Clients); i++ {
		c := ClientInfo{
			Address:     staker.Clients[i].Address,
			Amount:      (*math.HexOrDecimal256)(staker.Clients[i].Amount),
			StakeAmount: (*math.HexOrDecimal256)(staker.Clients[i].StakeAmount),
			QuitEpoch:   staker.Clients[i].QuitEpoch,
		}
		stakeJson.Clients = append(stakeJson.Clients, c)
	}
	stakeJson.Partners = make([]PartnerInfo, 0)
	for i := 0; i < len(staker.Partners); i++ {
		p := PartnerInfo{
			Address:      staker.Partners[i].Address,
			Amount:       (*math.HexOrDecimal256)(staker.Partners[i].Amount),
			StakeAmount:  (*math.HexOrDecimal256)(staker.Partners[i].StakeAmount),
			Renewal:      staker.Partners[i].Renewal,
			LockEpochs:   staker.Partners[i].LockEpochs,
			StakingEpoch: staker.Partners[i].StakingEpoch,
		}
		stakeJson.Partners = append(stakeJson.Partners, p)
	}
	stakeJson.PubSec256 = hexutil.Encode(staker.PubSec256)
	stakeJson.PubBn256 = hexutil.Encode(staker.PubBn256)
	return &stakeJson
}
