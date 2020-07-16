package vm

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"errors" // this is not match with other
	"fmt"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/crypto/ecies"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/util"
	posutil "github.com/wanchain/go-wanchain/pos/util"
	"math/big"
	"strings"
)


var (
	// pos staking contract abi definition
solEnhanceDef = `[
	{
		"constant": false,
		"inputs": [
			{
				"name": "input",
				"type": "bytes"
			}
		],
		"name": "bn256Pairing",
		"outputs": [
			{
				"name": "result",
				"type": "bytes32"
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
		"name": "bn256add",
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
		"name": "s256add",
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
				"name": "scalar",
				"type": "uint256"
			}
		],
		"name": "bn256MulG",
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
		"inputs": [
			{
				"name": "hash",
				"type": "bytes32"
			},
			{
				"name": "r",
				"type": "bytes32"
			},
			{
				"name": "s",
				"type": "bytes32"
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
				"name": "smgDeposit",
				"type": "uint256"
			},
			{
				"name": "smgStartTime",
				"type": "uint256"
			},
			{
				"name": "crossChainCoefficient",
				"type": "uint256"
			},
			{
				"name": "chainTypeCoefficient",
				"type": "uint256"
			}
		],
		"name": "getMinIncentive",
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
				"name": "blockTime",
				"type": "uint256"
			}
		],
		"name": "getEpochId",
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
				"name": "pk",
				"type": "bytes"
			}
		],
		"name": "s256CalPolyCommit",
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
		"name": "bn256CalPolyCommit",
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
		"inputs": [
			{
				"name": "groupStartTime",
				"type": "uint256"
			},
			{
				"name": "curTime",
				"type": "uint256"
			}
		],
		"name": "getPosAvgReturn",
		"outputs": [
			{
				"name": "result",
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
				"name": "scalar",
				"type": "uint256"
			},
			{
				"name": "xPk",
				"type": "uint256"
			},
			{
				"name": "yPk",
				"type": "uint256"
			}
		],
		"name": "s256ScalarMul",
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
		"inputs": [
			{
				"name": "rbpri",
				"type": "bytes32"
			},
			{
				"name": "iv",
				"type": "bytes32"
			},
			{
				"name": "mes",
				"type": "uint256"
			},
			{
				"name": "pub",
				"type": "bytes"
			}
		],
		"name": "enc",
		"outputs": [
			{
				"name": "",
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
			},
			{
				"name": "xPk",
				"type": "uint256"
			},
			{
				"name": "yPk",
				"type": "uint256"
			}
		],
		"name": "mulPk",
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
		"inputs": [
			{
				"name": "crossChainCoefficient",
				"type": "uint256"
			},
			{
				"name": "chainTypeCoefficient",
				"type": "uint256"
			},
			{
				"name": "time",
				"type": "uint256"
			}
		],
		"name": "getHardCap",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			},
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
				"name": "scalar",
				"type": "uint256"
			},
			{
				"name": "xPk",
				"type": "uint256"
			},
			{
				"name": "yPk",
				"type": "uint256"
			}
		],
		"name": "bn256ScalarMul",
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
	}
]`
	// pos staking contract abi object
	solenhanceAbi, errInit = abi.JSON(strings.NewReader(solEnhanceDef))


	getPosAvgReturnId 		[4]byte
	s256Addid				[4]byte
	s256MulGid				[4]byte
	checkSigid				[4]byte
	encid					[4]byte
	hardCapid				[4]byte
	s256MulPkid				[4]byte
	s256CalPolyCommitid		[4]byte
	bn256CalPolyCommitid	[4]byte
	bn256MulGid				[4]byte
)

