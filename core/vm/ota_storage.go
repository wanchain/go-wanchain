package vm

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"math/rand"

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

// GetOtaBalanceFromWanAddr retrieve ota balance from ota WanAddr
func GetOtaBalanceFromWanAddr(statedb StateDB, otaWanAddr []byte) (*big.Int, error) {
	if statedb == nil || len(otaWanAddr) != common.WAddressLength {
		return nil, errors.New("invalid input param!")
	}

	otaAX, _ := GetAXFromWanAddr(otaWanAddr)
	balance := statedb.GetStateByteArray(otaBalanceStorageAddr, common.BytesToHash(otaAX))
	if len(balance) == 0 {
		return common.Big0, nil
	}

	return new(big.Int).SetBytes(balance), nil
}

// ChechOTAExist checks the OTA exist or not
func CheckOTAExist(statedb StateDB, otaAX []byte) (exist bool, balance *big.Int, err error) {
	if statedb == nil || len(otaAX) < common.HashLength {
		return false, nil, errors.New("invalid input param!")
	}

	otaAddrKey := common.BytesToHash(otaAX)
	balance, err = GetOtaBalanceFromAX(statedb, otaAddrKey[:])
	if err != nil {
		return false, nil, err
	} else if balance.Cmp(common.Big0) == 0 {
		return false, nil, nil
	}

	mptAddr := OTABalance2ContractAddr(balance)

	otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
	if otaValue != nil && len(otaValue) != 0 {
		return true, balance, nil
	}

	return false, nil, nil
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

// OTAInSet checks ota in set or not
func OTAInSet(otaSet [][]byte, ota []byte) bool {
	for _, exist := range otaSet {
		if bytes.Equal(exist, ota) {
			return true
		}
	}

	return false
}

func randomSelOTA(loopTimes *int, rnd *int, otaWanAddrs *[][]byte, value []byte) bool {
	*loopTimes++
	if (*loopTimes)%(*rnd) == 0 {
		*otaWanAddrs = append(*otaWanAddrs, value)
		*rnd = rand.Intn(100) + 1
		return true
	} else {
		return false
	}
}

func isAXPointToWanAddr(otaAX []byte, otaWanAddr []byte) bool {
	findAX, _ := GetAXFromWanAddr(otaWanAddr)
	return bytes.Equal(findAX, otaAX)
}

func doOTAStorageTravelCallBack(setNum *int, otaAX []byte, getNum *int, passNum *int, mptEleCount *int,
	loopTimes *int, rnd *int, otaWanAddrs *[][]byte, value []byte) (bool, error) {

	if *passNum == 0 {
		// first pass, record total count
		*mptEleCount++

		// find self, continue
		if isAXPointToWanAddr(otaAX, value) {
			return true, nil
		}

		// random select
		if bGet := randomSelOTA(loopTimes, rnd, otaWanAddrs, value); bGet {
			*getNum++
			return *getNum < *setNum, nil

		} else {
			return true, nil
		}

	} else if *passNum == 1 {
		// second pass
		if *setNum >= *mptEleCount {
			if !OTAInSet(*otaWanAddrs, value) {
				// mpt ele less than set num, reap the one don't exist in set
				*otaWanAddrs = append(*otaWanAddrs, value)
				*getNum++

				// not enough, return true to continue
				return *getNum < *setNum, nil
			} else {
				// return true to continue
				return true, nil
			}
		} else if !OTAInSet(*otaWanAddrs, value) {
			// find self, continue
			if isAXPointToWanAddr(otaAX, value) {
				return true, nil
			}

			// random select
			if bGet := randomSelOTA(loopTimes, rnd, otaWanAddrs, value); bGet {
				*getNum++
				return *getNum < *setNum, nil

			} else {
				return true, nil
			}

		} else {
			return true, nil
		}

	} else {
		// third or more pass
		if *setNum >= *mptEleCount {
			// random select
			if bGet := randomSelOTA(loopTimes, rnd, otaWanAddrs, value); bGet {
				*getNum++
				return *getNum < *setNum, nil

			} else {
				return true, nil
			}
		} else if !OTAInSet(*otaWanAddrs, value) {
			// find self, continue
			if isAXPointToWanAddr(otaAX, value) {
				return true, nil
			}

			// random select
			if bGet := randomSelOTA(loopTimes, rnd, otaWanAddrs, value); bGet {
				*getNum++
				return *getNum < *setNum, nil

			} else {
				return true, nil
			}
		} else {
			return true, nil
		}
	}
}

// GetOTASet retrieve the setNum of same balance OTA address of the input OTA setting by otaAX.
// As far as possible return the OTA set does not contain duplicate items and input otaAX self.
func GetOTASet(statedb StateDB, otaAX []byte, setNum int) (otaWanAddrs [][]byte, balance *big.Int, err error) {
	if statedb == nil || len(otaAX) != common.HashLength {
		return nil, nil, errors.New("invalid input param!")
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		return nil, nil, err
	} else if balance == nil || balance.Cmp(common.Big0) == 0 {
		return nil, nil, errors.New("cant find ota address balance!")
	}

	mptAddr := OTABalance2ContractAddr(balance)
	log.Debug("GetOTASet", "mptAddr", common.ToHex(mptAddr[:]))

	otaWanAddrs = make([][]byte, 0, setNum)
	rnd := rand.Intn(100) + 1

	getNum := 0      // selected ota count
	loopTimes := 0   // looped ota in storage total num, used to compare to rand number
	passNum := 0     // travel ota mpt times
	mptEleCount := 0 // total number of ota containing in mpt

	for {
		statedb.ForEachStorageByteArray(mptAddr, func(key common.Hash, value []byte) bool {
			if len(value) != common.WAddressLength {
				log.Error("invalid OTA address!", "balance", balance, "value", value)
				err = errors.New(fmt.Sprint("invalid OTA address! balance:", balance, ", ota:", value))
				return false
			}

			bContinue, err := doOTAStorageTravelCallBack(&setNum, otaAX, &getNum, &passNum,
				&mptEleCount, &loopTimes, &rnd, &otaWanAddrs, value)
			if err != nil {
				return false
			} else {
				return bContinue
			}
		})

		if err != nil {
			return nil, nil, err
		}

		if mptEleCount == 0 {
			return nil, balance, errors.New("no ota address exist! balance:" + balance.String())
		}

		if getNum >= setNum {
			return otaWanAddrs, balance, nil
		}

		passNum++
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
