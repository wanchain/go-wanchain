package posapi

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/pos/epochLeader"
)

type ValidatorActivity struct {
	EpLeader   []common.Address `json:"epLeader"`
	EpActivity []int            `json:"epActivity"`
	RpLeader   []common.Address `json:"rpLeader"`
	RpActivity []int            `json:"rpActivity"`
}

type SlotActivity struct {
	SltLeader   []common.Address `json:"sltLeader"`
	SlBlocks    []int            `json:"slBlocks"`
	SlActivity  float64          `json:"slActivity"`
	SlCtrlCount int              `json:"slCtrlCount"`
}

type Activity struct {
	EpLeader    []common.Address `json:"epLeader"`
	EpActivity  []int            `json:"epActivity"`
	RpLeader    []common.Address `json:"rpLeader"`
	RpActivity  []int            `json:"rpActivity"`
	SltLeader   []common.Address `json:"sltLeader"`
	SlBlocks    []int            `json:"slBlocks"`
	SlActivity  float64          `json:"slActivity"`
	SlCtrlCount int              `json:"slCtrlCount"`
}

type EpRnpActivity struct {
	EpLeader   []common.Address `json:"epLeader"`
	EpActivity []int            `json:"epActivity"`
	RpLeader   []common.Address `json:"rpLeader"`
	RpActivity []int            `json:"rpActivity"`
}

type ValidatorInfo struct {
	Address       common.Address        `json:"address"`
	WalletAddress common.Address        `json:"stakeInFromAddr"`
	Incentive     *math.HexOrDecimal256 `json:"incentive"`
	Type          string                `json:"type"`
	Delegators    []DelegatorInfo       `json:"delegators"`
}
type DelegatorInfo struct {
	Address   common.Address        `json:"address"`
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
	Address     common.Address        `json:"address"`
	Amount      *math.HexOrDecimal256 `json:"amount"`
	StakeAmount *math.HexOrDecimal256 `json:"votingPower"`
	QuitEpoch   uint64                `json:"quitEpoch"`
}
type PartnerInfo struct {
	Address      common.Address        `json:"address"`
	Amount       *math.HexOrDecimal256 `json:"amount"`
	StakeAmount  *math.HexOrDecimal256 `json:"votingPower"`
	Renewal      bool                  `json:"renewal"`
	LockEpochs   uint64                `json:"lockEpochs"`
	StakingEpoch uint64                `json:"stakingEpoch"`
}
type StakerJson struct {
	Address   common.Address `json:"address"`
	PubSec256 string         `json:"pubSec256"` //stakeholder’s wan public key
	PubBn256  string         `json:"pubBn256"`  //stakeholder’s bn256 public key

	Amount         *math.HexOrDecimal256 `json:"amount"`
	StakeAmount    *math.HexOrDecimal256 `json:"votingPower"`
	LockEpochs     uint64                `json:"lockEpochs"`     //lock time which is input by user. 0 means unexpired.
	NextLockEpochs uint64                `json:"nextLockEpochs"` //lock time which is input by user. 0 means unexpired.
	From           common.Address        `json:"from"`

	StakingEpoch uint64 `json:"stakingEpoch"` //the user’s staking time
	FeeRate      uint64 `json:"feeRate"`
	//NextFeeRate  uint64
	Clients  []ClientInfo  `json:"clients"`
	Partners []PartnerInfo `json:"partners"`

	MaxFeeRate          uint64 `json:"maxFeeRate"`
	FeeRateChangedEpoch uint64 `json:"feeRateChangedEpoch"`
}

type RefundInfo struct {
	Addr   common.Address        `json:"address"`
	Amount *math.HexOrDecimal256 `json:"amount"`
}

func convertReundInfo(info []epochLeader.RefundInfo) []RefundInfo {
	refund := make([]RefundInfo, 0)
	for i := 0; i < len(info); i++ {
		record := RefundInfo{
			Addr:   info[i].Addr,
			Amount: (*math.HexOrDecimal256)(info[i].Amount),
		}
		refund = append(refund, record)
	}
	return refund
}

type PosInfoJson struct {
	FirstEpochId     uint64 `json:"firstEpochId"`
	FirstBlockNumber uint64 `json:"firstBlockNumber"`
}
type LeaderJson struct {
	Type      uint8          `json:"type"`
	SecAddr   common.Address `json:"secAddr"`
	PubSec256 string         `json:"pubSec256"`
	PubBn256  string         `json:"pubBn256"`
}

func ToLeaderJson(leader []vm.Leader) []LeaderJson {
	lj := make([]LeaderJson, len(leader))
	for i := 0; i < len(leader); i++ {
		lj[i].Type = leader[i].Type
		lj[i].SecAddr = leader[i].SecAddr
		lj[i].PubSec256 = hexutil.Encode(leader[i].PubSec256)
		lj[i].PubBn256 = hexutil.Encode(leader[i].PubBn256)
	}
	return lj
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

	stakeJson.MaxFeeRate = uint64(0)

	return &stakeJson
}
