package vm

import (
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
	"math/rand"
	"fmt"
)

// Reserved contracts address for ota info storage
var (
	ReserveContractOTABalance = big.NewInt(300)
	ReserveContractOTAImage   = big.NewInt(301)
	ReserveContractOTABalanceAddr = common.BytesToAddress(ReserveContractOTABalance.Bytes())
	ReserveContractOTAImageAddr   = common.BytesToAddress(ReserveContractOTAImage.Bytes())
)

// ota balance to ota storage address
//
// 1 wancoin --> (bigint)1000000000000000000 --> "0x0000000000000000000001000000000000000000"
//
func OTABalance2ContractAddr(balance * big.Int) common.Address {
	if balance == nil {
		return common.Address{}
	}

	return common.HexToAddress(balance.String())
	//	return common.BigToAddress(balance)
}

func GetOtaBalanceFromAX(statedb StateDB, otaAX []byte) (*big.Int, error) {
	if statedb == nil || otaAX == nil || len(otaAX) != common.HashLength {
		return nil, errors.New("GetOtaBalanceFromAX. invalid input param!")
	}

	balance := statedb.GetStateByteArray(ReserveContractOTABalanceAddr, common.BytesToHash(otaAX))
	if balance == nil || len(balance) == 0 {
		return common.Big0, nil
	}

	return new(big.Int).SetBytes(balance), nil
}

func SetOtaBalanceToAX(statedb StateDB, otaAX []byte, balance *big.Int) error {
	if statedb == nil || otaAX == nil || len(otaAX) != common.HashLength || balance == nil {
		return errors.New("SetOtaBalanceToAX. invalid input param!")
	}

	statedb.SetStateByteArray(ReserveContractOTABalanceAddr, common.BytesToHash(otaAX), balance.Bytes())
	return nil
}

func GetOtaBalanceFromWanAddr(statedb StateDB, otaWanAddr []byte) (*big.Int, error) {
	if statedb == nil || otaWanAddr == nil || len(otaWanAddr) != common.WAddressLength {
		return nil, errors.New("GetOtaBalanceFromWanAddr， invalid input param!")
	}

	otaAX := otaWanAddr[1 : 1+common.HashLength]
	balance := statedb.GetStateByteArray(ReserveContractOTABalanceAddr, common.BytesToHash(otaAX))
	if balance == nil || len(balance) == 0 {
		return common.Big0, nil
	}

	return new(big.Int).SetBytes(balance), nil
}

