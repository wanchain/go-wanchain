package vm

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"math/rand"

	"strconv"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
)

// OTABalance2ContractAddr convert ota balance to ota storage address
//
// 1 wancoin --> (bigint)1000000000000000000 --> "0x0000000000000000000001000000000000000000"
//
func OTABalance2ContractAddr(balance *big.Int) common.Address {
	if balance == nil {
		return common.Address{}
	}

	return common.HexToAddress(balance.String())
	//	return common.BigToAddress(balance)
}

// GetAXFromWanAddr retrieve ota AX from ota WanAddr
func GetAXFromWanAddr(otaWanAddr []byte) ([]byte, error) {
	if len(otaWanAddr) != common.WAddressLength {
		return nil, errors.New("invalid input param!")
	}

	return otaWanAddr[1 : 1+common.HashLength], nil
}

// IsAXPointToWanAddr check whether AX point to otaWanAddr or not
func IsAXPointToWanAddr(AX []byte, otaWanAddr []byte) bool {
	findAX, err := GetAXFromWanAddr(otaWanAddr)
	if err != nil {
		return false
	}

	return bytes.Equal(findAX, AX)
}

// GetOtaBalanceFromAX retrieve ota balance from ota AX
func GetOtaBalanceFromAX(statedb StateDB, otaAX []byte) (*big.Int, error) {
	if statedb == nil || len(otaAX) != common.HashLength {
		return nil, errors.New("invalid input param!")
	}

	balance := statedb.GetStateByteArray(otaBalanceStorageAddr, common.BytesToHash(otaAX))
	if len(balance) == 0 {
		return common.Big0, nil
	}

	return new(big.Int).SetBytes(balance), nil
}

// SetOtaBalanceToAX set ota balance as 'balance'. Overwrite if ota balance exist already.
func SetOtaBalanceToAX(statedb StateDB, otaAX []byte, balance *big.Int) error {
	if statedb == nil || len(otaAX) != common.HashLength || balance == nil {
		return errors.New("invalid input param!")
	}

	statedb.SetStateByteArray(otaBalanceStorageAddr, common.BytesToHash(otaAX), balance.Bytes())
	return nil
}

// ChechOTAExist checks the OTA exist or not.
//
// In order to avoid additional ota have conflict with existing,
// even if AX exist in balance storage already, will return true.
func CheckOTAExist(statedb StateDB, otaAX []byte) (exist bool, balance *big.Int, err error) {
	if statedb == nil || len(otaAX) < common.HashLength {
		return false, nil, errors.New("invalid input param!")
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX[:common.HashLength])
	if err != nil {
		return false, nil, err
	}

	if balance.Cmp(common.Big0) == 0 {
		return false, nil, nil
	}

	return true, balance, nil
}

// BatCheckOTAExist batch check the OTAs exist or not.
//
// return true means all OTAs exist and their have same balance
//
func BatCheckOTAExist(statedb StateDB, otaAXs [][]byte) (exist bool, balance *big.Int, unexistOta []byte, err error) {
	if statedb == nil || len(otaAXs) == 0 {
		return false, nil, nil, errors.New("invalid input param!")
	}

	for _, otaAX := range otaAXs {
		if len(otaAX) < common.HashLength {
			return false, nil, otaAX, errors.New("invalid input ota AX!")
		}

		balanceTmp, err := GetOtaBalanceFromAX(statedb, otaAX[:common.HashLength])
		if err != nil {
			return false, nil, otaAX, err
		} else if balanceTmp.Cmp(common.Big0) == 0 {
			return false, nil, otaAX, errors.New("ota balance is 0! ota:" + common.ToHex(otaAX))
		} else if balance == nil {
			balance = balanceTmp
			continue
		} else if balance.Cmp(balanceTmp) != 0 {
			return false, nil, otaAX, errors.New("otas have different balances! ota:" + common.ToHex(otaAX))
		}
	}

	mptAddr := OTABalance2ContractAddr(balance)
	for _, otaAX := range otaAXs {
		otaAddrKey := common.BytesToHash(otaAX)
		otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
		if len(otaValue) == 0 {
			return false, nil, otaAX, errors.New("ota doesn't exist:" + common.ToHex(otaAX))
		}
	}

	return true, balance, nil, nil
}