const (
	POLY_CIMMIT_ITEM_LEN = 64
)
//
// package initialize
//
func init() {
	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(getPosAvgReturnId[:], solenhanceAbi.Methods["getPosAvgReturn"].Id())
	copy(s256Addid[:],solenhanceAbi.Methods["add"].Id())
	copy(s256MulGid[:],solenhanceAbi.Methods["mulG"].Id())
	copy(checkSigid[:],solenhanceAbi.Methods["checkSigid"].Id())
	copy(encid[:],solenhanceAbi.Methods["enc"].Id())
	copy(hardCapid[:],solenhanceAbi.Methods["getHardCap"].Id())
	copy(s256MulPkid[:],solenhanceAbi.Methods["mulPk"].Id())

	copy(s256CalPolyCommitid[:],solenhanceAbi.Methods["s256CalPolyCommit"].Id())
	copy(bn256CalPolyCommitid[:],solenhanceAbi.Methods["bn256CalPolyCommit"].Id())
	copy(bn256MulGid[:],solenhanceAbi.Methods["bn256MulG"].Id())

	mulGidStr := common.Bytes2Hex(bn256MulGid[:])
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

	epid,_ := posutil.CalEpochSlotID(evm.Time.Uint64())
	if epid < posconfig.StoremanEpochid {
		return nil,errors.New("not reach forked epochid")
	}

	if len(input) < 4 {
		return nil, errors.New("parameter is wrong")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == getPosAvgReturnId {
		return s.getPosAvgReturn(input[4:], contract, evm)
	} else if  methodId == s256Addid{
		return s.s256Add(input[4:], contract, evm)
	} else if  methodId == s256MulGid {
		return s.s256MulG(input[4:], contract, evm)
	} else if methodId == s256CalPolyCommitid {
		return s.s256CalPolyCommit(input[4:], contract, evm)
	} else if methodId == checkSigid {
		return s.checkSig(input[4:], contract, evm)
	} else if methodId == encid {
		return s.encrypt(input[4:], contract, evm)
	} else if methodId == hardCapid {
		return s.getPosTotalRet(input[4:], contract, evm)
	} else if methodId == s256MulPkid {
		return s.s256MulPk(input[4:], contract, evm)
	} else if methodId == bn256CalPolyCommitid {
		return s.bn256CalPolyCommit(input[4:], contract, evm)
	} else if methodId == bn256MulGid {
		return s.bn256MulG(input[4:], contract, evm)
	}

	mid := common.Bytes2Hex(methodId[:])
	fmt.Println(""+mid)

	return nil, errMethodId
}


func (s *SolEnhance) getPosAvgReturn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(payload) < 64 {
		return nil,errors.New("wrong data length")
	}

	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, 0)
	common.LeftPadBytes(buf, 32)

	eid, _ := util.CalEpochSlotID(evm.Time.Uint64())
	if eid < posconfig.StoremanEpochid {
		return buf,errors.New("not reach forked epochid")
	}

	//to do
	groupStartTime := new(big.Int).SetBytes(getData(payload, 0, 32)).Uint64()
	targetTime := new(big.Int).SetBytes(getData(payload, 32, 32)).Uint64()

	////for test/////////////////////////////////
	//groupStartTime = uint64(time.Now().Unix())
	//targetTime = groupStartTime

	groupStartEpochId,_ := posutil.CalEpochSlotID(groupStartTime)
	groupStartEpochId--

	targetEpochId,_ := posutil.CalEpochSlotID(targetTime)
	targetEpochId--
	/////////////////////////////////////////

	if  groupStartEpochId <= posconfig.FirstEpochId ||
		targetEpochId <= groupStartEpochId ||
		groupStartEpochId > eid ||
		targetEpochId > eid {
		return buf,errors.New("wrong epochid")
	}

	inst := posutil.PosAvgRetInst()
	if inst == nil {
		return buf,errors.New("not initialzied for pos return ")
	}


	p2,err := inst.GetOneEpochAvgReturnFor90LockEpoch(groupStartEpochId);
	if err != nil {
		return buf,err
	}

	stakeBegin,err := inst.GetAllStakeAndReturn(targetEpochId - 1)
	if err != nil {
		return buf,err
	}

	stakeEnd,err := inst.GetAllStakeAndReturn(targetEpochId)
	if err != nil {
		return buf,err
	}


	p2Big := big.NewInt(int64(p2))

	p1Mul := p2Big.Mul(p2Big,stakeBegin)

	p1 := p1Mul.Div(p1Mul,stakeEnd).Uint64()


	binary.BigEndian.PutUint64(buf, p1)
	return common.LeftPadBytes(buf, 32), nil
}


