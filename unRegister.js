
personal.unlockAccount(eth.accounts[0],"wanglu",99999);
personal.unlockAccount(eth.accounts[1],"wanglu",99999);

var tranValue = 10

var secpub  = '0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70'
var g1pub   = '0x17efae606177ac93ea5a0ee1735ee6858f305d0206421b98052a0923b515eb1811621f0cd113468888aef823c3a701fdfabe57e053c36b41681ded14b13d5b20'

var cscDefinition = [{"constant":false,"type":"function","stateMutability":"nonpayable","inputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}],"name":"stakeIn","outputs":[{"name":"Pubs","type":"string"},{"name":"LockTime","type":"uint256"}]},{"constant":false,"type":"function","inputs":[{"name":"Pub","type":"string"},{"name":"Value","type":"uint256"}],"name":"stakeOut","outputs":[{"name":"Pub","type":"string"},{"name":"Value","type":"uint256"}]}]
/////////////////////////////////register staker////////////////////////////////////////////////////////////////////////
var datapks = '' + secpub


var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000d2";
var coinContract = contractDef.at(cscContractAddr);

var lockTime = web3.toWin(3600)

var payload = coinContract.stakeOut.getData(datapks,web3.toWin(tranValue))

var tx = eth.sendTransaction({from:eth.accounts[1], to:cscContractAddr, value:web3.toWin(0), data:payload, gas: 200000, gasprice:'0x' + (20000000000).toString(16)});

console.log("tx=" + tx)

/////////////////////////////////unregister staker//////////////////////////////////////////////////////////////////////
