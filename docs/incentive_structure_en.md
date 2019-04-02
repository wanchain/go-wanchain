# 1. Incentive mechanism design 

# 2. Contents

<!-- TOC -->

- [1. Incentive mechanism design](#1-incentive-mechanism-design)
- [2. Contents](#2-contents)
- [3. Overview](#3-overview)
- [4. Functional module](#4-functional-module)
- [5. Interface design](#5-interface-design)
  - [5.1. Foundation funding interface](#51-foundation-funding-interface)
  - [5.2. Transaction fee collection interface](#52-transaction-fee-collection-interface)
  - [5.3. Trigger execution interface](#53-trigger-execution-interface)
  - [5.4. Chain Query Interface](#54-chain-query-interface)
  - [5.5. Stake Information Query Interface](#55-stake-information-query-interface)
  - [5.6. Stake Account Information save interface](#56-stake-account-information-save-interface)
- [6. Functional module design](#6-functional-module-design)
  - [6.1. Incentive collection](#61-incentive-collection)
  - [6.2. Statistic and calculation](#62-statistic-and-calculation)
  - [6.3. Information collection](#63-information-collection)
  - [6.4. Reward payment](#64-reward-payment)
  - [6.5. Gas fee collection](#65-gas-fee-collection)
  - [6.6. Trigger execution](#66-trigger-execution)
- [7. Workflow design](#7-workflow-design)

<!-- /TOC -->

# 3. Overview

The purpose of the reward mechanism is to reward the executor of the POS agreement. Reward the executor of the transaction, and reward the shareholders who remain online and out of the block. And through various mechanisms to achieve: "The evildoer has no chance to get a higher reward by doing evil" to ensure that rational participants will not do evil.

# 4. Functional module

After breaking down by function, as shown below:

![img](./incentive_en_img/1.png)

As shown in the figure above, the reward pool section can be divided into the following six sections:

- Reward collection: collect foundation funds and transaction fees, and calculate each epoch reward share;
- Statistics and cost-effectiveness: bonuses and distributions based on activity, address, probability, dividend ratio, total capital ratio, etc.;
- Information inquiry: query chain information, obtain information such as miner address, protocol executor address, activity level, etc., according to the agent's situation, dividend ratio and the total proportion of the address;
- Reward allocation: According to the result of the calculation, the reward accounting information of each address is written to the Stake account book;

The miner node part is mainly divided into the following two parts:

- Collection of transaction fees: When the miners execute the transaction, collect all transaction fees and send them to the reward pool interface;
- Trigger execution: When each miner comes out of the block and other nodes check the block, the execution of the bonus is calculated and issued.  Each epoch only needs to be executed once;

# 5. Interface design

The interface is the six external arrows in the above figure. mainly divided:

- Foundation funding interface
- Transaction fee collection interface
- Trigger execution interface
- Chain Query Interface
- Stake Information Query Interface
- Stake account Information save interface

## 5.1. Foundation funding interface

The current plan uses the foundation to directly burn a certain amount of funds, and then generate funds in the reward time code to achieve the purpose of fund balance.

## 5.2. Transaction fee collection interface

The transaction fee interface collects the transaction fee originally awarded to the miner for the unified distribution of the reward pool.

The original transaction fee collection code location is as follows:

```
\\ File: core/state_transaction.go, Line: 311
st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(usedGas, st.gasPrice))
```

Replace this line of code. Replace with the reward collection function call provided in the precompiled contract.

Input parameters:

- transaction fee amount

Return parameters:

- none

## 5.3. Trigger execution interface

The trigger execution interface triggers execution when the miner packs the block while other nodes check the block. An instance of StateDB is required for access and data writing.

```
// File: consensus.go Line: 81
// Finalize runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
// Note: The block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
Finalize(chain ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
  uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error)

```

Input parameters：

- StateDB

- epoch ID

Output parameters：

- success/failed

## 5.4. Chain Query Interface

The chain query interface just requires the current StateDB.

## 5.5. Stake Information Query Interface

The Stake account query interface invokes the interface provided by the Staker module to implement:

Input parameters:

- Address
- epoch ID

Return parameters:

- An array of data structures, including the address and the corresponding probability value: {addr string, probility *big.Int}, similar to:
```
[{address, probability}, {address, probability}, {address, probability}, {address, probability}, {address, probability}, {address, probability}, {address, probability}
The first address corresponds to the proxy address.
```
- Proxy dividend ratio (if the dividend ratio is 100%, it is an independent running node, not acting for others), the value range is 1~100. (we aslo called feerate).
- Total proportion, the sum of the total probabilities


## 5.6. Stake Account Information save interface

The Stake Account Information save interface is used to record the amount of revenue to each revenue address. Call the interface implementation provided by the Staker module.

Input parameters:

- A two-dimensional array of data structures, including a list of each agent and its corresponding amount. Similar to:
```
[
[{address, amount}, {address, amount}, {address, amount}, {address, amount}],
[{address, amount}, {address, amount}],
[{address, amount}, {address, amount}, {address, amount}],
]
```
The first address of each line is the proxy address.

# 6. Functional module design

## 6.1. Incentive collection

The bonus pool of the reward mechanism is mainly divided into two parts: the 10% mining award fund reserved by the Foundation and the Gas Fee transaction fee for all transactions executed on the chain.

Among them, the foundation part will be paid 5% for the first five years, and then each five-year award will be halved.

So in the first 5 years, the annual rewards are:
```
P = 21 million * 50% / 5 years = 2.1 million / year
```

The total prize money is:

![img](./incentive_en_img/2.png)

a is the first 5-year award: a = 2.1 million

b = 50%

Starting from the second five years, in addition to the original prize pool, the remaining amount of the last five years has not been added. The annual reward for the second five years is:
```
P = (21 million * 50% * 50% + S) / 5 years = (105 + a) million / year
```

Where S is the remaining reward for the last 5 years. a is a bonus every year after accounting.

The reward for the nth year is: (n ∈ [0,1,2,3,4,5,6,7,8,9...])
```
P = ( a * pow(b, floor(n / 5)) + S ) / 5
```
Therefore, the total amount of awards that should be issued for each epoch should be:
```
G = P / N + T
```
Where N is the total number of epochs within 1 year and T is the total transaction fee collected within the epoch.

After each epoch reward is completed, the actual amount of the award is recorded for the remaining use of the total reward after 5 years.

Or refer to BTCD (bitcoin's Go language client) code, calculate the reward as follows:

```
// CalcBlockSubsidy returns the subsidy amount a block at the provided height
// should have. This is mainly used for determining how much the coin base for
// newly generated blocks awards as well as validating the coin base for blocks
// has the expected value.
//
// The subsidy is halved every SubsidyReductionInterval blocks.  Mathematically
// this is: baseSubsidy / 2^(height/SubsidyReductionInterval)
//
// At the target block generation rate for the main network, this is
// approximately every 4 years.
func CalcBlockSubsidy(height int32, chainParams *chaincfg.Params) int64 {
    if chainParams.SubsidyReductionInterval == 0 {
        return baseSubsidy
    }

    // Equivalent to: baseSubsidy / 2^(height/subsidyHalvingInterval)
    return baseSubsidy >> uint(height/chainParams.SubsidyReductionInterval)
}
```

For: P = basic reward / (2^(block height/half-life))

Here, to adapt the WanPoS protocol, the block height is replaced with an epoch ID.

This module provides an interface for transaction fee injection.

In addition to the infinite number of divisions, return to the total prize pool, and add to the next 5 years for distribution.

## 6.2. Statistic and calculation

Statistics and calculation Calculate the specific reward share assigned to each reward address based on the input information.

The three rewarded characters are Epoch Leader, Random Proposer, Slot leaders.

The statistics and the total score are executed in the following 10 steps. The reward pool refers to the reward pool of the epoch to be rewarded. The following steps are explained in detail in steps 3 and 5:

1. Calculate the portion of the reward pool that belongs to the foundation;

2. Obtain the transaction fee amount in the reward pool and add it to the previous step to get the epoch total reward pool;

3. Get the address list and corresponding activity information by 3 roles;

4. Divide the total reward into three according to the bonus factor of the three roles, and complete the reward calculation separately;

5. According to the total share of the stake and the ceiling factor, further increasing the amount of the reward and deduct the deductible portion;

6. Deduct the agency fee according to whether it is an agent;

7. Proportionally allocate the remaining prize amount according to the probability value of the address and address;

8. Accumulate the total amount of the verification is correct; (What if the verification fails?)

9. The remainder is returned to the total reward pool;

10. Invoke the billing interface for billing distribution rewards.

The actual encoding can be appropriately classified according to the type of operation.

In the third step, when obtaining the address list and the activity level, it is necessary to pay attention to the number of outbound blocks of each address.  And the activity is the total value. The activity of the other two roles is associated with each address.

In the fifth step, the corresponding bonus value should be assigned according to the address of the third and fourth steps. And then the total share of the ceiling factor is used for each address. And the excess reward is deducted. The next step is to calculate the distribution of proxy benefits.

![img](./incentive_en_img/3.png)

## 6.3. Information collection

The information collection needs to obtain the necessary information from the chain and the Staker module. The information mainly includes the following contents:

1. Count each blocker in the epoch, count the address list and the number of blocks, and the epoch block activity;

2. Statistics agreement transaction information, obtain epoch leader and random proposer information and activity;

3. Query the total score of the address by address;

4. Query the agent rate, the list of agents and the probability;

## 6.4. Reward payment

After the reward is completed, the bonus will be directly credited to the address. Use the AddBalance function to increase the balance of each address to be assigned.

Also, the wan coin of the foundation part is burned from the reward pool. Transfer to 0 address.

Before and after the award is issued, it is necessary to verify whether the payment amount is correct and reasonable.

Calling the billing interface provided by Staker for reward accounting in Staker.

## 6.5. Gas fee collection

The transaction fee collection function uses the transaction fee amount to write to the state DB, using the precompiled contract address and epoch ID as the index.

At each write, the original value is obtained first, and the accumulation is performed and then written again.

Can be considered for the interface for post-inquiry.

## 6.6. Trigger execution

When the miner node packs the block, it triggers the execution of rewards and distribution.

When the remaining nodes verify the block, the execution of the bonus is calculated and issued.

# 7. Workflow design

The Coinbase transaction triggers statistics and cost-effectiveness and collects information to complete the processing flow. The process can be referred to in [5.2. Transaction fee collection interface](#52-transaction-fee-collection-interface)

![img](./incentive_en_img/4.png)