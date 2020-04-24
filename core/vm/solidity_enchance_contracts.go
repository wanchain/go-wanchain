package vm

import (
	"encoding/binary"
	"errors" // this is not match with other
	"github.com/wanchain/go-wanchain/core/types"
	"math/big"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/pos/util"
	posutil "github.com/wanchain/go-wanchain/pos/util"
)


const (

)

var (
	// pos staking contract abi definition
solEnhanceDef = ` [
	{
		"constant": true,
		"inputs": [],
		"name": "DIVISOR",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "polyCommit",
				"type": "bytes"
			},
			{
				"name": "x",
				"type": "uint256"
			}
		],
		"name": "calPolyCommit",
		"outputs": [
			{
				"name": "sx",
				"type": "uint256"
			},
			{
				"name": "sy",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "r",
				"type": "uint256"
			},
			{
				"name": "M",
				"type": "uint256"
			},
			{
				"name": "K",
				"type": "uint256"
			}
		],
		"name": "enc",
		"outputs": [
			{
				"name": "c",
				"type": "bytes"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "groupStartTime",
				"type": "uint256"
			},
			{
				"name": "targetTime",
				"type": "uint256"
			}
		],
		"name": "getPosAvgReturn",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "scalar",
				"type": "uint256"
			}
		],
		"name": "mulG",
		"outputs": [
			{
				"name": "x",
				"type": "uint256"
			},
			{
				"name": "y",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "x1",
				"type": "uint256"
			},
			{
				"name": "y1",
				"type": "uint256"
			},
			{
				"name": "x2",
				"type": "uint256"
			},
			{
				"name": "y2",
				"type": "uint256"
			}
		],
		"name": "add",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			},
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	}
]`
	// pos staking contract abi object
	solenhanceAbi, errInit = abi.JSON(strings.NewReader(cscDefinition))


	getPosAvgReturnId [4]byte

)
//
// package initialize
//
func init() {
	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(getPosAvgReturnId[:], cscAbi.Methods["getPosAvgReturn"].Id())
}

/////////////////////////////
type SolEnhance struct {

}

//
// contract interfaces
//
func (s *SolEnhance) RequiredGas(input []byte) uint64 {
	return 0
}

func (s *SolEnhance) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

func (s *SolEnhance) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(input) < 4 {
		return nil, errors.New("parameter is wrong")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == getPosAvgReturnId {
		return s.getPosAvgReturn(input[4:], contract, evm)
	}

	return nil, errMethodId
}


func (s *SolEnhance) getPosAvgReturn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid < posconfig.StoremanEpochid {
		return []byte{0},errors.New("not reach forked epochid")
	}


	//to do
	groupStartTime := new(big.Int).SetBytes(getData(payload, 0, 32)).Uint64()
	targetTime := new(big.Int).SetBytes(getData(payload, 32, 32)).Uint64()

	////for test/////////////////////////////////
	groupStartTime = uint64(time.Now().Unix())
	targetTime = groupStartTime

	groupStartEpochId,_ := posutil.CalEpochSlotID(groupStartTime)
	groupStartEpochId--

	targetEpochId,_ := posutil.CalEpochSlotID(targetTime)
	targetEpochId--
	/////////////////////////////////////////

	if groupStartEpochId > eid || targetEpochId > eid {
		return []byte{0},errors.New("wrong epochid")
	}

	inst := posutil.PosAvgRetInst()
	if inst == nil {
		return []byte{0},errors.New("not initialzied for pos return ")
	}

	retTotal := uint64(0);
	for i:=uint64(0);i<posconfig.TARGETS_LOCKED_EPOCH;i++ {

		ret,err := inst.GetOneEpochAvgReturnFor90LockEpoch(groupStartEpochId - i)
		if err!= nil {
			continue
		}

		retTotal += ret
	}

	p2 := uint64(retTotal/posconfig.TARGETS_LOCKED_EPOCH)

	stakeBegin,err := inst.GetAllStakeAndReturn(targetEpochId - 1)
	if err != nil {
		return []byte{0},err
	}

	stakeEnd,err := inst.GetAllStakeAndReturn(targetEpochId)
	if err != nil {
		return []byte{0},err
	}


	p2Big := big.NewInt(int64(p2))

	p1Mul := p2Big.Mul(p2Big,stakeBegin)

	p1 := p1Mul.Div(p1Mul,stakeEnd).Uint64()

	////convert to byte array
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, p1)

	return common.LeftPadBytes(buf, 32), nil
}


