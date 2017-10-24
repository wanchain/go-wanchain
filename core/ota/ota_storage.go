package ota

import (
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/trie"
	"math/big"
	"math/rand"
)

var (
	OTAImageStorageAddr = crypto.Keccak256([]byte(string("OTA image storage addr")))
)

func GetOtaBalance(statedb vm.StateDB, otaAX []byte) (*big.Int, error) {
	if statedb == nil || otaAX == nil || len(otaAX) != common.HashLength {
		return nil, errors.New("getOtaBalance， invalid input param!")
	}

	balance := statedb.GetBalance(common.BytesToAddress(otaAX))

	return balance, nil
}

// ChechOTAExit chech the OTA exit in db or not
func CheckOTAExit(statedb vm.StateDB, otaAX []byte) (exit bool, balance *big.Int, err error) {
	if statedb == nil || otaAX == nil || len(otaAX) < common.HashLength {
		return false, nil, errors.New("CheckOTAExit, invalid input param!")
	}

	otaAddrKey := common.BytesToHash(otaAX)
	balance, err = GetOtaBalance(statedb, otaAddrKey[:])
	if err != nil {
		return false, nil, err
	} else if balance.Cmp(big.NewInt(0)) == 0 {
		return false, nil, nil
	}

	mptAddr := common.HexToAddress(balance.String())

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
func BatCheckOTAExit(statedb vm.StateDB, otaAXs [][]byte) (exit bool, balance *big.Int, unexitOta []byte, err error) {
	if statedb == nil || otaAXs == nil || len(otaAXs) == 0 {
		return false, nil, nil, errors.New("BatCheckOTAExit, invalid input param!")
	}

	for _, otaAX := range otaAXs {
		if otaAX == nil || len(otaAX) < common.HashLength {
			return false, nil, otaAX, errors.New("BatCheckOTAExit, invalid input ota short address!")
		}

		otaAddrKey := common.BytesToHash(otaAX)
		balanceTmp, err := GetOtaBalance(statedb, otaAddrKey[:])
		if err != nil {
			return false, nil, otaAX, err
		} else if balanceTmp.Cmp(big.NewInt(0)) == 0 {
			return false, nil, otaAX, errors.New("BatCheckOTAExit, ota balance is 0!")
		} else if balance == nil {
			balance = balanceTmp
			continue
		} else if balance.Cmp(balanceTmp) != 0 {
			return false, nil, otaAX, errors.New("BatCheckOTAExit, otas have different balances!")
		}
	}

	mptAddr := common.HexToAddress(balance.String())
	for _, otaAX := range otaAXs {
		otaAddrKey := common.BytesToHash(otaAX)
		otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
		if otaValue == nil || len(otaValue) == 0 {
			return false, nil, otaAX, nil
		}
	}

	return true, balance, nil, nil
}

func SetOTA(statedb vm.StateDB, balance *big.Int, otaShortAddr []byte) error {
	if statedb == nil || balance == nil || otaShortAddr == nil || len(otaShortAddr) != common.WAddressLength {
		return errors.New("SetOTA, invalid input param!")
	}

	//fmt.Println("otaLongAddr:", common.ToHex(otaLongAddr))
	otaAddrKey := otaShortAddr[1 : 1+common.HashLength]
	//fmt.Println("otaAddrKey:", common.ToHex(otaAddrKey))
	balanceOld, err := GetOtaBalance(statedb, otaAddrKey)
	if err != nil {
		return err
	}

	if balanceOld != nil && balanceOld.Cmp(big.NewInt(0)) != 0 {
		return errors.New("SetOTA, ota balance is not 0! old balance:" + balanceOld.String())
	}

	mptAddr := common.HexToAddress(balance.String())
	statedb.SetStateByteArray(mptAddr, common.BytesToHash(otaAddrKey), otaShortAddr)
	statedb.AddBalance(common.BytesToAddress(otaAddrKey), balance)

	/////////////////test
	// CountTrieItemCount(statedb, balance)
	/////////////////test

	return nil
}

/////////////////test
func CountTrieItemCount(statedb vm.StateDB, balance *big.Int) {
	mptAddr := common.HexToAddress(balance.String())

	dataTrie := statedb.StorageVmTrie(mptAddr)
	if dataTrie == nil {
		fmt.Println("CountTrieItemCount. cant find trie, mptAddr:", balance.String())
	}

	count := 0
	it := trie.NewIterator(dataTrie.NodeIterator(nil))
	for it.Next() {
		count++
	}

	fmt.Println("CountTrieItemCount. count:", count, ", mptAddr:", balance.String())
}

/////////////////test

func AddOTAIfNotExit(statedb vm.StateDB, balance *big.Int, otaShortAddr []byte) (bool, error) {
	if statedb == nil || balance == nil || otaShortAddr == nil || len(otaShortAddr) != common.WAddressLength {
		return false, errors.New("AddOTAIfNotExit, invalid input param!")
	}

	otaAddrKey := common.BytesToHash(otaShortAddr[1 : 1+common.HashLength])
	exit, _, err := CheckOTAExit(statedb, otaAddrKey[:])
	if err != nil {
		return false, err
	}

	if exit {
		return false, nil
	}

	err = SetOTA(statedb, balance, otaShortAddr)
	if err != nil {
		return false, err
	}

	return true, nil
}

func GetOTAInfoFromAX(statedb vm.StateDB, otaAX []byte) (otaShortAddr []byte, balance *big.Int, err error) {
	if statedb == nil || otaAX == nil || len(otaAX) < common.HashLength {
		return nil, nil, errors.New("GetOTAInfoFromAX, invalid input param!")
	}

	otaAddrKey := common.BytesToHash(otaAX)
	balance, err = GetOtaBalance(statedb, otaAddrKey[:])
	if err != nil {
		return nil, nil, err
	}

	if balance == nil || balance.Cmp(big.NewInt(0)) == 0 {
		return nil, nil, errors.New("GetOTAInfoFromAX, ota balance is 0!")
	}

	mptAddr := common.HexToAddress(balance.String())

	otaValue := statedb.GetStateByteArray(mptAddr, otaAddrKey)
	if otaValue != nil && len(otaValue) != 0 {
		return otaValue, balance, nil
	}

	return nil, balance, nil
}

func GetOTASet(statedb vm.StateDB, otaAX []byte, otaNum int) (otaShortAddrs [][]byte, balance *big.Int, err error) {
	if statedb == nil || otaAX == nil || len(otaAX) != common.HashLength {
		return nil, nil, errors.New("GetOTASet, invalid input param!")
	}

	balance, err = GetOtaBalance(statedb, otaAX)
	if err != nil {
		return nil, nil, err
	} else if balance == nil || balance.Cmp(big.NewInt(0)) == 0 {
		return nil, nil, errors.New("GetOTASet, cant find ota address balance!")
	}

	mptAddr := common.HexToAddress(balance.String())
	fmt.Println("GetOTASet, mptAddr:", common.ToHex(mptAddr[:]))
	dataTrie := statedb.StorageVmTrie(mptAddr)
	if dataTrie == nil {
		return nil, balance, errors.New("GetOTASet, can't find ota trie from the given ota address!")
	}

	otaShortAddrs = make([][]byte, 0)
	rnd := rand.Intn(100) + 1

	count := 0
	i := 0
	loop := 0
	for {
		it := trie.NewIterator(dataTrie.NodeIterator(nil))

		for it.Next() {
			fmt.Println("GetOTASet for loop.., i:", i, ", loop:", loop)
			if it.Value == nil || len(it.Value) != common.WAddressLength {
				return nil, balance, errors.New("GetOTASet, found invalid ota addr len from trie!")
			}

			loop++
			count++
			if count%rnd == 0 {
				otaShortAddrs = append(otaShortAddrs, it.Value)
				i++
				rnd = rand.Intn(100) + 1

				if i >= otaNum {
					return otaShortAddrs, balance, nil
				}
			}
		}

		if loop == 0 {
			return nil, balance, errors.New("GetOTASet, no ota adress in the trie! trie balance:" + balance.String())
		}
	}

	return nil, nil, nil
}

func CheckOTAImageExit(statedb vm.StateDB, otaImage []byte) (bool, []byte, error) {
	if statedb == nil || otaImage == nil || len(otaImage) == 0 {
		return false, nil, errors.New("CheckOTAImageExit, invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	mptAddr := common.BytesToAddress(OTAImageStorageAddr)

	otaImageValue := statedb.GetStateByteArray(mptAddr, otaImageKey)
	if otaImageValue != nil && len(otaImageValue) != 0 {
		return true, otaImageValue, nil
	}

	return false, nil, nil
}

func AddOTAImage(statedb vm.StateDB, otaImage []byte, value []byte) error {
	if statedb == nil || otaImage == nil || len(otaImage) == 0 || value == nil || len(value) == 0 {
		return errors.New("AddOTAImage, invalid input param!")
	}

	otaImageKey := crypto.Keccak256Hash(otaImage)
	mptAddr := common.BytesToAddress(OTAImageStorageAddr)

	statedb.SetStateByteArray(mptAddr, otaImageKey, value)
	return nil
}
