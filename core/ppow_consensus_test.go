// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"
	"github.com/wanchain/go-wanchain/consensus/ethash"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
	"testing"
	"github.com/wanchain/go-wanchain/core/types"
)

/*
Test
VerifyHeader

VerifyHeaders
VerifyPPOWReorg


verifySignerIdentity and snapshot are called by
对被测试接口的数据状态的分解
*/

var (
	genesisBlock * types.Block
)
// newTestBlockChain creates a blockchain without validation.
func newTestBlockChainEx(fake bool) (*BlockChain, *ChainEnv) {
	db, _ := ethdb.NewMemDatabase()
	gspec := DefaultPPOWTestingGenesisBlock()
	gspec.ExtraData = make([]byte, 0)
	for k := range signerSet{
		gspec.ExtraData = append(gspec.ExtraData, k.Bytes()...)
	}
	gspec.Difficulty = big.NewInt(1)
	genesisBlock = gspec.MustCommit(db)
	engine := ethash.NewFullFaker(db)
	if !fake {
		engine = ethash.NewTester(db)
	}
	blockchain, err := NewBlockChain(db, gspec.Config, engine, vm.Config{})
	if err != nil {
		panic(err)
	}
	chainEnv := NewChainEnv(params.TestChainConfig, gspec, engine, blockchain, db)
	blockchain.SetValidator(bproc{})
	return blockchain, chainEnv
}

func TestVerifyHeader(t *testing.T)  {
	blockchain, chainEnv := newTestBlockChainEx(false)
	blocks, _ := chainEnv.GenerateChainEx(genesisBlock, []int{1,2,3}, nil)
	err := blockchain.engine.VerifyHeader(blockchain, blocks[0].Header(), true)
	if err != nil {
		t.Error("valid block verify failed unexpect")
	}

	headers := make([]*types.Header, len(blocks))
	seals := make([]bool, 0)
	for i, block := range blocks{
		headers[i] = block.Header()
		seals = append(seals, true)
	}

	abort, results := blockchain.engine.VerifyHeaders(blockchain, headers, seals)
	defer close(abort)

	for _ = range headers {
		err := <- results
		if err != nil {
			t.Error("valid headers verify failed unexpected")
		}
	}
}

func TestVerifyHeadersFailed(t *testing.T)  {
	blockchain, chainEnv := newTestBlockChainEx(false)
	blocks, _ := chainEnv.GenerateChainEx(genesisBlock, []int{1,2,3,4, 1,1,1}, nil)

	headers := make([]*types.Header, len(blocks))
	seals := make([]bool, 0)
	for i, block := range blocks{
		headers[i] = block.Header()
		seals = append(seals, true)
	}

	abort, results := blockchain.engine.VerifyHeaders(blockchain, headers, seals)
	defer close(abort)

	isFail := false
	for _ = range headers {
		err := <- results
		//fmt.Printf("verify %s \n", headers[i].Coinbase.String())
		if err != nil {
			isFail = true
			break
		}
	}
	if isFail != true {
		t.Error("invalid headers verify passed unexpected")
	}
}


func TestVerifyHeaderLoadSnapshot(t *testing.T)  {
	blockchain, chainEnv := newTestBlockChainEx(true)
	signerSeq := make([]int, 0)
	for i := 0 ; i < 100; i++ {
		signerSeq = append(signerSeq, i%20)
	}
	blocks, _ := chainEnv.GenerateChainEx(genesisBlock, signerSeq, nil)
	blockchain.InsertChain(blocks[:99])

	err := blockchain.engine.VerifyHeader(blockchain, blocks[99].Header(), true)
	if err != nil {
		t.Error("valid block verify failed unexpect")
	}
}


func create2ChainContextSameGenesis()(*BlockChain, *ChainEnv, *BlockChain, *ChainEnv) {
	db, _ := ethdb.NewMemDatabase()
	gspec := DefaultPPOWTestingGenesisBlock()
	gspec.ExtraData = make([]byte, 0)
	for k := range signerSet{
		gspec.ExtraData = append(gspec.ExtraData, k.Bytes()...)
	}
	gspec.Difficulty = big.NewInt(1)
	genesisBlock = gspec.MustCommit(db)

	engine := ethash.NewFaker(db)
	blockchain, err := NewBlockChain(db, gspec.Config, engine, vm.Config{})
	if err != nil {
		panic(err)
	}
	chainEnv := NewChainEnv(params.TestChainConfig, gspec, engine, blockchain, db)
	blockchain.SetValidator(bproc{})

	newDb, _ := ethdb.NewMemDatabase()
	gspec.MustCommit(newDb)
	newBlockChain, err := NewBlockChain(newDb, gspec.Config, engine, vm.Config{})
	if err != nil {
		panic(err)
	}
	newChainEnv := NewChainEnv(params.TestChainConfig, gspec, engine, newBlockChain, newDb)

	return blockchain, chainEnv, newBlockChain, newChainEnv

}
func TestVerifyPPOWReorgSuccess(t *testing.T){
	blockchain, chainEnv, _, longChainEnv := create2ChainContextSameGenesis()
	signerSeq := make([]int, 0)
	for i := 0 ; i < 100; i++ {
		signerSeq = append(signerSeq, i%10)
	}
	blocks, _ := chainEnv.GenerateChainEx(genesisBlock, signerSeq, nil)
	blockchain.InsertChain(blocks[:99])

	longSignerSeq := make([]int, 0)
	for i := 0 ; i < 110; i++ {
		longSignerSeq = append(longSignerSeq, (i+1)%20)
	}
	longBlocks, _ := longChainEnv.GenerateChainEx(genesisBlock, longSignerSeq, nil)
	_, err := blockchain.InsertChain(longBlocks)
	if err != nil{
		t.Errorf("valid reorg failed: %s\n" , err.Error())
	}
}

func TestVerifyPPOWReorgFailed(t *testing.T){
	blockchain, chainEnv, _, longChainEnv := create2ChainContextSameGenesis()
	signerSeq := make([]int, 0)
	for i := 0 ; i < 100; i++ {
		signerSeq = append(signerSeq, i%20)
	}
	blocks, _ := chainEnv.GenerateChainEx(genesisBlock, signerSeq, nil)
	blockchain.InsertChain(blocks[:99])

	longSignerSeq := make([]int, 0)
	for i := 0 ; i < 110; i++ {
		longSignerSeq = append(longSignerSeq, (i+1)%10)
	}
	longBlocks, _ := longChainEnv.GenerateChainEx(genesisBlock, longSignerSeq, nil)
	_, err := blockchain.InsertChain(longBlocks)
	if err == nil{
		t.Errorf("invalid reorg shouldn't sucess: %s\n" , err.Error())
	}
}
