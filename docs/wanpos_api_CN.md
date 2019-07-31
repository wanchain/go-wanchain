# 1. 万维链星系共识节点PoS-API手册

PoS-API是用户通过IPC或RPC链接到gwan节点后，可以调用的POS相关的API接口。

本手册对全部POS-API进行了列举和解释说明。

# 2. 目录
<!-- TOC -->

- [1. 万维链星系共识节点PoS-API手册](#1-万维链星系共识节点pos-api手册)
- [2. 目录](#2-目录)
- [3. PoS-API](#3-pos-api)
    - [3.1. 基础信息查询](#31-基础信息查询)
        - [3.1.1. version](#311-version)
        - [3.1.2. getPosInfo](#312-getposinfo)
        - [3.1.3. getEpochID](#313-getepochid)
        - [3.1.4. getEpochBlkCnt](#314-getepochblkcnt)
        - [3.1.5. getEpochIDByTime](#315-getepochidbytime)
        - [3.1.6. getEpochIdByBlockNumber](#316-getepochidbyblocknumber)
        - [3.1.7. getSlotID](#317-getslotid)
        - [3.1.8. getSlotCount](#318-getslotcount)
        - [3.1.9. getSlotIDByTime](#319-getslotidbytime)
        - [3.1.10. getSlotTime](#3110-getslottime)
        - [3.1.11. getChainQuality](#3111-getchainquality)
        - [3.1.12. getLocalPK](#3112-getlocalpk)
        - [3.1.13. getMaxStableBlkNumber](#3113-getmaxstableblknumber)
        - [3.1.14. getReorgState](#3114-getreorgstate)
        - [3.1.15. getTimeByEpochID](#3115-gettimebyepochid)
        - [3.1.16. getWhiteListConfig](#3116-getwhitelistconfig)
        - [3.1.17. getWhiteListbyEpochID](#3117-getwhitelistbyepochid)
    - [3.2. 奖励信息查询](#32-奖励信息查询)
        - [3.2.1. getEpochIncentivePayDetail](#321-getepochincentivepaydetail)
        - [3.2.2. getEpochIncentiveBlockNumber](#322-getepochincentiveblocknumber)
        - [3.2.3. getEpochIncentive](#323-getepochincentive)
        - [3.2.4. getEpochGasPool](#324-getepochgaspool)
        - [3.2.5. getEpochRemain](#325-getepochremain)
        - [3.2.6. getIncentivePool](#326-getincentivepool)
    - [3.3. 选举信息查询](#33-选举信息查询)
        - [3.3.1. getEpochStakerInfo](#331-getepochstakerinfo)
        - [3.3.2. getEpochStakerInfoAll](#332-getepochstakerinfoall)
        - [3.3.3. getEpochLeadersAddrByEpochID](#333-getepochleadersaddrbyepochid)
        - [3.3.4. getEpochLeadersByEpochID](#334-getepochleadersbyepochid)
        - [3.3.5. calProbability](#335-calprobability)
        - [3.3.6. getEpochStakeOut](#336-getepochstakeout)
        - [3.3.7. getLeaderGroupByEpochID](#337-getleadergroupbyepochid)
        - [3.3.8. getRandomProposersAddrByEpochID](#338-getrandomproposersaddrbyepochid)
        - [3.3.9. getRandomProposersByEpochID](#339-getrandomproposersbyepochid)
        - [3.3.10. getSlStage](#3310-getslstage)
        - [3.3.11. getRbSignatureCount](#3311-getrbsignaturecount)
        - [3.3.12. getRbStage](#3312-getrbstage)
        - [3.3.13. getSlotLeaderByEpochIDAndSlotID](#3313-getslotleaderbyepochidandslotid)
        - [3.3.14. getStakerInfo](#3314-getstakerinfo)
        - [3.3.15. getValidRBCnt](#3315-getvalidrbcnt)
        - [3.3.16. getValidSMACnt](#3316-getvalidsmacnt)
    - [3.4. 随机数查询](#34-随机数查询)
        - [3.4.1. getRandom](#341-getrandom)
    - [3.5. 活跃度查询](#35-活跃度查询)
        - [3.5.1. getActivity](#351-getactivity)
        - [3.5.2. getSlotActivity](#352-getslotactivity)
        - [3.5.3. getValidatorActivity](#353-getvalidatoractivity)
    - [3.6. 废弃的API](#36-废弃的api)
        - [3.6.1. getBootNodePK](#361-getbootnodepk)
        - [3.6.2. getIncentiveRunTimes](#362-getincentiveruntimes)
        - [3.6.3. getRBAddress](#363-getrbaddress)
        - [3.6.4. getSlotCreateStatusByEpochID](#364-getslotcreatestatusbyepochid)
        - [3.6.5. getSlotScCallTimesByEpochID](#365-getslotsccalltimesbyepochid)
        - [3.6.6. getSmaByEpochID](#366-getsmabyepochid)
        - [3.6.7. getTotalIncentive](#367-gettotalincentive)
        - [3.6.8. getTotalRemain](#368-gettotalremain)

<!-- /TOC -->

# 3. PoS-API

## 3.1. 基础信息查询

### 3.1.1. version
获取POS-API的版本信息
```
> pos.version()
"1.0"
>
```
### 3.1.2. getPosInfo
获取从POW协议升级到POS协议的升级位置信息
```
> pos.getPosInfo()
{
  firstBlockNumber: 3560000,
  firstEpochId: 18078
}
```
其中`firstBlockNumber`是第一个POS的块号

`firstEpochId`是第一个POS协议下的Epoch ID，在POW阶段，此值为0

### 3.1.3. getEpochID
获取当前Epoch ID
```
> pos.getEpochID()
18108
```
### 3.1.4. getEpochBlkCnt
获取指定epoch的出块数量，输入参数为Epoch ID
```
> pos.getEpochBlkCnt(18107)
13753
```
### 3.1.5. getEpochIDByTime
根据时间推算Epoch ID，输入UTC时间秒数，返回Epoch ID
```
> Date.now()
1564546408833
> Date.now()/1000
1564546412.857
> pos.getEpochIDByTime(1564546412)
18108
```
### 3.1.6. getEpochIdByBlockNumber
根据块号获取Epoch ID
```
> eth.blockNumber
4017608
> pos.getEpochIdByBlockNumber(4017608)
18108
```
### 3.1.7. getSlotID
获取当前Slot ID
```
> pos.getSlotID()
3072
```
### 3.1.8. getSlotCount
获取一个epoch内的slot数量
```
> pos.getSlotCount()
17280
```
### 3.1.9. getSlotIDByTime
根据时间推算Slot ID，输入UTC时间秒数，返回Slot ID
```
> Date.now()
1564546408833
> Date.now()/1000
1564546412.857
> pos.getSlotIDByTime(1564546412)
3042
```
### 3.1.10. getSlotTime
获取一个slot的时间跨度，单位是秒
```
> pos.getSlotTime()
5
```
### 3.1.11. getChainQuality
获取链质量信息。输入epoch ID和slot ID，返回链质量值为千分数，例如770意为77.0%
```
> pos.getChainQuality(18108,3072)
770
```
### 3.1.12. getLocalPK
获取本地节点的挖矿账号的公钥
```
> pos.getLocalPK()
"04088b71907178ad7392736e7b817f1945364d0798665279f9d829299726828285366a0107a75c53d1e0f90b5251f0e33ab3abf4ef907fe28d0493bfeaa81ba676"
>
```
### 3.1.13. getMaxStableBlkNumber
获取当前最大稳定块号（不会回滚）
```
> pos.getMaxStableBlkNumber()
4018017
```
### 3.1.14. getReorgState
获取当前epoch回滚状态，输入epoch id，返回回滚次数和最大回滚长度
```
> pos.getReorgState(18108)
[0, 0]
>
```
### 3.1.15. getTimeByEpochID
获取指定epoch的开始时间，返回UTC时间秒
```
> pos.getTimeByEpochID(18108)
1564531200
> t=new Date(1564531200000)
<Date Wed, 31 Jul 2019 08:00:00 CST>
>
```
### 3.1.16. getWhiteListConfig
获取受控节点的配置信息，分别是生效epochID，受控节点数量，和起始序号
```
> pos.getWhiteListConfig()
[{
    EpochId: 0,
    WlCount: 26,
    WlIndex: 0
}]
```
### 3.1.17. getWhiteListbyEpochID
获取指定epoch的受控节点公钥列表
```
> pos.getWhiteListbyEpochID(18108)
["0x0451cffffa2fb947261efca509564768d909a4fefd450c0e00effc8d7cb848dbd08939e163a6a41bde571f4ae0056b876c2b01c18e1e2d6b7a4745b49f5f5912c0", 
......
"0x04fdb485b566c2ddb40e2f4341b1e5746479a7c45e3d8101b1360b8bdba6206deee520ceecc9e9897e3b05b53e3ffa6fa659bef47c384984c0bc021a843df10847"]
```
## 3.2. 奖励信息查询

### 3.2.1. getEpochIncentivePayDetail
获取指定epoch的奖励信息，输入epochID，返回为所有在此epoch工作的验证节点及委托人的奖励支付详情（包含RNP奖励、EL奖励和出块奖励）
```
>pos.getEpochIncentivePayDetail(18106)
[{
    address: "0xfb3b101776390f993f118cb959f38135c562c52a",
    delegators: [{
        address: "0x19ac9bb112cb2f903fe866b35c5eb59c4278fcbd",
        incentive: "0x71e72f24a7e92afe",
        type: "delegator"
    }],
    incentive: "0x271dbee21dc6d3e17",
    stakeInFromAddr: "0x56664f3b65cc5daf4098ed10b66c4a86e58e21a4",
    type: "validator"
},
......
]
```
### 3.2.2. getEpochIncentiveBlockNumber
获取指定epoch的奖励发放块号, 输入参数为epochID
```
> pos.getEpochIncentiveBlockNumber(18106)
4003788
```
### 3.2.3. getEpochIncentive
获取指定epoch所发放的总奖励金额，输入epochID返回奖励金额，单位为Wei
```
> pos.getEpochIncentive(18106)
"3710904768743286494978"
> web3.fromWin(3710904768743286494978)
"3710.9047687432865"
```
### 3.2.4. getEpochGasPool
获取指定epoch的总的交易费数额，单位为Wei
```
> pos.getEpochGasPool(18106)
"22306530114000000000"
```
### 3.2.5. getEpochRemain
获取指定epoch未发放的剩余奖励。此奖励将累计到下一年发放
```
> pos.getEpochRemain(18106)
"3160716829863864189953"
```
### 3.2.6. getIncentivePool
获取指定epoch的奖励池大小, 返回值分别为总奖励，基金会奖励，交易费奖励
```
> pos.getIncentivePool(18106)
["6871621598607150684931", "6849315068493150684931", "22306530114000000000"]
```


## 3.3. 选举信息查询

### 3.3.1. getEpochStakerInfo
获取指定epoch的指定验证人的选举权重信息，输入参数为epochID和验证人地址

其中TotalProbability为此验证人被选中的总权重值

Infors字段包含总权重中的各个分项，第一个分项为验证人自身，其余为它的委托人

```
> pos.getEpochStakerInfo(18106,'0x17d47c6ac4f72d43420f5e9533b526b2dee626a6')
{
  Addr: "0x17d47c6ac4f72d43420f5e9533b526b2dee626a6",
  FeeRate: 1000,
  Infors: [{
      Addr: "0x17d47c6ac4f72d43420f5e9533b526b2dee626a6",
      Probability: "0x297116712be7b468800000"
  }, {
      Addr: "0x4e6b5f1abdd517739889334df047113bd736c546",
      Probability: "0x849d149d594bdae800000"
  }],
  TotalProbability: "0x31bae7bb017c7217000000"
}
```
### 3.3.2. getEpochStakerInfoAll
获取指定epoch的所有验证节点信息，输入参数为epochID

字段含义与getEpochStakerInfo相同
```
>pos.getEpochStakerInfoAll(18106)
[{
    Addr: "0xa36576c856fe69faf1be738252febc3268075619",
    FeeRate: 10000,
    Infors: [{
        Addr: "0xa36576c856fe69faf1be738252febc3268075619",
        Probability: "0x84a079b60afeadbe80000"
    }],
    TotalProbability: "0x84a079b60afeadbe80000"
}, {
    Addr: "0x158bae682e6278a16d09d7c7311074585d38b54d",
    FeeRate: 1,
    Infors: [{
        Addr: "0x158bae682e6278a16d09d7c7311074585d38b54d",
        Probability: "0x29db4e3b8931016a000000"
    }],
    TotalProbability: "0x29db4e3b8931016a000000"
}]
```
### 3.3.3. getEpochLeadersAddrByEpochID
获取指定epoch的epoch leader地址列表，输入参数为EpochID
### 3.3.4. getEpochLeadersByEpochID
获取指定epoch的epoch leader公钥列表, 输入参数为EpochID
### 3.3.5. calProbability
计算选举权重，输入参数为金额和锁定时间，单位分别是wan和天
### 3.3.6. getEpochStakeOut
获取指定epoch的本金退还记录
```
> pos.getEpochStakeOut(18106)
[{
    address: "0x74b7505ef4ee4a4783f446df8964b6cdd4c61843",
    amount: "0x8f1d5c1cae3740000"
}]
```
### 3.3.7. getLeaderGroupByEpochID
获取指定epoch的EL和RNP地址与公钥列表

type为0表示Epoch Leader, 1表示Random Number Proposer

pubBn256值为0x的为受控节点

```
pos.getLeaderGroupByEpochID(18106)
[{
    pubBn256: "0x26f35218edefaf8e1547e9a463d14dd884cdc3dfc7e56b26167a50cb038367a10b9e3eaca8fa11624845f3d29f57219661df5b1ae388879815a01f920e838704",
    pubSec256: "0x0459ce8b55d547f24c2c88a4a642a755ca714f5749e5f0e0d8ea5237f8efdb36063c449a8883bc3c36bd263fc1256f3484b33b1598340ab176faecd4499e9bcbba",
    secAddr: "0x882c9c16c05496d7b5374840936aec1af2a16553",
    type: 0
}]
```
### 3.3.8. getRandomProposersAddrByEpochID
获取指定epoch的Random Number Proposer的地址列表
### 3.3.9. getRandomProposersByEpochID
获取指定epoch的Random Number Proposer的公钥2列表
### 3.3.10. getSlStage
获取指定slot的EL工作阶段, 输入参数为slotID
```
if slotId <= posconfig.Sma1End {
    return 1
} else if slotId < posconfig.Sma2Start {
    return 2
} else if slotId <= posconfig.Sma2End {
    return 3
} else if slotId < posconfig.Sma3Start {
    return 4
} else if slotId <= posconfig.Sma3End {
    return 5
} else {
    return 6
}
```
### 3.3.11. getRbSignatureCount
获取指定epoch中RNP签名的数量, 输入epochID和blockNumber，blockNumber为-1则使用最新块
```
> pos.getRbSignatureCount(18106, -1)
15
```
### 3.3.12. getRbStage
获取指定slot的RNP工作阶段, 输入参数为slotID, 共6个阶段
```
RbDkg1Stage
RbDkg1ConfirmStage
RbDkg2Stage
RbDkg2ConfirmStage
RbSignStage
RbSignConfirmStage
```
### 3.3.13. getSlotLeaderByEpochIDAndSlotID
获取指定epoch指定slot的Slot Leader公钥
```
> pos.getSlotLeaderByEpochIDAndSlotID(18106,1)
"0484caedf55f668c4cbd966f4aa3c0c1a064e33b1c3ab6c0bc4de7323583893e0fc2cbfe6f4fa3a53f2b57a24f0420b337bcc2f7fa9f9fc4e5e9d95b5c3874360d"
>
```
### 3.3.14. getStakerInfo
获取指定blockNumber的验证人详细信息

amount：为本金数量

clients: 为委托人列表

votingPower: 为选举权重

quitEpoch: 为委托人退款epoch

feeRate: 为佣金费率，万分数

feeRateChangedEpoch: 最近一次修改feeRate的epochID

from: 注册资金的来源账号

lockEpochs: 当前的锁定周期，单位为epoch

maxFeeRate: 最大佣金费率

nextLockEpochs: 下一个周期的锁定周期，当此值为0时，将在下一个周期退出，退还本金

stakingEpoch: 本次锁定周期开始工作的epochID

```
> pos.getStakerInfo(eth.blockNumber)
[{
    address: "0xf92ba56ac2506cb97c1d9ce55a54c595e0599ebd",
    amount: 5e+22,
    clients: [{
        address: "0x28f86db797a302b46fa04749faafb1b1c901ff19",
        amount: 100000000000000000000,
        quitEpoch: 0,
        votingPower: 1.002e+23
    }, {
        address: "0xc91e50c0ce32bb024e7e359ae2e829c7f2451e0b",
        amount: 1.248e+21,
        quitEpoch: 0,
        votingPower: 1.250496e+24
    }, {
        address: "0x7ed6135f81453059776ecbf3a838853103f3bf9d",
        amount: 2e+21,
        quitEpoch: 0,
        votingPower: 2.004e+24
    }, {
        address: "0xb3850a2c15c208075197645fc9a4010f8f7634a0",
        amount: 1.11e+22,
        quitEpoch: 0,
        votingPower: 1.11222e+25
    }, {
        address: "0x13944221112c8109be7dcd2adb6d47545dc45be3",
        amount: 1.249e+21,
        quitEpoch: 0,
        votingPower: 1.251498e+24
    }, {
        address: "0x4e6b5f1abdd517739889334df047113bd736c546",
        amount: 2.1e+23,
        quitEpoch: 18109,
        votingPower: 2.1042e+26
    }],
    feeRate: 2000,
    feeRateChangedEpoch: 18088,
    from: "0xf92ba56ac2506cb97c1d9ce55a54c595e0599ebd",
    lockEpochs: 30,
    maxFeeRate: 2000,
    nextLockEpochs: 30,
    partners: [],
    pubBn256: "0x1c466cedd50c33fac011ad4a8f14a177ef5d243e0d7add5c231935f545b30eb80015184d74bc7295b512ffdf9c69824c9db536ae07f1ab14f7eb2eed9a4f1b19",
    pubSec256: "0x041cd717ce3d97ff93d5dcd5f80d78897956dcded35dbaf7c7180bdaff3beb84b900c48b1fbd0c52feaef9aa5e3aae87707cc02eb0a0203b3b6f7911c2fb2bccdf",
    stakingEpoch: 18088,
    votingPower: 5.7e+25
}
```
### 3.3.15. getValidRBCnt
获取指定epoch中执行协议的RNP数量

输入epochID，返回值分别为DKG1, DKG2, 和SIGN等3个阶段的RNP有效参与数量
```
> pos.getValidRBCnt(18106)
[14, 14, 15]
```
### 3.3.16. getValidSMACnt
获取指定epoch的EL参与POS协议执行的数量，返回值分别为SMA1和SMA2的有效参与人数量
```
> pos.getValidSMACnt(18106)
[40, 40]
```
## 3.4. 随机数查询

### 3.4.1. getRandom
查询指定epoch指定blockNumber的随机数生成结果, 输入epochID和blockNumber，blockNumber如果为-1则使用最新块

如果随机数不存在，则返回Error: no random number exists，表示该epoch的随机数还未生成，或者因为低于门限值，生成失败。失败时会使用默认随机数。
```
> pos.getRandom(18106,-1)
Error: no random number exists
    at web3.js:3145:20
    at web3.js:6381:15
    at web3.js:5083:36
    at <anonymous>:1:1

> pos.getRandom(18107,-1)
"0x7241916d8f2a68937783cc577a373e22d74686a6a36b72937c2cbe3c6b58529c"
>
```


## 3.5. 活跃度查询

### 3.5.1. getActivity
获取指定epoch的活跃度信息，历史epoch为固定值，当前epoch会实时更新最新的当前值

其中epLeader为该epoch的epoch leader列表，epActivity为对应列表中的epoch leader是否完成全部EL的协议工作

rpLeader为该epoch的random number proposer列表，rpActivity为对应列表中的random number proposer是否完成全部RNP的协议工作

sltLeader为选中的出块人列表，其中不包含受控节点。slBlocks为对应sltLeader列表中每一个人的实际出块数量。

slActivity为此epoch的出块活跃度，其值为总出块数量除以epoch的总slot数量。

slCtrlCount为此epoch中受控节点出块的数量。

```
> pos.getActivity(18106)
{
  epActivity: [0, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 1],
  epLeader: ["0x882c9c16c05496d7b5374840936aec1af2a16553", "0x3628bf135f36c6e26a824ec9152885505f3fbc2a", "0x4add297a1c2eea65e1ab5fd67e79647ecea8f36c", "0x4bf9fd7308d0849a62c3a7dd71c5190e57c28756", "0xb58230a7923a6a1941016aa1682e212def899ed1", "0x1779a2002402319821e05977ad989e1cc0d3fbc3", "0x93c8ea0326ef334bdc3011e74cd1a6d78ce0594d", "0x2bfd98be771eeeb4d69dd8767d200ba58252d925", "0x28c12c7b51860b9d5aec3a0ceb63c6e187c00aac", "0x882c9c16c05496d7b5374840936aec1af2a16553", "0x1b7740df685f9d34773d5a2aba6ab3a2c1407f40", "0xee1ad9c4f9d81f900221e95ee04246b6254b0c6f", "0x6273ce1f6f32e129f295f138d6e4ba6f0e19333e", "0x0b80f69fcb2564479058e4d28592e095828d24aa", "0x9ce4664e9d7346869797b7d9fc8c7a0212d5ff44", "0x2f78203c3161f1139edf2ba4b17b4e430ad2cbfa", "0x17d47c6ac4f72d43420f5e9533b526b2dee626a6", "0x742d898d2ee28a338f03af79c47762a908281a6a", "0xb901829c7e8b7d1de44d8bce086e7a5b0bcc7957", "0x39140deffdbd7c3b2415c29a40e0571365819f57", "0x60528316c553df7cae86d1294ca0d381ebb65cf0", "0x052e421be8e93d6f6c4d3d99defed914920fb3c4", "0x2c72d7a8c02752fcfafbaea5a63c53056cfaf547", "0x3dabf8331afbc553a1e458e37a6c9c819c452d55"],
  rpActivity: [0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1],
  rpLeader: ["0x20e5203a97b2e08c3dcc22c1c32e0dde3cc41da8", "0xbdada4f58d17ce602cb0d2db2a55c3e4f47e397f", "0xa923ac48439add7124763b3682f4505044c81ae3", "0x94ecbf26582455f5a7c88ab65a5a4ac05f6fe231", "0xdcefae3fdb94815f5d15111b46a5761a39b6ec9d", "0xf92ba56ac2506cb97c1d9ce55a54c595e0599ebd", "0x1b7740df685f9d34773d5a2aba6ab3a2c1407f40", "0xa4ebf5bbb131179b69bbf33319257728cdada5cf", "0x1a95e85e8ffcfd28eb61ee53a542dc98c57b337a", "0x266ddcfdbe3ded75e0e511e6356bca052b221c6b", "0xa4626e2bb450204c4b34bcc7525e585e8f678c0d", "0x533c13658591caa8a188211e73097adea7b94010", "0x0b80f69fcb2564479058e4d28592e095828d24aa", "0x1b7740df685f9d34773d5a2aba6ab3a2c1407f40", "0xa4626e2bb450204c4b34bcc7525e585e8f678c0d", "0xeeb157fdf2a72959f2f8be75ff500cf7a2104fbb", "0x4729672067e1ad8ca7f5770e3273747fe52affad", "0xcf34eb7f491fa7d18ba132938d7208e39da4b509", "0xcd54e0c35b122860d8fe2eb41f2e8e3e79c085ba", "0x28c12c7b51860b9d5aec3a0ceb63c6e187c00aac", "0xb64b60ba915bc16dc71ea59c9950c1538dcead9c", "0x36fad9acaf51a13527375b1ffc3d5a749153efdb", "0xdfd7aa554653ca236c197ad746edc2954ca172df", "0x4add297a1c2eea65e1ab5fd67e79647ecea8f36c", "0xc7afae3c9e99af27fe3eaa10f6ec73cd2dbe003b"],
  slActivity: 0.794675925925926,
  slBlocks: [331, 1338, 666, 338, 338, 323, 341, 364, 349, 368],
  slCtrlCount: 8976,
  sltLeader: ["0xfb3b101776390f993f118cb959f38135c562c52a", "0xee1ad9c4f9d81f900221e95ee04246b6254b0c6f", "0x026e37c00451428027ebbbc2c81dce7e280ae97d", "0x1a95e85e8ffcfd28eb61ee53a542dc98c57b337a", "0xc7afae3c9e99af27fe3eaa10f6ec73cd2dbe003b", "0x533c13658591caa8a188211e73097adea7b94010", "0x4bf9fd7308d0849a62c3a7dd71c5190e57c28756", "0xda8fa1aee77709d37f59fb96afd4cf10ccaeb6ce", "0xb019a99f0653973ddb2d983a26e0970587d08447", "0x2f78203c3161f1139edf2ba4b17b4e430ad2cbfa"]
}
>
```
### 3.5.2. getSlotActivity
获取指定epoch的出块活跃信息

sltLeader为选中的出块人列表，其中不包含受控节点。slBlocks为对应sltLeader列表中每一个人的实际出块数量。

slActivity为此epoch的出块活跃度，其值为总出块数量除以epoch的总slot数量。

slCtrlCount为此epoch中受控节点出块的数量。
```
> pos.getSlotActivity(18106)
{
  slActivity: 0.794675925925926,
  slBlocks: [338, 364, 349, 341, 331, 368, 323, 1338, 666, 338],
  slCtrlCount: 8976,
  sltLeader: ["0x1a95e85e8ffcfd28eb61ee53a542dc98c57b337a", "0xda8fa1aee77709d37f59fb96afd4cf10ccaeb6ce", "0xb019a99f0653973ddb2d983a26e0970587d08447", "0x4bf9fd7308d0849a62c3a7dd71c5190e57c28756", "0xfb3b101776390f993f118cb959f38135c562c52a", "0x2f78203c3161f1139edf2ba4b17b4e430ad2cbfa", "0x533c13658591caa8a188211e73097adea7b94010", "0xee1ad9c4f9d81f900221e95ee04246b6254b0c6f", "0x026e37c00451428027ebbbc2c81dce7e280ae97d", "0xc7afae3c9e99af27fe3eaa10f6ec73cd2dbe003b"]
}
```
### 3.5.3. getValidatorActivity
获取指定Epoch的EL和RNP的活跃信息, 当前epoch或未来epoch返回null

其中epLeader为该epoch的epoch leader列表，epActivity为对应列表中的epoch leader是否完成全部EL的协议工作

rpLeader为该epoch的random number proposer列表，rpActivity为对应列表中的random number proposer是否完成全部RNP的协议工作
```
> pos.getValidatorActivity(18106)
{
  epActivity: [0, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 1],
  epLeader: ["0x882c9c16c05496d7b5374840936aec1af2a16553", "0x3628bf135f36c6e26a824ec9152885505f3fbc2a", "0x4add297a1c2eea65e1ab5fd67e79647ecea8f36c", "0x4bf9fd7308d0849a62c3a7dd71c5190e57c28756", "0xb58230a7923a6a1941016aa1682e212def899ed1", "0x1779a2002402319821e05977ad989e1cc0d3fbc3", "0x93c8ea0326ef334bdc3011e74cd1a6d78ce0594d", "0x2bfd98be771eeeb4d69dd8767d200ba58252d925", "0x28c12c7b51860b9d5aec3a0ceb63c6e187c00aac", "0x882c9c16c05496d7b5374840936aec1af2a16553", "0x1b7740df685f9d34773d5a2aba6ab3a2c1407f40", "0xee1ad9c4f9d81f900221e95ee04246b6254b0c6f", "0x6273ce1f6f32e129f295f138d6e4ba6f0e19333e", "0x0b80f69fcb2564479058e4d28592e095828d24aa", "0x9ce4664e9d7346869797b7d9fc8c7a0212d5ff44", "0x2f78203c3161f1139edf2ba4b17b4e430ad2cbfa", "0x17d47c6ac4f72d43420f5e9533b526b2dee626a6", "0x742d898d2ee28a338f03af79c47762a908281a6a", "0xb901829c7e8b7d1de44d8bce086e7a5b0bcc7957", "0x39140deffdbd7c3b2415c29a40e0571365819f57", "0x60528316c553df7cae86d1294ca0d381ebb65cf0", "0x052e421be8e93d6f6c4d3d99defed914920fb3c4", "0x2c72d7a8c02752fcfafbaea5a63c53056cfaf547", "0x3dabf8331afbc553a1e458e37a6c9c819c452d55"],
  rpActivity: [0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1],
  rpLeader: ["0x20e5203a97b2e08c3dcc22c1c32e0dde3cc41da8", "0xbdada4f58d17ce602cb0d2db2a55c3e4f47e397f", "0xa923ac48439add7124763b3682f4505044c81ae3", "0x94ecbf26582455f5a7c88ab65a5a4ac05f6fe231", "0xdcefae3fdb94815f5d15111b46a5761a39b6ec9d", "0xf92ba56ac2506cb97c1d9ce55a54c595e0599ebd", "0x1b7740df685f9d34773d5a2aba6ab3a2c1407f40", "0xa4ebf5bbb131179b69bbf33319257728cdada5cf", "0x1a95e85e8ffcfd28eb61ee53a542dc98c57b337a", "0x266ddcfdbe3ded75e0e511e6356bca052b221c6b", "0xa4626e2bb450204c4b34bcc7525e585e8f678c0d", "0x533c13658591caa8a188211e73097adea7b94010", "0x0b80f69fcb2564479058e4d28592e095828d24aa", "0x1b7740df685f9d34773d5a2aba6ab3a2c1407f40", "0xa4626e2bb450204c4b34bcc7525e585e8f678c0d", "0xeeb157fdf2a72959f2f8be75ff500cf7a2104fbb", "0x4729672067e1ad8ca7f5770e3273747fe52affad", "0xcf34eb7f491fa7d18ba132938d7208e39da4b509", "0xcd54e0c35b122860d8fe2eb41f2e8e3e79c085ba", "0x28c12c7b51860b9d5aec3a0ceb63c6e187c00aac", "0xb64b60ba915bc16dc71ea59c9950c1538dcead9c", "0x36fad9acaf51a13527375b1ffc3d5a749153efdb", "0xdfd7aa554653ca236c197ad746edc2954ca172df", "0x4add297a1c2eea65e1ab5fd67e79647ecea8f36c", "0xc7afae3c9e99af27fe3eaa10f6ec73cd2dbe003b"]
}
```

## 3.6. 废弃的API

下列API接口已被废弃，将在后续的版本中移除

### 3.6.1. getBootNodePK
### 3.6.2. getIncentiveRunTimes
### 3.6.3. getRBAddress
### 3.6.4. getSlotCreateStatusByEpochID
### 3.6.5. getSlotScCallTimesByEpochID
### 3.6.6. getSmaByEpochID
### 3.6.7. getTotalIncentive
### 3.6.8. getTotalRemain
