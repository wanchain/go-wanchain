
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

var secpub  = '0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70'
var g1pub   = '0x150b2b3230d6d6c8d1c133ec42d82f84add5e096c57665ff50ad071f6345cf45191fd8015cea72c4591ab3fd2ade12287c28a092ac0abf9ea19c13eb65fd4910'

var cscDefinition =
[
	{
		"constant": false,
		"inputs": [
			{
				"name": "sPub",
				"type": "string"
			},
			{
				"name": "bnPub",
				"type": "string"
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
		"payable": false,
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "delegateSpub",
				"type": "string"
			}
		],
		"name": "delegateIn",
		"outputs": [],
		"payable": false,
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

/////////////////////////////////unregister staker//////////////////////////////////////////////////////////////////////
