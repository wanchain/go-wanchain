package vm

import (
	"errors"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare" // this is not match with other
	"math/big"
	"strings"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
)

/* the contract interface described by solidity.

contract stake {
	function stakeIn(bytes memory secPk, bytes memory bn256Pk, uint256 lockEpochs, uint256 feeRate) public payable {}
	function stakeUpdate(address addr, uint256 lockEpochs) public {}
	function stakeAppend(address addr) public payable {}
	function delegateIn(address delegateAddress) public payable {}
	function delegateOut(address delegateAddress) public {}
}

*/
const (
	PSMinEpochNum = 7
	PSEpochNum_1 = 15 // 1.1 times
	PSEpochNum_2 = 45 // 1.3 times
	PSEpochNum_3 = 90 // 1.5 times

	PSMaxEpochNum = 90
	PSMinStakeholderStake = 10000
	PSMinValidatorStake = 100000
	PSMinDelegatorStake = 100
	PSMinFeeRate = 0
	PSMaxFeeRate = 100
	PSNodeleFeeRate = 100
	maxTimeDelegate   = 5

	PSOutKeyHash = 700
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
	}
]
`
	// pos staking contract abi object
	cscAbi, errCscInit = abi.JSON(strings.NewReader(cscDefinition))

	// function "stakeIn" "delegateIn" 's solidity binary id
	stakeInId [4]byte
	stakeUpdateId [4]byte
	stakeAppendId [4]byte
	delegateInId [4]byte
	delegateOutId [4]byte

	maxEpochNum         = big.NewInt(PSMaxEpochNum)
	minEpochNum         = big.NewInt(PSMinEpochNum)
	minStakeholderStake = new(big.Int).Mul(big.NewInt(PSMinStakeholderStake), ether)
	minValidatorStake 	= new(big.Int).Mul(big.NewInt(PSMinValidatorStake), ether)
	minDelegatorStake = new(big.Int).Mul(big.NewInt(PSMinDelegatorStake), ether)
	minFeeRate = big.NewInt(PSMinFeeRate)
	maxFeeRate = big.NewInt(PSMaxFeeRate)
	noDelegateFeeRate = big.NewInt(PSNodeleFeeRate)
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
}
type StakeUpdateParam struct {
	Addr    common.Address   //stakeholder’s bn256 pairing public key
	LockEpochs *big.Int //lock time which is input by user
}
type DelegateParam struct {
	DelegateAddress common.Address //delegation’s address
}

//
// storage structures
//
type StakerInfo struct {
	Address   common.Address
	PubSec256 []byte //stakeholder’s wan public key
	PubBn256  []byte //stakeholder’s bn256 public key

	Amount     *big.Int //staking wan value
	StakeAmount     *big.Int //staking wan value
	LockEpochs uint64   //lock time which is input by user. 0 means unexpired.
	NextLockEpochs uint64   //lock time which is input by user. 0 means unexpired.
	From       common.Address

	StakingEpoch uint64 //the first epoch in which stakerHolder might be selected.
	FeeRate      uint64
	//NextFeeRate  uint64
	Clients      []ClientInfo
}

type ClientInfo struct {
	Address      common.Address
	Amount       *big.Int
	StakeAmount     *big.Int //staking wan value
	QuitEpoch uint64
}

//
// public helper structures
//
type Leader struct {
	PubSec256     []byte
	PubBn256      []byte
	SecAddr       common.Address
}

type ClientProbability struct {
	Addr        common.Address
	Probability *big.Int
}

type ClientIncentive struct {
	Addr      common.Address
	Incentive *big.Int
}

//
// package initialize
//
func init() {
	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(stakeInId[:], cscAbi.Methods["stakeIn"].Id())
	copy(stakeAppendId[:], cscAbi.Methods["stakeAppend"].Id())
	copy(stakeUpdateId[:], cscAbi.Methods["stakeUpdate"].Id())
	copy(delegateInId[:], cscAbi.Methods["delegateIn"].Id())
	copy(delegateOutId[:], cscAbi.Methods["delegateOut"].Id())
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

	if methodId == stakeInId {
		ret,err :=  p.StakeIn(input[4:], contract, evm)
		if err != nil {
			log.Info("stakein failed", "err", err)
		}
		return ret, err
	} else if methodId == stakeUpdateId {
		return p.StakeUpdate(input[4:], contract, evm)
	} else if methodId == stakeAppendId {
		return p.StakeAppend(input[4:], contract, evm)
	} else if methodId == delegateInId {
		return p.DelegateIn(input[4:], contract, evm)
	} else if methodId == delegateOutId {
		return p.DelegateOut(input[4:], contract, evm)
	}
	return nil, errMethodId
}

func (p *PosStaking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	input := tx.Data()
	if len(input) < 4 {
		return errors.New("parameter is too short")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeInId {
		err := p.stakeInParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakein verify failed")
		}
		return nil
	} else if methodId == stakeAppendId {
		err := p.stakeAppendParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakeout verify failed " + err.Error())
		}
		return nil
	} else if methodId == stakeUpdateId {
		err := p.stakeUpdateParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakeout verify failed " + err.Error())
		}
		return nil
	} else if methodId == delegateInId {
		err := p.delegateInParseAndValid(input[4:])
		if err != nil {
			return errors.New("delegateIn verify failed")
		}
		return nil
	} else if methodId == delegateOutId {
		err := p.delegateOutParseAndValid(input[4:])
		if err != nil {
			return errors.New("delegateOut verify failed")
		}
		return nil
	}

	return errParameters
}

//
// contract's methods
//
func (p *PosStaking) StakeUpdate(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var info StakeUpdateParam
	err := cscAbi.UnpackInput(&info, "stakeUpdate", payload)
	if err != nil {
		return nil, err
	}

	//  Lock time >= min epoch, <= max epoch
	if info.LockEpochs.Cmp(minEpochNum) < 0 || info.LockEpochs.Cmp(maxEpochNum) > 0 {
		return nil, errors.New("invalid lock time")
	}

	//  0 <= FeeRate <= 100
	// if info.FeeRate.Cmp(maxFeeRate) > 0 || info.FeeRate.Cmp(minFeeRate) < 0 {
	// 	return nil, errors.New("fee rate should between 0 to 100")
	// }

	key := GetStakeInKeyHash(info.Addr)
	stakerBytes, err := GetInfo(evm.StateDB, StakersInfoAddr, key)
	if stakerBytes == nil {
		return nil, errors.New("item doesn't exist")
	}
	var stakerInfo StakerInfo
	err = rlp.DecodeBytes(stakerBytes, &stakerInfo)
	if err != nil {
		return nil, errors.New("parse staker info error")
	}
	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eidNow > stakerInfo.StakingEpoch+stakerInfo.LockEpochs -3 {
		return nil, errors.New("cannot change at the last 3 epoch.")
	}

	// if info.FeeRate.Cmp(noDelegateFeeRate) != 0 &&  stakerInfo.Amount.Cmp(minValidatorStake) < 0 {
	// 	return nil, errors.New("need more Wan to be a validator")
	// }
	stakerInfo.NextLockEpochs = info.LockEpochs.Uint64()
	//stakerInfo.NextFeeRate = info.FeeRate.Uint64()
	infoBytes, err := rlp.EncodeToBytes(stakerInfo)
	if err != nil {
		return nil, err
	}
	res := StoreInfo(evm.StateDB, StakersInfoAddr, key, infoBytes)
	if res != nil {
		return nil, res
	}

	return nil, nil
}
func (p *PosStaking) StakeAppend(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var addr common.Address
	err := cscAbi.UnpackInput(&addr, "stakeAppend", payload)
	if err != nil {
		return nil, err
	}

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
	// add origen Amount
	stakerInfo.Amount.Add(stakerInfo.Amount, contract.Value())
	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())
	realLockEpoch := eidNow+2 - stakerInfo.StakingEpoch + stakerInfo.LockEpochs
	weight := CalLocktimeWeight(realLockEpoch)
	stakerInfo.StakeAmount.Mul(stakerInfo.Amount, big.NewInt(int64(weight)))
	infoBytes, err := rlp.EncodeToBytes(stakerInfo)
	if err != nil {
		return nil, err
	}
	res := StoreInfo(evm.StateDB, StakersInfoAddr, key, infoBytes)
	if res != nil {
		return nil, res
	}

	return nil, nil
}
func (p *PosStaking) StakeIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var info StakeInParam
	err := cscAbi.UnpackInput(&info, "stakeIn", payload)
	if err != nil {
		return nil, err
	}

	// 1. SecPk is valid
	if info.SecPk == nil {
		return nil, errors.New("wrong secPk for stakeIn")
	}
	pub := crypto.ToECDSAPub(info.SecPk)
	if nil == pub {
		return nil, errors.New("secPk is invalid")
	}

	// 2. Bn256Pk is valid
	if info.Bn256Pk == nil {
		return nil, errors.New("wrong bn256Pk for stakeIn")
	}
	var g1 bn256.G1
	_, err = g1.Unmarshal(info.Bn256Pk)
	if err != nil {
		return nil, errors.New("wrong point for bn256Pk")
	}

	// 3. Lock time >= min epoch, <= max epoch
	if info.LockEpochs.Cmp(minEpochNum) < 0 || info.LockEpochs.Cmp(maxEpochNum) > 0 {
		return nil, errors.New("invalid lock time")
	}

	// 4. 0 <= FeeRate <= 100
	if info.FeeRate.Cmp(maxFeeRate) > 0 || info.FeeRate.Cmp(minFeeRate) < 0 {
		return nil, errors.New("fee rate should between 0 to 100")
	}

	// TODO: need max?
	// 5. amount >= PSMinStakeholderStake,
	if contract.value.Cmp(minStakeholderStake) < 0 {
		return nil, errors.New("need more Wan to be a stake holder")
	}

	if info.FeeRate.Cmp(noDelegateFeeRate) != 0 &&  contract.value.Cmp(minValidatorStake) < 0 {
		return nil, errors.New("need more Wan to be a validator")
	}

	secAddr := crypto.PubkeyToAddress(*pub)

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
	stakeholder := &StakerInfo{
		Address:      secAddr,
		PubSec256:    info.SecPk,
		PubBn256:     info.Bn256Pk,
		Amount:       contract.value,
		LockEpochs:   info.LockEpochs.Uint64(),
		FeeRate:      info.FeeRate.Uint64(),
		NextLockEpochs:   info.LockEpochs.Uint64(),
		//NextFeeRate:      info.FeeRate.Uint64(),
		From:         contract.CallerAddress,
		StakingEpoch: eidNow+2,
	}
	stakeholder.StakeAmount = big.NewInt(0)
	stakeholder.StakeAmount.Mul(stakeholder.Amount, big.NewInt(int64(weight)))
	infoBytes, err := rlp.EncodeToBytes(stakeholder)
	if err != nil {
		return nil, err
	}

	//store stake info
	res := StoreInfo(evm.StateDB, StakersInfoAddr, key, infoBytes)
	if res != nil {
		return nil, res
	}

	return nil, nil
}

// one wants to choose a delegation to join the pos
func (p *PosStaking) DelegateIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var addr common.Address
	err := cscAbi.UnpackInput(&addr, "delegateIn", payload)
	if err != nil {
		return nil, err
	}

	//  mandatory is a valid stakeholder
	sKey := GetStakeInKeyHash(addr)
	stakerBytes, err := GetInfo(evm.StateDB, StakersInfoAddr, sKey)
	if stakerBytes == nil {
		return nil, errors.New("mandatory doesn't exist")
	}

	var stakerInfo StakerInfo
	err = rlp.DecodeBytes(stakerBytes, &stakerInfo)
	if err != nil {
		return nil, errors.New("parse staker info error")
	}

	//  sender has not delegated by this
	var  info*ClientInfo
	length := len(stakerInfo.Clients)
	totalDelegated := big.NewInt(0)
	for i := 0; i < length; i++ {
		if stakerInfo.Clients[i].Address == contract.CallerAddress {
			weight := CalLocktimeWeight(0)
			info = &stakerInfo.Clients[i]
			info.Amount.Add(info.Amount, contract.Value())
			info.StakeAmount.Add(info.StakeAmount, big.NewInt(0).Mul(contract.Value(), big.NewInt(int64(weight))))
			totalDelegated.Add(totalDelegated, info.Amount)
		}
	}
	// check the totalDelegated <= 5*stakerInfo.Amount
	if totalDelegated.Cmp(big.NewInt(0).Mul(stakerInfo.Amount,big.NewInt(maxTimeDelegate))) > 0 {
		return nil, errors.New("over delegate limitation")
	}
	if info == nil {
		// only first delegatein check amount is valid.
		if contract.value.Cmp(minDelegatorStake) < 0 {
			return nil, errors.New("low amount")
		}
		// save
		weight := CalLocktimeWeight(0)
		info := &ClientInfo{
			Address:      contract.CallerAddress,
			Amount:       contract.value,
			StakeAmount:       big.NewInt(0).Mul(contract.value, big.NewInt(int64(weight))),
			QuitEpoch: 0,
		}
		stakerInfo.Clients = append(stakerInfo.Clients, *info)
	}

	stakerInfoBytes, err := rlp.EncodeToBytes(stakerInfo)
	if err != nil {
		return nil, err
	}

	res := StoreInfo(evm.StateDB, StakersInfoAddr, sKey, stakerInfoBytes)
	if res != nil {
		return nil, res
	}

	return nil, nil
}

func (p *PosStaking) DelegateOut(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var addr common.Address
	err := cscAbi.UnpackInput(&addr, "delegateOut", payload)
	if err != nil {
		return nil, err
	}


	sKey := GetStakeInKeyHash(addr)
	stakerBytes, err := GetInfo(evm.StateDB, StakersInfoAddr, sKey)
	if stakerBytes == nil {
		return nil, errors.New("mandatory doesn't exist")
	}

	var stakerInfo StakerInfo
	err = rlp.DecodeBytes(stakerBytes, &stakerInfo)
	if err != nil {
		return nil, errors.New("parse staker info error")
	}


	length := len(stakerInfo.Clients)
	eidNow, _ := util.CalEpochSlotID(evm.Time.Uint64())

	found := false
	for i := 0; i < length; i++ {
		if stakerInfo.Clients[i].Address == contract.CallerAddress {
			// check if delegater has existed.
			if stakerInfo.Clients[i].QuitEpoch != 0 {
				return nil,  errors.New("delegater has existed")
			}
			stakerInfo.Clients[i].QuitEpoch = eidNow+3
			found = true
			break
		}
	}
	if ! found {
		return nil,  errors.New("item doesn't exist")
	}

	stakerInfoBytes, err := rlp.EncodeToBytes(stakerInfo)
	if err != nil {
		return nil, err
	}

	res := StoreInfo(evm.StateDB, StakersInfoAddr, sKey, stakerInfoBytes)
	if res != nil {
		return nil, res
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
	return 960+lockEpoch*6
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


func StakeoutSetEpoch(stateDb *state.StateDB,epochID uint64) {
	b := big.NewInt(int64(epochID))
	StoreInfo(stateDb, StakingCommonAddr, StakersInfoStakeOutKeyHash, b.Bytes())
}

func StakeoutIsFinished(stateDb *state.StateDB,epochID uint64) (bool) {
	epochByte,err := GetInfo(stateDb, StakingCommonAddr, StakersInfoStakeOutKeyHash)
	if err != nil {
		return false
	}
	finishedEpochId := big.NewInt(0).SetBytes(epochByte).Uint64()
	return finishedEpochId >= epochID
}

//
// package param check helper functions
//
func (p *PosStaking) stakeInParseAndValid(payload []byte) error {
	var info StakeInParam
	err := cscAbi.UnpackInput(&info, "stakeIn", payload)
	if err != nil {
		return err
	}
	return nil
}
func (p *PosStaking) stakeUpdateParseAndValid(payload []byte) error {
	var info StakeInParam
	err := cscAbi.UnpackInput(&info, "stakeUpdate", payload)
	if err != nil {
		return err
	}
	return nil
}
func (p *PosStaking) stakeAppendParseAndValid(payload []byte) error {
	var addr common.Address
	err := cscAbi.UnpackInput(&addr, "stakeAppend", payload)
	if err != nil {
		return err
	}

	return nil
}
func (p *PosStaking) delegateInParseAndValid(payload []byte) error {
	var delegateParam common.Address
	err := cscAbi.UnpackInput(&delegateParam, "delegateIn", payload)
	if err != nil {
		return err
	}

	return nil
}
func (p *PosStaking) delegateOutParseAndValid(payload []byte) error {
	var delegateParam common.Address
	err := cscAbi.UnpackInput(&delegateParam, "delegateOut", payload)
	if err != nil {
		return err
	}

	return nil
}