// ChechOTAExit chech the OTA exit in db or not
func CheckOTAExit(statedb StateDB, otaAX []byte) (exit bool, balance *big.Int, err error) {
	if statedb == nil || otaAX == nil || len(otaAX) < common.HashLength {
		return false, nil, errors.New("CheckOTAExit, invalid input param!")
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

// BatCheckOTAExit batch chech the OTA exit in db or not
//
// return ota as []byte when find first not exit one
//
func BatCheckOTAExit(statedb StateDB, otaAXs [][]byte) (exit bool, balance *big.Int, unexitOta []byte, err error) {
	if statedb == nil || otaAXs == nil || len(otaAXs) == 0 {
		return false, nil, nil, errors.New("BatCheckOTAExit, invalid input param!")
	}

	for _, otaAX := range otaAXs {
		if otaAX == nil || len(otaAX) < common.HashLength {
			return false, nil, otaAX, errors.New("BatCheckOTAExit, invalid input ota AX!")
		}

		otaAddrKey := common.BytesToHash(otaAX)
		balanceTmp, err := GetOtaBalanceFromAX(statedb, otaAddrKey[:])
		if err != nil {
			return false, nil, otaAX, err
		} else if balanceTmp.Cmp(common.Big0) == 0 {
			return false, nil, otaAX, errors.New("BatCheckOTAExit, ota balance is 0!")
		} else if balance == nil {
			balance = balanceTmp
			continue
		} else if balance.Cmp(balanceTmp) != 0 {
			return false, nil, otaAX, errors.New("BatCheckOTAExit, otas have different balances!")
		}
	}

	mptAddr := OTABalance2ContractAddr(balance)
	for _, otaAX := range otaAXs {
		otaAddrKey := common.BytesToHash(otaAX)
		otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
		if otaValue == nil || len(otaValue) == 0 {
			return false, nil, otaAX, nil
		}
	}

	return true, balance, nil, nil
}

func SetOTA(statedb StateDB, balance *big.Int, otaWanAddr []byte) error {
	if statedb == nil || balance == nil || otaWanAddr == nil || len(otaWanAddr) != common.WAddressLength {
		return errors.New("SetOTA, invalid input param!")
	}

	otaAX := otaWanAddr[1 : 1+common.HashLength]
	balanceOld, err := GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		return err
	}

	if balanceOld != nil && balanceOld.Cmp(common.Big0) != 0 {
		return errors.New("SetOTA, ota balance is not 0! old balance:" + balanceOld.String())
	}

	mptAddr := OTABalance2ContractAddr(balance)
	statedb.SetStateByteArray(mptAddr, common.BytesToHash(otaAX), otaWanAddr)
	return SetOtaBalanceToAX(statedb, otaAX, balance)
}

func AddOTAIfNotExit(statedb StateDB, balance *big.Int, otaWanAddr []byte) (bool, error) {
	if statedb == nil || balance == nil || otaWanAddr == nil || len(otaWanAddr) != common.WAddressLength {
		return false, errors.New("AddOTAIfNotExit, invalid input param!")
	}

	otaAddrKey := common.BytesToHash(otaWanAddr[1 : 1+common.HashLength])
	exit, _, err := CheckOTAExit(statedb, otaAddrKey[:])
	if err != nil {
		return false, err
	}

	if exit {
		return false, nil
	}

	err = SetOTA(statedb, balance, otaWanAddr)
	if err != nil {
		return false, err
	}

	return true, nil
}

func GetOTAInfoFromAX(statedb StateDB, otaAX []byte) (otaWanAddr []byte, balance *big.Int, err error) {
	if statedb == nil || otaAX == nil || len(otaAX) < common.HashLength {
		return nil, nil, errors.New("GetOTAInfoFromAX, invalid input param!")
	}

	otaAddrKey := common.BytesToHash(otaAX)
	balance, err = GetOtaBalanceFromAX(statedb, otaAddrKey[:])
	if err != nil {
		return nil, nil, err
	}

	if balance == nil || balance.Cmp(common.Big0) == 0 {
		return nil, nil, errors.New("GetOTAInfoFromAX, ota balance is 0!")
	}

	mptAddr := OTABalance2ContractAddr(balance)

	otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
	if otaValue != nil && len(otaValue) != 0 {
		return otaValue, balance, nil
	}

	return nil, balance, nil
}

func GetOTASet(statedb StateDB, otaAX []byte, otaNum int) (otaWanAddrs [][]byte, balance *big.Int, err error) {
	if statedb == nil || otaAX == nil || len(otaAX) != common.HashLength {
		return nil, nil, errors.New("GetOTASet, invalid input param!")
	}

	balance, err = GetOtaBalanceFromAX(statedb, otaAX)
	if err != nil {
		return nil, nil, err
	} else if balance == nil || balance.Cmp(common.Big0) == 0 {
		return nil, nil, errors.New("GetOTASet, cant find ota address balance!")
	}

	mptAddr := OTABalance2ContractAddr(balance)
	log.Debug("GetOTASet, mptAddr:", common.ToHex(mptAddr[:]))

	otaWanAddrs = make([][]byte, 0)
	rnd := rand.Intn(100) + 1

	i := 0
	loop := 0
	for {
		statedb.ForEachStorageByteArray(mptAddr, func(key common.Hash, value []byte) bool {
			log.Trace("GetOTASet for loop.., i:", i, ", loop:", loop)
			if value == nil || len(value) != common.WAddressLength {
				log.Error("invalid OTA address!", balance, value)
				err = errors.New(fmt.Sprint("invalid OTA address! balance:", balance, ", ota:", value))
				return false
			}

			loop++
			if loop%rnd == 0 {
				otaWanAddrs = append(otaWanAddrs, value)
				i++
				rnd = rand.Intn(100) + 1

				if i >= otaNum {
					return false
				}
			}

			// Return true, indicating we'd like to continue.
			return true
		})

		if err != nil {
			return nil, nil, err
		}

		if loop == 0 {
			return nil, balance, errors.New("GetOTASet, no ota adress in the trie! trie balance:" + balance.String())
		}

		if  i >= otaNum {
			return otaWanAddrs, balance, nil
		}
	}

	return nil, nil, nil
}

func CheckOTAImageExit(statedb StateDB, otaImage []byte) (bool, []byte, error) {
	if statedb == nil || otaImage == nil || len(otaImage) == 0 {
		return false, nil, errors.New("CheckOTAImageExit, invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	otaImageValue := statedb.GetStateByteArray(ReserveContractOTAImageAddr, otaImageKey)
	if otaImageValue != nil && len(otaImageValue) != 0 {
		return true, otaImageValue, nil
	}

	return false, nil, nil
}

func AddOTAImage(statedb StateDB, otaImage []byte, value []byte) error {
	if statedb == nil || otaImage == nil || len(otaImage) == 0 || value == nil || len(value) == 0 {
		return errors.New("AddOTAImage, invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	statedb.SetStateByteArray(ReserveContractOTAImageAddr, otaImageKey, value)
	return nil
}
