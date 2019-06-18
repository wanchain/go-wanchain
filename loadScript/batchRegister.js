
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

var wait = function (conditionFunc) {
    var loopLimit = 30000;
    var loopTimes = 0;
    while (!conditionFunc()) {
        admin.sleep(2);
        loopTimes++;
        if(loopTimes>=loopLimit){
            console.log(Error("wait timeout! conditionFunc:" + conditionFunc))
            break
        }
    }
}


personal.unlockAccount(eth.accounts[0],"wanglu",99999);

var tranValue = 1000

var regCount = 10

for (idx=0;idx<regCount;idx++) {

    var address = personal.newAccount("wanglu")
    var tx = eth.sendTransaction({
        from: eth.accounts[0],
        to:address,
        value: web3.toWin(tranValue),
        gas: 200000,
        gasprice: '0x' + (200000000000).toString(16)
    });

    console.log("tx=" + tx)
    console.log("wait tx in blockchain")
    wait(function(){return eth.getTransaction(tx).blockNumber != null;});

    pubs = personal.showPublicKey(address,"wanglu")
    var secpub = pubs[0]
    var g1pub = pubs[1]

    var cscDefinition = [{
        "constant": false,
        "type": "function",
        "stateMutability": "nonpayable",
        "inputs": [{"name": "Pubs", "type": "string"}, {"name": "LockEpochs", "type": "uint256"}],
        "name": "stakeIn",
        "outputs": [{"name": "Pubs", "type": "string"}, {"name": "LockEpochs", "type": "uint256"}]
    }, {
        "constant": false,
        "type": "function",
        "inputs": [{"name": "Pub", "type": "string"}],
        "name": "stakeOut",
        "outputs": [{"name": "Pub", "type": "string"}]
    }]

/////////////////////////////////register staker////////////////////////////////////////////////////////////////////////
    var datapks = secpub + g1pub


    var contractDef = eth.contract(cscDefinition);
    var cscContractAddr = "0x00000000000000000000000000000000000000DA";
    var coinContract = contractDef.at(cscContractAddr);

    var lockTime = web3.toWin(3600*24*100)

    var payload = coinContract.stakeIn.getData(datapks, lockTime)

    var tx = eth.sendTransaction({
        from: eth.accounts[0],
        to: cscContractAddr,
        value: web3.toWin(tranValue),
        data: payload,
        gas: 200000,
        gasprice: '0x' + (20000000000).toString(16)
    });

    console.log("tx=" + tx)
    console.log("wait tx in blockchain")

    wait(function(){return eth.getTransaction(tx).blockNumber != null;});
}

console.log("regiter successfully")
