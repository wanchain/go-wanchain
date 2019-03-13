package vm

import (
	"errors"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/postools"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
	"strings"
)

/* the contract interface described by solidity.
contract stake {
	function stakeIn( string memory Pubs, uint256 LockEpochs) public {}
	function stakeOut(string memory Pub, uint256 Value) public pure {}
}

contract stake {
	function stakeIn( string memory secPk, string memory bnPub, uint256 lockEpochs, uint256 feeRate) public payable {}
	// function stakeOut(string memory sPub) public pure {} // TODO: need it?
	function delegateIn(address delegateAddr, uint256 lockEpochs) public payable {}
}

*/

var (
//	cscDefinition = `
//[
//	{
//		"constant": false,
//		"inputs": [
//			{
//				"name": "secPk",
//				"type": "bytes"
//			},
//			{
//				"name": "bn256Pk",
//				"type": "bytes"
//			},
//			{
//				"name": "lockEpochs",
//				"type": "uint256"
//			},
//			{
//				"name": "feeRate",
//				"type": "uint256"
//			}
//		],
//		"name": "stakeIn",
//		"outputs": [
//			{
//				"name": "secPk",
//				"type": "bytes"
//			},
//			{
//				"name": "bn256Pk",
//				"type": "bytes"
//			},
//			{
//				"name": "lockEpochs",
//				"type": "uint256"
//			},
//			{
//				"name": "feeRate",
//				"type": "uint256"
//			}
//		],
//		"payable": true,
//		"stateMutability": "payable",
//		"type": "function"
//	},
//	{
//		"constant": false,
//		"inputs": [
//			{
//				"name": "delegateAddr",
//				"type": "address"
//			},
//			{
//				"name": "lockEpochs",
//				"type": "uint256"
//			}
//		],
//		"name": "delegateIn",
//		"outputs": [
//			{
//				"name": "delegateAddr",
//				"type": "address"
//			},
//			{
//				"name": "lockEpochs",
//				"type": "uint256"
//			}
//		],
//		"payable": true,
//		"stateMutability": "payable",
//		"type": "function"
//	}
//]
//`
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
        "outputs": [],
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
			}
		],
		"name": "delegateIn",
        "outputs": [],
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

	//errStakeInAbiParse  = errors.New("error in stakein abi parse ")
	//
	//posStartTime int64

	kindStakeIn   = []byte{100}

	//posEpochGap = uint64(2)
	//posDelegateEpochGap = uint64(4)
	maxEpochNum = uint64(1000)
	minEpochNum = uint64(1)
	minStakeholderStake = big.NewInt(10000)
	minDelegateStake = big.NewInt(10000)
	minFeeRate = big.NewInt(0)
	maxFeeRate = big.NewInt(100)

	//epochInterval uint64

	//isRanFake = false
	//FakeCh = make(chan int)
)

type StakerInfo struct {
	Address	    common.Address
	PubSec256   []byte //stakeholder’s wan public key
	PubBn256    []byte //stakeholder’s bn256 public key

	Amount      *big.Int //staking wan value
	LockEpochs   uint64   //lock time which is input by user
	From        common.Address

	StakingEpochs uint64 //the user’s staking time
	FeeRate     uint64
	Clients      []ClientInfo
}

type Leader struct {
	PubSec256     []byte
	PubBn256      []byte
	SecAddr       common.Address
	FromAddr      common.Address
	Probabilities *big.Int
}

type ClientInfo struct {
	Address common.Address
	Amount   *big.Int
	//LockEpochs uint64
}

type ClientProbability struct {
	Addr        common.Address
	Probability *big.Int
}

type ClientIncentive struct {
	Addr      common.Address
	Incentive *big.Int
}

type StakeInParam struct {
	SecPk      []byte   //stakeholder’s original public key
	Bn256Pk    []byte   //stakeholder’s bn256 pairing public key
	LockEpochs *big.Int //lock time which is input by user
	FeeRate    *big.Int //lock time which is input by user
}

type DelegateInParam struct {
	DelegateAddress common.Address   //delegation’s address
	//LockEpochs    *big.Int //lock time which is input by user
}

