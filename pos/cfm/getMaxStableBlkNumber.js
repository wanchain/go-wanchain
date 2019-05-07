const web3ext = require('web3ext');
const Web3 = require('web3');

function getMaxStalbeBlockNumber(){
	let maxBlockStableNumber = 0
	try{
		
		let web3 = new Web3(new Web3.providers.HttpProvider("http://127.0.0.1:8545"));
		web3ext.extend(web3)
		maxBlockStableNumber = web3.pos.getMaxStableBlkNumber()
		
	}catch(e){
		console.log("intial web3 error! %s",e)
	}
	return maxBlockStableNumber
}

console.log(getMaxStalbeBlockNumber())
