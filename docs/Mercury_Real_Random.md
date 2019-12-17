# Introduce of Mercury Real Random nubmer

## Smart Contract ABI

We have added three functions in random number protocol ABIs:

- getEpochId 
  - You can get epochId from a input timestamp.
- getRandomNumberByEpochId
  - You can get the random of input epochId. The random protocol will generate one real random number in each epoch at the end of epoch.
- getRandomNumberByTimestamp
  - You can get the random of input Timestamp.

The unit of timestamp is second of UTC.

ABI define:
```
[{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg1","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg2","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"sigShare","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"timestamp","type":"uint256"}],"name":"getEpochId","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"timestamp","type":"uint256"}],"name":"getRandomNumberByTimestamp","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"epochId","type":"uint256"}],"name":"getRandomNumberByEpochId","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]
```

## The random number Smart Contract Address

The address of precompiled smart contract of random nubmer is:

```
0x0000000000000000000000000000000000000262
```

## Try it in mywanwallet.io

In Contract page, input abi and address.

you can get the epochId and random number in the web site.