func (s *SolEnhance) s256Add(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

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
	if rx == nil || ry == nil {
		return []byte{0},errors.New("errors in caculation")
	}

	var buf = make([]byte, 64)
	copy(buf,common.LeftPadBytes(rx.Bytes(),32))
	copy(buf[32:],common.LeftPadBytes(ry.Bytes(),32))

	return buf, nil

}




func (s *SolEnhance) s256MulPk(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 96{
		return []byte{0},errors.New("the data length is not correct")
	}

	scalar := payload[:32]
	xPK := payload[32:64]
	yPK := payload[64:96]

	if !crypto.S256().IsOnCurve(big.NewInt(0).SetBytes(xPK),big.NewInt(0).SetBytes(yPK)) {
		return []byte{0},errors.New("the point is not on curve")
	}

	//fmt.Println("x="+common.ToHex(xPK))
	//fmt.Println("y="+common.ToHex(yPK))
	//fmt.Println("scalar=" + common.Bytes2Hex(scalar))

	rx,ry := crypto.S256().ScalarMult(big.NewInt(0).SetBytes(xPK),big.NewInt(0).SetBytes(yPK),scalar)
	if rx == nil || ry == nil {
		return []byte{0},errors.New("errors in caculation")
	}

	var buf = make([]byte, 64)

	copy(buf,common.LeftPadBytes(rx.Bytes(),32))
	copy(buf[32:],common.LeftPadBytes(ry.Bytes(),32))

	return buf, nil
}


func (s *SolEnhance) s256MulG(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 32 {
		return []byte{0},errors.New("the data length is not correct")
	}

	k := payload[:32]
	rx,ry := crypto.S256().ScalarBaseMult(k);

	if rx == nil || ry == nil {
		return []byte{0},errors.New("k value is not correct")
	}

	var buf = make([]byte, 64)

	copy(buf,common.LeftPadBytes(rx.Bytes(),32))
	copy(buf[32:],common.LeftPadBytes(ry.Bytes(),32))

	return buf, nil
}



func (s *SolEnhance) bn256MulG(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 32 {
		return []byte{0},errors.New("the data length is not correct")
	}

	k := payload[:32]
	gk := new(bn256.G1).ScalarBaseMult(big.NewInt(0).SetBytes(k));
	if gk==nil {
		return nil,errors.New("errors in g1 base mult")
	}

	return gk.Marshal(), nil
}


func (s *SolEnhance) s256CalPolyCommit(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	//4 point and one ok
	if len < 64*4 || len%64 != 0 {
		return []byte{0},errors.New("payload length is not correct")
	}

	degree := len/64 - 1;

	if len < (degree + 1)*POLY_CIMMIT_ITEM_LEN {
		return []byte{0},errors.New("payload is not enough")
	}

	f := make([]*ecdsa.PublicKey,degree);
	i := 0

	for ;i< degree;i++ {		//set the oxo4 prevalue for publickey
		byte65 := make([]byte,0)
		byte65 = append(byte65,4)
		byte65 = append(byte65,payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN]...)
		f[i] = crypto.ToECDSAPub(byte65)
		if f[i] == nil {
			return []byte{0},errors.New("commit data is not correct")
		}
	}


	hashx := sha256.Sum256(payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN])
	bigx := big.NewInt(0).SetBytes(hashx[:])
	bigx = bigx.Mod(bigx, crypto.S256().Params().N)

	res,err := s.EvalByPolyG(f,degree - 1,bigx)
	if err != nil || res == nil{
		return []byte{0},errors.New("error in caculate poly")
	}

	//fmt.Println(common.Bytes2Hex(crypto.FromECDSAPub(res)))


	var buf = make([]byte, 64)
	copy(buf,common.LeftPadBytes(res.X.Bytes(),32))
	copy(buf[32:],common.LeftPadBytes(res.Y.Bytes(),32))

	return buf,nil
}



