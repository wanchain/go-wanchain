# 1. Galaxy Consensus getting started manual

# 2. Introduction
This is a manual for helping getting started as a Wanchain Galaxy Consensus node operator. You can follow along with this manual and help test the proof of concept version of Galaxy Consensus.

**Software Environment**
- We recommend using Linux or MacOS
- Docker services
- Install Golang from https://golang.org/ and set GO environment variables `$GOPATH` and `$GOROOT`

# 3. Contents

<!-- TOC -->

- [1. Galaxy Consensus getting started manual](#1-wanpos-getting-started-manual)
- [2. Introduction](#2-introduction)
- [3. Contents](#3-contents)
- [4. Quick start from Docker](#4-quick-start-from-docker)
    - [4.1. Step by step node setup](#41-step-by-step-to-be-a-miner)
    - [4.2. Step by step delegation guide](#42-step-by-step-to-delegate-wan-coins)
- [5. Download and run](#5-download-and-run)
    - [5.1. Run from Docker](#51-run-from-docker)
    - [5.2. Download](#52-download)
        - [5.2.1. Download BIN](#521-download-bin)
        - [5.2.2. Download code and compile](#522-download-code-and-compile)
    - [5.3. Run](#53-run)
        - [5.3.1. Non-staking node](#531-run-as-a-synchronize-node)
        - [5.3.2. Staking node](#532-run-as-a-miner-node)
- [6. Common Operations](#6-operations)
    - [6.1. PoS account creation](#61-pos-account-creation)
    - [6.2. Check balance](#62-check-balance)
    - [6.3. Get test WAN](#63-get-test-wan-coins-of-pos)
    - [6.4. Registration and delegation](#64-stake-register-and-delegate)
    - [6.5. Check rewards](#65-check-incentive)
    - [6.6. Unregister and unlock](#66-stake-unregister-and-unlock)
- [7. Results of internal testing](#7-test-result-of-incentive)

<!-- /TOC -->

# 4. Quick start from Docker

## 4.1. Step by step node setup

**Step 1:** Install docker (Ubuntu):
```
$ sudo wget -qO- https://get.docker.com/ | sh

$ sudo usermod -aG docker YourUserName

$ exit
```

**Step 2:** Start GWAN with Docker and create account:
```
$ docker pull wanchain/wanpos

$ docker run -d -v /home/YourUserName/.wanchain:/root/.wanchain wanchain/wanpos /bin/gwan --pluto

YourContainerID

$ docker exec -it YourContainerID /bin/bash

root> gwan attach .wanchain/pluto/gwan.ipc

> personal.newAccount('YourPassword')

"YourAccountAddress"

> personal.showPublicKey("YourAccountAddress", 'YourPassword')

["YourPK1", "YourPK2"]

> exit

root> echo "YourPassword" > /root/.wanchain/pw.txt

root> exit

```

![img](./img_get_start/1.png)

**Step 3:** Get test WAN for "YourAccountAddress":

Follow [6.3. Get test wan coins of PoS](#63-get-test-wan-coins-of-pos) to get test WAN.

And after receiving test WAN, continue to step 4.

![img](./img_get_start/4.png)

**Step 4:** Create a script file in path: `/home/YourUserName/.wanchain/minerRegister.js`

```
//minerRegister.js

// If you want to register as a miner you can modify and use this script.


//-------INPUT PARAMS SHOULD BE REPLACED WITH YOURS--------------------

// tranValue is the value you want to stake - minValue is 100000 
var tranValue = "100000"

// secpub is the miner node's secpub value
var secpub    = "YourPK1"

// g1pub is the miner node's g1pub value
var g1pub     = "YourPK2"

// feeRate is the percent of the reward kept by the node in delegation - 100 indicates the node does not accept delegation.
var feeRate   = 100

// lockTime is the length of stake locking time measured in epochs - minimum required locking time of 5 epochs
var lockTime  = 30

// baseAddr is the stake funding source account
var baseAddr  = "YourAccountAddress"

// passwd is the stake funding source account password
var passwd    = "YourPassword"

//-------INPUT PARAMS SHOULD BE REPLACED WITH YOURS--------------------


//------------------RUN CODE DO NOT MODIFY------------------
personal.unlockAccount(baseAddr, passwd)
var cscDefinition = [{"constant":false,"inputs":[{"name":"addr","type":"address"},{"name":"lockEpochs","type":"uint256"}],"name":"stakeUpdate","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"stakeAppend","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"secPk","type":"bytes"},{"name":"bn256Pk","type":"bytes"},{"name":"lockEpochs","type":"uint256"},{"name":"feeRate","type":"uint256"}],"name":"stakeIn","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"delegateAddress","type":"address"}],"name":"delegateIn","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"delegateAddress","type":"address"}],"name":"delegateOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}];


var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000d2";
var coinContract = contractDef.at(cscContractAddr);

var payload = coinContract.stakeIn.getData(secpub, g1pub, lockTime, feeRate)
var tx = eth.sendTransaction({from:baseAddr, to:cscContractAddr, value:web3.toWin(tranValue), data:payload, gas: 200000, gasprice:'0x' + (200000000000).toString(16)});
console.log("tx=" + tx)
//------------------RUN CODE DO NOT MODIFY------------------

```
![img](./img_get_start/2.png)


![img](./img_get_start/3.png)

**Step 5:** Run the registration script in GWAN

If you have not closed the Docker script from **Step 2**, continue with the commands below, otherwise restart the Docker script.

```
$ docker exec -it YourContainerID /bin/gwan attach .wanchain/pluto/gwan.ipc

> loadScript("/root/.wanchain/minerRegister.js")

> exit

$ docker stop YourContainerID

$ docker run -d -p 17717:17717 -p 17717:17717/udp -v /home/YourUserName/.wanchain:/root/.wanchain wanchain/wanpos /bin/gwan --pluto --etherbase "YourAccountAddress" --unlock "YourAccountAddress" --password /root/.wanchain/pw.txt --mine --minerthreads=1 

```

Setup is now complete, mining will begin as soon as syncing is finished.

![img](./img_get_start/5.png)


![img](./img_get_start/6.png)

## 4.2. Step by step delegation guide

**Step 1:** Install Docker (Ubuntu):
```
$ sudo wget -qO- https://get.docker.com/ | sh

$ sudo usermod -aG docker YourUserName

$ exit
```

**Step 2:** Start GWAN with Docker, create account, and view delegate node list:
```
$ docker run -d -v /home/YourUserName/.wanchain:/root/.wanchain wanchain/wanpos /bin/gwan --pluto

YourContainerID

$ docker exec -it YourContainerID /bin/bash

root> gwan attach .wanchain/pluto/gwan.ipc

> personal.newAccount('YourPassword')

"YourAccountAddress"

> pos.getStakerInfo(eth.blockNumber)
[
	{...},
	{...},
	{	Address: "DelegateAddress",
    Amount: 2e+23,
    Clients: [],
    FeeRate: 10,
    From: "...",
    LockEpochs: 30,
    PubBn256: "...",
    PubSec256: "...",
    StakingEpoch: 117
	}
]
```

`YourAccountAddress` and `DelegateAddress` are found from the step above along with the `FeeRate`.

**Step 3:** Get test WAN for "YourAccountAddress"

Follow [6.3. Get test wan coins of PoS](#63-get-test-wan-coins-of-pos) to get test WAN.

**Step 4:** Create a script file in path: `/home/YourUserName/.wanchain/sendDelegate.js`

```
//sendDelegate.js

// If you want to send to a delegate you can modify and use this script.


//-------INPUT PARAMS YOU SHOULD MODIFY TO YOURS--------------------

// tranValue is the value you want to stake in minValue is 100
var tranValue = "100000"

// delegateAddr is the validator address copied from the list of validators generated in Step 4
var delegateAddr = "DelegateAddress"

// baseAddr is the fund source account.
var baseAddr  = "YourAccountAddress"

// passwd is the fund source account password.
var passwd    = "YourPassword"

//-------INPUT PARAMS SHOULD BE REPLACED WITH YOURS--------------------


//------------------RUN CODE DO NOT MODIFY------------------
personal.unlockAccount(baseAddr, passwd)
var cscDefinition = [{"constant":false,"inputs":[{"name":"addr","type":"address"},{"name":"lockEpochs","type":"uint256"}],"name":"stakeUpdate","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"stakeAppend","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"secPk","type":"bytes"},{"name":"bn256Pk","type":"bytes"},{"name":"lockEpochs","type":"uint256"},{"name":"feeRate","type":"uint256"}],"name":"stakeIn","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"delegateAddress","type":"address"}],"name":"delegateIn","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"delegateAddress","type":"address"}],"name":"delegateOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}];


var contractDef = eth.contract(cscDefinition);
var cscContractAddr = "0x00000000000000000000000000000000000000d2";
var coinContract = contractDef.at(cscContractAddr);

var payloadDelegate = coinContract.delegateIn.getData(delegateAddr)
var tx2 = eth.sendTransaction({from:baseAddr, to:cscContractAddr, value:web3.toWin(tranValue), data:payloadDelegate, gas: 200000, gasprice:'0x' + (200000000000).toString(16)});
console.log("tx2=" + tx2)
//------------------RUN CODE DO NOT MODIFY------------------
```

**Step 5:** Run the registration script in GWAN

Load the script in GWAN to complete delegation.

```
> loadScript("/root/.wanchain/sendDelegate.js")

```

# 5. Download and run

## 5.1. Run from Docker

You can run a node from a Docker image.

```
// Install the Docker service

$ sudo wget -qO- https://get.docker.com/ | sh

$ sudo usermod -aG docker YourUserName
```

For a non-staking node:

```
//On MacOS:
$ docker run -d -v /Users/YourUserName/Library/Wanchain/:/root/.wanchain wanchain/wanpos /bin/gwan --pluto

//On Ubuntu
$ docker run -d -v /home/YourUserName/.wanchain:/root/.wanchain wanchain/wanpos /bin/gwan --pluto
```

For a staking-node, you should create a account and start like this:
```
$ docker run -d -v /home/YourUserName/.wanchain:/root/.wanchain wanchain/wanpos /bin/gwan --pluto --etherbase "YourAccountAddress" --unlock "YourAccountAddress" --password YourPasswordTxtFile --mine --minerthreads=1 
```

The `YourPasswordTxtFile` is a txt file with your miner account password in it in Docker.

Such as the file put in the path `/home/YourUserName/.wanchain/pw.txt` 

You should start Docker with this command:

```
$ docker run -d -v /home/YourUserName/.wanchain:/root/.wanchain wanchain/wanpos /bin/gwan --pluto --etherbase "YourAccountAddress" --unlock "YourAccountAddress" --password /root/.wanchain/pw.txt --mine --minerthreads=1 
```

## 5.2. Download

You can download a binary file or code to run a node.

### 5.2.1. Download BIN

You can download the compiled binary file from the download links below:

(Not ready now, please use docker)

| OS            | URL            | MD5             | SHA256
| --------------  | :------------  | :-------------: | :--: |
|Ubuntu|gwan.tar.gz| XXXXXXXXXXXXXXXX |XXXXXXXXXXXXXXXXXXXXXXXXX
|Windows|gwan.tar.gz| XXXXXXXXXXXXXXXX |XXXXXXXXXXXXXXXXXXXXXXXXX
|MacOS|gwan.tar.gz| XXXXXXXXXXXXXXXX |XXXXXXXXXXXXXXXXXXXXXXXXX


### 5.2.2. Download Code and Compile

If you want to compile the Galaxy Consensus code, you should first to install the Golang development environment and config $GOPATH and $GOROOT:

https://golang.org/

You can download the code file and compile to run with the following steps:

If you already have a golang compile and run environment, and you have configured $GOPATH , you can get the code as below:

```
$ go get github.com/wanchain/go-wanchain

$ cd $GOPATH/src/github.com/wanchain/go-wanchain

$ git checkout posalpha

$ git pull

$ make
```

Or you can clone from github.com as below:

```
$ mkdir -p $GOPATH/src/github.com/wanchain/

$ cd $GOPATH/src/github.com/wanchain/

$ git clone https://github.com/wanchain/go-wanchain.git

$ cd go-wanchain

$ git checkout posalpha

$ git pull

$ make
```

Then you can find the binary file in path `build/bin/gwan`

## 5.3. Run

You can run a node in two different modes, staking and non staking.

### 5.3.1. Non-staking node

```
$ gwan --pluto --rpc --syncmode "full"
```

### 5.3.2. Staking-node

In the following command, you should replace the `0x8d8e7c0813a51d3bd1d08246af2a8a7a57d8922e` with your own account address and replace the `/tmp/pw.txt` file with your own password file with your password string in it.

```
$ gwan --pluto --rpc --etherbase "0x8d8e7c0813a51d3bd1d08246af2a8a7a57d8922e" --unlock "0x8d8e7c0813a51d3bd1d08246af2a8a7a57d8922e" --password /tmp/pw.txt --rpc  --mine --minerthreads=1 --syncmode "full"
```

# 6. Common Operations

## 6.1. PoS account creation

Before you run a PoS node you should create an account.

```
$ gwan --pluto console --exec "personal.newAccount('Your Password')"

// Or run after ipc attach
$ personal.newAccount('Your Password')
```

You can see your address created and printed in the screen, then you can press `Ctrl+C` to exit.

You will get a keystore file with three crypto key words in your path `~/.wanchain/pluto/keystore/` in Ubuntu or `~/Library/Wanchain/pluto/keystore/` in Mac OS.

And you can use a command to get the `Address Public Key` and `G1 Public Key` of your account.

```
$ gwan --pluto console --exec "personal.showPublicKey('Your Address', 'Your Password')"

// Or run after ipc attach
$ personal.showPublicKey('Your Address', 'Your Password')
```

These public keys will be used in staking registration.

## 6.2. Check balance

You can check your balance in the address when you attach a GWAN console in the `ipc` file or use a console mode at GWAN start.

```
// In ubuntu
$ gwan attach ~/.wanchain/pluto/gwan.ipc

// In MacOS
$ gwan attach ~/Library/Wanchain/pluto/gwan.ipc

```

After the node synchronization is finished you can check your balance using the following command.

```
$ eth.getBalance("Your Address Fill Here")

// Such as address example shown above.
$ eth.getBalance("0x8c35B69AC00EC3dA29a84C40842dfdD594Bf5d27")
```

## 6.3. Get test WAN

If you want to get some test WAN to experiment with Galaxy Consensus, you can send an email with your WAN PoS test account address to the email shown below with your request, and we will transfer the test WAN to you within 3 working days.


| Index            | Email         | 
| --------------  | :------------  | 
|1| techsupport@wanchain.org| 



## 6.4. Registration and delegation

If you have an account with WAN coins and you want to create a Galaxy Consensus miner, you should do it as in the diagram below:

![img](./img_get_start/99.png)

You can register as a staking node through Stake register.

We have given a smart contract for registration and unregistration.

The contract interface is shown below.
```
var cscDefinition = [{"constant":false,"inputs":[{"name":"addr","type":"address"},{"name":"lockEpochs","type":"uint256"}],"name":"stakeUpdate","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"stakeAppend","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"secPk","type":"bytes"},{"name":"bn256Pk","type":"bytes"},{"name":"lockEpochs","type":"uint256"},{"name":"feeRate","type":"uint256"}],"name":"stakeIn","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"delegateAddress","type":"address"}],"name":"delegateIn","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":false,"inputs":[{"name":"delegateAddress","type":"address"}],"name":"delegateOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]
```

In the smart contract input parameters, the `feeRate` indicates the percentage of reward kept by the validator from the delegators' reward. 100 indicates that the validator does not accept delegations.

If you want to be an delegator and accept delegations from others, you need to set a reasonable percentage for your `feeRate` to attract others to invest.

The `feeRate`'s value ranges from 0 to 100 and indicates the amount of reward kept by the validator (10 means the validator will take a 10% fee, and the delegator will keep 90% of the reward).

You can register your stake with a custom script or just modify the module's script in `loadScript/minerRegister.js`.

The JavaScript file `loadScript/register.js` is used by validators for registration, and `loadScript/sendDelegate.js` is used by test WAN holders for sending their delegation.

In the script file, the password should be replaced with your own in `personal.unlockAccount`.

`secpub`, `secAddr`, `g1pub` should be filled with your account's address public key, account address, and G1 public key. These public keys can be found using the function `personal.showPublicKey` shown above.

`lockTime` should be filled with the stake locking time. The unit of time is epoch. Epoch time is equal to SlotTime * SlotCount. 

The `tranValue` should be filled with the amount of WAN you want to lock in the smart contract for stake registration. You can't get it back until the locking time is up.

This script can be run in an attached IPC session.

```
// This path is a relative path for your run.
$ loadScript('loadScript/register.js')
```

If you don't want to be a validator, you can delegate WAN to a validator who will stake for you and share the block rewards.

The reward percent is related to the stake amount and the `feeRate`.

The delegation method is also in `register.js`, it is in the last 3 lines.

You can input the delegator's address to make a delegation.

The lock time for delegations does not work in the proof of concept version, it will follow the delegator's lock time.

## 6.5. Check rewards

You can check your balance as shown above to verify whether you have received a reward, and you can use the commands shown below to see which address was awarded and the reward amount for the specified epoch ID.

```
// In an attached IPC session to run for epoch 123.
$ pos.getEpochIncentivePayDetail(123)
```

## 6.6. Unregister and Unlock

Your locked WAN will be automatically sent back when the time is up. 

# 7. Results of internal testing

We depolyed some PoS validator nodes to participate in staking.

We used different stake values and different locktimes to test.

The locktime is measured by epoch counts.

The epoch time is 20 minutes for one epoch. So 6 epochs means 120 minutes.

The total stake is about 6000000 ~ 8000000 WAN on the testnet.

The reward sent to the addresses is shown below:

| Address     | stake | locktime | ep 1| ep 2 | ep 3 | ep 4 | ep 5 | total incentive |
| ----------  | ---- | :---: | --- | --- | ----| ---- | ---- | ---- | 
|0xbec1f01f5cbe494279a3c1455644a16aebfd700d| 100000 | 6 |0 |0.32 |1.07 |1.02 |1.94 | 4.35|
|0xa38c0aafc0b4ee45e006814e5769f17fda60f994| 200000 | 6 |0.32 |1.39 |4.40 |3.06 |2.33 |11.5 |
|0x711a9967d0b61ab92a86e14102de1233d3de5ead| 500000 | 6 |2.49 |6.03 |9.62 |10.32 |5.14 |33.6 | 
|0x52eee1ccb29adc742449a3e87fe7acaad605bd4c| 200000 | 12 |1.93 |4.81 |1.08 |1.17 |0.32 |9.31 |


If the epoch incentive is 0, it means that address has not been selected.


![img](./img_get_start/7.png)
