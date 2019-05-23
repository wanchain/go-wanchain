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

var PlutoAllocJson = `{
	"0xcf696d8eea08a311780fb89b20d4f0895198a489": {"balance": "10000000000000000000000000000000000000",
		"staking":{
			"amount":"4000000000000000000000000",
			"s256pk":"0x04dc40d03866f7335e40084e39c3446fe676b021d1fcead11f2e2715e10a399b498e8875d348ee40358545e262994318e4dcadbc865bcf9aac1fc330f22ae2c786",
			"bn256pk":"0x259b72895ca4bcafceb189b0ea8f6460fd568f688e1dbe345392702e4b000e0f12f5f783218e4334113883f9e18c9ec7b167ddf52a64e6a22dd87f1e3b0db21c"
		}
	},
	"0x6e6f37b8463b541fd6d07082f30f0296c5ac2118": {"balance": "10000000000000000000000000000000000000",
		"staking":{
			"amount":"1000000000000000000000000",
			"s256pk":"0x0438da60706de12194d9d94aabd1b81dd2c5595f00317fd03f96131ca529788e59bb19daf478f8a6fc936c1e0c105a334c11e952c9c1d9e87213fdbf153150f5e3",
			"bn256pk":"0x1124aaa8d4304eb517aaf95e45e636455f97262319d49906cbe75b42017f58ae29d8863ae89e6f1120568bf7b3aa4dab03f2d7d732c250d5ca97422534a92395"
		}
	},

	"0xd1d1079cdb7249eee955ce34d90f215571c0781d": {"balance": "10000000000000000000000000000000000000",
		"staking":{
			"amount":"1000000000000000000000000",
			"s256pk":"0x04e78171373e7d4671fe7a0ab7c3983f46874fab1db2cce81ce512059e5b7e94373abf9875a1dd339aca8d36bdaad6d7542f3243f488155fc12e3a26c6e2f753cd",
			"bn256pk":"0x01050cd3e98b85e7c42b5fa60d4b459522014c9b9dcf4bfab4ae62c35627f01e1ef20d4c65fba61a083dc6c04208bcf6259da61b6eec29bc93e4c11e7199f2e6"
		}
	},
	"0x9cd8230d43464ae97f60bad6de9566a064990e55": {"balance": "100000000000000000000000000000000000026"},
	"0x18dfab02c42b0bbb33034120f2f96dc8ad99308a": {"balance": "10000000000000000000000"},
	"0xb0daf2a0a61b0f721486d3b88235a0714d60baa6": {"balance": "10000000000000000000000"}

 }`

var PlutoDevAllocJson = `{
	"0xcf696d8eea08a311780fb89b20d4f0895198a489": {"balance": "100000000000000000000000000000000000012"},
	"0x18dfab02c42b0bbb33034120f2f96dc8ad99308a": {"balance": "10000000000000000000000"},
	"0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e": {"balance": "10000000000000000000000000000000000000",
		"staking":{
			"amount":"4000000000000000000000000",
			"s256pk":"0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70",
			"bn256pk":"0x2b18624b28a714be61ca85bbbcc22dd47717494493fdb86fcae04928119fb00d15cd5544da7c76d347d4d27b2552e5044ae0e4cff2836ad7bb0de6afe46124a9"
		}
	  }	
 }`

const devAllocData = "\xf9\x01\x94\xc2\x01\x01\xc2\x02\x01\xc2\x03\x01\xc2\x04\x01\xf0\x94\x1a&3\x8f\r\x90^)_\u0337\x1f\xa9\ua11f\xfa\x12\xaa\xf4\x9a\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x94.\xf4q\x00\xe0x{\x91Q\x05\xfd^?O\xf6u y\xd5\u02da\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x94l8jK&\xf7<\x80/4g?rH\xbb\x11\x8f\x97BJ\x9a\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x94\xb9\xc0\x15\x91\x8b\u06ba$\xb4\xff\x05z\x92\xa3\x87=n\xb2\x01\xbe\x9a\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x94\xcd*=\x9f\x93\x8e\x13\u0354~\xc0Z\xbc\u007f\xe74\u07cd\xd8&\x9a\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x94\xdb\xdb\xdb,\xbd#\xb7\x83t\x1e\x8d\u007f\xcfQ\xe4Y\xb4\x97\u499a\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x94\xe4\x15{4\xea\x96\x15\u03fd\xe6\xb4\xfd\xa4\x19\x82\x81$\xb7\fx\x9a\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x94\xe6qo\x95D\xa5lS\r\x86\x8eK\xfb\xac\xb1r1[\u07ad\x9a\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

