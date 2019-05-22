// Copyright 2018 Wanchain Foundation Ltd

package core

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/consensus/ethash"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
)

const (
	coinSCDefinition = `
	[{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [{"name": "OtaAddr","type":"string"},{"name": "Value","type": "uint256"}],"name": "buyCoinNote","outputs": [{"name": "OtaAddr","type":"string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","inputs": [{"name":"RingSignedData","type": "string"},{"name": "Value","type": "uint256"}],"name": "refundCoin","outputs": [{"name": "RingSignedData","type": "string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [],"name": "getCoins","outputs": [{"name":"Value","type": "uint256"}]}]`
)

var (
	wanCoinSCAddr = common.BytesToAddress([]byte{100})

	otaBalanceStorageAddr = common.BytesToAddress(big.NewInt(300).Bytes())
)

var (
	errOTAGen = errors.New("Fail to generate OTA")
)

func TestGasOrdinaryCoinTransfer(t *testing.T) {
	var (
		initialBalance = big.NewInt(1000000000)
		// value of Wan coin to transfer
		transferValue = big.NewInt(100000)
		// gas price
		gp = big.NewInt(100)
		// gas used by the transaction
		gasUsed   = new(big.Int)
		db, _     = ethdb.NewMemDatabase()
		engine    = ethash.NewFaker(db)
		sk, _     = crypto.GenerateKey()
		rk, _     = crypto.GenerateKey()
		ck, _     = crypto.GenerateKey()
		sender    = crypto.PubkeyToAddress(sk.PublicKey)
		recipient = crypto.PubkeyToAddress(rk.PublicKey)
		coinbase  = crypto.PubkeyToAddress(ck.PublicKey)
	)

	// initialize valid signers to write blocks
	l := 5
	extraData := append(make([]byte, 0), coinbase[:]...)
	keySlice, addrSlice := make([]*ecdsa.PrivateKey, l), make([]common.Address, l)
	for i := 0; i < l; i++ {
		keySlice[i], _ = crypto.GenerateKey()
		addrSlice[i] = crypto.PubkeyToAddress(keySlice[i].PublicKey)
		extraData = append(extraData, addrSlice[i].Bytes()...)
	}

	// make the transaction
	gspec := &Genesis{
		Config:     params.TestChainConfig,
		GasLimit:   0x47b760,
		ExtraData:  extraData,
		Difficulty: big.NewInt(1),
		Alloc:      GenesisAlloc{sender: {Balance: initialBalance}},
	}
	genesis := gspec.MustCommit(db)

	blockchain, _ := NewBlockChain(db, gspec.Config, engine, vm.Config{})
	defer blockchain.Stop()

	chainEnv := NewChainEnv(params.TestChainConfig, gspec, engine, blockchain, db)

	signer := types.NewEIP155Signer(big.NewInt(gspec.Config.ChainId.Int64()))
	chain, _ := chainEnv.GenerateChainMulti(genesis, 1, func(i int, gen *BlockGen) {
		gen.SetCoinbase(coinbase)
		tx, _ := types.SignTx(types.NewTransaction(gen.TxNonce(sender), recipient, transferValue, new(big.Int).SetUint64(params.TxGas), gp, nil), signer, sk)
		gasUsed = gen.AddTxAndCalcGasUsed(tx)
	})

	if i, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("insert error (block %d): %v\n", chain[i].NumberU64(), err)
		return
	}

	// retrieve current state and account balances after the transaction
	state, _ := blockchain.State()

	senderBalance := state.GetBalance(sender)
	recipientBalance := state.GetBalance(recipient)
	coinbaseBalance := state.GetBalance(coinbase)

	if blockchain.CurrentBlock().Number().Cmp(big.NewInt(1)) != 0 {
		t.Fatal("fail to generate new block")
	}

	if coinbaseBalance.Cmp(new(big.Int).Mul(gp, gasUsed)) != 0 {
		t.Fatal("coinbase rewards error")
	}

	if initialBalance.Cmp(new(big.Int).Add(senderBalance, new(big.Int).Add(recipientBalance, coinbaseBalance))) != 0 {
		t.Fatal("wrong total balance")
	}
}