func  (s *SolEnhance)  EvalByPolyG(pks []*ecdsa.PublicKey,degree int,x *big.Int) (*ecdsa.PublicKey, error) {
	if len(pks) < 1 || degree != len(pks) - 1 {
		return nil, errors.New("invalid polynomial len")
	}

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


func (s *SolEnhance) bn256CalPolyCommit(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	//fmt.Println(common.Bytes2Hex(payload))

	//4 point and one ok
	if len < 64*4 || len%64 != 0 {
		return []byte{0},errors.New("payload length is not correct")
	}

	degree := len/64 - 1;
	if len < (degree + 1)*POLY_CIMMIT_ITEM_LEN {
		return []byte{0},errors.New("payload is not enough")
	}

	f := make([][]byte,degree)
	i := 0
	for ;i< degree;i++ {
		f[i] = payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN]
	}

	//fmt.Println(common.Bytes2Hex(payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN]))
	hashx := sha256.Sum256(payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN])
	//fmt.Println(common.Bytes2Hex(hashx[:]))
	bigx := big.NewInt(0).SetBytes(hashx[:])
	bigx = bigx.Mod(bigx, bn256.Order)

	res,err := s.bn256EvalByPolyG(f,degree - 1,bigx)
	if err != nil {
		return []byte{0},errors.New("error in caculate poly")
	}
	//fmt.Println(common.Bytes2Hex(res))

	return res,nil
}


func  (s *SolEnhance)  bn256EvalByPolyG(pkbytes [][]byte, degree int, x *big.Int) ([]byte, error) {
	if len(pkbytes) == 0 || x.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("len(pks)==0 or xvalue is zero")
	}

	if len(pkbytes) != int(degree+1) {
		return nil, errors.New("degree is not content with the len(pks)")
	}

	pks := make([]*bn256.G1,0)
	for _, val := range pkbytes {
		//fmt.Println(common.Bytes2Hex(val))
		pk,err := newCurvePoint(val)
		if err != nil {
			return nil,err
		}
		pks = append(pks,pk)
	}

	sumPk := new(bn256.G1)
	for i := 0; i < int(degree)+1; i++ {

		temp1 := new(big.Int).Exp(x, big.NewInt(int64(i)), bn256.Order)
		temp1.Mod(temp1, bn256.Order)

		temp1Pk := new(bn256.G1).ScalarMult(pks[i], temp1)
		sumPk.Add(sumPk, temp1Pk)
	}

	return sumPk.Marshal(), nil

}


func (s *SolEnhance) encrypt(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 160 {
		return []byte{0},errors.New("data is not enough")
	}

	rb := payload[0:32]
	iv := payload[32:64]
	msg := payload[64:96]

	prv := hexKey(common.Bytes2Hex(rb))

	pkb := payload[96:]
	pk := new(ecdsa.PublicKey)
	pk.Curve = prv.PublicKey.Curve
	pk.X = new(big.Int).SetBytes(pkb[:32])
	pk.Y = new(big.Int).SetBytes(pkb[32:])



	//fmt.Println("rbPriv"+common.Bytes2Hex(rb))
	//fmt.Println(common.Bytes2Hex(iv))
	//fmt.Println(common.Bytes2Hex(msg))
	//fmt.Println(common.Bytes2Hex(pkb))

	res,error := ecies.EncryptWithRandom(prv, ecies.ImportECDSAPublic(pk),iv[16:],  msg, nil, nil)

	if error != nil {
		return []byte{0},error
	}

	//fmt.Println(common.Bytes2Hex(res))
	return res,nil
}


func (s *SolEnhance) checkSig(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	if len < 64 + 32*3 {
		return []byte{0},errors.New("wrong data length")
	}

	hash := payload[:32]
	sr := big.NewInt(0).SetBytes(payload[32:64])
	ss := big.NewInt(0).SetBytes(payload[64:96])

	//fmt.Println(common.Bytes2Hex(payload[64:96]))
	payload[95] = byte(4)

	//fmt.Println(common.Bytes2Hex(payload[95:160]))
	pub := crypto.ToECDSAPub(payload[95:160])
	if pub == nil {
		return []byte{0},errors.New("wrong data for publlic key")
	}

	res := ecdsa.Verify(pub,hash,sr,ss)

	var buf = make([]byte, 32)
	if res {
		buf[31] = byte(1);
	} else {
		buf[31] = byte(0);
	}

	return buf,nil

}


