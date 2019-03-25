

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
	}
]

/////////////////////////////////register staker////////////////////////////////////////////////////////////////////////

var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000d2";
var coinContract = contractDef.at(cscContractAddr);

var lockTime = 30
var feeRate = 79

var payload = coinContract.stakeIn.getData(secpub, g1pub, lockTime, feeRate)
console.log("payload: ", payload)
var tx = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx=" + tx)

var payloadDelegate = coinContract.delegateIn.getData(secAddr)
var tx2 = eth.sendTransaction({from:eth.coinbase, to:cscContractAddr, value:web3.toWin(tranValue), data:payloadDelegate, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx2=" + tx2)

/////////////////////////////////unregister staker//////////////////////////////////////////////////////////////////////
