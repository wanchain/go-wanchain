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
    var loopLimit = 300;
    var loopTimes = 0;
    while (!conditionFunc()) {
        admin.sleep(1);
        loopTimes++;
        if(loopTimes>=loopLimit){
            console.log(Error("wait timeout! conditionFunc:" + conditionFunc))
            break
        }
    }
}

personal.unlockAccount(eth.accounts[1],"wanglu",99999);
personal.unlockAccount(eth.accounts[2],"wanglu",99999);
personal.unlockAccount(eth.accounts[9],"wanglu",99999);

cnt = 0;
//eth.sendTransaction({from:eth.accounts[9], to: eth.accounts[1], value: web3.toWin(100000)});



for(;;) {


    console.log("begin to loop test coin index=" + cnt++)

    var tranValue = 10;

    abiDef = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}],"name":"buyCoinNote","outputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}],"name":"refundCoin","outputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"inputs":[],"name":"getCoins","outputs":[{"name":"Value","type":"uint256"}]}];

    contractDef = eth.contract(abiDef);
    coinContractAddr = "0x0000000000000000000000000000000000000064";
    coinContract = contractDef.at(coinContractAddr);

    var acc1OldBalance = parseFloat(wanBalance(eth.accounts[1]))
    var acc2OldBalance = parseFloat(wanBalance(eth.accounts[2]))

    wanUnlock(eth.accounts[1]);
    wanUnlock(eth.accounts[2]);

    var wanAddr = wan.getWanAddress(eth.accounts[2]);
    var otaAddr = wan.generateOneTimeAddress(wanAddr);

    txBuyData = coinContract.buyCoinNote.getData(otaAddr, web3.toWin(tranValue));
    buyCoinTx = eth.sendTransaction({from:eth.accounts[1], to:coinContractAddr, value:web3.toWin(tranValue), data:txBuyData, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});

    console.log("buy coin index")
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
    var refundTx = eth.sendTransaction({from:eth.accounts[2], to:coinContractAddr, value:0, data:txRefundData, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});
    console.log("refund index")

    wait(function(){return eth.getTransaction(refundTx).blockNumber != null;});

    console.log("New balance of ", eth.accounts[2], " is ", web3.fromWin(eth.getBalance(eth.accounts[2])));

    var acc1NewBalance = parseInt(wanBalance(eth.accounts[1]))
    var acc2NewBalance = parseInt(wanBalance(eth.accounts[2]))
    if (acc2NewBalance < acc2OldBalance || acc2NewBalance > (acc2OldBalance + tranValue)) {
        console.log(Error("acc2OldBalance:" + acc2OldBalance + ", acc2NewBalance:" + acc2NewBalance + ", tranValue:" + tranValue))
    }

    if (acc1NewBalance > acc1OldBalance - tranValue || acc1NewBalance < acc1OldBalance - tranValue - 1) {
        console.log( Error("acc1OldBalance:" + acc1OldBalance + ", acc1NewBalance:" + acc1NewBalance + ", tranValue:" + tranValue))
    }  else {
        console.log("test coin success!!")
    }


    ///////////////////////////////////////////////////////////////////////////////

