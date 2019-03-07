How to Deploy your Dapp onto Wanchain 

Requirement:

1: a working Wanchain client, go to the github site, https://github.com/wanchain/go-wanchain, to get the latest version
2: remix https://remix.ethereum.org, which is an amazing online smart contract development IDE
3: your awesome Dapp consists of one or multiple smart contracts

Steps:

1: go to remix, copy and paste your smart contract code, make static syntax analysis, and compile it
2: click Details on the right panel of remix, copy all the code of WEB3DEPLOY section from the pop-up
3: open your favorite editor and comment out whatever inside contracts/demo/deploy.js
4: paste those scripts from step-3 into a javascript file, say, /some/directory/deploy.js
5: switch to root directory of go-wanchain project, 
6: launch a wanchain client console, make sure it is connected with either various wanchain public networks, or your private blockchain network
7: run loadScript('/path/to/your/javascript/script/in/step/4/deploy.js') in the console, what you are basically doing is sending a transaction to the Wanchain infra to deploy the contract
8: the transaction id and contract address (hash values starting with '0x') will be printed out onto the console after few seconds
9: now, you can play with your Dapp 

You can locate a demo wanchain token contract and involved scripts under contracts/demo/ directory