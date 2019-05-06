package vm

import (
	"errors"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
)

/* the contract interface described by solidity.

pragma solidity ^0.5.1;

contract posControl {
	function upgradeWhiteEpochLeader(uint256 EpochId, uint256 wlIndex, uint256 wlCount ) public  {}
}
*/

var (
	posControlDefinition = `
[
	{
		"constant": false,
		"inputs": [
			{
				"name": "EpochId",
				"type": "uint256"
			},
			{
				"name": "wlIndex",
				"type": "uint256"
			},
			{
				"name": "wlCount",
				"type": "uint256"
			}
		],
		"name": "upgradeWhiteEpochLeader",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
`

	posControlAbi             abi.ABI
	upgradeWhiteEpochLeaderId [4]byte

	maxWlindex = len(posconfig.WhiteList)
	minWlIndex = 0
)

type UpgradeWhiteEpochLeaderParam struct {
	EpochId *big.Int
	WlIndex *big.Int
	WlCount *big.Int
}

var UpgradeWhiteEpochLeaderDefault = UpgradeWhiteEpochLeaderParam{
	EpochId: big.NewInt(0),
	WlIndex: big.NewInt(0),
	WlCount: big.NewInt(26),
}

//
// package initialize
//
func init() {
	posControlAbi, errCscInit = abi.JSON(strings.NewReader(posControlDefinition))
	if errCscInit != nil {
		panic("err in posControl abi initialize ")
	}

	copy(upgradeWhiteEpochLeaderId[:], posControlAbi.Methods["upgradeWhiteEpochLeader"].Id())
}

type PosControl struct {
}

//
// contract interfaces
//
func (p *PosControl) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *PosControl) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(input) < 4 {
		return nil, errors.New("parameter is wrong")
	}

	// check only the owner could run it.
	if contract.Caller()  != posconfig.PosOwnerAddr {
		return nil, errParameters
	}
	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == upgradeWhiteEpochLeaderId {
		info, err := p.upgradeWhiteEpochLeaderParseAndValid(input[4:], evm.Time.Uint64())
		if err != nil {
			return nil, err
		}
		return p.upgradeWhiteEpochLeader(info, contract, evm)
	}

	return nil, errMethodId
}

func (p *PosControl) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	input := tx.Data()
	if len(input) < 4 {
		return errors.New("parameter is too short")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == upgradeWhiteEpochLeaderId {
		_, err := p.upgradeWhiteEpochLeaderParseAndValid(input[4:], uint64(time.Now().Unix()))
		if err != nil {
			return errors.New("upgradeWhiteEpochLeaderParseAndValid verify failed")
		}
		return nil
	}
	return errParameters
}

func posControlCheckEpoch(epochId uint64, time uint64) bool {
	eid, _ := util.CalEpochSlotID(time)
	if  eid+posconfig.PosUpgradeEpochID >  epochId { // must send tx some epochs in advance.
		return false
	}
	return true
}

func (p *PosControl) upgradeWhiteEpochLeader(info *UpgradeWhiteEpochLeaderParam, contract *Contract, evm *EVM) ([]byte, error) {
	infoBytes, err := rlp.EncodeToBytes(info)
	if err != nil {
		return nil, err
	}

	res := StoreInfo(evm.StateDB, PosControlPrecompileAddr, common.BigToHash(info.EpochId), infoBytes)
	if res != nil {
		return nil, res
	}

	return nil, nil
}

func (p *PosControl) upgradeWhiteEpochLeaderParseAndValid(payload []byte, time uint64) (*UpgradeWhiteEpochLeaderParam, error) {
	var info UpgradeWhiteEpochLeaderParam
	err := posControlAbi.UnpackInput(&info, "upgradeWhiteEpochLeader", payload)
	if err != nil {
		return nil, err
	}

	// check epoch valid
	wlIndex := info.WlIndex.Uint64()
	wlCount := info.WlCount.Uint64()
	if wlIndex+wlCount >= uint64(len(posconfig.WhiteList)) {
		return nil, errors.New("wlIndex out of range")
	}
	if wlCount < posconfig.MinEpHold || wlCount > posconfig.MaxEpHold {
		return nil, errors.New("wlCount out of range")
	}
	if !posControlCheckEpoch(info.EpochId.Uint64(), time) {
		return nil, errors.New("wrong epoch for upgradeWhiteEpochLeader")
	}
	return &info, nil
}

type WhiteInfos []UpgradeWhiteEpochLeaderParam

func (s WhiteInfos) Len() int {
	return len(s)
}

func (s WhiteInfos) Less(i, j int) bool {
	return s[i].EpochId.Cmp(s[j].EpochId) < 0
}

func (s WhiteInfos) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func GetWlConfig(stateDb StateDB) WhiteInfos {
	infos := make(WhiteInfos, 0)
	infos = append(infos, UpgradeWhiteEpochLeaderDefault)
	stateDb.ForEachStorageByteArray(PosControlPrecompileAddr, func(key common.Hash, value []byte) bool {
		info := UpgradeWhiteEpochLeaderParam{}
		err := rlp.DecodeBytes(value, &info)
		if err == nil {
			infos = append(infos, info)
		}
		return true
	})
	// sort
	sort.Stable(infos)
	return infos
}
func GetEpochWLInfo(stateDb StateDB, epochId uint64) *UpgradeWhiteEpochLeaderParam {
	infos := GetWlConfig(stateDb)
	index := len(infos) - 1
	for i := 0; i < len(infos); i++ {
		if infos[i].EpochId.Cmp(big.NewInt(int64(epochId))) == 0 {
			index = i
			break
		} else if infos[i].EpochId.Cmp(big.NewInt(int64(epochId))) > 0 {
			index = i - 1
			break
		}
	}
	return &infos[index]
}
