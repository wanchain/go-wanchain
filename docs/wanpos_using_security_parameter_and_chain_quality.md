# Using Security parameter and Chain quality metrics

## Using Security parameter 
According WanPoS security model, there is a block security parameter K.
We define blockSecurityParam K, slotSecurityParam 2K

### Using security parameter in WanPos protocol transactions
Transactions for WanPos protocol interaction such as random beacon generating and slot leaders selection should check validity for in their stage.

![](media/0bc7faf1eeb11016dfe6c35b72f19a1e.png)

### confirm chain quality when creating block
When a honest node as create block as selected slot leader, It should check local working chain quality is enough
block_count / (new_slot - deep_slot)

when honest miner node calculate slot leaders, also should confirm chain quality is meaningful

### consensus related: roll back
according our security model, a honest node had a valid chain prefix should never rollback  more than  security parameter length blocks


## Chain quality monitor

chainQualityK : latest  K block chain quality

overallChainQuality: 

fixedTimeChainQuality:

EpochChainQuality:
