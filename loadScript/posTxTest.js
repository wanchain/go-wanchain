var numberTPS = 3
var randomAccountCount = 10
var tranValue = 0.01



var toAddresses = []

for (var i = 0; i < randomAccountCount; i++) {
  var address = personal.newAccount("wanglu")
  toAddresses.push(address)
}

personal.unlockAccount(eth.accounts[0], "wanglu", 99999);

setInterval(sendTx, 1000, null);

var totalSendTx = 0;

function sendTx() {
  console.time("sendTx");
  for (var i = 0; i < numberTPS; i++) {
    tranValue += 0.00000001*totalSendTx;
    var address = toAddresses[Math.floor(Math.random() * randomAccountCount)]
    var tx = eth.sendTransaction({
      from: eth.accounts[0],
      to: address,
      value: web3.toWin(tranValue),
      gas: 200000,
      gasprice: '0x' + (20000000000).toString(16)
    });

    console.log("tx=" + tx)
    //console.log("wait tx in blockchain")
    //wait(function () { return eth.getTransaction(tx).blockNumber != null; });
    console.log("tx send finish " + totalSendTx++)
  }
  console.timeEnd("sendTx")
}

// var wait = function (conditionFunc) {
//   var loopLimit = 30000;
//   var loopTimes = 0;
//   while (!conditionFunc()) {
//     admin.sleep(2);
//     loopTimes++;
//     if (loopTimes >= loopLimit) {
//       console.log(Error("wait timeout! conditionFunc:" + conditionFunc))
//       break
//     }
//   }
// }