func init() {
	if errCscInit != nil {
		panic("err in csc abi initialize ")
	}

	copy(stakeInId[:], cscAbi.Methods["stakeIn"].Id())
	copy(stakeOutId[:], cscAbi.Methods["stakeOut"].Id())
	copy(delegateId[:], cscAbi.Methods["delegateIn"].Id())
}

type PosStaking struct {
}

func (p *PosStaking) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *PosStaking) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(input) < 4 {
		return nil, errors.New("parameter is wrong")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeInId {
		return p.StakeIn(input[4:], contract, evm)
	} else if methodId == delegateId {
		return p.DelegateIn(input[4:], contract, evm)
	}
	//else if methodId == stakeOutId {
	//	return p.StakeOut(input[4:], contract, evm)
	//}

	return nil, nil
}

func GetStakeInKeyHash(address common.Address) common.Hash {
	return common.BytesToHash(address[:])
}

func (p *PosStaking) StakeIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var info StakeInParam
	err := cscAbi.UnpackInput(&info, "stakeIn", payload)
	if err != nil {
		return nil, err
	}

	// 1. SecPk is valid
	if info.SecPk == nil {
		return nil, errors.New("wrong parameter for stakeIn")
	}
	pub := crypto.ToECDSAPub(info.SecPk)
	if nil == pub {
		return nil, errors.New("secPub is invalid")
	}

	// 2. Lock time >= min epoch, <= max epoch
	lockTime := info.LockEpochs.Uint64()
	if lockTime < minEpochNum || lockTime > maxEpochNum {
		return nil, errors.New("invalid lock time")
	}

	// 3. 0 <= FeeRate <= 100
	if info.FeeRate.Cmp(maxFeeRate) > 0 || info.FeeRate.Cmp(minFeeRate) < 0 {
		return nil, errors.New("fee rate should between 0 to 100")
	}

	// TODO: need max?
	// 4. amount >= min, (<= max ------- amount = self + delegate's, not to do)
	if contract.value.Cmp(minStakeholderStake) < 0 {
		return nil, errors.New("need more Wan to be a stake holder")
	}
	secAddr := crypto.PubkeyToAddress(*pub)

	// 5. secAddr has not join the pos or has finished
	key := GetStakeInKeyHash(secAddr)
	oldInfo, err := GetInfo(evm.StateDB, StakersInfoAddr, key)
	// a. is secAddr joined?
	if oldInfo != nil {
		return nil, errors.New("public Sec address is waiting for settlement")
	}

	// create stakeholder's information
	eidNow, _ := postools.CalEpochSlotID(evm.Time.Uint64())
	stakeholder := &StakerInfo{
		Address:     secAddr,
		PubSec256:   info.SecPk,
		PubBn256:    info.Bn256Pk,
		Amount:      contract.value,
		LockEpochs:    lockTime,
		From:        contract.CallerAddress,
		StakingEpochs: eidNow,
	}
	infoBytes, err := rlp.EncodeToBytes(stakeholder)
	if err != nil {
		return nil, err
	}

	//store stake info
	res := StoreInfo(evm.StateDB, StakersInfoAddr, key, infoBytes)
	if res != nil {
		return nil, res
	}

	return nil, nil
}

func (p *PosStaking) DelegateIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var delegateInParam DelegateInParam
	err := cscAbi.UnpackInput(&delegateInParam, "delegateIn", payload)
	if err != nil {
		return nil, err
	}

	// 1. amount is valid
	if contract.value.Cmp(minDelegateStake) < 0 {
		return nil, errors.New("")
	}

	// 2. mandatory is a valid stakeholder
	addr := delegateInParam.DelegateAddress
	sKey := GetStakeInKeyHash(addr)
	stakerBytes, err := GetInfo(evm.StateDB, StakersInfoAddr, sKey)
	if stakerBytes == nil {
		return nil, errors.New("mandatory doesn't exist")
	}

	var stakerInfo StakerInfo
	err = rlp.DecodeBytes(stakerBytes, &stakerInfo)
	if err != nil {
		return nil, errors.New("parse staker info error")
	}

	// 3. epoch is valid
	//lockEpochs := delegateInParam.LockEpochs.Uint64()
	//eidNow, _ := postools.CalEpochSlotID(evm.Time.Uint64())
	//eidEnd := eidNow + lockEpochs + posEpochGap
	//
	//dEidEnd := stakerInfo.StakingEpochs + stakerInfo.LockEpochs + posEpochGap - posDelegateEpochGap
	//if eidNow < stakerInfo.StakingEpochs || eidNow > dEidEnd || eidEnd > dEidEnd {
	//	return nil, errors.New("it's too late for your to delegate")
	//}

	// 4. sender has not delegated by this
	length := len(stakerInfo.Clients)
	for i:=0; i<length; i++ {
		if stakerInfo.Clients[i].Address == contract.CallerAddress {
			return nil, errors.New("duplicate delegate")
		}
	}

	// save
	info := &ClientInfo {
		Address: contract.CallerAddress,
		Amount: contract.value,
		//LockEpochs: uint64(0),
	}
	stakerInfo.Clients = append(stakerInfo.Clients, *info)

	stakerInfoBytes, err := rlp.EncodeToBytes(stakerInfo)
	if err != nil {
		return nil, err
	}

	res := StoreInfo(evm.StateDB, StakersInfoAddr, sKey, stakerInfoBytes)
	if res != nil {
		return nil, res
	}

	return nil, nil
}

