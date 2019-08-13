

var tranValue = 100000
var passwd = "wanglu"

var secAddr = "0x23fc2eda99667fd3df3caa7ce7e798d94eec06eb" // this is a testnet validator
var wallet = "0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8"
//var secAddr = personal.newAccount(passwd)
console.log("secAddr: ", secAddr)

var pubs = personal.showPublicKey(secAddr,passwd)
console.log("pubs: ", pubs)
var secpub  = pubs[0]
var g1pub   = pubs[1]

// for pos trsaction gas fee
//personal.sendTransaction({from:wallet, to: secAddr, value: web3.toWin(1)}, passwd)

var cscDefinition = [
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "stakeAppend",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			}
		],
		"name": "stakeUpdate",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "secPk",
				"type": "bytes"
			},
			{
				"name": "bn256Pk",
				"type": "bytes"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			},
			{
				"name": "feeRate",
				"type": "uint256"
			}
		],
		"name": "stakeIn",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "secPk",
				"type": "bytes"
			},
			{
				"name": "bn256Pk",
				"type": "bytes"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
			},
			{
				"name": "feeRate",
				"type": "uint256"
			},
			{
				"name": "maxFeeRate",
				"type": "uint256"
			}
		],
		"name": "stakeRegister",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			},
			{
				"name": "renewal",
				"type": "bool"
			}
		],
		"name": "partnerIn",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "delegateAddress",
				"type": "address"
			}
		],
		"name": "delegateIn",
		"outputs": [],
		"payable": true,
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "delegateAddress",
				"type": "address"
			}
		],
		"name": "delegateOut",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			},
			{
				"name": "feeRate",
				"type": "uint256"
			}
		],
		"name": "stakeUpdateFeeRate",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "feeRate",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "lockEpoch",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "maxFeeRate",
				"type": "uint256"
			}
		],
		"name": "stakeRegister",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "feeRate",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "lockEpoch",
				"type": "uint256"
			}
		],
		"name": "stakeIn",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			}
		],
		"name": "stakeAppend",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "lockEpoch",
				"type": "uint256"
			}
		],
		"name": "stakeUpdate",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			},
			{
				"indexed": false,
				"name": "renewal",
				"type": "bool"
			}
		],
		"name": "partnerIn",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "v",
				"type": "uint256"
			}
		],
		"name": "delegateIn",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			}
		],
		"name": "delegateOut",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "sender",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "posAddress",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "feeRate",
				"type": "uint256"
			}
		],
		"name": "stakeUpdateFeeRate",
		"type": "event"
	}
]

/////////////////////////////////register staker////////////////////////////////////////////////////////////////////////

var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000DA";
var coinContract = contractDef.at(cscContractAddr);

var lockTime = 7
var feeRate = 9800

// add validator
var payload = coinContract.stakeIn.getData(secpub, g1pub, lockTime, feeRate)
console.log("payload: ", payload)
var tx = personal.sendTransaction({from:wallet, to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
console.log("tx= " + tx)
// var payload = coinContract.stakeRegister.getData(secpub, g1pub, lockTime, feeRate, 9999)
// console.log("payload: ", payload)
// var tx = personal.sendTransaction({from:wallet, to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
// console.log("tx= " + tx)

var addr2 = "0x435b316a70cdb8143d56b3967aacdb6392fd6125"
var pubs2 = personal.showPublicKey(addr2,passwd)
console.log("pubs2: ", pubs2)
var maxFeeRate = 9900
var payload = coinContract.stakeRegister.getData(pubs2[0], pubs2[1], lockTime, feeRate, maxFeeRate)
console.log("payload: ", payload)
tx = personal.sendTransaction({from:wallet, to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
console.log("stakeRegister tx= " + tx)

// add delegator
var tranValue2 = 200
var payloadDelegate = coinContract.delegateIn.getData(secAddr)
var tx2 = personal.sendTransaction({from:wallet, to:cscContractAddr, value:web3.toWin(tranValue2), data:payloadDelegate, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
console.log("tx2= " + tx2)

// append delegate
var payloadDelegate3 = coinContract.delegateIn.getData(secAddr)
var tranValue3 = 100
var tx3 = personal.sendTransaction({from:wallet, to:cscContractAddr, value:web3.toWin(tranValue3), data:payloadDelegate3, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
console.log("tx3= " + tx3)

var payloadDelegate31 = coinContract.delegateOut.getData(secAddr)
var tx31 = personal.sendTransaction({from:wallet, to:cscContractAddr, value:'0x00', data:payloadDelegate31, gas: 200000, gasprice:'0x' + (200000000000).toString(16)}, passwd);
console.log("tx31= " + tx31)

// append validator
var tranValue4 = 1000
var payload4 = coinContract.stakeAppend.getData(secAddr)
console.log("payload: ", payload)
var tx = personal.sendTransaction({from:wallet, to:cscContractAddr, value:web3.toWin(tranValue4), data:payload4, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
console.log("tx4=" + tx)



// update validator in pow phase, this can't work.
var payload5 = coinContract.stakeUpdate.getData(secAddr, 0)
console.log("payload5: ", payload5)
tx = personal.sendTransaction({from:wallet, to:cscContractAddr, value:'0x00', data:payload5, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
console.log("tx5=" + tx)

var payload6 = coinContract.stakeUpdateFeeRate.getData(secAddr, 50)
console.log("payload6: ", payload6)
tx = personal.sendTransaction({from:wallet, to:cscContractAddr, value:'0x00', data:payload6, gas: 200000, gasprice:'0x' + (200000000000).toString(16)},passwd);
console.log("stakeUpdateFeeRate tx6=" + tx)

var bContinue = true
tranValue = 20000
var payload7 = coinContract.partnerIn.getData(secAddr, bContinue)
console.log("payload7: ", payload7)
tx = personal.sendTransaction({ from: wallet, to: cscContractAddr, value: web3.toWin(tranValue), data: payload7, gas: 200000, gasprice: '0x' + (200000000000).toString(16)}, passwd);
console.log("partnerIn tx=" + tx)
/////////////////////////////////unregister staker//////////////////////////////////////////////////////////////////////
