package vm

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"errors" // this is not match with other
	"fmt"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/ecies"
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
solEnhanceDef = `[
	{
		"constant": true,
		"inputs": [],
		"name": "calPolyCommitTest",
		"outputs": [
			{
				"name": "sx",
				"type": "uint256"
			},
			{
				"name": "sy",
				"type": "uint256"
			},
			{
				"name": "success",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
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
		"inputs": [],
		"name": "addTest",
		"outputs": [
			{
				"name": "retx",
				"type": "uint256"
			},
			{
				"name": "rety",
				"type": "uint256"
			},
			{
				"name": "success",
				"type": "bool"
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
				"name": "hash",
				"type": "bytes32"
			},
			{
				"name": "r",
				"type": "bytes"
			},
			{
				"name": "s",
				"type": "bytes"
			},
			{
				"name": "pk",
				"type": "bytes"
			}
		],
		"name": "checkSig",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "data",
				"type": "string"
			}
		],
		"name": "hexStr2bytes",
		"outputs": [
			{
				"name": "",
				"type": "bytes"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
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
				"name": "r",
				"type": "uint256"
			},
			{
				"name": "M",
				"type": "uint256"
			},
			{
				"name": "K",
				"type": "bytes"
			}
		],
		"name": "enc",
		"outputs": [
			{
				"name": "c",
				"type": "bytes"
			},
			{
				"name": "success",
				"type": "bool"
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
			},
			{
				"name": "success",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "mulGTest",
		"outputs": [
			{
				"name": "retx",
				"type": "uint256"
			},
			{
				"name": "rety",
				"type": "uint256"
			},
			{
				"name": "success",
				"type": "bool"
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
				"name": "retx",
				"type": "uint256"
			},
			{
				"name": "rety",
				"type": "uint256"
			},
			{
				"name": "success",
				"type": "bool"
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
				"name": "pk",
				"type": "bytes"
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
			},
			{
				"name": "success",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	}
]`
	// pos staking contract abi object
	solenhanceAbi, errInit = abi.JSON(strings.NewReader(solEnhanceDef))


	getPosAvgReturnId 	[4]byte
	addid				[4]byte
	mulGid				[4]byte
	calPolyCommitid		[4]byte
	checkSigid			[4]byte


)

const (
	POLY_CIMMIT_ITEM_LEN = 65
)
//
// package initialize
//
func init() {
	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(getPosAvgReturnId[:], solenhanceAbi.Methods["getPosAvgReturn"].Id())
	copy(addid[:],solenhanceAbi.Methods["add"].Id())
	copy(mulGid[:],solenhanceAbi.Methods["mulG"].Id())
	copy(calPolyCommitid[:],solenhanceAbi.Methods["calPolyCommit"].Id())
	copy(checkSigid[:],solenhanceAbi.Methods["checkSigid"].Id())

	mulGidStr := common.Bytes2Hex(checkSigid[:])
	fmt.Println(""+mulGidStr)
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
	} else if  methodId == addid{
		return s.add(input[4:], contract, evm)
	} else if  methodId == mulGid {
		return s.mulG(input[4:], contract, evm)
	} else if methodId == calPolyCommitid {
		return s.calPolyCommit(input[4:], contract, evm)
	} else if methodId == checkSigid {
		return s.checkSig(input[4:], contract, evm)
	}


	mid := common.Bytes2Hex(methodId[:])
	fmt.Println(""+mid)



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


func (s *SolEnhance) add(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 128 {
		return []byte{0},errors.New("the point is not on curve")
	}
	x1 := big.NewInt(0).SetBytes(payload[:32])
	y1 := big.NewInt(0).SetBytes(payload[32:64])

	x2 := big.NewInt(0).SetBytes(payload[64:96])
	y2 := big.NewInt(0).SetBytes(payload[96:128])




	if !crypto.S256().IsOnCurve(x1,y1) || !crypto.S256().IsOnCurve(x2,y2) {
		return []byte{0},errors.New("the point is not on curve")
	}

	rx,ry := crypto.S256().Add(x1,y1,x2, y2)

	var buf = make([]byte, 64)
	copy(buf,rx.Bytes())
	copy(buf[32:],ry.Bytes())

	return buf, nil

}



