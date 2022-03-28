package vm

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"errors" // this is not match with other
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	"github.com/ethereum/go-ethereum/pos/util"
	posutil "github.com/ethereum/go-ethereum/pos/util"
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
				"name": "targetSecond",
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

	getPosAvgReturnId    [4]byte
	s256Addid            [4]byte
	s256MulGid           [4]byte
	checkSigid           [4]byte
	checkSigid2          [4]byte
	encid                [4]byte
	hardCapid            [4]byte
	s256MulPkid          [4]byte
	s256CalPolyCommitid  [4]byte
	bn256CalPolyCommitid [4]byte
	bn256MulGid          [4]byte
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

	copy(getPosAvgReturnId[:], solenhanceAbi.Methods["getPosAvgReturn"].ID)
	copy(s256Addid[:], solenhanceAbi.Methods["add"].ID)
	copy(s256MulGid[:], solenhanceAbi.Methods["mulG"].ID)
	copy(checkSigid[:], solenhanceAbi.Methods["checkSig"].ID)
	copy(checkSigid2[:], common.Hex2Bytes("861731d5"))
	copy(encid[:], solenhanceAbi.Methods["enc"].ID)
	copy(hardCapid[:], solenhanceAbi.Methods["getHardCap"].ID)
	copy(s256MulPkid[:], solenhanceAbi.Methods["mulPk"].ID)

	copy(s256CalPolyCommitid[:], solenhanceAbi.Methods["s256CalPolyCommit"].ID)
	copy(bn256CalPolyCommitid[:], solenhanceAbi.Methods["bn256CalPolyCommit"].ID)
	copy(bn256MulGid[:], solenhanceAbi.Methods["bn256MulG"].ID)
}

/////////////////////////////
type SolEnhance struct {
	contract *Contract
	evm      *EVM
}

//
// contract interfaces
//
func (s *SolEnhance) RequiredGas(input []byte) uint64 {
	return params.GasForSolEnhance
}

func (s *SolEnhance) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

func (s *SolEnhance) Run(input []byte) ([]byte, error) {
	contract := s.contract
	evm  := s.evm
	epid, _ := posutil.CalEpochSlotID(evm.Time().Uint64())
	if epid < posconfig.Cfg().MarsEpochId {
		// return nil,errors.New("not reach forked epochid")
		return nil, nil
	}

	if len(input) < 4 {
		return nil, errors.New("parameter is wrong")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == getPosAvgReturnId {
		return s.getPosAvgReturn(input[4:], contract, evm)
	} else if methodId == s256Addid {
		return s.s256Add(input[4:], contract, evm)
	} else if methodId == s256MulGid {
		return s.s256MulG(input[4:], contract, evm)
	} else if methodId == s256CalPolyCommitid {
		return s.s256CalPolyCommit(input[4:], contract, evm)
	} else if methodId == checkSigid || methodId == checkSigid2 {
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
	fmt.Println("" + mid)

	return nil, errMethodId
}

func (s *SolEnhance) getPosAvgReturn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(payload) < 32 {
		return nil, errors.New("wrong data length")
	}

	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, 0)
	common.LeftPadBytes(buf, 32)

	eid, _ := util.CalEpochSlotID(evm.Time().Uint64())
	if eid < posconfig.Cfg().MarsEpochId {
		return buf, errors.New("not reach forked epochid")
	}

	targetTime := new(big.Int).SetBytes(getData(payload, 0, 32)).Uint64()

	targetEpochId, _ := posutil.CalEpochSlotID(targetTime)
	/////////////////////////////////////////

	if targetEpochId <= posconfig.FirstEpochId ||
		targetEpochId > eid {
		return buf, errors.New("wrong epochid")
	}

	inst := posutil.PosAvgRetInst()
	if inst == nil {
		return buf, errors.New("not initialzied for pos return ")
	}

	p2, err := inst.GetPosAverageReturnRate(targetEpochId)
	if err != nil {
		return buf, err
	}

	binary.BigEndian.PutUint64(buf, p2)
	return common.LeftPadBytes(buf, 32), nil
}

