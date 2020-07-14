# 如何使用POS真随机数

本文介绍对于DApp开发者来说，如何在自己的DApp中使用POS真随机数。

POS随机数的特点如下：

1. 生成频率: 为每个epoch生成一个;
2. 数值范围: 0 ~ 2^256 - 1;
3. 生成时间: 大约每日UTC 16点以后，时间不固定;
4. 特点: 25个RandomLeader使用密码原理多方计算共同生成，不可预测;
5. 工作代码: 以预编译合约的方式预置在go-wanchain节点内部;
6. 预编译合约地址: 0x262;

## 使用方法

使用方法分为2个方面：链上智能合约获取、链下获取。

### 1. 在智能合约中获取POS随机数

开发者可自行调用预编译合约获取，或使用如下辅助获取合约代码获取。

开发者在链上部署下面的合约代码，或者使现有的合约继承此代码，即可使用代码中的接口获取到POS随机数。

***注意：***

1）代码中的**randomPrecompileAddr**参数，请固定填写**address(0x262)**;

2）代码默认获取输入epochId上一个epoch生成的随机数，如果要获取当前epoch刚刚产生的随机数，请将传入的**epochId设为当前Epoch+1;**



```
pragma solidity 0.4.26;


contract PosHelper {

    function callWith32BytesReturnsUint256(
        address to,
        bytes32 functionSelector,
        bytes32 param1
    ) private view returns (uint256 result, bool success) {
        assembly {
            let freePtr := mload(0x40)

            mstore(freePtr, functionSelector)
            mstore(add(freePtr, 4), param1)

            // call ERC20 Token contract transfer function
            success := staticcall(gas, to, freePtr, 36, freePtr, 32)

            result := mload(freePtr)
        }
    }

    function getRandomByEpochId(uint256 epochId, address randomPrecompileAddr)
        public
        view
        returns (uint256)
    {
        bytes32 functionSelector = keccak256(
            "getRandomNumberByEpochId(uint256)"
        );

        (uint256 result, bool success) = callWith32BytesReturnsUint256(
            randomPrecompileAddr,
            functionSelector,
            bytes32(epochId)
        );

        require(success, "ASSEMBLY_CALL getRandomByEpochId failed");

        return result;
    }

    function getRandomByBlockTime(
        uint256 blockTime,
        address randomPrecompileAddr
    ) public view returns (uint256) {
        bytes32 functionSelector = keccak256(
            "getRandomNumberByTimestamp(uint256)"
        );

        (uint256 result, bool success) = callWith32BytesReturnsUint256(
            randomPrecompileAddr,
            functionSelector,
            bytes32(blockTime)
        );

        require(success, "ASSEMBLY_CALL getRandomByBlockTime failed");

        return result;
    }

    function getEpochId(uint256 blockTime, address randomPrecompileAddr)
        public
        view
        returns (uint256)
    {
        bytes32 functionSelector = keccak256("getEpochId(uint256)");

        (uint256 result, bool success) = callWith32BytesReturnsUint256(
            randomPrecompileAddr,
            functionSelector,
            bytes32(blockTime)
        );

        require(success, "ASSEMBLY_CALL getEpochId failed");

        return result;
    }
}

```

### 2. 链下获取POS随机数

如果需要在DApp页面中直接获取并使用随机数，可使用当前已经部署的Jack's Pot合约中的获取随机数接口。

使用这个合约地址：**0x76b074d91f546914c6765ef81cbdc6f9c7da5685**

使用下面的ABI，即可用web3接口或者其它手段在链外直接读取链上随机。

```
[
    {
      "constant": true,
      "inputs": [
        {
          "name": "epochId",
          "type": "uint256"
        },
        {
          "name": "randomPrecompileAddr",
          "type": "address"
        }
      ],
      "name": "getRandomByEpochId",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [
        {
          "name": "blockTime",
          "type": "uint256"
        },
        {
          "name": "randomPrecompileAddr",
          "type": "address"
        }
      ],
      "name": "getRandomByBlockTime",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [
        {
          "name": "blockTime",
          "type": "uint256"
        },
        {
          "name": "randomPrecompileAddr",
          "type": "address"
        }
      ],
      "name": "getEpochId",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    }
  ]
```

例如，使用web3js获取：

```
const Web3 = require('web3');

// You can get YOUR-API-KEY from iwan.wanchain.org
let iWanUrl = "wss://api.wanchain.org:8443/ws/v3/YOUR-API-KEY";

let web3 = new Web3(new Web3.providers.WebsocketProvider(iWanUrl));

// fill abi shown above
let abi = "YOUR-ABI";

let addr = "0x76b074d91f546914c6765ef81cbdc6f9c7da5685";

let sc = new web3.eth.Contract(abi, addr);

let epochId = await sc.methods.getEpochId().call();

let random = await sc.methods.getRandomByEpochId(epochId + 1).call();

```

如果获取到的随机数值为0，表示当前epoch的随机数还未生成。