func (s *SolEnhance) mulG(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) == 0 || len(payload) > 32 {
		return []byte{0},errors.New("the data length is not correct")
	}

	k := payload[:32]
	rx,ry := crypto.S256().ScalarBaseMult(k);

	var buf = make([]byte, 64)

	copy(buf,rx.Bytes())
	copy(buf[32:],ry.Bytes())

	return buf, nil
}




func (s *SolEnhance) calPolyCommit(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
//	fmt.Println(common.Bytes2Hex(payload))

	degree := int(payload[len -1])

	if len < (degree + 1)*POLY_CIMMIT_ITEM_LEN {
		return []byte{0},errors.New("payload is not enough")
	}

	f := make([]*ecdsa.PublicKey,degree);
	i := 0
	for ;i< degree;i++ {
		value := payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN]
		f[i] = crypto.ToECDSAPub(value)
	}

	pb := payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN]

	hashx := sha256.Sum256(pb)

	bigx := big.NewInt(0).SetBytes(hashx[:])
	bigx = bigx.Mod(bigx, crypto.S256().Params().N)

	res,err := s.EvalByPolyG(f,degree - 1,bigx)
	if err != nil {
		return []byte{0},errors.New("error in caculate poly")
	}

	fmt.Println(common.Bytes2Hex(crypto.FromECDSAPub(res)))


	var buf = make([]byte, 64)
	copy(buf,res.X.Bytes())
	copy(buf[32:],res.Y.Bytes())

	return buf,nil
}



func  (s *SolEnhance)  EvalByPolyG(pks []*ecdsa.PublicKey,degree int,x *big.Int) (*ecdsa.PublicKey, error) {
	// check input parameters

	sumPk := new(ecdsa.PublicKey)
	sumPk.Curve = crypto.S256()
	sumPk.X, sumPk.Y = pks[0].X, pks[0].Y

	for i := 1; i < int(degree)+1; i++ {

		temp1 := new(big.Int).Exp(x, big.NewInt(int64(i)), crypto.S256().Params().N)
		temp1.Mod(temp1, crypto.S256().Params().N)

		temp1Pk := new(ecdsa.PublicKey)
		temp1Pk.Curve = crypto.S256()

		temp1Pk.X, temp1Pk.Y = crypto.S256().ScalarMult(pks[i].X,pks[i].Y,temp1.Bytes())

		sumPk.X, sumPk.Y = crypto.S256().Add(sumPk.X,sumPk.Y,temp1Pk.X,temp1Pk.Y)

	}
	return sumPk,nil
}



func (s *SolEnhance) encrypt(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	rb := payload[0:32]
	message := payload[32:64]
	pkb := payload[64:]

	pk := new(ecdsa.PublicKey)
	pk.Curve = crypto.S256()
	pk.X = new(big.Int).SetBytes(pkb[:32])
	pk.Y = new(big.Int).SetBytes(pkb[32:])


	res,error := ecies.EncryptWithRandom(rb, ecies.ImportECDSAPublic(pk), message, nil, nil)

	if error != nil {
		return []byte{0},error
	}

	return res,nil
}


func (s *SolEnhance) checkSig(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	if len < 64 + 32*3 {
		return []byte{0},nil
	}

	hash := payload[:32]
	sr := big.NewInt(0).SetBytes(payload[32:64])
	ss := big.NewInt(0).SetBytes(payload[64:96])
	payload[95] = byte(4)
	pub := crypto.ToECDSAPub(payload[95:160])


	res := ecdsa.Verify(pub,hash,sr,ss)

	var buf = make([]byte, 32)
	if res {
		buf[31] = byte(1);
	} else {
		buf[31] = byte(0);
	}

	return buf,nil

}
