# WanPos Frame

# summary
WanPos has the following modules
![img](./frame/frame.png)

# Storage

## pos_storage
pos_storage supply some of common metheds to record information in local DB, include epoch's epoch Leader, random beacon group, etc. currently wanpos has 3 local DBs
*pos: store the information when pos protocol run. include the random of slot leader selection. 
*rblocaldb: store random beacon group for every epoch
*eplocaldb: store epoch leaders for every epoch.


## stateDB
stateDB: store blockchain state information.

# Smart Contract

## staking delegate contract
Staking delegate contract used for deposit wancoin and the user node will become a stakeholder or a delegate.  
[detail](./wanpos_staking_delegate.md)
 


# Wanpos testnet start

build/bin/gwan --testnet --nodiscover --etherbase  "0xcf696d8eea08a311780fb89b20d4f0895198a489"  --unlock "0xcf696d8eea08a311780fb89b20d4f0895198a489" --password ./pw.txt  --mine --minerthreads=1
## parameter 
* --testnet: specify the consensus.
* --datadir: datadir and ipc path
* --etherbase: specify the account used for mining.
* --mine --minerthreads: specify enable mining and minerthreads is 1.
* --unlock: unlock the specified account.


## PosInit PosInitMiner
Pos module has 2 initial function.
* PosInit: this one will invoke both mining and non-mining node. because non-mining node need verify pos transaction too.
* PosInitMiner: this one is used only for mining node. 

## epoch Leader selection
because wanpos protocol need select epoch leader from all staker, there are some staker builtin the genesis. At the first epoch, all the slot leader is the builtin staker. At the same time, 50 public keys will be selected as epochLeader, which will send transaction to select slot leader, and 21 public keys will be selected as random beacon group, which will generate the random number for epoch.

|       |Genesis|Epoch 0 | Epoch 1  | Epoch 2...|
|  ---  | ---   | ---    | ---      | ---       |
|staker |builtin|allow register  | allow register  |allow register     |
|epochLeader|NA |select from builtin|select from builtin|select from epoch 0|
|slotLeader|NA|the same one|select from epoch 0 transactions|select from epoch 1 transactions|



# miner
we add a go routine timer loop in miner module.  

BackendTimerLoop lanch a goroutine.

the timer will invoke slot_selection and random_beacon 
![img](./frame/BackendTimerLoop.png) 


# epoch/RB leader selection

## The last block in a epoch
Because the blockchain's blockNumber is not corresponding to time(allow there is no block in a slot), we cache the last block number for a epoch. the function GetTargetBlkNumber() will return the last block number of a epoch


## when to generate epoch/RB leader
when download a new block(insertChain in blockchain.go) or generate a new block(Seal in pluto.go), we update the last block number for a epoch, if the epoch id updated, we start to caculate the epoch Leader and rb group(GetEpocherInst().SelectLeadersLoop()) and store them in local DB.

