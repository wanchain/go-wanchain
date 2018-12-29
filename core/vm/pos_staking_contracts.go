package vm

import (
	"github.com/wanchain/go-wanchain/accounts/abi"
	"strings"
	"math/big"
	"time"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"encoding/json"
	"github.com/wanchain/go-wanchain/core/types"
	"errors"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/cloudflare"
	"crypto/rand"
	mrand "math/rand"
)

var (
		//cscDefinition = `[{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}],"name":"stakeIn","outputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"Pub","type":"string"}],"name":"stakeOut","outputs":[{"name":"Pub","type":"string"}]}]`
		cscDefinition = `[{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}],"name":"stakeIn","outputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"Pub","type":"string"},{"name":"Value","type":"uint256"}],"name":"stakeOut","outputs":[{"name":"Pub","type":"string"},{"name":"Value","type":"uint256"}]}]`

		cscAbi, errCscInit  = abi.JSON(strings.NewReader(cscDefinition))

		stakeInId  [4]byte
		stakeOutId [4]byte

		errStakeInAbiParse  = errors.New("error in stakein abi parse ")
		errStakeInPubLen =  errors.New("error in getting stake public keys length")
		errStakeInCreatePub  = errors.New("error in stakein creating pub")

		errStakeOutAbiParse  = errors.New("error in stakeout abi parse")

		epochId  uint64
		//this just for test
		posStartTime int64

		epochInterval uint64

		isRanFake = false
		FakeCh = make(chan int)
	)


type StakerInfo struct {
	PubSec256 		    []byte  	//staker’s ethereum public key
	PubBn256 		    []byte  	//staker’s bn256 public key

	Amount      	*big.Int		    //staking wan value
	LockTime	 	uint64			//lock time which is input by user
	StakingTime		int64			//the user’s staking time
}

func init() {

	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(stakeInId[:], 	cscAbi.Methods["stakeIn"].Id())
	copy(stakeOutId[:], cscAbi.Methods["stakeOut"].Id())

	//posStartTime = pos.Cfg().PosStartTime
	//epochInterval = pos.Cfg().EpochInterval
	posStartTime = time.Now().Unix()
	epochInterval = 3600
}


type pos_staking  struct{

}


func (p *pos_staking) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *pos_staking) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {

	if len(input) < 4 {
		return nil,errors.New("parameter is wrong")
	}

	var methodId [4]byte
    copy(methodId[:], input[:4])

	if methodId == stakeInId {
		return p.stakeIn(input[4:], contract, evm)
	} else if methodId == stakeOutId {
		return p.stakeOut(input[4:], contract, evm)
	}

	return nil,nil
}


func (p *pos_staking) stakeIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	secpub,bn256pub,lt := p.stakeInParseAndValid(payload)
	if secpub == nil {
		return nil,errors.New("wrong parameter for stakeIn")
	}

	pukHash := common.BytesToHash(secpub)
	lkperiod := (lt/epochInterval)*epochInterval

	//create staker's information
	staker := &StakerInfo{
		PubSec256:secpub,
		PubBn256:bn256pub,
		Amount:	contract.value,
		LockTime:lkperiod,
		StakingTime: time.Now().Unix(),
	}

	gotInfoArray,err := GetInfo(evm.StateDB,StakersInfoAddr,pukHash)
	if err != nil {
		return nil, err
	} else if(gotInfoArray != nil) {
		var gotStaker StakerInfo
		error := json.Unmarshal(gotInfoArray,&gotStaker)
		if error != nil {
			return nil, error
		}
		//if staker existed already,update value
		if gotStaker.PubSec256 != nil {
			staker.Amount = staker.Amount.Add(staker.Amount,gotStaker.Amount)
			if staker.LockTime < gotStaker.LockTime {
				staker.LockTime = gotStaker.LockTime
			}
		}
	}

	infoArray,err := json.Marshal(staker)
	if err != nil {
		return nil, err
	}

	//store stake info
	res := StoreInfo(evm.StateDB,StakersInfoAddr,pukHash,infoArray)
	if res != nil {
		return nil,res
	}

	if isRanFake {
		runFake(evm.StateDB)
		isRanFake = true
	}

	return nil,nil

}

func (p *pos_staking) stakeOut(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	staker,pubHash,err := p.stakeOutParseAndValid(evm.StateDB,payload)
	if err != nil {
		return nil,err
	}

	//if the time already go beyong staker's staking time, staker can stake out
	if  ( time.Now().Unix() > staker.StakingTime + int64(staker.LockTime))  {

		scBal := evm.StateDB.GetBalance(WanCscPrecompileAddr)
		if scBal.Cmp(staker.Amount) >= 0 {
			evm.StateDB.AddBalance(contract.CallerAddress, staker.Amount)
			evm.StateDB.SubBalance(WanCscPrecompileAddr, staker.Amount)
		} else {
			return nil,errors.New("whole stakes is not enough to pay")
		}

	} else {
		return nil,errors.New("lockTIme did not reach")
	}

	//store staker info to nil
	nilValue := &StakerInfo{
		PubSec256:nil,
		PubBn256:nil,
		Amount:	big.NewInt(0),
		LockTime:0,
		StakingTime: 0,
	}

	nilArray,err := json.Marshal(nilValue)
	if err != nil {
		return nil,err
	}


	err = UpdateInfo(evm.StateDB,StakersInfoAddr,pubHash,nilArray)
	if err != nil {
		return nil,err
	}

	return nil,nil
}

