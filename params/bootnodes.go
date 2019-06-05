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
	// Wanchain Foundation Bootnodes,SG NK001
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@13.250.146.205:17717",
	// Wanchain Foundation Bootnodes,SG NK002
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@13.250.146.14:17717",

	//Wanchain Foundation Bootnodes,US EAST NK003
	"enode://52d23259b4b30eb3d8e2c8cb7aaf2debac9d731476dcb516d11fa4931dd39d4158f474bb1aff47e8897b50820ba404c3927f1ec833707e594d4b9238411ac465@18.216.140.128:17717",
	//Wanchain Foundation Bootnodes,US EAST NK004
	"enode://262f3b1db1652c21dfd21df7b07ad244c81d4f92dc9523281ac657b0723f64a80c8e01d8518d6fe7ca3981149234a54ab6b549005f9741e509b5c186122f6d5f@52.15.203.224:17717",

	//Wanchain Foundation Bootnodes,US WEST NK005
	"enode://cd003df4a1883493b5f86c1d1d1bd6c39dc2ab171b76d66fce2526e22eda83517a892724d7b8200f0edf9e0d9c7f1b1d4e207faa68847eafd41e3fd325ecc82d@34.211.235.236:17717",
	//Wanchain Foundation Bootnodes,US WEST NK006
	"enode://e82ad9b30bd10d3359c1db0d6be72a49b783b5c8bc040b3c0d09651fdc7ff0874156c89284fb70b03b0c520caf904f5f92442aad7a705e56067556ffd6f15fed@35.165.177.61:17717",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
var TestnetBootnodes = []string{
	"enode://b2174fde9d40c3d5775afcc84963bb9055981ce7fdbdfdf8abdca1dd1f942a2eedd52d97cec76480c91e635210daed1eaad26f40b0c5c6b236cca2726fd04da8@54.183.204.219:17717", //nk050
	//"enode://23208f45ddc7c96f635459a1740011746abfc3a74c02794512d84537adfcf51e8de89391227e84a09e2a3fdc954b247a3307788bc7cb576b80c6ef3bc6e456b3@35.167.106.109:17717",//nk049
	//"enode://01b31508ed81c2c6f1bb941faee1f6b0c5445994311a8bcc916c04f5693e19d8a4fdc7b4bfbd1aa65bcf32dc4a222fa5709e52e5fc624b81ef8edd79ca281e88@52.89.169.52:17717", //nk048
	//"enode://84c15ab07550c70b5b96d1e40d2347a718bb08500f804a4ca621a07ad66bb78503bd1e1711ff9ae585ad5bf154d72f314aed1b49bc42f65614a4ebadeb8cba0d@35.160.61.159:17717",//nk047
}

var InternalBootnodes = []string{
	"enode://dea09d1ded799044d3b8b5c66e28e584ea3fdaae12e0e39bb3491ac99424cc6c098f32e978c4aef1c3382c3c4492d7a33d720eabdee78cddba28541d6bef1bdc@54.193.85.171:17717", //
	//"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",
}

// PlutoBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Pluto test network.
var PlutoBootnodes = []string{
	"enode://81ffab14284d29f9a87737780717719666af814d78057ec4b6799b9d275c41e8041887c733a95fba13ae9fba4fb3026d5f53993143b83ab6648cae3b1e5e9c35@18.236.236.189:17717",
	"enode://ca5496aa6eda6403f4ac41e7841d1ae6d963a321afcb8c59c0f2935f837bd2300ff258ab94bf4db375257f29898e5d0ea5903c28a0e9a41a4aba4e100b4b2ed0@34.212.171.224:17717",
	"enode://86989aacffbc22640dee74864ac0f17fb4987ab0b6792a6fd14801557e7f7ff6447d77945d64463d5e6e0bed5ac257c842d9631abe8e68110e1aa9233ad4e3a1@54.184.26.209:27717",
}
var PlutoV5ootnodes = []string{
	"enode://81ffab14284d29f9a87737780717719666af814d78057ec4b6799b9d275c41e8041887c733a95fba13ae9fba4fb3026d5f53993143b83ab6648cae3b1e5e9c35@18.236.236.189:17717?discport=17718",
}

var InternalV5Bootnodes = []string{
	"enode://81ffab14284d29f9a87737780717719666af814d78057ec4b6799b9d275c41e8041887c733a95fba13ae9fba4fb3026d5f53993143b83ab6648cae3b1e5e9c35@18.236.236.189:17717",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	"enode://dfa95c2be31b3541895df452678355ef4b38988863959fe56d1a217cd9fdeee27024cb13688cab56373ac597968aa2faf0da8cd87f4238366ddb41f03fc78884@118.190.33.68:17717", // IE
	"enode://9e41c167954d33f5f5b7740a0f6a03b90ddab423cfd4e1fc6c844feff32e3a5d82e76c20d1823915676b58505efb6d33ea1fa6f7e6e22812b1d7ae7a90874881@118.190.33.102:17717",
}
