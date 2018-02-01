var wanBalance = function(addr){
	return web3.fromWin(web3.eth.getBalance(addr));
}

var wanUnlock = function(addr){
    return personal.unlockAccount(addr,"wanglu",99999);	
}

var sendWanFromUnlock = function (From, To , V){
	return eth.sendTransaction({from:From, to: To, value: web3.toWin(V)});
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

var tranValue = 10;

abiDef = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}],"name":"buyCoinNote","outputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}],"name":"refundCoin","outputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"inputs":[],"name":"getCoins","outputs":[{"name":"Value","type":"uint256"}]}];

contractDef = eth.contract(abiDef);
coinContractAddr = "0x0000000000000000000000000000000000000064";
coinContract = contractDef.at(coinContractAddr);

wanUnlock(eth.accounts[1]);
wanUnlock(eth.accounts[2]);

for (i = 0; i < 3; i++) {
    var wanAddr = wan.getWanAddress(eth.accounts[2]);
    var otaAddr = wan.generateOneTimeAddress(wanAddr);

    txBuyData = coinContract.buyCoinNote.getData(otaAddr, web3.toWin(tranValue));
    buyCoinTx = eth.sendTransaction({from:eth.accounts[1], to:coinContractAddr, value:web3.toWin(tranValue), data:txBuyData, gas: 1000000, gasprice:'0x' + (20000000000).toString(16)});
    wait(function(){return eth.getTransaction(buyCoinTx).blockNumber != null;});
}

var acc1OldBalance = parseFloat(wanBalance(eth.accounts[1]))
var acc2OldBalance = parseFloat(wanBalance(eth.accounts[2]))


var wanAddr = wan.getWanAddress(eth.accounts[2]);
var otaAddr = wan.generateOneTimeAddress(wanAddr);

txBuyData = coinContract.buyCoinNote.getData(otaAddr, web3.toWin(tranValue));
buyCoinTx = eth.sendTransaction({from:eth.accounts[1], to:coinContractAddr, value:web3.toWin(tranValue), data:txBuyData, gas: 1000000, gasprice:'0x' + (20000000000).toString(16)});
wait(function(){return eth.getTransaction(buyCoinTx).blockNumber != null;});


var mixWanAddresses = wan.getOTAMixSet(otaAddr,2);
var mixSetWith0x = []
for (i = 0; i < mixWanAddresses.length; i++){
	mixSetWith0x.push(mixWanAddresses[i])
}

keyPairs = wan.computeOTAPPKeys(eth.accounts[2], otaAddr).split('+');
privateKey = keyPairs[0];

console.log("Balance of ", eth.accounts[2], " is ", web3.fromWin(eth.getBalance(eth.accounts[2])));
var ringSignData = personal.genRingSignData(eth.accounts[2], privateKey, mixSetWith0x.join("+"))
var txRefundData = coinContract.refundCoin.getData(ringSignData, web3.toWin(tranValue))
var refundTx = eth.sendTransaction({from:eth.accounts[2], to:coinContractAddr, value:0, data:txRefundData, gas: 2000000, gasprice:'0x' + (20000000000).toString(16)});
wait(function(){return eth.getTransaction(refundTx).blockNumber != null;});

console.log("New balance of ", eth.accounts[2], " is ", web3.fromWin(eth.getBalance(eth.accounts[2])));

var acc1NewBalance = parseFloat(wanBalance(eth.accounts[1]))
var acc2NewBalance = parseFloat(wanBalance(eth.accounts[2]))
if (acc2NewBalance < acc2OldBalance || acc2NewBalance > (acc2OldBalance + tranValue)) {
	throw Error("acc2OldBalance:" + acc2OldBalance + ", acc2NewBalance:" + acc2NewBalance + ", tranValue:" + tranValue)
}

if (acc1NewBalance > acc1OldBalance - tranValue || acc1NewBalance < acc1OldBalance - tranValue - 1) {
	throw Error("acc1OldBalance:" + acc1OldBalance + ", acc1NewBalance:" + acc1NewBalance + ", tranValue:" + tranValue)
}


