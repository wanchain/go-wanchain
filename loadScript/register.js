

var tranValue = 100000
var passwd = "wanglu"

var secAddr = personal.newAccount(passwd)
console.log("secAddr: ", secAddr)

var pubs = personal.showPublicKey(secAddr,passwd)
console.log("pubs: ", pubs)
var secpub  = pubs[0]
var g1pub   = pubs[1]

// for pos trsaction gas fee
eth.sendTransaction({from:eth.coinbase, to: secAddr, value: web3.toWin(1)})

var cscDefinition = [
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
			},
			{
				"name": "feeRate",
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
	}
]

/////////////////////////////////register staker////////////////////////////////////////////////////////////////////////

var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000DA";
var coinContract = contractDef.at(cscContractAddr);

var lockTime = 7
var feeRate = 79

// add validator
var payload = coinContract.stakeIn.getData(secpub, g1pub, lockTime, feeRate)
console.log("payload: ", payload)
var tx = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx= " + tx)

// add delegator
var payloadDelegate = coinContract.delegateIn.getData(secAddr)
var tx2 = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:web3.toWin(tranValue), data:payloadDelegate, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx2= " + tx2)

// append delegate
var payloadDelegate3 = coinContract.delegateIn.getData(secAddr)
var tranValue3 = 140
var tx3 = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:web3.toWin(tranValue3), data:payloadDelegate3, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx3= " + tx3)

// append validator
var tranValue4 = 11111
var payload4 = coinContract.stakeAppend.getData(secAddr)
console.log("payload: ", payload)
var tx = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:web3.toWin(tranValue4), data:payload4, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx4=" + tx)


// delegateOut
var payloadDelegate5 = coinContract.delegateOut.getData(secAddr)
var tx5 = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:'0x00', data:payloadDelegate5, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx5= " + tx5)

// update validator
var payload5 = coinContract.stakeUpdate.getData(secAddr, 12, 31)
console.log("payload5: ", payload5)
var tx = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:'0x00', data:payload5, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx5=" + tx)

/////////////////////////////////unregister staker//////////////////////////////////////////////////////////////////////
