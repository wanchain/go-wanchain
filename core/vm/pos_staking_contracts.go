package vm

import (
	"crypto/ecdsa"
	"errors" // this is not match with other
	"github.com/wanchain/go-wanchain/params"
	"math/big"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
)

/* the contract interface described by solidity.

contract stake {
	function stakeIn(bytes memory secPk, bytes memory bn256Pk, uint256 lockEpochs, uint256 feeRate) public payable {}
	function stakeUpdate(address addr, uint256 lockEpochs) public {}
	function stakeUpdateFeeRate(address addr, uint256 feeRate) public {}
	function stakeAppend(address addr) public payable {}
	function partnerIn(address addr, bool renewal) public payable {}
	function delegateIn(address delegateAddress) public payable {}
	function delegateOut(address delegateAddress) public {}

	event stakeIn(address indexed sender, address indexed posAddress, uint indexed value, uint256 feeRate, uint256 lockEpoch);
	event stakeAppend(address indexed sender, address indexed posAddress, uint indexed value);
	event stakeUpdate(address indexed sender, address indexed posAddress, uint indexed lockEpoch);
	event delegateIn(address indexed sender, address indexed posAddress, uint indexed value);
	event delegateOut(address indexed sender, address indexed posAddress);
	event stakeUpdateFeeRate(address indexed sender, address indexed posAddress, uint indexed feeRate);
	event partnerIn(address indexed sender, address indexed posAddress, uint indexed value, bool renewal);
}

*/
const (
	PSMinEpochNum = 7
	PSMaxEpochNum = 90

	PSMaxStake            = 10500000
	PSMinStakeholderStake = 10000
	PSMinValidatorStake   = 50000
	PSMinDelegatorStake   = 100
	PSMinFeeRate          = 0
	PSMaxFeeRate          = 10000
	PSFeeRateStep		  = 100
	PSNodeleFeeRate       = 10000
	PSMinPartnerIn        = 10000
	MaxTimeDelegate       = 10
	UpdateDelay           = 3
	QuitDelay             = 3
	JoinDelay             = 2
	PSOutKeyHash          = 700
	maxPartners           = 5
)

var (
	// pos staking contract abi definition
	cscDefinition = `
[
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "stakeAppend",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			}
		],
		"name": "stakeUpdate",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "secPk",
				"type": "bytes"
			},
			{
				"name": "bn256Pk",
				"type": "bytes"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			},
			{
				"name": "feeRate",
				"type": "uint256"
			}
		],
		"name": "stakeIn",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "secPk",
				"type": "bytes"
			},
			{
				"name": "bn256Pk",
				"type": "bytes"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			},
			{
				"name": "feeRate",
				"type": "uint256"
			},
			{
				"name": "maxFeeRate",
				"type": "uint256"
			}
		],
		"name": "stakeRegister",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			},
			{
				"name": "renewal",
				"type": "bool"
			}
		],
		"name": "partnerIn",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "delegateAddress",
				"type": "address"
			}
		],
		"name": "delegateIn",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "delegateAddress",
				"type": "address"
			}
		],
		"name": "delegateOut",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			},
			{
				"name": "feeRate",
				"type": "uint256"
			}
		],
		"name": "stakeUpdateFeeRate",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "feeRate",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "lockEpoch",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "maxFeeRate",
				"type": "uint256"
			}
		],
		"name": "stakeRegister",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "feeRate",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "lockEpoch",
				"type": "uint256"
			}
		],
		"name": "stakeIn",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			}
		],
		"name": "stakeAppend",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "lockEpoch",
				"type": "uint256"
			}
		],
		"name": "stakeUpdate",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "renewal",
				"type": "bool"
			}
		],
		"name": "partnerIn",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			}
		],
		"name": "delegateIn",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			}
		],
		"name": "delegateOut",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "feeRate",
				"type": "uint256"
			}
		],
		"name": "stakeUpdateFeeRate",
		"type": "event"
	}
]
`
	// pos staking contract abi object
	cscAbi, errCscInit = abi.JSON(strings.NewReader(cscDefinition))

	// function "stakeIn" "delegateIn" 's solidity binary id
	stakeRegisterId [4]byte
	stakeInId     [4]byte
	stakeUpdateId [4]byte
	stakeAppendId [4]byte
	partnerInId   [4]byte
	delegateInId  [4]byte
	delegateOutId [4]byte
	stakeUpdateFeeRateId [4]byte

	maxEpochNum                = big.NewInt(PSMaxEpochNum)
	minEpochNum                = big.NewInt(PSMinEpochNum)
	minStakeholderStake        = new(big.Int).Mul(big.NewInt(PSMinStakeholderStake), ether)
	MinValidatorStake          = new(big.Int).Mul(big.NewInt(PSMinValidatorStake), ether)
	minDelegatorStake          = new(big.Int).Mul(big.NewInt(PSMinDelegatorStake), ether)
	maxTotalStake              = new(big.Int).Mul(big.NewInt(PSMaxStake), ether)
	minFeeRate                 = big.NewInt(PSMinFeeRate)
	maxFeeRate                 = big.NewInt(PSMaxFeeRate)
	minPartnerIn               = new(big.Int).Mul(big.NewInt(PSMinPartnerIn), ether)
	noDelegateFeeRate          = big.NewInt(PSNodeleFeeRate)
	StakersInfoStakeOutKeyHash = common.BytesToHash(big.NewInt(PSOutKeyHash).Bytes())
)

