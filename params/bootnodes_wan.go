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
	//// Wanchain Foundation Bootnodes,SG NK001
	//"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@13.250.146.205:17717",
	//// Wanchain Foundation Bootnodes,SG NK002
	//"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@13.250.146.14:17717",
	//
	////Wanchain Foundation Bootnodes,US EAST NK003
	//"enode://52d23259b4b30eb3d8e2c8cb7aaf2debac9d731476dcb516d11fa4931dd39d4158f474bb1aff47e8897b50820ba404c3927f1ec833707e594d4b9238411ac465@18.216.140.128:17717",
	////Wanchain Foundation Bootnodes,US EAST NK004
	//"enode://262f3b1db1652c21dfd21df7b07ad244c81d4f92dc9523281ac657b0723f64a80c8e01d8518d6fe7ca3981149234a54ab6b549005f9741e509b5c186122f6d5f@52.15.203.224:17717",
	//
	////Wanchain Foundation Bootnodes,US WEST NK005
	//"enode://cd003df4a1883493b5f86c1d1d1bd6c39dc2ab171b76d66fce2526e22eda83517a892724d7b8200f0edf9e0d9c7f1b1d4e207faa68847eafd41e3fd325ecc82d@34.211.235.236:17717",
	////Wanchain Foundation Bootnodes,US WEST NK006
	//"enode://e82ad9b30bd10d3359c1db0d6be72a49b783b5c8bc040b3c0d09651fdc7ff0874156c89284fb70b03b0c520caf904f5f92442aad7a705e56067556ffd6f15fed@35.165.177.61:17717",

	// new pos
	"enode://9a0539d2777c33532b3450b88343266637daa93776126b580bfdabc2d7a566a553be68079e14a37d3803e5bdb65055c3ac9342b03f1939c4fb2e413ecbc102b1@52.24.132.78:17717",
	"enode://11c018f57c1c4dbb89f57832936dacf5f7ad677b33b86fbb9fe268d3c97126a625661e06e894896613edf5167e9e77867f9c0be8620e150ac6f2b92e5c0efc3d@52.41.157.63:17717",
	"enode://8980a56b1c1580080d2c1e96c9bbb2a9a50528cf18cbff82cc73bee1c8f3821fc65b098ac910a1a567153fd4cb7478b727f6ab0dd726afdbeec1f6b94cac4cd7@54.245.68.228:17717",
	"enode://0b324af8b1489202f09a48f582df6c42319041f1367d8d8d4363caa105b040b8ba73561e41691e40bfcd41e849b89b077beb38dfa188517e4e67b33dd115e4a4@35.162.114.208:17717",
	"enode://fc182f486efadf112cbb4a3e552e1f3f25811fa863352ee6b9ae3f1d9855b4018af47d55ea747d808a12159e9bbd7e7c74726cd8bc3dfe2dbb2b0a1d07a31578@54.148.77.52:17717",
	"enode://fdbdf600ed6eb35d177d0f971360ead71f21d20b31c8924d8d61fabf07fb720b60bb3d5ced5d85a30fce8815479c520a9cc9dcbf5416fff366ba24c37cb1b980@34.218.127.55:17717",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
var TestnetBootnodes = []string{
	"enode://e5fcbadaaa482b50c5538b8ffb6527522c98c11635bfd5d780f9a36dd96e79c08c83620e74b2b6335f9dd5a2e08b2d5d82874415925a1858e0e3fdfcebccdae1@54.203.165.174:17717",
	//"enode://ce45ef8eb81b66c6bfdd255cd0cad8e7f75bcfea39edd130012ca6f437a9518b9d61447adab25e189fa39673a33107efa1978e332851fcb407f2fbee2fe61961@52.39.32.90:17717",   //nk050
	//"enode://23208f45ddc7c96f635459a1740011746abfc3a74c02794512d84537adfcf51e8de89391227e84a09e2a3fdc954b247a3307788bc7cb576b80c6ef3bc6e456b3@52.38.204.254:17717", //nk049
	//
	//"enode://01b31508ed81c2c6f1bb941faee1f6b0c5445994311a8bcc916c04f5693e19d8a4fdc7b4bfbd1aa65bcf32dc4a222fa5709e52e5fc624b81ef8edd79ca281e88@34.214.174.220:17717", //nk048
	//"enode://84c15ab07550c70b5b96d1e40d2347a718bb08500f804a4ca621a07ad66bb78503bd1e1711ff9ae585ad5bf154d72f314aed1b49bc42f65614a4ebadeb8cba0d@52.40.83.86:17717",    //nk047
}

var InternalBootnodes = []string{
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@118.190.33.68:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",
}

// PlutoBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Pluto test network.
var PlutoBootnodes = []string{
	"enode://9c6d6f351a3ede10ed994f7f6b754b391745bba7677b74063ff1c58597ad52095df8e95f736d42033eee568dfa94c5a7689a9b83cc33bf919ff6763ae7f46f8d@121.42.8.74:17717",
	//"enode://d4ec4e3208c17ee38d80121ac536f0a2f9086131d0878fa9ce80eae5ee2a2368c3de08c500cb28d8e621a7ae3d4238862033ea98622ff17bf505627141c6b4cf@127.0.0.1:17717",
	"enode://d4ec4e3208c17ee38d80121ac536f0a2f9086131d0878fa9ce80eae5ee2a2368c3de08c500cb28d8e621a7ae3d4238862033ea98622ff17bf505627141c6b4cf@34.216.68.63:17717",
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
