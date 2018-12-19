package vm

import (
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"strings"
	"math/big"
	"errors"
	"time"
	"fmt"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/common"
	"encoding/json"
)

var (
		cscDefinition = `[{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"pk","type":"string"},{"name":"lockTime","type":"uint256"}],"name":"stakeIn","outputs":[{"name":"pk","type":"string"},{"name":"lockTime","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"pk","type":"string"}],"name":"stakeOut","outputs":[{"name":"pk","type":"string"}]}]`
		cscAbi, errCscInit  = abi.JSON(strings.NewReader(coinSCDefinition))

		stakeInId  [4]byte
		stakeOutId [4]byte

		errStakeInAbiParse  = errors.New("error in stakein abi parse ")
		errStakeInPubLen =  errors.New("error in getting stake public keys length")
		errStakeInCreatePub  = errors.New("error in stakein creating pub")

		errStakeOutAbiParse  = errors.New("error in stakeout abi parse")

		epochId  uint64
		//this just for test
		posStartTime = time.Now().Unix()

		stakersInfoAddr   = common.BytesToAddress(big.NewInt(400).Bytes())
	)

const (
		 epochInterval = 60
	  )

type stakerInfo struct {
	pubSec256 		    []byte  	//staker’s ethereum public key
	pubBn256 		    []byte  	//staker’s bn256 public key

	amount      	*big.Int		    //staking wan value
	lockPeriod	 	int64			//lock time which is input by user
	stakingTime		int64			//the user’s staking time
}

func init() {

	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(stakeInId[:], 	cscAbi.Methods["stakeIn"].Id())
	copy(stakeOutId[:], cscAbi.Methods["stakeOut"].Id())
}

//this is test time
func scsTiming()  {
	//timer,
	ticker := time.NewTicker(epochInterval * time.Second)
	for {
		time := <-ticker.C

		epochId = uint64(time.Unix() - posStartTime)/epochInterval

		fmt.Println("timer====>",time.String())
	}
}

func getEpochIdAddress(pepochId int64) common.Address {
	msg := fmt.Sprintf("wanchainepochid:%v", pepochId)
	return common.BytesToAddress(crypto.Keccak256([]byte(msg)))
}


type pos_staking  struct{}

func (p *pos_staking) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *pos_staking) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {

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

	var info struct {
		pubs 		    string  		//staker’s original public key + bn256 pairing public key
		lockPeriod	 	*big.Int		//lock time which is input by user
	}

	err := cscAbi.Unpack(&info, "stakeIn", payload)
	if err != nil {
		return nil, errStakeInAbiParse
	}

	//get public keys
	ss := strings.Split(info.pubs, "+")
	if len(ss) < 2 {
		return nil,errStakeInPubLen
	}

	 lkperiod := (info.lockPeriod.Int64()/epochInterval)*epochInterval
	//create staker's information
	staker := &stakerInfo{
							pubSec256:common.FromHex(ss[0]),
							pubBn256:common.FromHex(ss[1]),
							amount:	contract.value,
							lockPeriod:lkperiod,
							stakingTime: time.Now().Unix(),
						}


	infoArray,err := json.Marshal(staker)
	pukHash := common.BytesToHash(common.FromHex(ss[0]))

	//store stake info
	res := StoreInfo(evm.StateDB,stakersInfoAddr,pukHash,infoArray)

	if res != nil {
		return nil,res
	}

	return nil,nil

}

func (p *pos_staking) stakeOut(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {

	var info struct {
		pub 		    string  		//staker’s original public key
	}

	err := cscAbi.Unpack(&info, "stakeOut", payload)
	if err != nil {
		return nil, errStakeInAbiParse
	}

	pukHash := common.BytesToHash(common.FromHex(info.pub))

	infoArray,err := GetInfo(evm.StateDB,stakersInfoAddr,pukHash)

	var staker stakerInfo
	error := json.Unmarshal(infoArray,&staker)
	if error != nil {
		return nil, error
	}


	if  (staker.stakingTime + staker.lockPeriod) > time.Now().Unix() {
		evm.StateDB.AddBalance(contract.CallerAddress, staker.amount)
		evm.StateDB.SubBalance(wanCscPrecompileAddr,staker.amount)
	}

	return nil,nil
}

func (p *pos_staking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}