func TestGasCoinMint(t *testing.T) {
	var (
		initialBalance = big.NewInt(0)
		// value of Wan coin to transfer
		transferValue = big.NewInt(0)
		// gasLimit
		gl = new(big.Int).SetUint64(params.SstoreSetGas * 20)
		// gas price
		gp = big.NewInt(100)
		// gas used by the transaction
		gasUsed  = new(big.Int)
		db, _    = ethdb.NewMemDatabase()
		engine   = ethash.NewFaker(db)
		sk, _    = crypto.GenerateKey()
		rkA, _   = crypto.GenerateKey()
		rkB, _   = crypto.GenerateKey()
		ck, _    = crypto.GenerateKey()
		sender   = crypto.PubkeyToAddress(sk.PublicKey)
		coinbase = crypto.PubkeyToAddress(ck.PublicKey)
	)

	initialBalance.SetString("20000000000000000000", 10)
	transferValue.SetString("10000000000000000000", 10)

	OTAStr, err := genOTAStr(&rkA.PublicKey, &rkB.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	mintCoinData, err := genBuyCoinData(OTAStr, transferValue)
	if err != nil {
		t.Fatal(err)
	}

	// initialize valid signers to write blocks
	l := 5
	extraData := append(make([]byte, 0), coinbase[:]...)
	keySlice, addrSlice := make([]*ecdsa.PrivateKey, l), make([]common.Address, l)
	for i := 0; i < l; i++ {
		keySlice[i], _ = crypto.GenerateKey()
		addrSlice[i] = crypto.PubkeyToAddress(keySlice[i].PublicKey)
		extraData = append(extraData, addrSlice[i].Bytes()...)
	}

	// make the transaction
	gspec := &Genesis{
		Config:     params.TestChainConfig,
		GasLimit:   0x47b760,
		ExtraData:  extraData,
		Difficulty: big.NewInt(1),
		Alloc:      GenesisAlloc{sender: {Balance: initialBalance}},
	}
	genesis := gspec.MustCommit(db)
	blockchain, _ := NewBlockChain(db, gspec.Config, engine, vm.Config{})
	defer blockchain.Stop()

	chainEnv := NewChainEnv(params.TestChainConfig, gspec, engine, blockchain, db)

	signer := types.NewEIP155Signer(big.NewInt(gspec.Config.ChainId.Int64()))
	chain, _ := chainEnv.GenerateChainMulti(genesis, 1, func(i int, gen *BlockGen) {
		gen.SetCoinbase(coinbase)
		tx, _ := types.SignTx(types.NewTransaction(gen.TxNonce(sender), wanCoinSCAddr, transferValue, gl, gp, mintCoinData), signer, sk)
		gasUsed = gen.AddTxAndCalcGasUsed(tx)
	})

	if i, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("insert error (block %d): %v\n", chain[i].NumberU64(), err)
		return
	}

	// retrieve current state and account balances after the transaction
	state, _ := blockchain.State()

	senderBalance := state.GetBalance(sender)
	OTABalance := getOTABalance(state, OTAStr)
	coinbaseBalance := state.GetBalance(coinbase)

	if blockchain.CurrentBlock().Number().Cmp(big.NewInt(1)) != 0 {
		t.Fatal("fail to generate new block")
	}

	if coinbaseBalance.Cmp(new(big.Int).Mul(gp, gasUsed)) != 0 {
		t.Fatal("coinbase rewards error")
	}

	if initialBalance.Cmp(new(big.Int).Add(senderBalance, new(big.Int).Add(OTABalance, coinbaseBalance))) != 0 {
		t.Fatal("wrong total balance")
	}

}