//
// param structures
//
type StakeInParam struct {
	SecPk      []byte   //stakeholder’s original public key
	Bn256Pk    []byte   //stakeholder’s bn256 pairing public key
	LockEpochs *big.Int //lock time which is input by user
	FeeRate    *big.Int
	pub        *ecdsa.PublicKey
}
type StakeRegisterParam struct {
	StakeInParam
	MaxFeeRate *big.Int
}
type StakeUpdateParam struct {
	Addr       common.Address //stakeholder’s bn256 pairing public key
	LockEpochs *big.Int       //lock time which is input by user
}
type PartnerInParam struct {
	Addr    common.Address //stakeholder’s bn256 pairing public key
	Renewal bool
}
type DelegateParam struct {
	DelegateAddress common.Address //delegation’s address
}
type UpdateFeeRateParam struct {
	Addr common.Address
	FeeRate *big.Int
}

//
// storage structures
//
type StakerInfo struct {
	Address   common.Address
	PubSec256 []byte //stakeholder’s wan public key
	PubBn256  []byte //stakeholder’s bn256 public key

	Amount         *big.Int //staking wan value
	StakeAmount    *big.Int //staking wan value
	LockEpochs     uint64   //lock time which is input by user. 0 means unexpired.
	NextLockEpochs uint64   //lock time which is input by user. 0 means unexpired.
	From           common.Address

	StakingEpoch uint64 //the first epoch in which stakerHolder might be selected.
	FeeRate      uint64
	//NextFeeRate  uint64
	Clients  []ClientInfo
	Partners []PartnerInfo
}

type ValidatorInfo struct {
	TotalProbability *big.Int
	FeeRate          uint64
	ValidatorAddr    common.Address
	WalletAddr       common.Address
	Infos            []ClientProbability // the position 0 is validator and others is delegators.
}
type ClientInfo struct {
	Address     common.Address
	Amount      *big.Int
	StakeAmount *big.Int //staking wan value
	QuitEpoch   uint64
}
type PartnerInfo struct {
	Address      common.Address
	Amount       *big.Int
	StakeAmount  *big.Int //staking wan value
	Renewal      bool
	LockEpochs   uint64
	StakingEpoch uint64
}

type UpdateFeeRate struct {
	ValidatorAddr    common.Address
	MaxFeeRate uint64
	FeeRate uint64
	ChangedEpoch uint64
}
//
// public helper structures
//
type Leader struct {
	Type      uint8          `json:"type"`
	SecAddr   common.Address `json:"secAddr"`
	PubSec256 []byte         `json:"pubSec256"`
	PubBn256  []byte         `json:"pubBn256"`
}

type ClientProbability struct {
	ValidatorAddr common.Address
	WalletAddr    common.Address
	Probability   *big.Int
}

type ClientIncentive struct {
	ValidatorAddr common.Address
	WalletAddr    common.Address
	Incentive     *big.Int
}

//
// package initialize
//
func init() {
	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(stakeRegisterId[:], cscAbi.Methods["stakeRegister"].Id())
	copy(stakeInId[:], cscAbi.Methods["stakeIn"].Id())
	copy(stakeAppendId[:], cscAbi.Methods["stakeAppend"].Id())
	copy(stakeUpdateId[:], cscAbi.Methods["stakeUpdate"].Id())
	copy(partnerInId[:], cscAbi.Methods["partnerIn"].Id())
	copy(delegateInId[:], cscAbi.Methods["delegateIn"].Id())
	copy(delegateOutId[:], cscAbi.Methods["delegateOut"].Id())
	copy(stakeUpdateFeeRateId[:], cscAbi.Methods["stakeUpdateFeeRate"].Id())
}

/////////////////////////////
//
// pos staking contract
//
type PosStaking struct {
}

//
// contract interfaces
//
func (p *PosStaking) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *PosStaking) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(input) < 4 {
		return nil, errors.New("parameter is wrong")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeRegisterId {
		return p.StakeRegister(input[4:], contract, evm)
	} else if methodId == stakeInId {
		ret, err := p.StakeIn(input[4:], contract, evm)
		if err != nil {
			log.Info("stakein failed", "err", err)
		}
		return ret, err
	} else if methodId == stakeUpdateId {
		return p.StakeUpdate(input[4:], contract, evm)
	} else if methodId == stakeAppendId {
		return p.StakeAppend(input[4:], contract, evm)
	} else if methodId == partnerInId {
		return p.PartnerIn(input[4:], contract, evm)
	} else if methodId == delegateInId {
		return p.DelegateIn(input[4:], contract, evm)
	} else if methodId == delegateOutId {
		return p.DelegateOut(input[4:], contract, evm)
	} else if methodId == stakeUpdateFeeRateId {
		return p.StakeUpdateFeeRate(input[4:], contract, evm)
	}
	return nil, errMethodId
}

