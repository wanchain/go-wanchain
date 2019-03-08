package vm

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/cloudflare"
)

/* the contract interface described by solidity.
contract stake {
	function stakeIn( string memory Pubs, uint256 LockTime) public {}
	function stakeOut(string memory Pub, uint256 Value) public pure {}
}

contract stake {
	function stakeIn( string memory secPk, string memory bnPub, uint256 lockEpochs, uint256 feeRate) public payable {}
	// function stakeOut(string memory sPub) public pure {} // TODO: need it?
	function delegateIn(address delegateAddr, uint256 lockEpochs) public payable {}
}

*/

var (
	//cscDefinition = `[{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}],"name":"stakeIn","outputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"Pub","type":"string"}],"name":"stakeOut","outputs":[{"name":"Pub","type":"string"}]}]`
	//cscDefinition = `[{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}],"name":"stakeIn","outputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"Pub","type":"string"},{"name":"Value","type":"uint256"}],"name":"stakeOut","outputs":[{"name":"Pub","type":"string"},{"name":"Value","type":"uint256"}]}]`
	//cscDefinition = `[{"constant":false,"inputs":[{"name":"sPub","type":"string"},{"name":"bnPub","type":"string"},{"name":"lockEpochs","type":"uint256"},{"name":"feeRate","type":"uint256"}],"name":"stakeIn","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"delegateSpub","type":"string"}],"name":"delegateIn","outputs":[],"payable":false,"type":"function"}]`
	cscDefinition = `
[
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
		"outputs": [
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
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "delegateAddr",
				"type": "address"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			}
		],
		"name": "delegateIn",
		"outputs": [
			{
				"name": "delegateAddr",
				"type": "address"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			}
		],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	}
]
`
	cscAbi, errCscInit = abi.JSON(strings.NewReader(cscDefinition))

	stakeInId  [4]byte
	stakeOutId [4]byte
	delegateId [4]byte

	errStakeInAbiParse  = errors.New("error in stakein abi parse ")
	errStakeInPubLen    = errors.New("error in getting stake public keys length")
	errStakeInCreatePub = errors.New("error in stakein creating pub")

	errStakeOutAbiParse = errors.New("error in stakeout abi parse")

	//epochId  uint64
	//this just for test
	posStartTime int64

	epochInterval uint64

	//isRanFake = false
	FakeCh    = make(chan int)
)

type StakerInfo struct {
	PubSec256 []byte //staker’s ethereum public key
	PubBn256  []byte //staker’s bn256 public key

	Amount      *big.Int //staking wan value
	LockTime    uint64   //lock time which is input by user
	From		common.Address
	StakingTime uint64    //the user’s staking time
	FeeRate		uint64
}

type ClientInfo struct {
	Delegate common.Address
	Amount  *big.Int
	LockTime	uint64
}
type ClientProbability struct {
	Addr common.Address
	Probability  *big.Int
}
type ClientIncentive struct {
	Addr common.Address
	Incentive  *big.Int
}
func init() {

	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(stakeInId[:], cscAbi.Methods["stakeIn"].Id())
	copy(stakeOutId[:], cscAbi.Methods["stakeOut"].Id())
	copy(delegateId[:], cscAbi.Methods["delegateIn"].Id())

	//posStartTime = pos.Cfg().PosStartTime
	//epochInterval = pos.Cfg().EpochInterval
	posStartTime = time.Now().Unix()
	epochInterval = 3600
}

type Pos_staking struct {
}

func (p *Pos_staking) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *Pos_staking) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(input) < 4 {
		return nil, errors.New("parameter is wrong")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeInId {
		return p.StakeIn(input[4:], contract, evm)
	} else if methodId == delegateId {
		return p.DelegateIn(input[4:], contract, evm)
	} else if methodId == stakeOutId {
		return p.StakeOut(input[4:], contract, evm)
	}

	return nil, nil
}

