var wanBalance = function(addr){
	return web3.fromWei(web3.eth.getBalance(addr));
}

var wanUnlock = function(addr){
    return personal.unlockAccount(addr,"wanglu",99999);	
}

var sendWanFromUnlock = function (From, To , V){
	return eth.sendTransaction({from:From, to: To, value: web3.toWei(V)});
}

var wait = function (conditionFunc) {
	var loopLimit = 100;
	var loopTimes = 0;
	while (!conditionFunc()) {
		admin.sleep(1);
		loopTimes++;
		if(loopTimes>=loopLimit){
			throw Error("wait timeout! conditionFunc:" + conditionFunc)
		}
	}
}

var tranValue = 1;

wanUnlock(eth.coinbase);
var sendTx = sendWanFromUnlock(eth.coinbase, eth.accounts[1], 100);
wait(function(){return eth.getTransaction(sendTx).blockNumber != null;});

abiDef = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}],"name":"buyCoinNote","outputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}],"name":"refundCoin","outputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"inputs":[],"name":"getCoins","outputs":[{"name":"Value","type":"uint256"}]}];

contractDef = eth.contract(abiDef);
coinContractAddr = "0x0000000000000000000000000000000000000006";
coinContract = contractDef.at(coinContractAddr);

var acc1OldBalance = parseFloat(wanBalance(eth.accounts[1]))
var acc2OldBalance = parseFloat(wanBalance(eth.accounts[2]))

personal.unlockAccount(eth.accounts[1],"wanglu",9999);
personal.unlockAccount(eth.accounts[2],"wanglu",9999);

var wanAddr = eth.getWanAddress(eth.accounts[2]);
var otaAddr = eth.generateOneTimeAddress(wanAddr);

txBuyData = coinContract.buyCoinNote.getData(otaAddr, web3.toWei(1));
buyCoinTx = eth.sendTransaction({from:eth.accounts[1], to:"0x0000000000000000000000000000000000000006", value:web3.toWei(tranValue), data:txBuyData, gas: 1000000});
wait(function(){return eth.getTransaction(buyCoinTx).blockNumber != null;});


var mixWanAddresses = eth.getOTAMixSet(otaAddr,2);
var mixSetWith0x = []
for (i = 0; i < mixWanAddresses.length; i++){
	mixSetWith0x.push('0x' + mixWanAddresses[i])
}

keyPairs = eth.computeOTAPPKeys(eth.accounts[2], otaAddr).split('+');
privateKey = keyPairs[0];

console.log("Balance of ", eth.accounts[2], " is ", web3.fromWei(eth.getBalance(eth.accounts[2])));
var ringSignData = eth.genRingSignData(eth.accounts[2], privateKey, mixSetWith0x.join("+"))
var txRefundData = coinContract.refundCoin.getData(ringSignData, web3.toWei(1))
var refundTx = eth.sendTransaction({from:eth.accounts[2], to:"0x0000000000000000000000000000000000000006", value:0, data:txRefundData, gas: 2000000});
wait(function(){return eth.getTransaction(refundTx).blockNumber != null;});

console.log("New balance of ", eth.accounts[2], " is ", web3.fromWei(eth.getBalance(eth.accounts[2])));

var acc1NewBalance = parseFloat(wanBalance(eth.accounts[1]))
var acc2NewBalance = parseFloat(wanBalance(eth.accounts[2]))
if (acc2NewBalance < acc2OldBalance || acc2NewBalance > (acc2OldBalance + tranValue)) {
	throw Error("acc2OldBalance:" + acc2OldBalance + ", acc2NewBalance:" + acc2NewBalance + ", tranValue:" + tranValue)
}

if (acc1NewBalance > acc1OldBalance - tranValue || acc1NewBalance < acc1OldBalance - tranValue - 1) {
	throw Error("acc1OldBalance:" + acc1OldBalance + ", acc1NewBalance:" + acc1NewBalance + ", tranValue:" + tranValue)
}