func (p *PosStaking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	input := tx.Data()
	if len(input) < 4 {
		return errors.New("parameter is too short")
	}

	if params.IsNoStaking() {
		return errors.New("noStaking specified")
	}
	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeRegisterId {
		eidNow, _ := util.CalEpochSlotID(uint64(time.Now().Unix()))
		if eidNow < posconfig.ApolloEpochID {
			return  errors.New("stakeRegister haven't enabled.")
		}
		_, err := p.stakeRegisterParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakeRegister verify failed")
		}
		return nil
	} else if methodId == stakeInId {
		_, err := p.stakeInParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakein verify failed")
		}
		return nil
	} else if methodId == stakeAppendId {
		_, err := p.stakeAppendParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakeAppend verify failed " + err.Error())
		}
		return nil
	} else if methodId == stakeUpdateId {
		_, err := p.stakeUpdateParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakeUpdate verify failed " + err.Error())
		}
		return nil
	} else if methodId == partnerInId {
		_, err := p.partnerInParseAndValid(input[4:])
		if err != nil {
			return errors.New("partnerIn verify failed " + err.Error())
		}
		return nil
	} else if methodId == delegateInId {
		_, err := p.delegateInParseAndValid(input[4:])
		if err != nil {
			return errors.New("delegateIn verify failed")
		}
		return nil
	} else if methodId == delegateOutId {
		_, err := p.delegateOutParseAndValid(input[4:])
		if err != nil {
			return errors.New("delegateOut verify failed")
		}
		return nil
	} else if methodId == stakeUpdateFeeRateId {
		eidNow, _ := util.CalEpochSlotID(uint64(time.Now().Unix()))
		if eidNow < posconfig.ApolloEpochID {
			return  errors.New("stakeUpdateFeeRateId haven't enabled.")
		}
		_, err := p.updateFeeRateParseAndValid(input[4:])
		if err != nil {
			return errors.New("update fee rate verify failed")
		}
		return nil
	}

	return errParameters
}

func (p *PosStaking) saveStakeInfo(evm *EVM, stakerInfo *StakerInfo) error {
	infoBytes, err := rlp.EncodeToBytes(stakerInfo)
	if err != nil {
		return err
	}
	key := GetStakeInKeyHash(stakerInfo.Address)
	res := StoreInfo(evm.StateDB, StakersInfoAddr, key, infoBytes)
	if res != nil {
		return res
	}
	return nil
}
func (p *PosStaking) getStakeInfo(evm *EVM, addr common.Address) (*StakerInfo, error) {
	key := GetStakeInKeyHash(addr)
	stakerBytes, err := GetInfo(evm.StateDB, StakersInfoAddr, key)
	if stakerBytes == nil {
		return nil, errors.New("item doesn't exist")
	}
	var stakerInfo StakerInfo
	err = rlp.DecodeBytes(stakerBytes, &stakerInfo)
	if err != nil {
		return nil, errors.New("parse staker info error")
	}
	return &stakerInfo, nil
}


func (p *PosStaking) getStakeFeeRate(evm *EVM, address common.Address) (*UpdateFeeRate, error) {
	key := GetStakeInKeyHash(address)
	feeBytes, err := GetInfo(evm.StateDB, StakersFeeAddr, key)
	if err != nil {
		return nil, err
	}
	if feeBytes == nil {
		return nil, nil
	}
	var feeRate UpdateFeeRate
	err = rlp.DecodeBytes(feeBytes, &feeRate)
	if err != nil {
		return nil, err
	}
	return &feeRate, nil
}

func (p *PosStaking) saveStakeFeeRate(evm *EVM, feeRate *UpdateFeeRate, address common.Address) error {
	feeBytes, err := rlp.EncodeToBytes(feeRate)
	if err != nil {
		return err
	}
	key := GetStakeInKeyHash(address)
	err = StoreInfo(evm.StateDB, StakersFeeAddr, key, feeBytes)
	if err != nil {
		return err
	}
	return nil
}

func (p *PosStaking) StakeUpdate(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	info, err := p.stakeUpdateParseAndValid(payload)
	if err != nil {
		return nil, err
	}

	stakerInfo, err := p.getStakeInfo(evm, info.Addr)
	if err != nil {
		return nil, err
	}
	if contract.CallerAddress != stakerInfo.From {
		return nil, errors.New("Cannot update from another account")
	}

	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eidNow > stakerInfo.StakingEpoch+stakerInfo.LockEpochs-UpdateDelay {
		return nil, errors.New("cannot change at the last 3 epoch.")
	}

	stakerInfo.NextLockEpochs = info.LockEpochs.Uint64()
	err = p.saveStakeInfo(evm, stakerInfo)
	if err != nil {
		return nil, err
	}
	p.stakeUpdateLog(contract, evm, stakerInfo)
	return nil, nil
}