func TestGasCoinRefund(t *testing.T) {
	var (
		initialBalance = big.NewInt(0)
		// value of Wan coin to transfer
		transferValue = big.NewInt(0)
		// gasLimit
		gl = new(big.Int).SetUint64(params.SstoreSetGas * 20)
		// gas price
		gp = big.NewInt(100)
		// gas used by the transaction
		gasUsed   = new(big.Int)
		gasUsedC1 = new(big.Int)
		gasUsedC2 = new(big.Int)
		// ota set elements count, which does not include the true OTA
		setSize           = 2
		db, _             = ethdb.NewMemDatabase()
		engine            = ethash.NewFaker(db)
		sk, _             = crypto.GenerateKey()
		skContributor1, _ = crypto.GenerateKey()
		skContributor2, _ = crypto.GenerateKey()
		rkA, _            = crypto.GenerateKey()
		rkB, _            = crypto.GenerateKey()
		ck, _             = crypto.GenerateKey()
		sender            = crypto.PubkeyToAddress(sk.PublicKey)
		contributor1      = crypto.PubkeyToAddress(skContributor1.PublicKey)
		contributor2      = crypto.PubkeyToAddress(skContributor2.PublicKey)
		coinbase          = crypto.PubkeyToAddress(ck.PublicKey)
	)

	initialBalance.SetString("20000000000000000000", 10)
	transferValue.SetString("10000000000000000000", 10)

	// generate OTAs
	OTAStr, err := genOTAStr(&rkA.PublicKey, &rkB.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	OTAC1, err := genOTAStr(&rkA.PublicKey, &rkB.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	OTAC2, err := genOTAStr(&rkA.PublicKey, &rkB.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	// generate ABI data
	mintCoinData, err := genBuyCoinData(OTAStr, transferValue)
	if err != nil {
		t.Fatal(err)
	}

	mintCoinDataContributor1, err := genBuyCoinData(OTAC1, transferValue)
	if err != nil {
		t.Fatal(err)
	}

	mintCoinDataContributor2, err := genBuyCoinData(OTAC2, transferValue)
	if err != nil {
		t.Fatal(err)
	}

	// initialize valid signers to write blocks
	l := 5
	extraData := append(make([]byte, 0), coinbase[:]...)
	keySlice, addrSlice := make([]*ecdsa.PrivateKey, l), make([]common.Address, l)
	for i := 0; i < l; i++ {
		keySlice[i], _ = crypto.GenerateKey()
		addrSlice[i] = crypto.PubkeyToAddress(keySlice[i].PublicKey)
		extraData = append(extraData, addrSlice[i].Bytes()...)
	}

	// make the transaction
	gspec := &Genesis{
		Config:     params.TestChainConfig,
		GasLimit:   0x47b760,
		ExtraData:  extraData,
		Difficulty: big.NewInt(1),
		Alloc: GenesisAlloc{
			sender:       {Balance: initialBalance},
			contributor1: {Balance: initialBalance},
			contributor2: {Balance: initialBalance},
		},
	}
	genesis := gspec.MustCommit(db)
	blockchain, _ := NewBlockChain(db, gspec.Config, engine, vm.Config{})
	defer blockchain.Stop()

	chainEnv := NewChainEnv(params.TestChainConfig, gspec, engine, blockchain, db)

	signer := types.NewEIP155Signer(big.NewInt(gspec.Config.ChainId.Int64()))
	signerC1 := types.NewEIP155Signer(big.NewInt(gspec.Config.ChainId.Int64()))
	signerC2 := types.NewEIP155Signer(big.NewInt(gspec.Config.ChainId.Int64()))
	chain, _ := chainEnv.GenerateChainMulti(genesis, 1, func(i int, gen *BlockGen) {
		// set coinbase
		gen.SetCoinbase(coinbase)

		// add transactions
		tx, _ := types.SignTx(types.NewTransaction(gen.TxNonce(sender), wanCoinSCAddr, transferValue, gl, gp, mintCoinData), signer, sk)
		txC1, _ := types.SignTx(types.NewTransaction(gen.TxNonce(contributor1), wanCoinSCAddr, transferValue, gl, gp, mintCoinDataContributor1), signerC1, skContributor1)
		txC2, _ := types.SignTx(types.NewTransaction(gen.TxNonce(contributor2), wanCoinSCAddr, transferValue, gl, gp, mintCoinDataContributor2), signerC2, skContributor2)

		// collect gas spent
		gasUsed = gen.AddTxAndCalcGasUsed(tx)
		gasUsedC1 = gen.AddTxAndCalcGasUsed(txC1)
		gasUsedC2 = gen.AddTxAndCalcGasUsed(txC2)
	})

	if i, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("insert error (block %d): %v\n", chain[i].NumberU64(), err)
		return
	}

	// retrieve current state and account balances after the coin-mint transaction
	state, _ := blockchain.State()

	senderBalance := state.GetBalance(sender)
	contributor1Balance := state.GetBalance(contributor1)
	contributor2Balance := state.GetBalance(contributor2)
	OTABalance := getOTABalance(state, OTAStr)
	OTAC1Balance := getOTABalance(state, OTAC1)
	OTAC2Balance := getOTABalance(state, OTAC2)
	coinbaseBalance := state.GetBalance(coinbase)

	if blockchain.CurrentBlock().Number().Cmp(big.NewInt(1)) != 0 {
		t.Fatal("fail to generate a new block")
	}

	if coinbaseBalance.Cmp(new(big.Int).Add(new(big.Int).Mul(gp, gasUsed), new(big.Int).Add(new(big.Int).Mul(gp, gasUsedC1), new(big.Int).Mul(gp, gasUsedC2)))) != 0 {
		t.Fatal("coinbase rewards error")
	}

	if new(big.Int).Add(new(big.Int).Add(new(big.Int).Add(new(big.Int).Add(new(big.Int).Add(new(big.Int).Add(senderBalance, OTAC1Balance), coinbaseBalance), contributor1Balance), contributor2Balance), OTABalance), OTAC2Balance).Cmp(new(big.Int).Mul(big.NewInt(3), initialBalance)) != 0 {
		t.Fatal("Wrong total balance")
	}

	OTASet, err := genOTASet(state, OTAStr, setSize)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range OTASet {
		fmt.Println(v)
	}

	keyPairs, err := computeOTAPubKeys(rkA, rkB, strings.Replace(OTAStr, "0x", "", -1))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(keyPairs)
}

// generate recipient's OTA for privary transaction
func genOTAStr(pk, pk1 *ecdsa.PublicKey) (string, error) {
	PKPair := hexutil.PKPair2HexSlice(pk, pk1)
	OTA, err := crypto.GenerateOneTimeKey(PKPair[0], PKPair[1], PKPair[2], PKPair[3])
	if err != nil {
		return "", errOTAGen
	}

	OTAStr := strings.Replace(strings.Join(OTA, ""), "0x", "", -1)
	OTARaw, err := hexutil.Decode("0x" + OTAStr)
	if err != nil {
		return "", errOTAGen
	}

	OTAWanFormatRaw, err := keystore.WaddrFromUncompressedRawBytes(OTARaw)
	if err != nil {
		return "", errOTAGen
	}

	return hexutil.Encode(OTAWanFormatRaw[:]), nil
}

// generate data for wan coin mint transaction
func genBuyCoinData(ota string, value *big.Int) ([]byte, error) {
	coinABI, _ := abi.JSON(strings.NewReader(coinSCDefinition))
	data, err := coinABI.Pack("buyCoinNote", ota, value)
	return data, err
}

// retrieve OTA's balance from state trie
func getOTABalance(db *state.StateDB, ota string) *big.Int {
	otaAX, _ := vm.GetAXFromWanAddr(common.FromHex(ota))
	balance := db.GetStateByteArray(otaBalanceStorageAddr, common.BytesToHash(otaAX))
	return new(big.Int).SetBytes(balance)
}

// return OTA set with num elements
func genOTASet(db *state.StateDB, ota string, num int) ([]string, error) {
	otaAX, _ := vm.GetAXFromWanAddr(common.FromHex(ota))
	otaSet, _, err := vm.GetOTASet(db, otaAX, num)
	if err != nil {
		return nil, err
	}
	ret := make([]string, 0)
	for _, ota := range otaSet {
		ret = append(ret, common.ToHex(ota))
	}

	return ret, err
}

func computeOTAPubKeys(pk, pk1 *ecdsa.PrivateKey, ota string) (string, error) {
	strs := [4]string{}
	l := 32
	for i := 0; i < len(ota)/l; i++ {
		strs[i] = "0x" + ota[i*l:(i+1)*l]
	}

	pub1, priv1, _, err := crypto.GenerteOTAPrivateKey(pk, pk1, strs[0], strs[1], strs[2], strs[3])
	if err != nil {
		return "", err
	}

	pub1X := hexutil.Encode(common.LeftPadBytes(pub1.X.Bytes(), 32))
	pub1Y := hexutil.Encode(common.LeftPadBytes(pub1.Y.Bytes(), 32))
	priv1D := hexutil.Encode(common.LeftPadBytes(priv1.D.Bytes(), 32))

	otaPub := pub1X + pub1Y[2:]
	otaPriv := priv1D

	sk, err := crypto.HexToECDSA(otaPriv[2:])
	if err != nil {
		return "", err
	}

	var addr common.Address
	pubkey := crypto.FromECDSAPub(&sk.PublicKey)
	copy(addr[:], crypto.Keccak256(pubkey[1:])[12:])

	return otaPriv + "+" + otaPub + "+" + hexutil.Encode(addr[:]), err
}

func genRingSignData(addr common.Address) {

}
