## Problem background
Currently the slot is accurately align with time.because according the security model, the block chain honest node never offline K slot time. Also assume the chain never down. But in pratice maybe there unexpected bug cause network crashed.
So how we react this scenario.

## Solution
The model paper didn't show how recovery the network when network crashed.
We make a proposal:
    all slot time using offset time to the epoch start time of slot. Every Epoch has its start time.
    all honest miner node(foundation and partners node), restart the node,using the last valid block of honest nodes, Cross epoch concat the blocks.
    
In the new implementation: Epoch has jumpEpochs for cross epoch time concat blocks

After WanPoS beta we will complete the code for network maintainable.
