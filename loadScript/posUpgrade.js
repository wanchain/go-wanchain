

var posControlDefinition = [
	{
		"constant": false,
		"inputs": [
			{
				"name": "EpochId",
				"type": "uint256"
			},
			{
				"name": "wlIndex",
				"type": "uint256"
			},
			{
				"name": "wlCount",
				"type": "uint256"
			}
		],
		"name": "upgradeWhiteEpochLeader",
		"outputs": [],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	}
]

var contractDef = eth.contract(posControlDefinition);
var ContractAddr = "0x0000000000000000000000000000000000000264";
var Contract = contractDef.at(ContractAddr);

var EpochId = 2
var wlIndex = 11
var wlCount = 26

var payload = Contract.upgradeWhiteEpochLeader.getData(EpochId, wlIndex, wlCount)
console.log("payload: ", payload)
var tx = eth.sendTransaction({from:eth.coinbase, to:ContractAddr, value:'0x00', data:payload, gas: 200000, gasprice:'0x' + (200000000000).toString(16)});
console.log("tx: ",tx)
