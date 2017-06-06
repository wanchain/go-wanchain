var path = require('path');
var Web3 = require('web3');
var events = require('events');

var Tx = require('ethereumjs-tx');
var ethUtil = require('ethereumjs-util');
ethUtil.crypto = require('crypto');

var web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));

var fs = require('fs');
var content = fs.readFileSync(path.join(__dirname, "privacyTokenBase.js"), 'utf8');
//var content = fs.readFileSync("beida.js", 'utf8');
var solc = require('solc');
var compiled = solc.compile(content, 1);
var myTestContract = web3.eth.contract(JSON.parse(compiled.contracts.PrivacyTokenBase.interface));

console.log(compiled.contracts.PrivacyTokenBase.interface);

var config_privatekey = 'daa2fbee5ee569bc64842f5a386e7037612e0736b52e41749d52b616beaca65e';
var config_pubkey = '0xc29258c409380d34c9255406e8204212da552f92'


	var constructorInputs = [];

	constructorInputs.push({ data: compiled.contracts.PrivacyTokenBase.bytecode});
	var txData = myTestContract.new.getData.apply(null, constructorInputs);

	//TODO: replace user's private key
	var privateKey = new Buffer(config_privatekey, 'hex');
	var amount = web3.toWei(0, 'ether');
	var bn = new web3.BigNumber(amount);
	var hexValue = '0x' + bn.toString(16);
	//TODO: replace with user address
	var serial = '0x' + web3.eth.getTransactionCount(config_pubkey).toString(16);
	var rawTx = { 
	  Txtype: '0x1',
	  nonce: serial,
	  gasPrice: '0x43bb88745', 
	  gasLimit: '0x400000',
	  to: '',
	  value: hexValue,
	  from: config_pubkey,
	  data: '0x' + txData
	};
	var tx = new Tx(rawTx);
	tx.sign(privateKey);
	var serializedTx = tx.serialize();
	console.log("serializedTx:" + serializedTx.toString('hex'));
	web3.eth.sendRawTransaction('0x' + serializedTx.toString('hex'), function(err, hash){
	   if(!err){
	   	console.log('tx hash');
	   	console.log(hash);
       }else {
	       console.log(err);
	   }
	});	