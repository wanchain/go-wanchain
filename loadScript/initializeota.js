var wanUnlock = function(addr){
    return personal.unlockAccount(addr,"password1",99999);
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

abiDef = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}],"name":"buyCoinNote","outputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}],"name":"refundCoin","outputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"inputs":[],"name":"getCoins","outputs":[{"name":"Value","type":"uint256"}]}];

contractDef = eth.contract(abiDef);
coinContractAddr = "0x0000000000000000000000000000000000000064";
coinContract = contractDef.at(coinContractAddr);

wanUnlock(eth.accounts[0]);
var tranValues = [ 10, 20, 50, 100, 200, 500, 1000, 5000, 50000 ];

for (a=0; a<tranValues.length; a++) {
    tranValue = tranValues[a]
    console.log(tranValue)
    for (i = 0; i < 9; i++) {
        var wanAddr = wan.getWanAddress(eth.accounts[0]);
        var otaAddr = wan.generateOneTimeAddress(wanAddr);

        txBuyData = coinContract.buyCoinNote.getData(otaAddr, web3.toWin(tranValue));
        buyCoinTx = eth.sendTransaction({from:eth.accounts[1], to:coinContractAddr, value:web3.toWin(tranValue), data:txBuyData, gas: 1000000, gasprice:'0x' + (20000000000).toString(16)});
        // wait(function(){return eth.getTransaction(buyCoinTx).blockNumber != null;});
        console.log("generate OTA " + i)
    }
}