func (s *SolEnhance) s256Add(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 128 {
		return []byte{0}, errors.New("the point is not on curve")
	}

	x1 := big.NewInt(0).SetBytes(payload[:32])
	y1 := big.NewInt(0).SetBytes(payload[32:64])

	x2 := big.NewInt(0).SetBytes(payload[64:96])
	y2 := big.NewInt(0).SetBytes(payload[96:128])

	if !crypto.S256().IsOnCurve(x1, y1) || !crypto.S256().IsOnCurve(x2, y2) {
		return []byte{0}, errors.New("the point is not on curve")
	}

	var rx, ry *big.Int
	if bytes.Equal(x1.Bytes(), x2.Bytes()) && bytes.Equal(y1.Bytes(), y2.Bytes()) {
		rx, ry = crypto.S256().Double(x1, y1)
	} else {
		rx, ry = crypto.S256().Add(x1, y1, x2, y2)
	}

	if rx == nil || ry == nil {
		return []byte{0}, errors.New("errors in caculation")
	}

	var buf = make([]byte, 64)
	copy(buf, common.LeftPadBytes(rx.Bytes(), 32))
	copy(buf[32:], common.LeftPadBytes(ry.Bytes(), 32))

	return buf, nil

}

func (s *SolEnhance) s256MulPk(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 96 {
		return []byte{0}, errors.New("the data length is not correct")
	}

	scalar := payload[:32]
	xPK := payload[32:64]
	yPK := payload[64:96]

	if !crypto.S256().IsOnCurve(big.NewInt(0).SetBytes(xPK), big.NewInt(0).SetBytes(yPK)) {
		return []byte{0}, errors.New("the point is not on curve")
	}

	//fmt.Println("x="+common.ToHex(xPK))
	//fmt.Println("y="+common.ToHex(yPK))
	//fmt.Println("scalar=" + common.Bytes2Hex(scalar))

	rx, ry := crypto.S256().ScalarMult(big.NewInt(0).SetBytes(xPK), big.NewInt(0).SetBytes(yPK), scalar)
	if rx == nil || ry == nil {
		return []byte{0}, errors.New("errors in caculation")
	}

	var buf = make([]byte, 64)

	copy(buf, common.LeftPadBytes(rx.Bytes(), 32))
	copy(buf[32:], common.LeftPadBytes(ry.Bytes(), 32))

	return buf, nil
}

func (s *SolEnhance) s256MulG(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 32 {
		return []byte{0}, errors.New("the data length is not correct")
	}

	k := payload[:32]
	rx, ry := crypto.S256().ScalarBaseMult(k)

	if rx == nil || ry == nil {
		return []byte{0}, errors.New("k value is not correct")
	}

	var buf = make([]byte, 64)

	copy(buf, common.LeftPadBytes(rx.Bytes(), 32))
	copy(buf[32:], common.LeftPadBytes(ry.Bytes(), 32))

	return buf, nil
}

func (s *SolEnhance) bn256MulG(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(payload) < 32 {
		return []byte{0}, errors.New("the data length is not correct")
	}

	k := payload[:32]
	gk := new(bn256.G1).ScalarBaseMult(big.NewInt(0).SetBytes(k))
	if gk == nil {
		return nil, errors.New("errors in g1 base mult")
	}

	return gk.Marshal(), nil
}

func (s *SolEnhance) s256CalPolyCommit(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	//4 point and one ok
	if len < 64*4 || len%64 != 0 {
		return []byte{0}, errors.New("payload length is not correct")
	}

	degree := len/64 - 1

	if len < (degree+1)*POLY_CIMMIT_ITEM_LEN {
		return []byte{0}, errors.New("payload is not enough")
	}

	f := make([]*ecdsa.PublicKey, degree)
	i := 0

	for ; i < degree; i++ { //set the oxo4 prevalue for publickey
		byte65 := make([]byte, 0)
		byte65 = append(byte65, 4)
		byte65 = append(byte65, payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN]...)
		f[i] = crypto.ToECDSAPub(byte65)
		if f[i] == nil {
			return []byte{0}, errors.New("commit data is not correct")
		}
	}

	hashx := sha256.Sum256(payload[i*POLY_CIMMIT_ITEM_LEN : (i+1)*POLY_CIMMIT_ITEM_LEN])
	bigx := big.NewInt(0).SetBytes(hashx[:])
	bigx = bigx.Mod(bigx, crypto.S256().Params().N)

	res, err := s.EvalByPolyG(f, degree-1, bigx)
	if err != nil || res == nil {
		return []byte{0}, errors.New("error in caculate poly")
	}

	//fmt.Println(common.Bytes2Hex(crypto.FromECDSAPub(res)))

	var buf = make([]byte, 64)
	copy(buf, common.LeftPadBytes(res.X.Bytes(), 32))
	copy(buf[32:], common.LeftPadBytes(res.Y.Bytes(), 32))

	return buf, nil
}