func (p *PosStaking) PartnerIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	info, err := p.partnerInParseAndValid(payload)
	if err != nil {
		return nil, err
	}

	stakerInfo, err := p.getStakeInfo(evm, info.Addr)
	if err != nil {
		return nil, err
	}

	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eidNow >= posconfig.ApolloEpochID &&  eidNow < posconfig.AugustEpochID{
		if contract.Value().Cmp(minPartnerIn) < 0 {
			return nil, errors.New("min wan amount should >= 10000")
		}
	}

	realLockEpoch := int64(stakerInfo.LockEpochs - (eidNow + JoinDelay - stakerInfo.StakingEpoch))
	if stakerInfo.StakingEpoch == 0 {
		realLockEpoch = int64(stakerInfo.LockEpochs)
	}
	if realLockEpoch < 0 || realLockEpoch > PSMaxEpochNum {
		return nil, errors.New("Wrong lock Epochs")
	}
	weight := CalLocktimeWeight(uint64(realLockEpoch))

	total := big.NewInt(0).Set(stakerInfo.Amount)
	for i := 0; i < len(stakerInfo.Clients); i++ {
		total.Add(total, stakerInfo.Clients[i].Amount)
	}
	total.Add(total, contract.Value())
	length := len(stakerInfo.Partners)
	found := false
	var partner *PartnerInfo = nil
	for i := 0; i < length; i++ {
		total.Add(total, stakerInfo.Partners[i].Amount)
		if stakerInfo.Partners[i].Address == contract.CallerAddress {
			partner = &stakerInfo.Partners[i]
			partner.Amount.Add(partner.Amount, contract.Value())
			partner.StakeAmount.Add(partner.StakeAmount, big.NewInt(0).Mul(contract.Value(), big.NewInt(int64(weight))))
			partner.Renewal = info.Renewal
			found = true
		}
	}
	// check stake + partner + delegate <= 10,500,000
	if total.Cmp(maxTotalStake) > 0 {
		return nil, errors.New("partner in failed, too much stake")
	}
	if found == false {
		if length >= maxPartners {
			return nil, errors.New("Too many partners")
		}

		if eidNow >= posconfig.ApolloEpochID {
			if contract.Value().Cmp(minPartnerIn) < 0 {
				return nil, errors.New("min wan amount should >= 10000")
			}
		}
		partner = &PartnerInfo{
			Address:      contract.CallerAddress,
			Amount:       contract.Value(),
			Renewal:      info.Renewal,
			StakingEpoch: eidNow + JoinDelay,
			LockEpochs:   uint64(realLockEpoch),
		}
		if posconfig.FirstEpochId == 0 {
			partner.StakingEpoch = 0
		}
		partner.StakeAmount = big.NewInt(0).Mul(partner.Amount, big.NewInt(int64(weight)))
		stakerInfo.Partners = append(stakerInfo.Partners, *partner)
	}

	err = p.saveStakeInfo(evm, stakerInfo)
	if err != nil {
		return nil, err
	}
	if partner != nil {
		err = p.partnerInLog(contract, evm, &info.Addr, info.Renewal)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
func (p *PosStaking) StakeAppend(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	addr, err := p.stakeAppendParseAndValid(payload)
	if err != nil {
		return nil, err
	}

	stakerInfo, err := p.getStakeInfo(evm, addr)
	if err != nil {
		return nil, err
	}
	if contract.CallerAddress != stakerInfo.From {
		return nil, errors.New("Cannot append from another account")
	}

	// add origen Amount
	stakerInfo.Amount.Add(stakerInfo.Amount, contract.Value())
	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())
	realLockEpoch := int64(stakerInfo.LockEpochs - (eidNow + JoinDelay - stakerInfo.StakingEpoch))
	if stakerInfo.StakingEpoch == 0 {
		realLockEpoch = int64(stakerInfo.LockEpochs)
	}
	if realLockEpoch < 0 || realLockEpoch > PSMaxEpochNum {
		return nil, errors.New("Wrong lock Epochs")
	}

	total := big.NewInt(0).Set(stakerInfo.Amount)
	for i := 0; i < len(stakerInfo.Clients); i++ {
		total.Add(total, stakerInfo.Clients[i].Amount)
	}
	for i := 0; i < len(stakerInfo.Partners); i++ {
		total.Add(total, stakerInfo.Partners[i].Amount)
	}
	// check stake + partner + delegate <= 10,500,000
	if total.Cmp(maxTotalStake) > 0 {
		return nil, errors.New("StakeAppend in failed, too much stake")
	}

	weight := CalLocktimeWeight(uint64(realLockEpoch))
	epochid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if epochid < posconfig.Cfg().VenusEpochId {
		stakerInfo.StakeAmount.Mul(stakerInfo.Amount, big.NewInt(int64(weight)))
	} else {
		stakerInfo.StakeAmount.Add(stakerInfo.StakeAmount, big.NewInt(0).Mul(contract.Value(), big.NewInt(int64(weight))))
	}
	err = p.saveStakeInfo(evm, stakerInfo)
	if err != nil {
		return nil, err
	}
	p.stakeAppendLog(contract, evm, stakerInfo.Address)
	return nil, nil
}

