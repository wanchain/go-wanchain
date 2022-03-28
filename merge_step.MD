# merge step

## 替换params
注释掉 
MainnetGenesisHash
MainnetChainConfig
MainnetBootnodes
这几个变量, 因为wanchain的也叫这几个名字.
wanchain的变量放在同名_wan后缀的文件里.


## 替换consensus
gwan第一步是ppow共识, 移植只需要能同步数据即可.
pos共识需要完整支持.


## 通信协议
gwan支持的是63 62.
geth只支持66 65, 已经不支持低版本了.
为了平滑升级, 必须支持新节点从老节点同步数据.所以需要把ETH33 移植到geth上.
另外geth支持了checkpoint 和snapshot同步. 看起来是好东西, 都是我们曾经想做而没有实现的, 如果可能, 都支持过来.

## tx格式变化.
eth在交易格式的开头增加了一个byte作为txtype, 目前只有两个值(0:传统, 1:accessList类型).
wan在交易格式的开头增加了一个uint64作为txtype, 目前有四个值(0:普通, 1:普通, 6:隐私 7:pos)
accessList还不确定是什么, 也行不需要支持.需要继续分析.

## stateDb格式修改
geth增加了一些方法，　wan也增加了一些方法, 需要整合起来.

## keystore 格式和解析的修改

## 隐私交易内置合约

## pos内置合约

  