func (s *SolEnhance) EvalByPolyG(pks []*ecdsa.PublicKey, degree int, x *big.Int) (*ecdsa.PublicKey, error) {
	if len(pks) < 1 || degree != len(pks)-1 {
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

		temp1Pk.X, temp1Pk.Y = crypto.S256().ScalarMult(pks[i].X, pks[i].Y, temp1.Bytes())

		if bytes.Equal(sumPk.X.Bytes(), temp1Pk.X.Bytes()) && bytes.Equal(sumPk.Y.Bytes(), temp1Pk.Y.Bytes()) {
			sumPk.X, sumPk.Y = crypto.S256().Double(sumPk.X, sumPk.Y)
		} else {
			sumPk.X, sumPk.Y = crypto.S256().Add(sumPk.X, sumPk.Y, temp1Pk.X, temp1Pk.Y)
		}

	}
	return sumPk, nil
}

func (s *SolEnhance) bn256CalPolyCommit(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	//fmt.Println(common.Bytes2Hex(payload))

	//4 point and one ok
	if len < 64*4 || len%64 != 0 {
		return []byte{0}, errors.New("payload length is not correct")
	}

	degree := len/64 - 1
	if len < (degree+1)*POLY_CIMMIT_ITEM_LEN {
		return []byte{0}, errors.New("payload is not enough")
	}

	f := make([][]byte, degree)
	i := 0
	for ; i < degree; i++ {
		f[i] = payload[i*POLY_CIMMIT_ITEM_LEN : (i+1)*POLY_CIMMIT_ITEM_LEN]
	}

	//fmt.Println(common.Bytes2Hex(payload[i*POLY_CIMMIT_ITEM_LEN:(i+1)*POLY_CIMMIT_ITEM_LEN]))
	hashx := sha256.Sum256(payload[i*POLY_CIMMIT_ITEM_LEN : (i+1)*POLY_CIMMIT_ITEM_LEN])
	//fmt.Println(common.Bytes2Hex(hashx[:]))
	bigx := big.NewInt(0).SetBytes(hashx[:])
	bigx = bigx.Mod(bigx, bn256.Order)

	res, err := s.bn256EvalByPolyG(f, degree-1, bigx)
	if err != nil {
		return []byte{0}, errors.New("error in caculate poly")
	}
	//fmt.Println(common.Bytes2Hex(res))

	return res, nil
}

func (s *SolEnhance) bn256EvalByPolyG(pkbytes [][]byte, degree int, x *big.Int) ([]byte, error) {
	if len(pkbytes) == 0 || x.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("len(pks)==0 or xvalue is zero")
	}

	if len(pkbytes) != int(degree+1) {
		return nil, errors.New("degree is not content with the len(pks)")
	}

	pks := make([]*bn256.G1, 0)
	for _, val := range pkbytes {
		//fmt.Println(common.Bytes2Hex(val))
		pk, err := newCurvePoint(val)
		if err != nil {
			return nil, err
		}
		pks = append(pks, pk)
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
		return []byte{0}, errors.New("data is not enough")
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

	res, error := ecies.EncryptWithRandom(prv, ecies.ImportECDSAPublic(pk), iv[16:], msg, nil, nil)

	if error != nil {
		return []byte{0}, error
	}

	//fmt.Println(common.Bytes2Hex(res))
	return res, nil
}

func (s *SolEnhance) checkSig(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	len := len(payload)
	if len < 64+32*3 {
		return []byte{0}, errors.New("wrong data length")
	}

	hash := payload[:32]
	sr := big.NewInt(0).SetBytes(payload[32:64])
	ss := big.NewInt(0).SetBytes(payload[64:96])

	//fmt.Println(common.Bytes2Hex(payload[64:96]))
	payload[95] = byte(4)

	//fmt.Println(common.Bytes2Hex(payload[95:160]))
	pub := crypto.ToECDSAPub(payload[95:160])
	if pub == nil {
		return []byte{0}, errors.New("wrong data for publlic key")
	}

	res := ecdsa.Verify(pub, hash, sr, ss)

	var buf = make([]byte, 32)
	if res {
		buf[31] = byte(1)
	} else {
		buf[31] = byte(0)
	}

	return buf, nil

}

