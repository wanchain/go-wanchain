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

var lockTimeSecond = 3600

var balanceSourceAddress = '0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e';

var toAddresses = []
console.log("wait for normal tx accout creating...")
for (var i = 0; i < randomAccountCount; i++) {
  var address = personal.newAccount("wanglu")
  toAddresses.push(address)
  console.log(i)
}
console.log("normal tx account create finish.")

// Start normal tx send.
setInterval(sendTx, 1000, null);

//--------------------------------------------------------------------------------------
var totalSendTx = 0;
function sendTx() {
  personal.unlockAccount(balanceSourceAddress, 'wanglu', 9999999)
  for (var i = 0; i < numberTPS; i++) {
    tranValue += 0.00000001 * totalSendTx;
    var address = toAddresses[Math.floor(Math.random() * randomAccountCount)]
    var tx = eth.sendTransaction({
      from: balanceSourceAddress,
      to: address,
      value: web3.toWin(tranValue),
      gas: 200000,
      gasprice: '0x' + (20000000000).toString(16)
    });
    console.log("tx=" + tx + " send finish " + totalSendTx++)
  }
  console.timeEnd("sendTx")
}