func (p *PosStaking) StakeRegister(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	info, err := p.stakeRegisterParseAndValid(payload)
	if err != nil {
		return nil, err
	}

	stakerInfo, err := p.doStakeIn(contract, evm, info.StakeInParam)
	if err != nil {
		return nil, err
	}

	maxFeeRate := info.MaxFeeRate.Uint64()
	feeUpdate := &UpdateFeeRate{
		ValidatorAddr: stakerInfo.Address,
		MaxFeeRate: maxFeeRate,
		FeeRate: stakerInfo.FeeRate,
		ChangedEpoch: uint64(0),
	}
	err = p.saveStakeFeeRate(evm, feeUpdate, stakerInfo.Address)
	if err != nil {
		return nil, err
	}
	err = p.stakeRegisterLog(contract, evm, stakerInfo, maxFeeRate)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (p *PosStaking) StakeIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	info, err := p.stakeInParseAndValid(payload)
	if err != nil {
		return nil, err
	}

	stakerInfo, err := p.doStakeIn(contract, evm, info)
	if err != nil {
		return nil, err
	}
	err = p.stakeInLog(contract, evm, stakerInfo)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
func (p *PosStaking) doStakeIn(contract *Contract, evm *EVM, info StakeInParam) (*StakerInfo, error) {
	// no max limit
	//  amount >= PSMinStakeholderStake,
	if contract.value.Cmp(minStakeholderStake) < 0 {
		return nil, errors.New("need more Wan to be a stake holder")
	}
	// TODO: or return value - 10,500,000 to the sender?
	if contract.value.Cmp(maxTotalStake) > 0 {
		return nil, errors.New("max stake is 10,500,000")
	}

	// NOTE: if a validator has no MinValidatorStake, but want delegate, he can partnerIn or stakeAppend later.
	// SO, don't need all in the first stakeIn.
	//if info.FeeRate.Cmp(noDelegateFeeRate) != 0 &&  contract.value.Cmp(MinValidatorStake) < 0 {
	//	return nil, errors.New("need more Wan to be a validator")
	//}
	secAddr := crypto.PubkeyToAddress(*(info.pub))

	// 6. secAddr has not join the pos or has finished
	key := GetStakeInKeyHash(secAddr)
	oldInfo, err := GetInfo(evm.StateDB, StakersInfoAddr, key)
	// a. is secAddr joined?
	if oldInfo != nil {
		return nil, errors.New("public Sec address has exist")
	}

	// create stakeholder's information
	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())
	weight := CalLocktimeWeight(info.LockEpochs.Uint64())
	stakerInfo := &StakerInfo{
		Address:        secAddr,
		PubSec256:      info.SecPk,
		PubBn256:       info.Bn256Pk,
		Amount:         contract.value,
		LockEpochs:     info.LockEpochs.Uint64(),
		FeeRate:        info.FeeRate.Uint64(),
		NextLockEpochs: info.LockEpochs.Uint64(),
		//NextFeeRate:      info.FeeRate.Uint64(),
		From:         contract.CallerAddress,
		StakingEpoch: eidNow + JoinDelay,
	}
	if posconfig.FirstEpochId == 0 {
		stakerInfo.StakingEpoch = 0
	}
	stakerInfo.StakeAmount = big.NewInt(0).Mul(stakerInfo.Amount, big.NewInt(int64(weight)))
	err = p.saveStakeInfo(evm, stakerInfo)
	if err != nil {
		return nil, err
	}
	return stakerInfo, nil
}

// one wants to choose a delegation to join the pos
func (p *PosStaking) DelegateIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	addr, err := p.delegateInParseAndValid(payload)
	if err != nil {
		return nil, err
	}

	stakerInfo, err := p.getStakeInfo(evm, addr)
	if err != nil {
		return nil, err
	}
	// check if the validator's feeRate is 100, can't delegatein
	if stakerInfo.FeeRate == noDelegateFeeRate.Uint64() {
		return nil, errors.New("Validator don't accept delegation.")
	}
	// check if the validator's amount(include partner) is not enough, can't delegatein
	total := big.NewInt(0).Set(stakerInfo.Amount)
	for i := 0; i < len(stakerInfo.Partners); i++ {
		total.Add(total, stakerInfo.Partners[i].Amount)
	}
	if total.Cmp(MinValidatorStake) < 0 {
		return nil, errors.New("Validator don't have enough amount.")
	}
	//  sender has not delegated by this
	var info *ClientInfo
	length := len(stakerInfo.Clients)

	totalDelegated := big.NewInt(0)
	totalDelegated.Add(totalDelegated, contract.Value())
	for i := 0; i < length; i++ {
		totalDelegated.Add(totalDelegated, stakerInfo.Clients[i].Amount)
		if stakerInfo.Clients[i].Address == contract.CallerAddress {
			if stakerInfo.Clients[i].QuitEpoch != 0 {
				return nil, errors.New("dalegater is quiting.")
			}
			weight := CalLocktimeWeight(PSMinEpochNum)
			info = &stakerInfo.Clients[i]
			info.Amount.Add(info.Amount, contract.Value())
			info.StakeAmount.Add(info.StakeAmount, big.NewInt(0).Mul(contract.Value(), big.NewInt(int64(weight))))
		}
	}
	// check self + partner + delegate <= 10,500,000
	if new(big.Int).Add(totalDelegated, total).Cmp(maxTotalStake) > 0 {
		return nil, errors.New("delegate over total stake limitation")
	}
	// check the totalDelegated <= 10*stakerInfo.Amount
	if totalDelegated.Cmp(big.NewInt(0).Mul(total, big.NewInt(MaxTimeDelegate))) > 0 {
		return nil, errors.New("over delegate limitation")
	}
	if info == nil {
		// only first delegatein check amount is valid.
		if contract.value.Cmp(minDelegatorStake) < 0 {
			return nil, errors.New("low amount")
		}
		// save
		weight := CalLocktimeWeight(PSMinEpochNum)
		info := &ClientInfo{
			Address:     contract.CallerAddress,
			Amount:      contract.value,
			StakeAmount: big.NewInt(0).Mul(contract.value, big.NewInt(int64(weight))),
			QuitEpoch:   0,
		}
		stakerInfo.Clients = append(stakerInfo.Clients, *info)
	}

	err = p.saveStakeInfo(evm, stakerInfo)
	if err != nil {
		return nil, err
	}
	p.delegateInLog(contract, evm, stakerInfo.Address)
	return nil, nil
}

