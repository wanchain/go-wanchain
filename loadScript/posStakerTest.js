// numberTPS means how many txs per second send.
var numberTPS = 3
// stakerRegisterPeriod means how long to register a account into staker.
var stakerRegisterPeriod = 1
// randomAccountCount is normal tx account count.
var randomAccountCount = 10
// tranValue is normal tx send value
var tranValue = 0.01
// stakeValue is staker balance to use
var stakeValue = 1000

var lockTimeSecond = 600

var balanceSourceAddress = '0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e';

// Start staker register
setTimeout(stakeRegisterTest, 1000 * stakerRegisterPeriod, null);

// var cscDefinition = [
//   {
//     "constant": false,
//     "type": "function",
//     "stateMutability": "nonpayable",
//     "inputs": [
//       {
//         "name": "Pubs",
//         "type": "string"
//       },
//       {
//         "name": "LockEpochs",
//         "type": "uint256"
//       }
//     ],
//     "name": "stakeIn",
//     "outputs": [
//       {
//         "name": "Pubs",
//         "type": "string"
//       },
//       {
//         "name": "LockEpochs",
//         "type": "uint256"
//       }
//     ]
//   },
//   {
//     "constant": false,
//     "type": "function",
//     "inputs": [
//       {
//         "name": "Pub",
//         "type": "string"
//       },
//       {
//         "name": "Value",
//         "type": "uint256"
//       }
//     ],
//     "name": "stakeOut",
//     "outputs": [
//       {
//         "name": "Pub",
//         "type": "string"
//       },
//       {
//         "name": "Value",
//         "type": "uint256"
//       }
//     ]
//   }
// ]
var cscDefinition = [
    {
        "constant": false,
        "inputs": [
            {
                "name": "secPk",
                "type": "bytes"
            },
            {
                "name": "bn256Pk",
                "type": "bytes"
            },
            {
                "name": "lockEpochs",
                "type": "uint256"
            },
            {
                "name": "feeRate",
                "type": "uint256"
            }
        ],
        "name": "stakeIn",
        "outputs": [],
        "payable": true,
        "stateMutability": "payable",
        "type": "function"
    },
    {
        "constant": false,
        "inputs": [
            {
                "name": "delegateAddress",
                "type": "address"
            }
        ],
        "name": "delegateIn",
        "outputs": [],
        "payable": true,
        "stateMutability": "payable",
        "type": "function"
    }
]

var stakeIndex = 0

function stakeRegisterTest() {
  personal.unlockAccount(balanceSourceAddress, 'wanglu', 9999999)

  var address = personal.newAccount("wanglu")
  var tx = eth.sendTransaction({
    from: balanceSourceAddress,
    to: address,
    value: web3.toWin(stakeValue*2),
    gas: 200000,
    gasprice: '0x' + (200000000000).toString(16)
  });

  console.log("stake index:" + stakeIndex)
  console.log("stake balance tx=" + tx)
  console.log("wait tx in blockchain")
  wait(function () { return eth.getTransaction(tx).blockNumber != null; });

  console.log("address:" + address + ", balance:" + eth.getBalance(address))
  personal.unlockAccount(address, 'wanglu', 9999999)

  pubKeys = personal.showPublicKey(address, "wanglu");
  console.log("pubKeys:" + pubKeys);
  if (!pubKeys) {
      console.log("showPublicKey,error")
      return
  }
  var secpub = pubKeys[0]
  var g1pub = pubKeys[1]

  console.log("pk:" + secpub + ", G1:" + g1pub)


  /////////////////////////////////register staker////////////////////////////////////////////////////////////////////////
  var contractDef = eth.contract(cscDefinition);
  var cscContractAddr = "0x00000000000000000000000000000000000000DA";
  var coinContract = contractDef.at(cscContractAddr);

  var lockEpochs = 10;
  var feeRate = 99;

  var payload = coinContract.stakeIn.getData(secpub, g1pub, lockEpochs, feeRate)

  console.log("payload:")
  console.log(payload)

  var tx = eth.sendTransaction({
    from: address,
    to: cscContractAddr,
    value: web3.toWin(stakeValue),
    data: payload,
    gas: 200000,
    gasprice: '0x' + (20000000000).toString(16)
  });

  console.log("stake register tx=" + tx)
  console.log("wait stake register tx in blockchain")

  wait(function () { return eth.getTransaction(tx).blockNumber != null; });

  console.log("stake register success")

  // setTimeout(stakeUnregister, lockTimeSecond * 1000 + 60 * 1000, address, stakeIndex)

  // Start staker register
  setTimeout(stakeRegisterTest, 1000 * stakerRegisterPeriod, null);

  stakeIndex++;
}

function stakeUnregister(address, index) {

  pubs = personal.showPublicKey(address, "wanglu")
  var secpub = pubs[0]
  var g1pub = pubs[1]

  /////////////////////////////////register staker////////////////////////////////////////////////////////////////////////
  var datapks = secpub + '';//'' + g1pub

  var contractDef = eth.contract(cscDefinition);
  var cscContractAddr = "0x00000000000000000000000000000000000000DA";
  var coinContract = contractDef.at(cscContractAddr);

  //var lockTime = web3.toWin(lockTimeSecond)

  var payload = coinContract.stakeOut.getData(datapks, web3.toWin(stakeValue))

  console.log("stakeOut index:" + index)
  console.log("stakeOut payload:")
  console.log(payload)
  console.log("G1")
  console.log(g1pub)

  var tx = eth.sendTransaction({
    from: address,
    to: cscContractAddr,
    value: web3.toWin(0),
    data: payload,
    gas: 200000,
    gasprice: '0x' + (20000000000).toString(16)
  });

  console.log("stake unregister tx=" + tx)
  console.log("wait stake unregister tx in blockchain")

  wait(function () { return eth.getTransaction(tx).blockNumber != null; });

  console.log("stake unregister success")
}

var wait = function (conditionFunc) {
  var loopLimit = 30000;
  var loopTimes = 0;
  while (!conditionFunc()) {
    admin.sleep(2);
    loopTimes++;
    if (loopTimes >= loopLimit) {
      console.log(Error("wait timeout! conditionFunc:" + conditionFunc))
      break
    }
  }
}
