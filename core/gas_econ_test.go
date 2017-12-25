package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/consensus/ethash"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
)

func TestGasOrdinaryCoinTransfer(t *testing.T) {
	var (
		initialBalance = big.NewInt(1000000000)
		// value ot Wan coin to transfer
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
