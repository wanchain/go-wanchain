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
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
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


type pos_staking  struct{}

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

	fmt.Println(""+ common.ToHex(payload))

	var Info struct {
		Pubs 		    string  		//staker’s original public key + bn256 pairing public key
		LockTime	 	*big.Int		//lock time which is input by user
	}

	err := cscAbi.Unpack(&Info, "stakeIn", payload)
	if err != nil {
		return nil, errStakeInAbiParse
	}

	//get public keys

	ss := strings.Split(strings.ToLower(Info.Pubs), "0x")
	fmt.Println("pk1=" + ss[1])
	fmt.Println("pk2=" + ss[2])

	lt := big.NewInt(0).Div(Info.LockTime,ether).Uint64()

	lkperiod := (lt/epochInterval)*epochInterval
	//create staker's information
	staker := &StakerInfo{
							PubSec256:common.FromHex(ss[1]),
							PubBn256:common.FromHex(ss[2]),
							Amount:	contract.value,
							LockTime:lkperiod,
							StakingTime: time.Now().Unix(),
						}


	infoArray,err := json.Marshal(staker)
	if err != nil {
		return nil, err
	}
	pukHash := common.BytesToHash(common.FromHex(ss[1]))

	//store stake info
	res := StoreInfo(evm.StateDB,StakersInfoAddr,pukHash,infoArray)
	if res != nil {
		return nil,res
	}

	if !isRanFake {
		runFake(evm.StateDB)
		isRanFake = true
	}

	return nil,nil

}

func (p *pos_staking) stakeOut(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	fmt.Println(""+ common.ToHex(payload))

	var Info struct {
		Pub 		    string  		//staker’s original public key
		Value	 		*big.Int
	}

	err := cscAbi.Unpack(&Info, "stakeOut", payload)
	if err != nil {
		return nil, errStakeInAbiParse
	}

	pukHash := common.BytesToHash(common.FromHex(Info.Pub))
	infoArray,err := GetInfo(evm.StateDB,StakersInfoAddr,pukHash)
	if err != nil {
		return nil, err
	}

	var staker StakerInfo
	error := json.Unmarshal(infoArray,&staker)
	if error != nil {
		return nil, error
	}

	if staker.PubSec256 == nil {
		return nil,errors.New("staker has unregistered already")
	}


	//store staker info to nil
	nilValue := &StakerInfo{
		PubSec256:nil,
		PubBn256:nil,
		Amount:	big.NewInt(0),
		LockTime:0,
		StakingTime: 0,
	}

	infoArray,err = json.Marshal(nilValue)
	if err != nil {
		return nil,err
	}

	pukHash = common.BytesToHash(staker.PubSec256)
	err = UpdateInfo(evm.StateDB,StakersInfoAddr,pukHash,infoArray)
	if err != nil {
		return nil,err
	}

	if  (staker.StakingTime + int64(staker.LockTime)) > time.Now().Unix() {
		evm.StateDB.AddBalance(contract.CallerAddress, staker.Amount)
		evm.StateDB.SubBalance(wanCscPrecompileAddr,staker.Amount)
	} else {
		return nil,errors.New("lockTIme did not reach")
	}

	return nil,nil
}

func (p *pos_staking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}


func runFake(statedb StateDB) error {
	Ns                         := 100 //num of publickey samples
	secpubs := fakeGenSecPublicKeys(Ns)
	g1pubs := fakeGenG1PublicKeys(Ns)

	for i:=0;i<Ns;i++ {
		staker := &StakerInfo{
			PubSec256:secpubs[i],
			PubBn256:g1pubs[i],
			Amount:	big.NewInt(0).Mul(big.NewInt(int64(mrand.Float32()*1000)),ether),
			LockTime:uint64(mrand.Float32()*10)*3600,
			StakingTime: time.Now().Unix(),
		}

		infoArray,_ := json.Marshal(staker)
		pukHash := common.BytesToHash(staker.PubSec256)

		fmt.Println("generate fake date %d",i)

		StoreInfo(statedb,StakersInfoAddr,pukHash,infoArray)
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