func (p *PosStaking) DelegateOut(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	addr, err := p.delegateOutParseAndValid(payload)
	if err != nil {
		return nil, err
	}
	stakerInfo, err := p.getStakeInfo(evm, addr)
	if err != nil {
		return nil, err
	}

	length := len(stakerInfo.Clients)
	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())

	found := false
	for i := 0; i < length; i++ {
		if stakerInfo.Clients[i].Address == contract.CallerAddress {
			// check if delegater has existed.
			if stakerInfo.Clients[i].QuitEpoch != 0 {
				return nil, errors.New("delegator has existed")
			}
			stakerInfo.Clients[i].QuitEpoch = eidNow + QuitDelay
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("item doesn't exist")
	}

	err = p.saveStakeInfo(evm, stakerInfo)
	if err != nil {
		return nil, err
	}
	p.delegateOutLog(contract, evm, stakerInfo.Address)
	return nil, nil
}



func (p *PosStaking) StakeUpdateFeeRate(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	feeRateParam, err := p.updateFeeRateParseAndValid(payload)
	if err != nil {
		return nil, err
	}
	stakeInfo, err := p.getStakeInfo(evm, feeRateParam.Addr)
	if err != nil {
		return nil, err
	}

	// if feeRate == 10000, can't change
	if stakeInfo.FeeRate == PSMaxFeeRate || feeRateParam.FeeRate.Uint64() == PSMaxFeeRate  {
		return nil, errors.New("feeRate equal 10000, can't change")
	}

	if stakeInfo.FeeRate == feeRateParam.FeeRate.Uint64() {
		return nil, errors.New("feeRate already same")
	}

	if contract.CallerAddress != stakeInfo.From {
		return nil, errors.New("cannot update fee from another account")
	}

	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	oldFee, err := p.getStakeFeeRate(evm, stakeInfo.Address)
	if err != nil {
		return nil, err
	}

	if oldFee == nil {
		oldFee = &UpdateFeeRate{
			ValidatorAddr: stakeInfo.Address,
			MaxFeeRate: stakeInfo.FeeRate,
			FeeRate: stakeInfo.FeeRate,
			ChangedEpoch: eid,
		}
	} else if oldFee.ChangedEpoch == eid {
		return nil, errors.New("one epoch can only change one time")
	}

	feeRate := feeRateParam.FeeRate.Uint64()
	// 0 <= fee <= maxFee
	if feeRate > oldFee.MaxFeeRate {
		return nil, errors.New("fee rate can't bigger than old")
	}
	if feeRate > stakeInfo.FeeRate + PSFeeRateStep {
		return nil, errors.New("0 <= newFeeRate <= oldFeerate + 100")
	}

	oldFee.FeeRate = feeRate
	oldFee.ChangedEpoch = eid

	stakeInfo.FeeRate = feeRate

	err = p.saveStakeInfo(evm, stakeInfo)
	if err != nil {
		return nil, err
	}
	err = p.saveStakeFeeRate(evm, oldFee, stakeInfo.Address)
	if err != nil {
		return nil, err
	}
	err = p.stakeUpdateFeeRateLog(contract, evm, oldFee)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

/*
the weight of 7 epoch:  a + 7*b ~= 1000
the weight of 90 epoch: a + 90*b ~= 1500
the time of maxEpoch/minEpoch is 1.5
so the a=960, b=6.
thus, weight of 7 epoch is 960+7*6=1002
the weight of 90 epoch is 960+90*6 = 1500
the time is 1500/1002, about 1.5
*/
func CalLocktimeWeight(lockEpoch uint64) uint64 {
	if lockEpoch == 0 { //builtin account.
		return 960 + PSMinEpochNum*6
	}
	return 960 + lockEpoch*6
}

func GetStakeInKeyHash(address common.Address) common.Hash {
	return common.BytesToHash(address[:])
}

func GetStakersSnap(stateDb *state.StateDB) []StakerInfo {
	stakeHolders := make([]StakerInfo, 0)
	stateDb.ForEachStorageByteArray(StakersInfoAddr, func(key common.Hash, value []byte) bool {
		var stakerInfo StakerInfo
		err := rlp.DecodeBytes(value, &stakerInfo)
		if err != nil {
			log.Error(err.Error())
			return true
		}
		stakeHolders = append(stakeHolders, stakerInfo)
		return true
	})
	return stakeHolders
}

func StakeoutSetEpoch(stateDb *state.StateDB, epochID uint64) {
	b := big.NewInt(int64(epochID))
	StoreInfo(stateDb, StakingCommonAddr, StakersInfoStakeOutKeyHash, b.Bytes())
}

func StakeoutIsFinished(stateDb *state.StateDB, epochID uint64) bool {
	epochByte, err := GetInfo(stateDb, StakingCommonAddr, StakersInfoStakeOutKeyHash)
	if err != nil {
		return false
	}
	finishedEpochId := big.NewInt(0).SetBytes(epochByte).Uint64()
	return finishedEpochId >= epochID
}

func (p *PosStaking) stakeRegisterParseAndValid(payload []byte) (StakeRegisterParam, error) {
	var info StakeRegisterParam
	err := cscAbi.UnpackInput(&info, "stakeRegister", payload)
	if err != nil {
		return info, err
	}
	if info.MaxFeeRate.Cmp(maxFeeRate) > 0 || info.MaxFeeRate.Cmp(minFeeRate) < 0 {
		return info, errors.New("max fee rate should between 0 to 100")
	}
	if info.FeeRate.Cmp(info.MaxFeeRate) > 0 {
		return info, errors.New("fee rate should le maxFeeRate")
	}
	if info.MaxFeeRate.Cmp(maxFeeRate) == 0 && info.FeeRate.Cmp(info.MaxFeeRate) != 0 {
		return info, errors.New("feeRate should be same with maxFeeRate, if maxFeeRate eq 100")
	}
	err = p.doStakeInParseAndValid(&info.StakeInParam)
	if err != nil {
		return info, err
	}

	return info, nil
}

func (p *PosStaking) doStakeInParseAndValid(info *StakeInParam) error {
	// 1. SecPk is valid
	if info.SecPk == nil {
		return errors.New("wrong secPk for stakeIn")
	}
	pub := crypto.ToECDSAPub(info.SecPk)
	if nil == pub {
		return errors.New("secPk is invalid")
	}
	info.pub = pub

	// 2. Bn256Pk is valid
	if info.Bn256Pk == nil {
		return errors.New("wrong bn256Pk for stakeIn")
	}
	var g1 bn256.G1
	_, err := g1.Unmarshal(info.Bn256Pk)
	if err != nil {
		return errors.New("wrong point for bn256Pk")
	}

	// 3. Lock time >= min epoch, <= max epoch
	if info.LockEpochs.Cmp(minEpochNum) < 0 || info.LockEpochs.Cmp(maxEpochNum) > 0 {
		return errors.New("invalid lock time")
	}

	// 4. 0 <= FeeRate <= 10000
	if info.FeeRate.Cmp(maxFeeRate) > 0 || info.FeeRate.Cmp(minFeeRate) < 0 {
		return errors.New("fee rate should between 0 to 100")
	}
	return nil
}
//
// package param check helper functions
//
func (p *PosStaking) stakeInParseAndValid(payload []byte) (StakeInParam, error) {
	var info StakeInParam
	err := cscAbi.UnpackInput(&info, "stakeIn", payload)
	if err != nil {
		return info, err
	}

	err = p.doStakeInParseAndValid(&info)
	if err != nil {
		return info, err
	}
	return info, nil
}
func (p *PosStaking) partnerInParseAndValid(payload []byte) (PartnerInParam, error) {
	var info PartnerInParam
	err := cscAbi.UnpackInput(&info, "partnerIn", payload)
	if err != nil {
		return info, err
	}
	return info, nil
}
func (p *PosStaking) stakeUpdateParseAndValid(payload []byte) (StakeUpdateParam, error) {
	var info StakeUpdateParam
	err := cscAbi.UnpackInput(&info, "stakeUpdate", payload)
	if err != nil {
		return info, err
	}
	//  Lock time >= min epoch, <= max epoch
	if info.LockEpochs.Uint64() != 0 && (info.LockEpochs.Cmp(minEpochNum) < 0 || info.LockEpochs.Cmp(maxEpochNum) > 0) {
		return info, errors.New("invalid lock time")
	}

	return info, nil
}
func (p *PosStaking) stakeAppendParseAndValid(payload []byte) (common.Address, error) {
	var addr common.Address
	err := cscAbi.UnpackInput(&addr, "stakeAppend", payload)
	if err != nil {
		return addr, err
	}

	return addr, nil
}
func (p *PosStaking) delegateInParseAndValid(payload []byte) (common.Address, error) {
	var addr common.Address
	err := cscAbi.UnpackInput(&addr, "delegateIn", payload)
	if err != nil {
		return addr, err
	}

	return addr, nil
}
func (p *PosStaking) delegateOutParseAndValid(payload []byte) (common.Address, error) {
	var addr common.Address
	err := cscAbi.UnpackInput(&addr, "delegateOut", payload)
	if err != nil {
		return addr, err
	}

	return addr, nil
}
func (p *PosStaking) updateFeeRateParseAndValid(payload []byte) (*UpdateFeeRateParam, error) {
	var updateFeeRateParam UpdateFeeRateParam
	err := cscAbi.UnpackInput(&updateFeeRateParam, "stakeUpdateFeeRate", payload)
	if err != nil {
		return nil, err
	}
	if updateFeeRateParam.FeeRate.Cmp(maxFeeRate) > 0 || updateFeeRateParam.FeeRate.Cmp(minFeeRate) < 0 {
		return nil, errors.New("fee rate should between 0 to 10000")
	}

	return &updateFeeRateParam, nil
}

func (p *PosStaking) stakeRegisterLog(contract *Contract, evm *EVM, info *StakerInfo, maxFeeRate uint64) error {
	// event stakeRegister(address indexed sender, address indexed posAddress, uint indexed v, uint feeRate, uint lockEpoch, uint maxFeeRate);
	params := make([]common.Hash, 3)
	params[0] = common.BytesToHash(contract.Caller().Bytes())
	params[1] = info.Address.Hash()
	params[2] = common.BigToHash(contract.Value())
	//
	data := make([]byte, 0)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(info.FeeRate)).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(info.LockEpochs)).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(maxFeeRate)).Bytes()...)
	sig := cscAbi.Events["stakeRegister"].Id().Bytes()
	return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, data)
}

