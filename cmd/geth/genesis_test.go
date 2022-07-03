// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var customGenesisTests = []struct {
	genesis string
	query   string
	result  string
}{
	// Genesis file with an empty chain configuration (ensure missing fields work)
	{
		genesis: `{
	"config": {
		"chainId": 6,
		"homesteadBlock": 0,
		"daoForkBlock": 0,
		"eip150Block": 0,
		"eip150Hash": "0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0",
		"eip155Block": 0,
		"eip158Block": 0,
		"byzantiumBlock": 0,
		"constantinopleBlock": 10000,
		"petersburgBlock": 10000,
		"istanbulBlock": 10000,
		"muirGlacierBlock": 10000,
		"berlinBlock": 10000,
		"londonBlock": 10000,
		"ethash": {},
		"posFirstBlock": 1,
		"pluto": {
			"period": 10,
			"epoch": 100
		}
	},
	"nonce": "0x0",
	"timestamp": "0x59f83144",
	"extraData": "0x00000000000000000000000000000000000000000000000000000000000000002d0e7c0813a51d3bd1d08246af2a8a7a57d8922e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	"gasLimit": "0x2cd29c0",
	"difficulty": "0x1",
	"mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	"coinbase": "0x0000000000000000000000000000000000000000",
	"alloc": {},
	"number": "0x0",
	"gasUsed": "0x0",
	"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	"baseFeePerGas": null
}`,
		query:  "eth.getBlock(0).difficulty",
		result: "1",
	},
	//Genesis file with specific chain configurations
	{
		genesis: `{
	"config": {
		"chainId": 6,
		"homesteadBlock": 0,
		"daoForkBlock": 0,
		"eip150Block": 0,
		"eip150Hash": "0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0",
		"eip155Block": 0,
		"eip158Block": 0,
		"byzantiumBlock": 0,
		"constantinopleBlock": 10000,
		"petersburgBlock": 10000,
		"istanbulBlock": 10000,
		"muirGlacierBlock": 10000,
		"berlinBlock": 10000,
		"londonBlock": 10000,
		"ethash": {},
		"posFirstBlock": 1,
		"pluto": {
			"period": 10,
			"epoch": 100
		}
	},
	"nonce": "0x0",
	"timestamp": "0x59f83149",
	"extraData": "0x00000000000000000000000000000000000000000000000000000000000000002d0e7c0813a51d3bd1d08246af2a8a7a57d8922e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	"gasLimit": "0x2cd29c0",
	"difficulty": "0x2",
	"mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	"coinbase": "0x0000000000000000000000000000000000000000",
	"alloc": {},
	"number": "0x0",
	"gasUsed": "0x0",
	"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	"baseFeePerGas": null
}`,
		query:  "eth.getBlock(0).difficulty",
		result: "2",
	},
}

// Tests that initializing Geth with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	for i, tt := range customGenesisTests {
		// Create a temporary data directory to use and inspect later
		datadir := tmpdir(t)
		defer os.RemoveAll(datadir)

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := ioutil.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGeth(t, "--datadir", datadir, "init", json).WaitExit()

		// Query the custom genesis block
		geth := runGeth(t, "--networkid", "1337", "--syncmode=full", "--cache", "16",
			"--datadir", datadir, "--maxpeers", "0", "--port", "0",
			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--exec", tt.query, "console")
		geth.ExpectRegexp(tt.result)
		geth.ExpectExit()
	}
}
