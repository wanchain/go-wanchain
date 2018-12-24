
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

personal.unlockAccount(eth.accounts[0],"wanglu",99999);
personal.unlockAccount(eth.accounts[1],"wanglu",99999);

var tranValue = 10

var secpub  = '0x04bc039c519a0054e6c61ae5be2a8755a45264cc333748a4c29c1f9480839ce1d245710bc400e4f1684337613aec9afd16efbde16abbd24ff527c0c019feec8ee4'
var g1pub   = '0x17efae606177ac93ea5a0ee1735ee6858f305d0206421b98052a0923b515eb1811621f0cd113468888aef823c3a701fdfabe57e053c36b41681ded14b13d5b20'

var cscDefinition = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}],"name":"stakeIn","outputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"Pub","type":"string"}],"name":"stakeOut","outputs":[{"name":"Pub","type":"string"}]}]

/////////////////////////////////register staker////////////////////////////////////////////////////////////////////////
var datapks = secpub + g1pub


var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000d2";
var coinContract = contractDef.at(cscContractAddr);

var lockTime = web3.toWin(3600)

var payload = coinContract.stakeIn.getData(datapks,lockTime)

var tx = eth.sendTransaction({from:eth.accounts[1], to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});

console.log("tx=" + tx)

/////////////////////////////////unregister staker//////////////////////////////////////////////////////////////////////