func (p *PosStaking) stakeInLog(contract *Contract, evm *EVM, info *StakerInfo) error {
	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid < posconfig.ApolloEpochID {
		params := make([]common.Hash, 5)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = common.BigToHash(contract.Value())
		params[2] = common.BigToHash(new(big.Int).SetUint64(info.FeeRate))
		params[3] = common.BigToHash(new(big.Int).SetUint64(info.LockEpochs))
		params[4] = info.Address.Hash()
		sig := crypto.Keccak256([]byte(cscAbi.Methods["stakeIn"].Sig()))
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	} else {
		// event stakeIn(address indexed sender, address indexed posAddress, uint indexed v, uint feeRate, uint lockEpoch);
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = info.Address.Hash()
		params[2] = common.BigToHash(contract.Value())
		//
		data := make([]byte, 0)
		data = append(data, common.BigToHash(new(big.Int).SetUint64(info.FeeRate)).Bytes()...)
		data = append(data, common.BigToHash(new(big.Int).SetUint64(info.LockEpochs)).Bytes()...)
		sig := cscAbi.Events["stakeIn"].Id().Bytes()
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, data)
	}
}

func (p *PosStaking) stakeAppendLog(contract *Contract, evm *EVM, validator common.Address) error {
	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid < posconfig.ApolloEpochID {
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = common.BigToHash(contract.Value())
		params[2] = validator.Hash()

		sig := crypto.Keccak256([]byte(cscAbi.Methods["stakeAppend"].Sig()))
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	} else {
		// event stakeAppend(address indexed sender, address indexed posAddress, uint indexed v);
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = validator.Hash()
		params[2] = common.BigToHash(contract.Value())

		sig := cscAbi.Events["stakeAppend"].Id().Bytes()
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	}
}

