package incentive

import (
	"math/big"

	"github.com/wanchain/go-wanchain/consensus"

	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/core/state"

	"github.com/wanchain/go-wanchain/common"

	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
)

var localDb *posdb.Db

var (
	dictAllTotal       = "all_total"
	dictEpochTotal     = "epoch_total"
	dictTotalRemain    = "total_remain"
	dictEpochRemain    = "epoch_remain"
	dictEpochBlock     = "epoch_block_number"
	dictRunTimes       = "run_times"
	dictEpochGasPool   = "epoch_gas_pool"
	dictAllGasPool     = "all_gas_pool"
	dictAddressAll     = "address_all"
	dictAddressEpoch   = "address_epoch"
	dictCurrentEpoch   = "current_epoch"
	dictAddressCnt     = "address_count"
	dictEpochPayDetail = "epoch_pay_detail"
)

func initLocalDb(dbName string) {
	localDb = posdb.NewDb(dbName)
}

func saveIncentiveHistory(epochID uint64, payments [][]vm.ClientIncentive) {
	if payments == nil {
		return
	}

	buf, err := rlp.EncodeToBytes(payments)
	if err != nil {
		log.SyslogErr(err.Error())
		return
	}
	localDb.Put(epochID, dictEpochPayDetail, buf)

	saveOtherInfomation(epochID, payments)
}

func saveTotalIncentive(epochID uint64, incentives [][]vm.ClientIncentive) {
	if incentives == nil {
		return
	}
	totalIncome := sumIncentive(incentives)
	localDbAddValue(0, dictAllTotal, totalIncome)
}

func saveEpochTotalIncentive(epochID uint64, incentives [][]vm.ClientIncentive) {
	if incentives == nil {
		return
	}
	totalIncome := sumIncentive(incentives)
	localDb.Put(epochID, dictEpochTotal, totalIncome.Bytes())
}

func saveRemain(epochID uint64, remain *big.Int) {
	if remain == nil {
		return
	}
	localDb.Put(epochID, dictEpochRemain, remain.Bytes())
	localDbAddValue(0, dictTotalRemain, remain)
}

func addRunTimes() {
	localDbAddValue(0, dictRunTimes, big.NewInt(1))
}

func saveOtherInfomation(epochID uint64, incentives [][]vm.ClientIncentive) {
	saveTotalIncentive(epochID, incentives)
	saveEpochTotalIncentive(epochID, incentives)
	addRunTimes()
}

func localDbGetValue(epochID uint64, key string) (*big.Int, error) {
	total, err := localDb.Get(epochID, key)
	if err != nil && err.Error() != "leveldb: not found" {
		log.SyslogErr(err.Error())
		return nil, err
	}

	if total == nil || len(total) == 0 {
		return big.NewInt(0), nil
	}

	return big.NewInt(0).SetBytes(total), nil
}
func localDbSetValue(epochID uint64, key string, value *big.Int) {
	localDb.Put(epochID, key, value.Bytes())
}

func localDbAddValue(epochID uint64, key string, value *big.Int) {
	total, err := localDb.Get(epochID, key)
	if err != nil && err.Error() != "leveldb: not found" {
		log.SyslogErr(err.Error())
		return
	}
	totalNum := big.NewInt(0)
	if total != nil && len(total) != 0 {
		totalNum.SetBytes(total)
	}
	totalNum.Add(totalNum, value)
	localDb.Put(0, key, totalNum.Bytes())
}

// GetEpochPayDetail use to get detail payment array
func GetEpochPayDetail(epochID uint64) ([][]vm.ClientIncentive, error) {
	buf, err := localDb.Get(epochID, dictEpochPayDetail)
	if err != nil {
		log.SyslogErr(err.Error())
		return nil, err
	}

	var payment [][]vm.ClientIncentive

	err = rlp.DecodeBytes(buf, &payment)
	if err != nil {
		log.SyslogErr(err.Error())
		return nil, err
	}

	return payment, nil
}

// GetTotalIncentive get total incentive of all epoch
func GetTotalIncentive() (*big.Int, error) {
	return localDbGetValue(0, dictAllTotal)
}

// GetEpochIncentive get total incentive of all epoch
func GetEpochIncentive(epochID uint64) (*big.Int, error) {
	return localDbGetValue(epochID, dictEpochTotal)
}

func GetEpochIncentiveBlockNumber(epochID uint64) (*big.Int, error) {
	return localDbGetValue(epochID, dictEpochBlock)
}

// GetEpochRemain get remain of epoch input
func GetEpochRemain(epochID uint64) (*big.Int, error) {
	return localDbGetValue(epochID, dictEpochRemain)
}

// GetTotalRemain get remain of epoch input
func GetTotalRemain() (*big.Int, error) {
	return localDbGetValue(0, dictTotalRemain)
}

// GetRunTimes returns incentive run times
func GetRunTimes() (*big.Int, error) {
	return localDbGetValue(0, dictRunTimes)
}

// GetEpochGasPool use to get epoch gas pool
func GetEpochGasPool(stateDb vm.StateDB, epochID uint64) *big.Int {
	return getEpochGas(stateDb, epochID)
}

// GetRBAddress use to get random proposer address list
func GetRBAddress(epochID uint64) []common.Address {
	if getRandomProposerAddress == nil {
		return nil
	}

	leaders := getRandomProposerAddress(epochID)
	addrs := make([]common.Address, len(leaders))
	for i := 0; i < len(leaders); i++ {
		addrs[i] = leaders[i].SecAddr
	}

	return addrs
}

// GetIncentivePool can get the total incentive, foundation part and gas pool part.
func GetIncentivePool(stateDb *state.StateDB, epochID uint64) (*big.Int, *big.Int, *big.Int) {
	return calculateIncentivePool(stateDb, epochID)
}

// GetEpochLeaderActivity can get the address and activity of epoch leaders
func GetEpochLeaderActivity(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int) {
	return getEpochLeaderActivity(stateDb, epochID)
}

// GetEpochRBLeaderActivity can get the address and activity of RB leaders
func GetEpochRBLeaderActivity(stateDb vm.StateDB, epochID uint64) ([]common.Address, []int) {
	return getRandomProposerActivity(stateDb, epochID)
}

// GetSlotLeaderActivity can get the address, blockCnt, and activity of slotleader
func GetSlotLeaderActivity(chain consensus.ChainReader, epochID uint64) ([]common.Address, []int, float64, int) {
	return getSlotLeaderActivity(chain, epochID, posconfig.SlotCount)
}
