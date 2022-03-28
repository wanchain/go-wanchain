// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const MainnetPow2PosUpgradeBlockNumber = 4046000
const TestnetPow2PosUpgradeBlockNumber = 3560000
const InternalPow2PosUpgradeBlockNumber = 200

const MAINNET_CHAIN_ID = 1
const TESTNET_CHAIN_ID = 3

// INTERNAL_CHAIN_ID is Private chain pow -> pos mode chainId
const INTERNAL_CHAIN_ID = 4

// PLUTO_CHAIN_ID is Private chain pos mode chainId
const PLUTO_CHAIN_ID = 6

// PLUTODEV_CHAIN_ID is Private chain pos mode single node chainId
const PLUTODEV_CHAIN_ID = 6

// JUPITER_MAINNET_CHAIN_ID is mainnet chainId after jupiter fork
const JUPITER_MAINNET_CHAIN_ID = 888

// JUPITER_TESTNET_CHAIN_ID is testnet chainId after jupiter fork
const JUPITER_TESTNET_CHAIN_ID = 999
const JUPITER_INTERNAL_CHAIN_ID = 777

//const JUPITER_PLUTO_CHAIN_ID = 6
const JUPITER_PLUTO_CHAIN_ID = 666

//const JUPITER_PLUTODEV_CHAIN_ID = 6
const JUPITER_PLUTODEV_CHAIN_ID = 555

// NOT_JUPITER_CHAIN_ID is used for compare
const NOT_JUPITER_CHAIN_ID = 0xffffffff

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash  = common.HexToHash("0x0376899c001618fc7d5ab4f31cfd7f57ca3a896ccc1581a57d8f129ecf40b840") // Mainnet genesis hash to enforce below configs on
	TestnetGenesisHash  = common.HexToHash("0xa37b811609a9d1e898fb49b3901728023e5e72e18e58643d9a7a82db483bfeb0") // Testnet genesis hash to enforce below configs on
	PlutoGenesisHash    = common.HexToHash("0x7b67a3f28e0d12b57e5fdaa445c4d6dbe68bffa9b808e944e5c67726669d62b6") // Pluto genesis hash to enforce below configs on
	InternalGenesisHash = common.HexToHash("0xb1dc31a86510003c23b9ddee0e194775807262529b8dafa6dc23d9315364d2b3")
)

