// before use the file, please desploy yourself contract and replace the contractAddr value with the new address!!!

var initPriBalance = '0x1000000000';
var priTranValue = 888;

var wanBalance = function(addr){
	return web3.fromWei(web3.eth.getBalance(addr));
}

var wanUnlock = function(addr){
    return personal.unlockAccount(addr,"wanglu",99999);
}

var sendWanFromUnlock = function (From, To , V){
	eth.sendTransaction({from:From, to: To, value: web3.toWei(V)});
}

var wait = function (conditionFunc) {
	var loopLimit = 120;
	var loopTimes = 0;
	while (!conditionFunc()) {
		admin.sleep(1);
		loopTimes++;
		if(loopTimes>=loopLimit){
			throw Error("wait timeout! conditionFunc:" + conditionFunc)
		}
	}
}

wanUnlock(eth.accounts[1])
wanUnlock(eth.accounts[2])

abiDefStamp = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}],"name":"buyStamp","outputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}],"name":"refundCoin","outputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[],"name":"getCoins","outputs":[{"name":"Value","type":"uint256"}]}];

contractDef = eth.contract(abiDefStamp);
stampContractAddr = "0x00000000000000000000000000000000000000c8";
stampContract = contractDef.at(stampContractAddr);

var wanAddr = eth.getWanAddress(eth.accounts[1]);
var otaAddrStamp = eth.generateOneTimeAddress(wanAddr);
txBuyData = stampContract.buyStamp.getData(otaAddrStamp, web3.toWei(0.01));


sendTx = eth.sendTransaction({from:eth.accounts[1], to:stampContractAddr, value:web3.toWei(0.01), data:txBuyData, gas: 1000000});
wait(function(){return eth.getTransaction(sendTx).blockNumber != null;});


keyPairs = eth.computeOTAPPKeys(eth.accounts[1], otaAddrStamp).split('+');
privateKeyStamp = keyPairs[0];

var mixStampAddresses = eth.getOTAMixSet(otaAddrStamp,2);
var mixSetWith0x = []
for (i = 0; i < mixStampAddresses.length; i++){
    mixSetWith0x.push('0x' + mixStampAddresses[i])
}



var erc20simple_contract = web3.eth.contract([{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_toKey","type":"bytes"},{"name":"_value","type":"uint256"}],"name":"otatransfer","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"privacyBalance","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":false,"inputs":[{"name":"initialBase","type":"address"},{"name":"baseKeyBytes","type":"bytes"},{"name":"value","type":"uint256"}],"name":"initPrivacyAsset","outputs":[],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"otabalanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"otaKey","outputs":[{"name":"","type":"bytes"}],"payable":false,"type":"function","stateMutability":"view"}]);
contractAddr = '0x18f940983efda661f29b8b18609daf28d0cd5bff';
erc20simple = erc20simple_contract.at(contractAddr)

		

var wanAddr = eth.getWanAddress(eth.accounts[1]);
var otaAddrTokenHolder = eth.generateOneTimeAddress(wanAddr);
keyPairs = eth.computeOTAPPKeys(eth.accounts[1], otaAddrTokenHolder).split('+');
privateKeyTokenHolder = keyPairs[0];
addrTokenHolder = keyPairs[2];
sendTx = erc20simple.initPrivacyAsset.sendTransaction(addrTokenHolder, otaAddrTokenHolder, initPriBalance,{from:eth.accounts[1], gas:1000000});
wait(function(){return eth.getTransaction(sendTx).blockNumber != null;});

ota1Balance = erc20simple.privacyBalance(addrTokenHolder)
if (ota1Balance != parseFloat(initPriBalance-0)) {
	throw Error('ota1 balance wrong! balance:' + ota1Balance + ', except:' + initPriBalance)
}


var hashMsg = addrTokenHolder
var ringSignData = eth.genRingSignData(hashMsg, privateKeyStamp, mixSetWith0x.join("+"))

var wanAddr = eth.getWanAddress(eth.accounts[2]);
var otaAddr4Account2 = eth.generateOneTimeAddress(wanAddr);
keyPairs = eth.computeOTAPPKeys(eth.accounts[2], otaAddr4Account2).split('+');
privateKeyOtaAcc2 = keyPairs[0];
addrOTAAcc2 = keyPairs[2];

cxtInterfaceCallData = erc20simple.otatransfer.getData(addrOTAAcc2, otaAddr4Account2, priTranValue);

glueContractDef = eth.contract([{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}],"name":"combine","outputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}]}]);
glueContract = glueContractDef.at("0x0000000000000000000000000000000000000000")
combinedData = glueContract.combine.getData(ringSignData, cxtInterfaceCallData)

sendTx = eth.sendPrivacyCxtTransaction({from:addrTokenHolder, to:contractAddr, value:0, data: combinedData}, privateKeyTokenHolder)
wait(function(){return eth.getTransaction(sendTx).blockNumber != null;});


ota2Balance = erc20simple.privacyBalance(addrOTAAcc2)
if (ota2Balance != priTranValue) {
	throw Error("ota2 balance wrong. balance:" + ota2Balance +  ", expect:" + priTranValue)
}

ota1Balance = erc20simple.privacyBalance(addrTokenHolder)
if (ota1Balance != initPriBalance - priTranValue) {
	throw Error("ota2 balance wrong. balance:" + ota1Balance +  ", expect:" + (initPriBalance - priTranValue))
}