func (p *Pos_staking) StakeIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	secpub, bn256pub, lt := p.stakeInParseAndValid(payload)
	if secpub == nil {
		return nil, errors.New("wrong parameter for stakeIn")
	}
	//bpub, err := hex.DecodeString(secpub)
	//if(nil != err){
	//	return nil, errors.New("secpub is invalid")
	//}
	pub := crypto.ToECDSAPub(secpub)
	if(nil == pub){
		return nil, errors.New("secpub is invalid")
	}

	secAddr := crypto.PubkeyToAddress(*pub)
	pukHash := common.BytesToHash(secAddr[:])
	lkperiod := (lt / epochInterval) * epochInterval

	//create staker's information
	staker := &StakerInfo{
		PubSec256:   secpub,
		PubBn256:    bn256pub,
		Amount:      contract.value,
		LockTime:    lkperiod,
		From:		 contract.CallerAddress,
		StakingTime: uint64(evm.Time.Int64()),
	}

	gotInfoArray, err := GetInfo(evm.StateDB, StakersInfoAddr, pukHash)
	if gotInfoArray != nil {
		return nil, errors.New("public key registed")
		//var gotStaker StakerInfo
		//error := json.Unmarshal(gotInfoArray, &gotStaker)
		//if error != nil {
		//	return nil, error
		//}
		////if staker existed already,update value
		//if gotStaker.PubSec256 != nil {
		//	staker.Amount = staker.Amount.Add(staker.Amount, gotStaker.Amount)
		//	if staker.LockTime < gotStaker.LockTime {
		//		staker.LockTime = gotStaker.LockTime
		//	}
		//}
	}

	infoArray, err := json.Marshal(staker)
	if err != nil {
		return nil, err
	}

	//store stake info
	res := StoreInfo(evm.StateDB, StakersInfoAddr, pukHash, infoArray)
	if res != nil {
		return nil, res
	}

	//if isRanFake {
	//	runFake(evm.StateDB)
	//	isRanFake = true
	//}

	return nil, nil

}
func (p *Pos_staking) DelegateIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	addr, err := p.delegateInParseAndValid(payload)
	if err != nil {
		return nil, err
	}

	delegateHash := common.BytesToHash(addr[:])
	gotInfoArray, err := GetInfo(evm.StateDB, StakersInfoAddr, delegateHash)
	if gotInfoArray == nil {
		return nil, errors.New("delegate doesn't exist")
	}

	clientAddr := contract.CallerAddress
	clientHash := common.BytesToHash(clientAddr[:])

	clientInfoArray, err := GetInfo(evm.StateDB, addr, clientHash)
	if clientInfoArray != nil {
		return nil, errors.New("address has registed in this delegate")
	}
	info := &ClientInfo{Delegate: addr, Amount: big.NewInt(0), LockTime: uint64(0)}
	infoArray, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	res := StoreInfo(evm.StateDB, addr, clientHash, infoArray)
	if res != nil {
		return nil, res
	}

	return nil, nil
}
func (p *Pos_staking) StakeOut(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	staker, pubHash, err := p.stakeOutParseAndValid(evm.StateDB, payload)
	if err != nil {
		return nil, err
	}

	//if the time already go beyong staker's staking time, staker can stake out
	if uint64(time.Now().Unix()) > staker.StakingTime+uint64(staker.LockTime) {

		scBal := evm.StateDB.GetBalance(WanCscPrecompileAddr)
		if scBal.Cmp(staker.Amount) >= 0 {
			evm.StateDB.AddBalance(contract.CallerAddress, staker.Amount)
			evm.StateDB.SubBalance(WanCscPrecompileAddr, staker.Amount)
		} else {
			return nil, errors.New("whole stakes is not enough to pay")
		}

	} else {
		return nil, errors.New("lockTIme did not reach")
	}

	//store staker info to nil
	nilValue := &StakerInfo{
		PubSec256:   nil,
		PubBn256:    nil,
		Amount:      big.NewInt(0),
		LockTime:    0,
		StakingTime: 0,
	}

	nilArray, err := json.Marshal(nilValue)
	if err != nil {
		return nil, err
	}

	err = UpdateInfo(evm.StateDB, StakersInfoAddr, pubHash, nilArray)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *Pos_staking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {

	input := tx.Data()
	if len(input) < 4 {
		return errors.New("parameter is too short")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeInId {
		secpub, _, _ := p.stakeInParseAndValid(input[4:])
		if secpub == nil {
			return errors.New("stakein verify failed")
		}
	} else if methodId == delegateId {
		// TODO validate DelegateIn
		_, err := p.delegateInParseAndValid(input[4:])
		if err != nil {
			return errors.New("delegateIn verify failed")
		}
	} else if methodId == stakeOutId {
		_, _, err := p.stakeOutParseAndValid(stateDB, input[4:])
		if err != nil {
			return errors.New("stakeout verify failed " + err.Error())
		}
	}

	return nil
}

func (p *Pos_staking) stakeInParseAndValid(payload []byte) ( []byte,  []byte,  uint64) {

	fmt.Println("" + common.ToHex(payload))
	var Info struct {
		SecPk     	[]byte   //staker’s original public key + bn256 pairing public key
		Bn256Pk     []byte   //staker’s original public key + bn256 pairing public key
		LockEpochs *big.Int //lock time which is input by user
		FeeRate *big.Int //lock time which is input by user
	}

	err := cscAbi.Unpack(&Info, "stakeIn", payload)
	if err != nil {
		return nil, nil, 0
	}

	////get public keys
	//ss := strings.Split(strings.ToLower(Info.Pubs), "0x")
	//
	//secPk = common.FromHex(ss[1])
	//pub := crypto.ToECDSAPub(secPk)
	//if pub == nil {
	//	return nil, nil, 0
	//}
	//
	//bn256Pk = common.FromHex(ss[2])
	//_, err = new(bn256.G1).Unmarshal(bn256Pk)
	//if err != nil {
	//	return nil, nil, 0
	//}
	//
	//lkt = big.NewInt(0).Div(Info.LockTime, ether).Uint64()

	return Info.SecPk, Info.Bn256Pk, Info.LockEpochs.Uint64()
}
func (p *Pos_staking) delegateInParseAndValid(payload []byte) ( common.Address, error) {

	fmt.Println("" + common.ToHex(payload))
	var Info struct {
		DelegateAddr common.Address
	}

	err := cscAbi.Unpack(&Info, "delegateIn", payload)
	if err != nil {
		return common.Address{}, err
	}

	return Info.DelegateAddr, nil
}
func (p *Pos_staking) stakeOutParseAndValid(stateDB StateDB, payload []byte) (str *StakerInfo, pubHash common.Hash, err error) {

	fmt.Println("" + common.ToHex(payload))

	var Info struct {
		Pub   string //staker’s original public key
		Value *big.Int
	}

	err = cscAbi.Unpack(&Info, "stakeOut", payload)
	if err != nil {
		return nil, common.Hash{}, errStakeInAbiParse
	}

	pub := common.FromHex(Info.Pub)

	pubHash = common.BytesToHash(pub)
	infoArray, err := GetInfo(stateDB, StakersInfoAddr, pubHash)
	if infoArray == nil {
		return nil, common.Hash{}, errors.New("not find staker staking info")
	}

	var staker StakerInfo
	err = json.Unmarshal(infoArray, &staker)
	if err != nil {
		return nil, common.Hash{}, err
	}

	if staker.PubSec256 == nil {
		return nil, common.Hash{}, errors.New("staker has unregistered already")
	}

	return &staker, pubHash, nil
}

func runFake(statedb StateDB) error {
	Ns := 100 //num of publickey samples
	secpubs := fakeGenSecPublicKeys(Ns)
	g1pubs := fakeGenG1PublicKeys(Ns)
	mrand.Seed(100000)

	for i := 0; i < Ns; i++ {

		staker := &StakerInfo{
			PubSec256:   secpubs[i],
			PubBn256:    g1pubs[i],
			Amount:      big.NewInt(0).Mul(big.NewInt(int64(mrand.Float32()*1000)), ether),
			LockTime:    uint64(mrand.Float32()*100) * 3600,
			StakingTime: uint64(time.Now().Unix()),
		}

		infoArray, _ := json.Marshal(staker)
		pukHash := common.BytesToHash(staker.PubSec256)

		StoreInfo(statedb, StakersInfoAddr, pukHash, infoArray)

		infoArray, _ = GetInfo(statedb, StakersInfoAddr, pukHash)

		fmt.Println("generate fake date ", infoArray)

	}

	FakeCh <- 1

	return nil
}

func fakeGenSecPublicKeys(x int) [][]byte {
	if x <= 0 {
		return nil
	}
	PublicKeys := make([][]byte, 0) //PublicKey Samples

	for i := 0; i < x; i++ {
		privateksample, err := crypto.GenerateKey()
		if err != nil {
			return nil
		}
		PublicKeys = append(PublicKeys, crypto.FromECDSAPub(&privateksample.PublicKey))
	}

	return PublicKeys
}

func fakeGenG1PublicKeys(x int) [][]byte {

	g1Pubs := make([][]byte, 0) //PublicKey Samples

	for i := 0; i < x; i++ {
		_, Pub, err := bn256.RandomG1(rand.Reader)
		if err != nil {
			continue
		}
		g1Pubs = append(g1Pubs, Pub.Marshal())
	}

	return g1Pubs
}