//func (p *PosStaking) StakeOut(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
//	stakeholder, pubHash, err := p.stakeOutParseAndValid(evm.StateDB, payload)
//	if err != nil {
//		return nil, err
//	}
//
//	//if the time already go beyong stakeholder's staking time, stakeholder can stake out
//	if uint64(time.Now().Unix()) > stakeholder.StakingEpochs + uint64(stakeholder.LockEpochs) {
//
//		scBal := evm.StateDB.GetBalance(WanCscPrecompileAddr)
//		if scBal.Cmp(stakeholder.Amount) >= 0 {
//			evm.StateDB.AddBalance(contract.CallerAddress, stakeholder.Amount)
//			evm.StateDB.SubBalance(WanCscPrecompileAddr, stakeholder.Amount)
//		} else {
//			return nil, errors.New("whole stakes is not enough to pay")
//		}
//
//	} else {
//		return nil, errors.New("lockTIme did not reach")
//	}
//
//	//store stakeholder info to nil
//	nilValue := &StakerInfo{
//		PubSec256:   nil,
//		PubBn256:    nil,
//		Amount:      big.NewInt(0),
//		LockEpochs:    0,
//		StakingEpochs: 0,
//	}
//
//	nilArray, err := json.Marshal(nilValue)
//	if err != nil {
//		return nil, err
//	}
//
//	err = UpdateInfo(evm.StateDB, StakersInfoAddr, pubHash, nilArray)
//	if err != nil {
//		return nil, err
//	}
//
//	return nil, nil
//}

func (p *PosStaking) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	input := tx.Data()
	if len(input) < 4 {
		return errors.New("parameter is too short")
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == stakeInId {
		err := p.stakeInParseAndValid(input[4:])
		if err != nil {
			return errors.New("stakein verify failed")
		}
	} else if methodId == delegateId {
		err := p.delegateInParseAndValid(input[4:])
		if err != nil {
			return errors.New("delegateIn verify failed")
		}
	}
	//else if methodId == stakeOutId {
	//	_, _, err := p.stakeOutParseAndValid(stateDB, input[4:])
	//	if err != nil {
	//		return errors.New("stakeout verify failed " + err.Error())
	//	}
	//}

	return nil
}

func (p *PosStaking) stakeInParseAndValid(payload []byte) error {
	var info StakeInParam
	err := cscAbi.UnpackInput(&info, "stakeIn", payload)
	if err != nil {
		return err
	}
	return nil
}

func (p *PosStaking) delegateInParseAndValid(payload []byte) error {
	var delegateInParam DelegateInParam
	err := cscAbi.UnpackInput(&delegateInParam, "delegateIn", payload)
	if err != nil {
		return err
	}

	return nil
}