func (p *pos_staking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {

	input := tx.Data()
	if len(input) < 4 {
		return errors.New("parameter is too short")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeInId {
		secpub,_,_ := p.stakeInParseAndValid(input[4:])
		if secpub == nil {
			return errors.New("stakein verify failed")
		}
	} else if methodId == stakeOutId {
		_,_,err := p.stakeOutParseAndValid(stateDB,input[4:])
		if err != nil {
			return errors.New("stakeout verify failed")
		}
	}

	return nil
}


func (p *pos_staking) stakeInParseAndValid(payload []byte)(secPk []byte,bn256Pk []byte,lkt uint64) {

	fmt.Println(""+ common.ToHex(payload))
	var Info struct {
		Pubs 		    string  		//staker’s original public key + bn256 pairing public key
		LockTime	 	*big.Int		//lock time which is input by user
	}

	err := cscAbi.Unpack(&Info, "stakeIn", payload)
	if err != nil {
		return nil, nil,0
	}

	//get public keys
	ss := strings.Split(strings.ToLower(Info.Pubs), "0x")

	secPk = common.FromHex(ss[1])
	pub := crypto.ToECDSAPub(secPk)
	if pub == nil {
		return nil, nil,0
	}

	bn256Pk = common.FromHex(ss[2])
	_,err = new(bn256.G1).Unmarshal(bn256Pk)
	if err != nil {
		return nil, nil,0
	}

	lkt = big.NewInt(0).Div(Info.LockTime,ether).Uint64()

	return secPk,bn256Pk,lkt
}

func (p *pos_staking) stakeOutParseAndValid(stateDB StateDB, payload []byte) (str *StakerInfo,pubHash common.Hash,err error) {

	fmt.Println(""+ common.ToHex(payload))

	var Info struct {
		Pub 		    string  		//staker’s original public key
		Value	 		*big.Int
	}

	err = cscAbi.Unpack(&Info, "stakeOut", payload)
	if err != nil {
		return nil,common.Hash{}, errStakeInAbiParse
	}

	pub := common.FromHex(Info.Pub)

	pubHash = common.BytesToHash(pub)
	infoArray,err := GetInfo(stateDB,StakersInfoAddr,pubHash)
	if err != nil {
		return nil,common.Hash{}, err
	}

	var staker StakerInfo
	error := json.Unmarshal(infoArray,&staker)
	if error != nil {
		return nil, common.Hash{},error
	}

	if staker.PubSec256 == nil {
		return nil,common.Hash{},errors.New("staker has unregistered already")
	}

	return &staker,pubHash,nil
}


func runFake(statedb StateDB) error {
	Ns                         := 100 //num of publickey samples
	secpubs := fakeGenSecPublicKeys(Ns)
	g1pubs := fakeGenG1PublicKeys(Ns)
	mrand.Seed(100000)

	for i:=0;i<Ns;i++ {

		staker := &StakerInfo{
			PubSec256:secpubs[i],
			PubBn256:g1pubs[i],
			Amount:	big.NewInt(0).Mul(big.NewInt(int64(mrand.Float32()*1000)),ether),
			LockTime:uint64(mrand.Float32()*100)*3600,
			StakingTime: time.Now().Unix(),
		}

		infoArray,_ := json.Marshal(staker)
		pukHash := common.BytesToHash(staker.PubSec256)

		StoreInfo(statedb,StakersInfoAddr,pukHash,infoArray)

		infoArray,_= GetInfo(statedb,StakersInfoAddr,pukHash)

		fmt.Println("generate fake date ",infoArray)

	}

	FakeCh <- 1

	return nil
}

func fakeGenSecPublicKeys(x int) ([][]byte) {
	if x <= 0 {
		return nil
	}
	PublicKeys := make([][]byte, 0) //PublicKey Samples

	for i := 0; i < x; i++ {
		privateksample, err := crypto.GenerateKey()
		if err != nil {
			return nil
		}
		PublicKeys = append(PublicKeys,crypto.FromECDSAPub(&privateksample.PublicKey))
	}

	return PublicKeys
}


func fakeGenG1PublicKeys(x int) ([][]byte) {

	g1Pubs := make([][]byte, 0) //PublicKey Samples

	for i := 0; i < x; i++ {
		_, Pub, err := bn256.RandomG1(rand.Reader)
		if err != nil {
			continue
		}
		g1Pubs = append(g1Pubs,Pub.Marshal())
	}

	return g1Pubs
}