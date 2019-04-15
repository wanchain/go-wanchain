# WanPos staking and Delegate Design

## Summary
Wan holders participate in WanPoS by sending a certain amount of wancoin to consensus smart contract CSC to lock for a period of time determined by themselves. In this way, wans are transformed into wanstake, which increases with wan amount and locking time. Meanwhile the left locking time also influences the stake amount.  




![image](./staking_delegate/staking.jpg)


$$ StakeRate = Amount*(10+lockEpoch/(maxLockEpoch/10)) * (2-e^{(t-1)} ) $$

the t is the left percentage of locking time

At the same time, for decreasing the scale of pos node, we introduced in the delegate mechanism.
User can specify if he want to be a delegate when he register into pos.
to be a delegate, you must register at least ?? wancoin.
to delegate your coin to another one, you must have at lease ?? wancoin.
to register to pos, the locked epoch is ?? ~ ??.

* the incentive is paied automatic and wancoin refund is automatic too.
* currently, the same publickey cann't join 2 times.
* when register, the msg.from may be different with the secPk. when refund the wancoin will return to the msg.from address and the incentive will pay to secAddress. 



## Coin staking and delegate contract
 This contract is implemented by precompiled contract. It will define several functions and structures which will be used in staking and delegate

### register interface solidity description.

```
contract stake {
	function stakeIn( bytes memory secPk, bytes memory bn256Pk, uint256 lockEpochs, uint256 feeRate) public payable {}
	function delegateIn(address delegateAddress) public payable {}
}

```
there
* sPub is the stakeholder's public  key.  
* bn256Pk is the stakeholder's bn256 public key.   
* lockEpochs is the number of locking epoch.  
* feeRate is delegate fee rate.  The range is 1~100.  100 means not to be a delegate.
* delegateAddr is the address you want to delegate to.  

### The structures 
 
Staker information which will be stored in db
``` 
type StakerInfo struct {
	Address	    common.Address
	PubSec256   []byte //stakeholder’s wan public key
	PubBn256    []byte //stakeholder’s bn256 public key

	Amount      *big.Int //staking wan value
	LockEpochs   uint64   //lock time which is input by user
	From        common.Address

	StakingEpoch uint64 //the user’s staking time
	FeeRate     uint64
	Clients      []ClientInfo
}

type ClientInfo struct {
	Address common.Address
	Amount   *big.Int
	StakingEpoch uint64
}

```

### Functions of contract

* StakeIn  
process user put wan into CSC  
```
func (p *Pos_staking) StakeIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) 
```

* DelegateIn  
process user put wan into a delegate.   
```
func (p *Pos_staking) DelegateIn(payload []byte, contract *Contract, evm *EVM) ([]byte, error) 
``` 

## Epoch leader random beacon group selection

### how to select the epoch leader and rb group.
refer to the Glaxy paper.

### wancoin to wanStake
1. formula

// TODO



### Functions for leader selection
* Create new epocher instance
```
func NewEpocher(blc *core.BlockChain) *Epocher  
```

* Start epoch leaders and random proposers selection
```
func (e *Epocher) SelectLeadersLoop(epochId uint64) error  
```

# module API
## wanStake check api
```
type ClientProbability struct {
	addr common.Address
	probability  *big.Int
}
func (e *Epocher)GetEpochProbability(epochId uint64,addr common.Address) (infors []vm.ClientProbability,  feeRate uint64, totalProbability *big.Int, err error) 
```

## incentive api
```
type ClientIncentive struct {
	addr common.Address
	Incentive  *big.Int
}
func (e *Epocher)SetEpochIncentive(epochId uint64, infors [][]vm.ClientIncentive) (err error)
```