// PlutoConfig is the consensus engine configs for proof-of-authority based sealing.
type PlutoConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *PlutoConfig) String() string {
	return "pluto"
}

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainneLondonBlockNumber int64 = 21000000
	MainnetChainConfig            = &ChainConfig{
		ChainID:             big.NewInt(MAINNET_CHAIN_ID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(MainneLondonBlockNumber),
		PetersburgBlock:     big.NewInt(MainneLondonBlockNumber),
		IstanbulBlock:       big.NewInt(MainneLondonBlockNumber),
		MuirGlacierBlock:    big.NewInt(MainneLondonBlockNumber),
		BerlinBlock:         big.NewInt(MainneLondonBlockNumber),
		LondonBlock:         big.NewInt(MainneLondonBlockNumber),
		Ethash:              new(EthashConfig),

		// add by Jacob
		PosFirstBlock: big.NewInt(MainnetPow2PosUpgradeBlockNumber), // set as n * epoch_length
		IsPosActive:   false,
		Pluto: &PlutoConfig{
			Period: 10,
			Epoch:  100,
		},
	}

	TestnetSaturnBlockNumber int64 = 18950000
	TestnetLondonBlockNumber       = TestnetSaturnBlockNumber
	TestnetChainConfig             = &ChainConfig{
		ChainID:             big.NewInt(TESTNET_CHAIN_ID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(TestnetLondonBlockNumber),
		PetersburgBlock:     big.NewInt(TestnetLondonBlockNumber),
		IstanbulBlock:       big.NewInt(TestnetLondonBlockNumber),
		MuirGlacierBlock:    big.NewInt(TestnetLondonBlockNumber),
		BerlinBlock:         big.NewInt(TestnetLondonBlockNumber),
		LondonBlock:         big.NewInt(TestnetLondonBlockNumber),
		Ethash:              new(EthashConfig),

		// add by Jacob
		PosFirstBlock: big.NewInt(TestnetPow2PosUpgradeBlockNumber), // set as n * epoch_length
		IsPosActive:   false,
		Pluto: &PlutoConfig{
			Period: 10,
			Epoch:  100,
		},
	}

	InternalLondonBlockNumber int64 = 22000000
	InternalChainConfig             = &ChainConfig{
		ChainID:             big.NewInt(INTERNAL_CHAIN_ID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(InternalLondonBlockNumber),
		PetersburgBlock:     big.NewInt(InternalLondonBlockNumber),
		IstanbulBlock:       big.NewInt(InternalLondonBlockNumber),
		MuirGlacierBlock:    big.NewInt(InternalLondonBlockNumber),
		BerlinBlock:         big.NewInt(InternalLondonBlockNumber),
		LondonBlock:         big.NewInt(InternalLondonBlockNumber),
		Ethash:              new(EthashConfig),

		// add by Jacob
		PosFirstBlock: big.NewInt(InternalPow2PosUpgradeBlockNumber), // set as n * epoch_length
		IsPosActive:   false,
		Pluto: &PlutoConfig{
			Period: 10,
			Epoch:  100,
		},
	}

	PlutoLondonBlockNumber int64 = 10000
	PlutoChainConfig             = &ChainConfig{
		ChainID:             big.NewInt(PLUTO_CHAIN_ID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(PlutoLondonBlockNumber),
		PetersburgBlock:     big.NewInt(PlutoLondonBlockNumber),
		IstanbulBlock:       big.NewInt(PlutoLondonBlockNumber),
		MuirGlacierBlock:    big.NewInt(PlutoLondonBlockNumber),
		BerlinBlock:         big.NewInt(PlutoLondonBlockNumber),
		LondonBlock:         big.NewInt(PlutoLondonBlockNumber),
		Ethash:              new(EthashConfig),

		// add by Jacob
		PosFirstBlock: big.NewInt(1), // set as n * epoch_length
		IsPosActive:   false,
		Pluto: &PlutoConfig{
			Period: 10,
			Epoch:  100,
		},
	}
	noStaking = false
)

func (c *ChainConfig) SetPosActive() {
	c.IsPosActive = true
	SetPosActive(c.IsPosActive)
}

func (c *ChainConfig) IsPosBlockNumber(n *big.Int) bool {
	return n.Cmp(c.PosFirstBlock) >= 0
}

var (
	isPosActive    = false
	isLondonForked = false
	TestnetChainId = TestnetChainConfig.ChainID.Int64()
	MainnetChainId = MainnetChainConfig.ChainID.Int64()
)

func IsPosActive() bool {
	return isPosActive
}

func SetPosActive(active bool) {
	isPosActive = active
}

func IsLondonActive() bool {
	return isLondonForked
}

func SetLondonActive(active bool) {
	isLondonForked = active
}

func IsNoStaking() bool {
	return noStaking
}
func SetNoStaking() {
	noStaking = true
}

func JupiterChainId(chainId uint64) uint64 {
	if chainId == MAINNET_CHAIN_ID {
		return JUPITER_MAINNET_CHAIN_ID
	}

	if chainId == TESTNET_CHAIN_ID {
		return JUPITER_TESTNET_CHAIN_ID
	}

	if chainId == INTERNAL_CHAIN_ID {
		return JUPITER_INTERNAL_CHAIN_ID
	}

	if chainId == PLUTO_CHAIN_ID {
		return JUPITER_PLUTO_CHAIN_ID
	}

	if chainId == PLUTODEV_CHAIN_ID {
		return JUPITER_PLUTODEV_CHAIN_ID
	}

	return NOT_JUPITER_CHAIN_ID
}

func IsOldChainId(chainId uint64) bool {
	if chainId == MAINNET_CHAIN_ID {
		return true
	}

	if chainId == TESTNET_CHAIN_ID {
		return true
	}

	if chainId == INTERNAL_CHAIN_ID {
		return true
	}

	if chainId == PLUTO_CHAIN_ID {
		return true
	}

	if chainId == PLUTODEV_CHAIN_ID {
		return true
	}

	return false
}