//func (p *PosStaking) stakeOutParseAndValid(stateDB StateDB, payload []byte) (str *StakerInfo, pubHash common.Hash, err error) {
//
//	fmt.Println("" + common.ToHex(payload))
//
//	var Info struct {
//		Pub   string //staker’s original public key
//		Value *big.Int
//	}
//
//	err = cscAbi.Unpack(&Info, "stakeOut", payload)
//	if err != nil {
//		return nil, common.Hash{}, errStakeInAbiParse
//	}
//
//	pub := common.FromHex(Info.Pub)
//
//	pubHash = common.BytesToHash(pub)
//	infoArray, err := GetInfo(stateDB, StakersInfoAddr, pubHash)
//	if infoArray == nil {
//		return nil, common.Hash{}, errors.New("not find staker staking info")
//	}
//
//	var staker StakerInfo
//	err = json.Unmarshal(infoArray, &staker)
//	if err != nil {
//		return nil, common.Hash{}, err
//	}
//
//	if staker.PubSec256 == nil {
//		return nil, common.Hash{}, errors.New("staker has unregistered already")
//	}
//
//	return &staker, pubHash, nil
//}
//
//func runFake(statedb StateDB) error {
//	// num of public key samples
//	Ns := 100
//	secPubs := fakeGenSecPublicKeys(Ns)
//	g1pubs := fakeGenG1PublicKeys(Ns)
//	mrand.Seed(100000)
//
//	for i := 0; i < Ns; i++ {
//		stakeholder := &StakerInfo{
//			PubSec256:   secPubs[i],
//			PubBn256:    g1pubs[i],
//			Amount:      big.NewInt(0).Mul(big.NewInt(int64(mrand.Float32()*1000)), ether),
//			LockEpochs:    uint64(mrand.Float32()*100) * 3600,
//			StakingEpochs: uint64(time.Now().Unix()),
//		}
//
//		infoArray, _ := json.Marshal(stakeholder)
//		pukHash := common.BytesToHash(stakeholder.PubSec256)
//
//		StoreInfo(statedb, StakersInfoAddr, pukHash, infoArray)
//
//		infoArray, _ = GetInfo(statedb, StakersInfoAddr, pukHash)
//
//		fmt.Println("generate fake date ", infoArray)
//
//	}
//
//	println(posStartTime)
//
//	FakeCh <- 1
//
//	return nil
//}
//
//func fakeGenSecPublicKeys(x int) [][]byte {
//	if x <= 0 {
//		return nil
//	}
//	PublicKeys := make([][]byte, 0) //PublicKey Samples
//
//	for i := 0; i < x; i++ {
//		privateKeySample, err := crypto.GenerateKey()
//		if err != nil {
//			return nil
//		}
//		PublicKeys = append(PublicKeys, crypto.FromECDSAPub(&privateKeySample.PublicKey))
//	}
//
//	return PublicKeys
//}
//
//func fakeGenG1PublicKeys(x int) [][]byte {
//
//	g1Pubs := make([][]byte, 0) //PublicKey Samples
//
//	for i := 0; i < x; i++ {
//		_, Pub, err := bn256.RandomG1(rand.Reader)
//		if err != nil {
//			continue
//		}
//		g1Pubs = append(g1Pubs, Pub.Marshal())
//	}
//
//	return g1Pubs
//}


func GetStakersSnap(stateDb *state.StateDB) ([]StakerInfo, error) {
	stakers := make([]StakerInfo,0)
	stateDb.ForEachStorageByteArray(StakersInfoAddr, func(key common.Hash, value []byte) bool {
		var stakerInfo StakerInfo
		err := rlp.DecodeBytes(value, &stakerInfo)
		if err != nil {
			log.Info(err.Error())
			return true
		}
		stakers = append(stakers, stakerInfo)
		return true
	})
	return stakers, nil
}
var 	StakersInfoStakeOutKeyHash      = common.BytesToHash(big.NewInt(700).Bytes())
func stakeoutSetEpoch(stateDb *state.StateDB,epochID uint64) {
	b := big.NewInt(int64(epochID))
	StoreInfo(stateDb, StakersInfoAddr, StakersInfoStakeOutKeyHash, b.Bytes())
}
func stakeoutIsFinished(stateDb *state.StateDB,epochID uint64) (bool) {
	epochByte,err := GetInfo(stateDb, StakersInfoAddr, StakersInfoStakeOutKeyHash)
	if err != nil {
		return false
	}
	finishedEpochId := big.NewInt(0).SetBytes(epochByte).Uint64()
	return finishedEpochId >= epochID
}
func stakeOutRun(chain consensus.ChainReader, stateDb *state.StateDB, epochID uint64, blockNumber uint64) bool {
	return true
}