// setOTA storage ota info, include balance and WanAddr. Overwrite if ota exist already.
func setOTA(statedb StateDB, balance *big.Int, otaWanAddr []byte) error {
	if statedb == nil || balance == nil || len(otaWanAddr) != common.WAddressLength {
		return errors.New("invalid input param!")
	}

	otaAX, _ := GetAXFromWanAddr(otaWanAddr)
	//balanceOld, err := GetOtaBalanceFromAX(statedb, otaAX)
	//if err != nil {
	//	return err
	//}
	//
	//if balanceOld != nil && balanceOld.Cmp(common.Big0) != 0 {
	//	return errors.New("ota balance is not 0! old balance:" + balanceOld.String())
	//}

	mptAddr := OTABalance2ContractAddr(balance)
	statedb.SetStateByteArray(mptAddr, common.BytesToHash(otaAX), otaWanAddr)
	return SetOtaBalanceToAX(statedb, otaAX, balance)
}

// AddOTAIfNotExist storage ota info if doesn't exist already.
func AddOTAIfNotExist(statedb StateDB, balance *big.Int, otaWanAddr []byte) (bool, error) {
	if statedb == nil || balance == nil || len(otaWanAddr) != common.WAddressLength {
		return false, errors.New("invalid input param!")
	}

	otaAX, _ := GetAXFromWanAddr(otaWanAddr)
	otaAddrKey := common.BytesToHash(otaAX)
	exist, _, err := CheckOTAExist(statedb, otaAddrKey[:])
	if err != nil {
		return false, err
	}

	if exist {
		return false, errors.New("ota exist already!")
	}

	err = setOTA(statedb, balance, otaWanAddr)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetOTAInfoFromAX retrieve ota info, include balance and WanAddr
func GetOTAInfoFromAX(statedb StateDB, otaAX []byte) (otaWanAddr []byte, balance *big.Int, err error) {
	if statedb == nil || len(otaAX) < common.HashLength {
		return nil, nil, errors.New("invalid input param!")
	}

	otaAddrKey := common.BytesToHash(otaAX)
	balance, err = GetOtaBalanceFromAX(statedb, otaAddrKey[:])
	if err != nil {
		return nil, nil, err
	}

	if balance == nil || balance.Cmp(common.Big0) == 0 {
		return nil, nil, errors.New("ota balance is 0!")
	}

	mptAddr := OTABalance2ContractAddr(balance)

	otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
	if otaValue != nil && len(otaValue) != 0 {
		return otaValue, balance, nil
	}

	return nil, balance, nil
}

type GetOTASetEnv struct {
	otaAX         []byte
	setNum        int
	getNum        int
	loopTimes     int
	rnd           int
	otaWanAddrSet [][]byte
}

func (env *GetOTASetEnv) OTAInSet(ota []byte) bool {
	for _, exist := range env.otaWanAddrSet {
		if bytes.Equal(exist, ota) {
			return true
		}
	}

	return false
}

func (env *GetOTASetEnv) UpdateRnd() {
	env.rnd = rand.Intn(100) + 1
}

func (env *GetOTASetEnv) IsSetFull() bool {
	return env.getNum >= env.setNum
}

func (env *GetOTASetEnv) RandomSelOTA(value []byte) bool {
	env.loopTimes++
	if env.loopTimes%env.rnd == 0 {
		env.otaWanAddrSet = append(env.otaWanAddrSet, value)
		env.getNum++
		env.UpdateRnd()
		return true
	} else {
		return false
	}
}

// doOTAStorageTravelCallBack implement ota mpt travel call back
func doOTAStorageTravelCallBack(env *GetOTASetEnv, value []byte) (bool, error) {
	// find self, return true to continue travel loop
	if IsAXPointToWanAddr(env.otaAX, value) {
		return true, nil
	}

	// ota contained in set already, return true to continue travel loop
	if env.OTAInSet(value) {
		return true, nil
	}

	// random select
	// if set full already, return false to stop travel loop
	if bGet := env.RandomSelOTA(value); bGet {
		return !env.IsSetFull(), nil
	} else {
		return true, nil
	}
}

// GetOTASet retrieve the setNum of same balance OTA address of the input OTA setting by otaAX, and ota balance.
// Rules:
//		1: The result can't contain otaAX self;
//		2: The result can't contain duplicate items;
//		3: No ota exist in the mpt, return error;
//		4: OTA total count in the mpt less or equal to the setNum, return error(returned set must
//		   can't contain otaAX self, so need more exist ota in mpt);
//		5: If find invalid ota wanaddr, return error;
//		6: Travel the ota mpt.Record loop exist ota cumulative times as loopTimes.
// 		   Generate a random number as rnd.
// 		   If loopTimes%rnd == 0, collect current exist ota to result set and update the rnd.
//		   Loop checking exist ota and loop traveling ota mpt, untile collect enough ota or find error.
//
func GetOTASet(statedb StateDB, otaAX []byte, setNum int) (otaWanAddrs [][]byte, balance *big.Int, err error) {
	if statedb == nil || len(otaAX) != common.HashLength {
		return nil, nil, errors.New("invalid input param!")
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		return nil, nil, err
	} else if balance == nil || balance.Cmp(common.Big0) == 0 {
		return nil, nil, errors.New("can't find ota address balance!")
	}

	mptAddr := OTABalance2ContractAddr(balance)
	log.Debug("GetOTASet", "mptAddr", common.ToHex(mptAddr[:]))

	env := GetOTASetEnv{otaAX, setNum, 0, 0, 0, nil}
	env.otaWanAddrSet = make([][]byte, 0, setNum)
	env.UpdateRnd()

	mptEleCount := 0 // total number of ota containing in mpt

	for {
		statedb.ForEachStorageByteArray(mptAddr, func(key common.Hash, value []byte) bool {
			mptEleCount++

			if len(value) != common.WAddressLength {
				log.Error("invalid OTA address!", "balance", balance, "value", value)
				err = errors.New(fmt.Sprint("invalid OTA address! balance:", balance, ", ota:", value))
				return false
			}

			bContinue, err := doOTAStorageTravelCallBack(&env, value)
			if err != nil {
				return false
			} else {
				return bContinue
			}
		})

		if env.IsSetFull() {
			return env.otaWanAddrSet, balance, nil
		} else if err != nil {
			return nil, nil, err
		} else if mptEleCount == 0 {
			return nil, balance, errors.New("no ota exist! balance:" + balance.String())
		} else if setNum >= mptEleCount {
			return nil, balance, errors.New("too more required ota number! balance:" + balance.String() +
				", exist count:" + strconv.Itoa(mptEleCount))
		} else {
			continue
		}
	}
}

// CheckOTAImageExist checks ota image key exist already or not
func CheckOTAImageExist(statedb StateDB, otaImage []byte) (bool, []byte, error) {
	if statedb == nil || len(otaImage) == 0 {
		return false, nil, errors.New("invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	otaImageValue := statedb.GetStateByteArray(otaImageStorageAddr, otaImageKey)
	if otaImageValue != nil && len(otaImageValue) != 0 {
		return true, otaImageValue, nil
	}

	return false, nil, nil
}

// AddOTAImage storage ota image key. Overwrite if exist already.
func AddOTAImage(statedb StateDB, otaImage []byte, value []byte) error {
	if statedb == nil || len(otaImage) == 0 || len(value) == 0 {
		return errors.New("invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	statedb.SetStateByteArray(otaImageStorageAddr, otaImageKey, value)
	return nil
}