func (s *SolEnhance) getPosTotalRet(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	if len < 32 {
		return []byte{0},nil
	}

	time := big.NewInt(0).SetBytes(payload[:32])
	epid,_ := posutil.CalEpochSlotID(time.Uint64())
	epid--

	inst := posutil.PosAvgRetInst()
	if inst == nil {
		return []byte{0},errors.New("not initialzied for pos return ")
	}


	totalIncentive,err := inst.GetAllIncentive(epid)
	if err != nil || totalIncentive == nil  {
		return []byte{0},nil
	}

	totalIncentive = totalIncentive.Mul(totalIncentive,big.NewInt(10000))//keep 4 dots parts
	totalIncentive = totalIncentive.Div(totalIncentive,ether)
	ret := totalIncentive.Uint64()
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, ret)

	return common.LeftPadBytes(buf, 32), nil
}

func hexKey(prv string) *ecies.PrivateKey {
	key, err := crypto.HexToECDSA(prv)
	if err != nil {
		panic(err)
	}
	return ecies.ImportECDSA(key)
}




// bn256Add implements a native elliptic curve point addition.
type s256Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (s *s256Add) RequiredGas(input []byte) uint64 {
	return 0
}

func (s *s256Add) Run(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	epid,_ := posutil.CalEpochSlotID(evm.Time.Uint64())
	if epid < posconfig.StoremanEpochid {
		return nil,errors.New("not reach forked epochid")
	}

	if len(payload) < 128 {
		return []byte{0},errors.New("the point is not on curve")
	}

	x1 := big.NewInt(0).SetBytes(payload[:32])
	y1 := big.NewInt(0).SetBytes(payload[32:64])

	x2 := big.NewInt(0).SetBytes(payload[64:96])
	y2 := big.NewInt(0).SetBytes(payload[96:128])

	if !crypto.S256().IsOnCurve(x1,y1) || !crypto.S256().IsOnCurve(x2,y2) {
		return []byte{0},errors.New("the point is not on 256 curve")
	}

	rx,ry := crypto.S256().Add(x1,y1,x2, y2)
	if rx == nil || ry == nil {
		return []byte{0},errors.New("errors in curve add")
	}

	var buf = make([]byte, 64)
	copy(buf,common.LeftPadBytes(rx.Bytes(),32))
	copy(buf[32:],common.LeftPadBytes(ry.Bytes(),32))

	return buf, nil


}

func (s *s256Add) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

// bn256ScalarMul implements a native elliptic curve scalar multiplication.
type s256ScalarMul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (s *s256ScalarMul) RequiredGas(input []byte) uint64 {
	return 0
}

func (s *s256ScalarMul) Run(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	epid,_ := posutil.CalEpochSlotID(evm.Time.Uint64())
	if epid < posconfig.StoremanEpochid {
		return nil,errors.New("not reach forked epochid")
	}

	if len(payload) < 96{
		return []byte{0},errors.New("the data length is not correct")
	}

	scalar := payload[:32]

	xPK := payload[32:64]
	yPK := payload[64:96]

	//fmt.Println("x="+common.ToHex(xPK))
	//fmt.Println("y="+common.ToHex(yPK))
	//fmt.Println("scalar=" + common.Bytes2Hex(scalar))

	if !crypto.S256().IsOnCurve(big.NewInt(0).SetBytes(xPK),big.NewInt(0).SetBytes(yPK)) {
		return []byte{0},errors.New("the point is not on curve")
	}


	rx,ry := crypto.S256().ScalarMult(big.NewInt(0).SetBytes(xPK),big.NewInt(0).SetBytes(yPK),scalar)

	if rx == nil || ry == nil {
		return []byte{0},errors.New("k value is not correct")
	}

	var buf = make([]byte, 64)

	copy(buf,common.LeftPadBytes(rx.Bytes(),32))
	copy(buf[32:],common.LeftPadBytes(ry.Bytes(),32))

	return buf, nil
}

func (s *s256ScalarMul) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}
