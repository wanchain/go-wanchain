## Problem background
Currently the slot is accurately align with time.because according the security model, the block chain honest node never offline K slot time. Also assume the chain never down. But in pratice maybe there unexpected bug cause network crashed.
So how we react this scenario.


The model paper didn't show how recovery the network when network crashed.
We make a proposal:

## Solution 1
* all slot time using offset time to the epoch start time of slot. Every Epoch has its start time.
* all honest miner node(foundation and partners node), restart the node,using the last valid block of honest nodes, Cross epoch concat the blocks.

### advantage
* the epoch is seems continuous for common user, whole chain keeps growning continously

### disadvantage
* slot time is offset time, the calculation turns complicated
* after recovering chain, the the epoch for the last valid block for recovering need to record more than one start time, which will bring more work to do for the block verify in downloading process
* for the current wanchain system, there are more modification for code,such as header verify, slot time caculation and epoch start time recording ...  
* difficult to coordinate with partners to upgrade gwan programe and keep online



## Solution 2

*  use relative time to pos genesis starting time to calculate epoch id and slot id
*  if whole nework stop, use pos starting block state to start a new epoch
*  use foundation nodes to start the new epoch to recover the block chain growing
*  after block block chain recovered, coordinate partiner to upgrade gwan node

### advantage
* epochid slotid caculation is simple and same with current code, do not need change
* code for recovering chain growing is simple, only need to set the initializing process, other system do not need make much modification
* do not need to care about if partners nodes is online or offline

### disadvantage
* epoch is not continous and the epoch that chain stop is not completed