// before use the file, please desploy yourself contract and replace the contractAddr value with the new address!!!
    console.log("begin to loop test token index=" + cnt++)

    var initPriBalance = '0x1000000000';
    var priTranValue = 888;


    abiDefStamp = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}],"name":"buyStamp","outputs":[{"name":"OtaAddr","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}],"name":"refundCoin","outputs":[{"name":"RingSignedData","type":"string"},{"name":"Value","type":"uint256"}]},{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[],"name":"getCoins","outputs":[{"name":"Value","type":"uint256"}]}];

    contractDef = eth.contract(abiDefStamp);
    stampContractAddr = "0x00000000000000000000000000000000000000c8";
    stampContract = contractDef.at(stampContractAddr);

    var wanAddr = wan.getWanAddress(eth.accounts[1]);
    var otaAddrStamp = wan.generateOneTimeAddress(wanAddr);
    txBuyData = stampContract.buyStamp.getData(otaAddrStamp, web3.toWin(0.005));


    sendTx = eth.sendTransaction({from:eth.accounts[1], to:stampContractAddr, value:web3.toWin(0.005), data:txBuyData, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});

    console.log("wait buy stamp")

    wait(function(){return eth.getTransaction(sendTx).blockNumber != null;});


    keyPairs = wan.computeOTAPPKeys(eth.accounts[1], otaAddrStamp).split('+');
    privateKeyStamp = keyPairs[0];

    var mixStampAddresses = wan.getOTAMixSet(otaAddrStamp,2);
    var mixSetWith0x = []
    for (i = 0; i < mixStampAddresses.length; i++){
        mixSetWith0x.push(mixStampAddresses[i])
    }



    var erc20simple_contract = web3.eth.contract([{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_toKey","type":"bytes"},{"name":"_value","type":"uint256"}],"name":"otatransfer","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"privacyBalance","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":false,"inputs":[{"name":"initialBase","type":"address"},{"name":"baseKeyBytes","type":"bytes"},{"name":"value","type":"uint256"}],"name":"initPrivacyAsset","outputs":[],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"otabalanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function","stateMutability":"view"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"otaKey","outputs":[{"name":"","type":"bytes"}],"payable":false,"type":"function","stateMutability":"view"}]);

    contractAddr = '0xa6000c50b8ccf77702c7fde117b02f79f9e1989e';

    erc20simple = erc20simple_contract.at(contractAddr)

    var wanAddr = wan.getWanAddress(eth.accounts[1]);
    var otaAddrTokenHolder = wan.generateOneTimeAddress(wanAddr);
    keyPairs = wan.computeOTAPPKeys(eth.accounts[1], otaAddrTokenHolder).split('+');
    privateKeyTokenHolder = keyPairs[0];
    addrTokenHolder = keyPairs[2];
    sendTx = erc20simple.initPrivacyAsset.sendTransaction(addrTokenHolder, otaAddrTokenHolder, initPriBalance,{from:eth.accounts[1], gas:200000, gasprice:'0x' + (20000000000).toString(16)});

    console.log("wait init token in contract")

    wait(function(){return eth.getTransaction(sendTx).blockNumber != null;});

    ota1Balance = erc20simple.privacyBalance(addrTokenHolder)
    if (ota1Balance != parseFloat(initPriBalance-0)) {
        console.log( Error('ota1 balance wrong! balance:' + ota1Balance + ', except:' + initPriBalance))
    }


    var hashMsg = addrTokenHolder
    var ringSignData = personal.genRingSignData(hashMsg, privateKeyStamp, mixSetWith0x.join("+"))

    var wanAddr = wan.getWanAddress(eth.accounts[2]);
    var otaAddr4Account2 = wan.generateOneTimeAddress(wanAddr);
    keyPairs = wan.computeOTAPPKeys(eth.accounts[2], otaAddr4Account2).split('+');
    privateKeyOtaAcc2 = keyPairs[0];
    addrOTAAcc2 = keyPairs[2];

    cxtInterfaceCallData = erc20simple.otatransfer.getData(addrOTAAcc2, otaAddr4Account2, priTranValue);

    glueContractDef = eth.contract([{"constant":false,"type":"function","inputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}],"name":"combine","outputs":[{"name":"RingSignedData","type":"string"},{"name":"CxtCallParams","type":"bytes"}]}]);
    glueContract = glueContractDef.at("0x0000000000000000000000000000000000000000")
    combinedData = glueContract.combine.getData(ringSignData, cxtInterfaceCallData)

    sendTx = personal.sendPrivacyCxtTransaction({from:addrTokenHolder, to:contractAddr, value:0, gas:200000, gasprice:'0x' + (20000000000).toString(16),data: combinedData}, privateKeyTokenHolder)

    console.log("wait privacy tx in blockchain")

    wait(function(){return eth.getTransaction(sendTx).blockNumber != null;});


    ota2Balance = erc20simple.privacyBalance(addrOTAAcc2)
    if (ota2Balance != priTranValue) {
        console.log( Error("ota2 balance wrong. balance:" + ota2Balance +  ", expect:" + priTranValue))
    }

    ota1Balance = erc20simple.privacyBalance(addrTokenHolder)
    if (ota1Balance != initPriBalance - priTranValue) {
        console.log(Error("ota2 balance wrong. balance:" + ota1Balance +  ", expect:" + (initPriBalance - priTranValue)))
    } else {
        console.log("test token success!!")
    }


}

