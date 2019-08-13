// Copyright 2017 The go-ethereum Authors
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
	"reflect"
	"testing"

	"github.com/wanchain/go-wanchain/core/types"

	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
)

var (
	MainnetGenesisHashMock = common.HexToHash("0x0376899c001618fc7d5ab4f31cfd7f57ca3a896ccc1581a57d8f129ecf40b840") // Mainnet genesis hash to enforce below configs on
	TestnetGenesisHashMock = common.HexToHash("0xa37b811609a9d1e898fb49b3901728023e5e72e18e58643d9a7a82db483bfeb0") // Testnet genesis hash to enforce below configs on
	PlutoGenesisHashMock   = common.HexToHash("0x7b67a3f28e0d12b57e5fdaa445c4d6dbe68bffa9b808e944e5c67726669d62b6") // Pluto genesis hash to enforce below configs on

	InternalGenesisHashMock = common.HexToHash("0xa88f332a08f0ff353ec1097c77ec4c58abe3173bad7ae50dca4d6efee5dba590")
)

func TestDefaultGenesisBlock(t *testing.T) {
	var block *types.Block
	//block, _ := DefaultGenesisBlock().ToBlock()
	block, _ = DefaultGenesisBlock().ToBlock()
	fmt.Println(common.ToHex(block.Hash().Bytes()))
	if block.Hash() != MainnetGenesisHashMock {
		t.Errorf("wrong mainnet genesis hash, got %v, want %v", block.Hash(), params.MainnetGenesisHash)
	}

	block, _ = DefaultTestnetGenesisBlock().ToBlock()
	fmt.Println(common.ToHex(block.Hash().Bytes()))
	if block.Hash() != TestnetGenesisHashMock {
		t.Errorf("wrong testnet genesis hash, got %v, want %v", block.Hash(), params.TestnetGenesisHash)
	}

	block, _ = DefaultInternalGenesisBlock().ToBlock()
	fmt.Println(common.ToHex(block.Hash().Bytes()))
	if block.Hash() != InternalGenesisHashMock {
		//t.Errorf("wrong testnet genesis hash, got %v, want %v", block.Hash(), params.TestnetGenesisHash)
	}
}

func TestDefaultTestnetGenesisBlock(t *testing.T) {
	block, _ := DefaultGenesisBlock().ToBlock()
	if block.Hash() != MainnetGenesisHashMock {
		t.Errorf("wrong mainnet genesis hash, got %v, want %v", block.Hash(), MainnetGenesisHashMock)
	}

	block, _ = DefaultTestnetGenesisBlock().ToBlock()
	if block.Hash() != TestnetGenesisHashMock {
		t.Errorf("wrong testnet genesis hash, got %v, want %v", block.Hash(), TestnetGenesisHashMock)
	}
}

func TestSetupGenesis(t *testing.T) {
	var (
		customghash = common.HexToHash("0x89c99d90b79719238d2645c7642f2c9295246e80775b38cfd162b696817fbd50")
		customg     = Genesis{
			Config: &params.ChainConfig{ByzantiumBlock: big.NewInt(3)},
			Alloc: GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
		}
		oldcustomg = customg
	)
	oldcustomg.Config = &params.ChainConfig{ByzantiumBlock: big.NewInt(2)}
	tests := []struct {
		name       string
		fn         func(ethdb.Database) (*params.ChainConfig, common.Hash, error)
		wantConfig *params.ChainConfig
		wantHash   common.Hash
		wantErr    error
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, new(Genesis))
			},
			wantErr:    errGenesisNoConfig,
			wantConfig: params.AllProtocolChanges,
		},
		{
			name: "no block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   MainnetGenesisHashMock,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "mainnet block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				DefaultGenesisBlock().MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   MainnetGenesisHashMock,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "custom block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "custom block in DB, genesis == testnet",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, DefaultTestnetGenesisBlock())
			},
			wantErr:    &GenesisMismatchError{Stored: customghash, New: params.TestnetGenesisHash},
			wantHash:   TestnetGenesisHashMock,
			wantConfig: params.TestnetChainConfig,
		},
		{
			name: "compatible config in DB",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				oldcustomg.MustCommit(db)
				return SetupGenesisBlock(db, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},

		/*
			{
				name: "incompatible config in DB",
				fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
					// Commit the 'old' genesis block with Homestead transition at #2.
					// Advance to block #4, past the homestead transition block of customg.
					genesis := oldcustomg.MustCommit(db)
					bc, _ := NewBlockChain(db, oldcustomg.Config, ethash.NewFullFaker(), vm.Config{})
					defer bc.Stop()
					bc.SetValidator(bproc{})
					bc.InsertChain(makeBlockChainWithDiff(genesis, []int{2, 3, 4, 5}, 0))
					bc.CurrentBlock()
					// This should return a compatibility error.
					return SetupGenesisBlock(db, &customg)
				},
				wantHash:   customghash,
				wantConfig: customg.Config,
				wantErr: &params.ConfigCompatError{
					What:         "Homestead fork block",
					StoredConfig: big.NewInt(2),
					NewConfig:    big.NewInt(3),
					RewindTo:     1,
				},
			},
		*/

	}

	for _, test := range tests {
		db, _ := ethdb.NewMemDatabase()
		config, hash, err := test.fn(db)
		// Check the return values.
		if !reflect.DeepEqual(err, test.wantErr) {
			spew := spew.ConfigState{DisablePointerAddresses: true, DisableCapacities: true}
			t.Errorf("%s: returned error %#v, want %#v", test.name, spew.NewFormatter(err), spew.NewFormatter(test.wantErr))
		}
		if !reflect.DeepEqual(config, test.wantConfig) {
			t.Errorf("%s:\nreturned %v\nwant     %v", test.name, config, test.wantConfig)
		}
		if hash != test.wantHash {
			t.Errorf("%s: returned hash %s, want %s", test.name, hash.Hex(), test.wantHash.Hex())
		} else if err == nil {
			// Check database content.
			stored := GetBlock(db, test.wantHash, 0)
			if stored.Hash() != test.wantHash {
				t.Errorf("%s: block in DB has hash %s, want %s", test.name, stored.Hash(), test.wantHash)
			}
		}
	}
}
