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
		posStartTime int64
		epochInterval uint64
	)


type StakerInfo struct {
	PubSec256 		    []byte  	//staker’s ethereum public key
	PubBn256 		    []byte  	//staker’s bn256 public key

	Amount      	*big.Int		    //staking wan value
	LockPeriod	 	uint64			//lock time which is input by user
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

	lkperiod := (info.lockPeriod.Uint64()/epochInterval)*epochInterval
	//create staker's information
	staker := &StakerInfo{
							PubSec256:common.FromHex(ss[0]),
							PubBn256:common.FromHex(ss[1]),
							Amount:	contract.value,
							LockPeriod:lkperiod,
							StakingTime: time.Now().Unix(),
						}


	infoArray,err := json.Marshal(staker)
	pukHash := common.BytesToHash(common.FromHex(ss[0]))

	//store stake info
	res := StoreInfo(evm.StateDB,StakersInfoAddr,pukHash,infoArray)

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

	infoArray,err := GetInfo(evm.StateDB,StakersInfoAddr,pukHash)

	var staker StakerInfo
	error := json.Unmarshal(infoArray,&staker)
	if error != nil {
		return nil, error
	}


	if  (staker.StakingTime + int64(staker.LockPeriod)) > time.Now().Unix() {
		evm.StateDB.AddBalance(contract.CallerAddress, staker.Amount)
		evm.StateDB.SubBalance(wanCscPrecompileAddr,staker.Amount)
	}

	return nil,nil
}

func (p *pos_staking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}