
function Bytes2HexString (b) {
    var hexs = "";
    for (var i = 0; i < b.length; i++) {
         var hex = b[i].toString(16);
        if (hex.length == 1) {
            hex = '0' + hex;
        }
        hexs += hex.toUpperCase();
    }
    console.log(hexs);
}

personal.unlockAccount(eth.accounts[0],"wanglu",600);
//personal.unlockAccount(eth.accounts[1],"wanglu",99999);

var tranValue = 10

var secpub  = '0x04a5946c1968bbe53bfd897c06d53555292bef6e71a4c8ed92b9c1de1b1b94f797c3984581307788ff0c2a564548901f83000b1aa65a1532dacca01214e1f3fa6c'
var secAddr = '0x23Fc2eDa99667fD3df3CAa7cE7e798d94Eec06eb'
var g1pub   = '0x1b4626213c1af35b38d226a386e3b661a98198794e52740d2be58c14315dc1a12d8e79a95c2b6a21653550b422bb3211e62b6af6b1afe09c3232bd6c6b601ea5'

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
				"name": "delegateAddr",
				"type": "address"
			},
			{
				"name": "lockEpochs",
				"type": "uint256"
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
console.log(JSON.stringify(cscDefinition) )

var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000d2";
var coinContract = contractDef.at(cscContractAddr);

var lockTime = 30
var feeRate = 100

var payload = coinContract.stakeIn.getData(secpub, g1pub, lockTime, feeRate)
console.log("payload: ", payload)
var tx = eth.sendTransaction({from:eth.accounts[0], to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx=" + tx)

var payloadDelegate = coinContract.delegateIn.getData(secAddr, lockTime)
var tx2 = eth.sendTransaction({from:eth.accounts[0], to:cscContractAddr, value:web3.toWin(tranValue), data:payloadDelegate, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
console.log("tx2=" + tx2)

/////////////////////////////////unregister staker//////////////////////////////////////////////////////////////////////
