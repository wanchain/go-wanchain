// Copyright 2015 The go-ethereum Authors
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

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	// Ethereum Foundation Go Bootnodes
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@118.190.33.68:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",

}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@118.190.33.68:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
	"enode://b969b3fb8365086cdccef44562945939863a578fab8d00eb1f40089cb9eff1e105e4c396eed856aacc6b6d3eaa15efa639d8c6fca2e0eacad6f99389f6d96bb8@118.190.33.66:17717",
	//"enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:17717",  // INFURA
}

// PlutoBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Pluto test network.
var PlutoBootnodes = []string{
	"enode://9c6d6f351a3ede10ed994f7f6b754b391745bba7677b74063ff1c58597ad52095df8e95f736d42033eee568dfa94c5a7689a9b83cc33bf919ff6763ae7f46f8d@121.42.8.74:17717",
}
var PlutoV5ootnodes = []string{
	"enode://9c6d6f351a3ede10ed994f7f6b754b391745bba7677b74063ff1c58597ad52095df8e95f736d42033eee568dfa94c5a7689a9b83cc33bf919ff6763ae7f46f8d@121.42.8.74:30303?discport=17718",
}

// RinkebyV5Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network for the experimental RLPx v5 topic-discovery network.
var RinkebyV5Bootnodes = []string{
	"enode://b969b3fb8365086cdccef44562945939863a578fab8d00eb1f40089cb9eff1e105e4c396eed856aacc6b6d3eaa15efa639d8c6fca2e0eacad6f99389f6d96bb8@118.190.33.66:17717",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@118.190.33.68:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",
}