func (p *PosStaking) stakeUpdateLog(contract *Contract, evm *EVM, info *StakerInfo) error {
	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid < posconfig.ApolloEpochID {
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = common.BigToHash(new(big.Int).SetUint64(info.NextLockEpochs))
		params[2] = info.Address.Hash()

		sig := crypto.Keccak256([]byte(cscAbi.Methods["stakeUpdate"].Sig()))
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	} else {
		// event stakeUpdate(address indexed sender, address indexed posAddress, uint indexed lockEpoch);
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = info.Address.Hash()
		params[2] = common.BigToHash(new(big.Int).SetUint64(info.NextLockEpochs))

		sig := cscAbi.Events["stakeUpdate"].Id().Bytes()
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	}
}

func (p *PosStaking) delegateInLog(contract *Contract, evm *EVM, validator common.Address) error {
	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid < posconfig.ApolloEpochID {
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = common.BigToHash(contract.Value())
		params[2] = validator.Hash()

		sig := crypto.Keccak256([]byte(cscAbi.Methods["delegateIn"].Sig()))
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	} else {
		// event delegateIn(address indexed sender, address indexed posAddress, uint indexed v);
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = validator.Hash()
		params[2] = common.BigToHash(contract.Value())

		sig := cscAbi.Events["delegateIn"].Id().Bytes()
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)

	}
}

func (p *PosStaking) delegateOutLog(contract *Contract, evm *EVM, validator common.Address) error {
	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid < posconfig.ApolloEpochID {
		params := make([]common.Hash, 2)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = validator.Hash()

		sig := crypto.Keccak256([]byte(cscAbi.Methods["delegateOut"].Sig()))
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	} else {
		// event delegateOut(address indexed sender, address indexed posAddress);
		params := make([]common.Hash, 2)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = validator.Hash()

		sig := cscAbi.Events["delegateOut"].Id().Bytes()
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	}
}

func (p *PosStaking) stakeUpdateFeeRateLog(contract *Contract, evm *EVM, feeInfo *UpdateFeeRate) error {
	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid >= posconfig.ApolloEpochID {
		// event stakeUpdateFeeRate(address indexed sender, address indexed posAddress, uint indexed feeRate);
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = common.BytesToHash(feeInfo.ValidatorAddr.Bytes())
		params[2] = common.BigToHash(new(big.Int).SetUint64(feeInfo.FeeRate))

		//data := make([]byte, 0)
		//data = append(data, common.BigToHash(new(big.Int).SetUint64(feeInfo.MaxFeeRate)).Bytes()...)
		//data = append(data, common.BigToHash(new(big.Int).SetUint64(feeInfo.ChangedEpoch)).Bytes()...)

		sig := cscAbi.Events["stakeUpdateFeeRate"].Id().Bytes()
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, nil)
	}
	return nil
}

func (p *PosStaking) partnerInLog(contract *Contract, evm *EVM, addr *common.Address, renew bool) error {
	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid >= posconfig.ApolloEpochID {
		// event partnerIn(address indexed sender, address indexed posAddress, uint indexed v, bool renewal);
		params := make([]common.Hash, 3)
		params[0] = common.BytesToHash(contract.Caller().Bytes())
		params[1] = common.BytesToHash(addr.Bytes())
		params[2] = common.BigToHash(contract.Value())

		var renewal = uint64(0)
		if renew {
			renewal = uint64(1)
		}
		data := make([]byte, 0)
		data = append(data, common.BigToHash(new(big.Int).SetUint64(renewal)).Bytes()...)

		sig := cscAbi.Events["partnerIn"].Id().Bytes()
		return precompiledScAddLog(contract.Address(), evm, common.BytesToHash(sig), params, data)
	}
	return nil
}