/*
* wanchain balance allocation

* wanchain allocation unit: 1 wan = 1000000000000000000 win

* wanchain wancoin total:210000000 wan


* wanchain foundattion balance total: 19%, 39900000 wan

* wanchain foundation address 1 balance:1０％,21000000 wan
* "0x4A2a82864c2Db7091142C2078543FaB0286251a9": {"balance": "21000000000000000000000000"},

* wanchain foundation address 2 balance:６％, 12600000 wan
* "0x0dA4512bB81fA50891533d9A956238dCC219aBCf": {"balance": "12600000000000000000000000"},

* wanchain foundation address 3 balance:３％, 6300000 wan
* "0xD209fe3073Ca6B61A1a8A7b895e1F4aD997103E1": {"balance": "6300000000000000000000000"},

* wanchain miner total: 10%, 21000000 wan
* "0x01d0689001F18637993948e487a15BF3064b16e4": {"balance": "21000000000000000000000000"},


* wanchain open sold balance:total 51%, 107100000 wan

* wanchain open sold address 1 balance:１０％,21000000 wan
* "0xb3a5c9789A4d882BceF63abBe9B7893aC505bf60": {"balance": "21000000000000000000000000"},

* wanchain open sold address 2 balance:10%,21000000 wan
* "0xc57FeeC601d5A473fE9d1D70Af26ac639e0c61a1": {"balance": "21000000000000000000000000"},

* wanchain open sold address 3 balance:１０％,21000000 wan
* "0xEeCABC0900998aFeE0B52438a6003F2388c78A62": {"balance": "21000000000000000000000000"},

* wanchain open sold address 4 balance:１０％,21000000 wan
* "0x2dC9A6A04Bc004a8f68f0e886a463AeF23D43030": {"balance": "21000000000000000000000000"},

* wanchain open sold address 5 balance:１０％,21000000 wan
* "0x5866dD6794B8996E5bC745D508AC6901FF3b0427": {"balance": "21000000000000000000000000"},

* wanchain open sold address 6 balance:1%,2100000 wan
* "0x89442477dC39A2503E30D1f8d7FFD4Ea5f87a2aF": {"balance":  "2100000000000000000000000"},


* wanchain develop team balance:20%,42000000 wan

* wanchain develop team address:10%,21000000 wan
* "0xae8d9B975eC8df8359eA79e50e89b18601816aC3": {"balance": "21000000000000000000000000"},

* wanchain develop team address:5%,10500000 wan
* "0x53D81A644a0d1081D6C6E8B25f807C2cFb6edE35": {"balance": "10500000000000000000000000"},

* wanchain develop team address:5%,10500000 wan
* "0x3B9289124f04194F0b3C4F8F862fE1Fbac59c978": {"balance": "10500000000000000000000000"}

 */

const wanchainAllocJson = `{
		"0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e": {"balance": "21000000000000000000000000"},
		"0x0dA4512bB81fA50891533d9A956238dCC219aBCf": {"balance": "12600000000000000000000000"},
		"0xD209fe3073Ca6B61A1a8A7b895e1F4aD997103E1": {"balance":  "6300000000000000000000000"},

		"0x01d0689001F18637993948e487a15BF3064b16e4": {"balance": "21000000000000000000000000"},

		"0xb3a5c9789A4d882BceF63abBe9B7893aC505bf60": {"balance": "21000000000000000000000000"},
		"0xc57FeeC601d5A473fE9d1D70Af26ac639e0c61a1": {"balance": "21000000000000000000000000"},
		"0xEeCABC0900998aFeE0B52438a6003F2388c78A62": {"balance": "21000000000000000000000000"},
		"0x2dC9A6A04Bc004a8f68f0e886a463AeF23D43030": {"balance": "21000000000000000000000000"},
		"0x5866dD6794B8996E5bC745D508AC6901FF3b0427": {"balance": "21000000000000000000000000"},
		"0x89442477dC39A2503E30D1f8d7FFD4Ea5f87a2aF": {"balance":  "2100000000000000000000000"},

		"0xae8d9B975eC8df8359eA79e50e89b18601816aC3": {"balance": "21000000000000000000000000"},
		"0x53D81A644a0d1081D6C6E8B25f807C2cFb6edE35": {"balance": "10500000000000000000000000"},
		"0x3B9289124f04194F0b3C4F8F862fE1Fbac59c978": {"balance": "10500000000000000000000000"}
}`

//miner reward
//public sale
//Team holding
//Foundation operation
const wanchainTestAllocJson = `{
	"0x4cb79c7868cd88629df6d4fa8637dda83d13ef27": {"balance": "21000000000000000000000000"},
	"0xeb71d33d5c7cf05d9177934200c51efa53057c27": {"balance": "107100000000000000000000000"},
	"0x6b4683cafa549d9f4c06815a2397cef5a540b919": {"balance": "42000000000000000000000000"},
	"0xbb9003ca8226f411811dd16a3f1a2c1b3f71825d": {"balance": "39900000000000000000000000"}
}`


const wanchainInternalAllocJson = `{
	"0x4cb79c7868cd88629df6d4fa8637dda83d13ef27": {"balance": "21000000000000000000000000"},
	"0xeb71d33d5c7cf05d9177934200c51efa53057c27": {"balance": "107100000000000000000000000"},
	"0x6b4683cafa549d9f4c06815a2397cef5a540b919": {"balance": "42000000000000000000000000"},
	"0xbb9003ca8226f411811dd16a3f1a2c1b3f71825d": {"balance": "39900000000000000000000000"},
	"0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e": {"balance": "10000000000000000000000000000000000000",
		"staking":{
			"amount":"4000000000000000000000000",
			"s256pk":"0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70",
			"bn256pk":"0x2b18624b28a714be61ca85bbbcc22dd47717494493fdb86fcae04928119fb00d15cd5544da7c76d347d4d27b2552e5044ae0e4cff2836ad7bb0de6afe46124a9"
		}
	}	
}`
const wanchainPPOWTestAllocJson = `{
	  "0xbd100cf8286136659a7d63a38a154e28dbf3e0fd": {"balance": "3000000000000000000000000000"},
	  "0xF9b32578b4420a36F132DB32b56f3831A7CC1804": {"balance": "3000000000000000000000000000"},
	  "0x1631447d041f929595a9c7b0c9c0047de2e76186": {"balance": "1000"}
}`

const wanchainPPOWDevAllocJson = `{
	  "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e": {"balance": "999999999999999999999999999999999999999999999999999999"},
	  "0x8b179c2b542f47bb2fb2dc40a3cf648aaae1df16": {"balance": "999999999999999999999999999999999999999999999999999999"},
	  "0x7a22d4e2dc5c135c4e8a7554157446e206835b05": {"balance": "999999999999999999999999999999999999999999999999999999"}
}`