func (s *SolEnhance) getPosTotalRet(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	len := len(payload)
	log.Info("getPosTotalRet", "payload", len)

	if len < 32 {
		log.Warn("getPosTotalRet", "payload", len)
		return []byte{0}, nil
	}

	time := big.NewInt(0).SetBytes(payload[:32])
	epid, _ := posutil.CalEpochSlotID(time.Uint64())
	epid--

	inst := posutil.PosAvgRetInst()
	if inst == nil {
		log.Warn("not initialzied for pos return", "time", time, "epid", epid)
		return []byte{0}, errors.New("not initialzied for pos return ")
	}
	var totalIncentive *big.Int
	if params.IsLondonActive() {
		inst := posutil.PosAvgRetInst()
		_totalIncentive := inst.GetYearReward(epid)
		totalIncentive = _totalIncentive.Div(_totalIncentive, big.NewInt(365))
	} else {
		_totalIncentive, err := inst.GetAllIncentive(epid)
		if err != nil || _totalIncentive == nil {
			log.Warn("GetAllIncentive failed", "err", err, "time", time, "epid", epid)
			return []byte{0}, nil
		}
		totalIncentive = _totalIncentive
	}

	totalIncentive = totalIncentive.Mul(totalIncentive, big.NewInt(10000)) //keep 4 dots parts
	totalIncentive = totalIncentive.Div(totalIncentive, ether)
	ret := totalIncentive.Uint64()
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, ret)
	log.Info("getPosTotalRet return")
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
type s256Add struct{
	contract *Contract
	evm      *EVM
}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (s *s256Add) RequiredGas(input []byte) uint64 {
	return params.S256AddGas
}

func (s *s256Add) Run(payload []byte) ([]byte, error) {
	//contract := s.contract
	evm  := s.evm
	if evm != nil {
		epid, _ := posutil.CalEpochSlotID(evm.Time().Uint64())
		if epid < posconfig.Cfg().MarsEpochId {
			return nil, nil
		}
	}

	if len(payload) < 128 {
		return []byte{0}, errors.New("the point is not on curve")
	}

	x1 := big.NewInt(0).SetBytes(payload[:32])
	y1 := big.NewInt(0).SetBytes(payload[32:64])

	x2 := big.NewInt(0).SetBytes(payload[64:96])
	y2 := big.NewInt(0).SetBytes(payload[96:128])

	if !crypto.S256().IsOnCurve(x1, y1) || !crypto.S256().IsOnCurve(x2, y2) {
		return []byte{0}, errors.New("the point is not on 256 curve")
	}

	var rx, ry *big.Int
	if bytes.Equal(x1.Bytes(), x2.Bytes()) && bytes.Equal(y1.Bytes(), y2.Bytes()) {
		rx, ry = crypto.S256().Double(x1, y1)
	} else {
		rx, ry = crypto.S256().Add(x1, y1, x2, y2)
	}

	if rx == nil || ry == nil {
		return []byte{0}, errors.New("errors in curve add")
	}

	var buf = make([]byte, 64)
	copy(buf, common.LeftPadBytes(rx.Bytes(), 32))
	copy(buf[32:], common.LeftPadBytes(ry.Bytes(), 32))

	return buf, nil

}

func (s *s256Add) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

// bn256ScalarMul implements a native elliptic curve scalar multiplication.
type s256ScalarMul struct{
	contract *Contract
	evm      *EVM
}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (s *s256ScalarMul) RequiredGas(input []byte) uint64 {
	return params.S256ScalarMulGas
}

func (s *s256ScalarMul) Run(payload []byte) ([]byte, error) {
	//contract := s.contract
	evm  := s.evm
	if evm != nil {
		epid, _ := posutil.CalEpochSlotID(evm.Time().Uint64())
		if epid < posconfig.Cfg().MarsEpochId {
			return nil, nil
		}
	}

	if len(payload) < 96 {
		return []byte{0}, errors.New("the data length is not correct")
	}

	scalar := payload[:32]

	xPK := payload[32:64]
	yPK := payload[64:96]

	//fmt.Println("x="+common.ToHex(xPK))
	//fmt.Println("y="+common.ToHex(yPK))
	//fmt.Println("scalar=" + common.Bytes2Hex(scalar))

	if !crypto.S256().IsOnCurve(big.NewInt(0).SetBytes(xPK), big.NewInt(0).SetBytes(yPK)) {
		return []byte{0}, errors.New("the point is not on curve")
	}

	rx, ry := crypto.S256().ScalarMult(big.NewInt(0).SetBytes(xPK), big.NewInt(0).SetBytes(yPK), scalar)

	if rx == nil || ry == nil {
		return []byte{0}, errors.New("k value is not correct")
	}

	var buf = make([]byte, 64)

	copy(buf, common.LeftPadBytes(rx.Bytes(), 32))
	copy(buf[32:], common.LeftPadBytes(ry.Bytes(), 32))

	return buf, nil
}

func (s *s256ScalarMul) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}
