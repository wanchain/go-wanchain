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
	// Wanchain Foundation Go Bootnodes
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@13.250.146.205:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@13.250.146.14:17717",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
var TestnetBootnodes = []string{
	"enode://ce45ef8eb81b66c6bfdd255cd0cad8e7f75bcfea39edd130012ca6f437a9518b9d61447adab25e189fa39673a33107efa1978e332851fcb407f2fbee2fe61961@52.39.32.90:17717",   //nk050
	"enode://23208f45ddc7c96f635459a1740011746abfc3a74c02794512d84537adfcf51e8de89391227e84a09e2a3fdc954b247a3307788bc7cb576b80c6ef3bc6e456b3@52.38.204.254:17717", //nk049

	"enode://01b31508ed81c2c6f1bb941faee1f6b0c5445994311a8bcc916c04f5693e19d8a4fdc7b4bfbd1aa65bcf32dc4a222fa5709e52e5fc624b81ef8edd79ca281e88@34.214.174.220:17717", //nk048
	"enode://84c15ab07550c70b5b96d1e40d2347a718bb08500f804a4ca621a07ad66bb78503bd1e1711ff9ae585ad5bf154d72f314aed1b49bc42f65614a4ebadeb8cba0d@52.40.83.86:17717",    //nk047
}

var InternalBootnodes = []string{
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@118.190.33.68:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",
}

// PlutoBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Pluto test network.
var PlutoBootnodes = []string{
	"enode://9c6d6f351a3ede10ed994f7f6b754b391745bba7677b74063ff1c58597ad52095df8e95f736d42033eee568dfa94c5a7689a9b83cc33bf919ff6763ae7f46f8d@121.42.8.74:17717",
}
var PlutoV5ootnodes = []string{
	"enode://9c6d6f351a3ede10ed994f7f6b754b391745bba7677b74063ff1c58597ad52095df8e95f736d42033eee568dfa94c5a7689a9b83cc33bf919ff6763ae7f46f8d@121.42.8.74:17717?discport=17718",
}

var InternalV5Bootnodes = []string{
	"enode://b969b3fb8365086cdccef44562945939863a578fab8d00eb1f40089cb9eff1e105e4c396eed856aacc6b6d3eaa15efa639d8c6fca2e0eacad6f99389f6d96bb8@118.190.33.66:17717",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@118.190.33.68:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",
}
