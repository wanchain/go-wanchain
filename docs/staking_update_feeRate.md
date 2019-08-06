# validator update feeRate

## introduction 

In our beta release, a validator can update his Locktime. 
For example, a validator register, and specify the lockTime to 10 epochs, feeRate to 90%
the interface is
```
stakeIn(secPK, bnPK, lockTime, feeRate)
```
Thus, this validator's lockTime is 10.   
Later, this validator can change his lockTime via send a transaction. 
```
stakeUpdate(validator_address, lockEpochs)
```

## new requirement
There is a new requirement that a validator should be allow to change his feeRate.   
And the feeRate specified when staking is the max feeRate,
the validator could change his new feeRate from 0 to max feeRate
Currently Gwan don't support this requirement.

## Difficulty
Because we have released beta, our new build must be compatible with old version.

### code change
we need add a new interface stakeUpdateFeeRate(address addr, uint256 feeRate)

currently, the stake interfaces is
```
contract stake {
	function stakeIn(bytes memory secPk, bytes memory bn256Pk, uint256 lockEpochs, uint256 feeRate) public payable {}
	function stakeUpdate(address addr, uint256 lockEpochs) public {}
	function stakeAppend(address addr) public payable {}
	function partnerIn(address addr, bool renewal) public payable {}
	function delegateIn(address delegateAddress) public payable {}
	function delegateOut(address delegateAddress) public {}
}
```

the new interface is:
```
contract stake {
    function stakeIn(bytes memory secPk, bytes memory bn256Pk, uint256 lockEpochs, uint256 feeRate) public payable {}
    function stakeUpdate(address addr, uint256 lockEpochs) public {}
    function stakeUpdateFeeRate(address addr, uint256 feeRate) public {}
    function stakeAppend(address addr) public payable {}
    function partnerIn(address addr, bool renewal) public payable {}
    function delegateIn(address delegateAddress) public payable {}
    function delegateOut(address delegateAddress) public {}
}
```

the function stakeUpdateFeeRate(address addr, uint256 feeRate) is new.
when this api is involved, if this is the first time, we create a new structure 
type UpdateFeeRate struct {
	ValidatorAddr    common.Address
	MaxFeeRate uint64
	FeeRate uint64
    EpochId uint64
}
the ValidatorAddr is the validator address.   
the EpochId is when the tx is called.
FeeRate is the new feeRate   
MaxFeeRate should copy from the stakerInfo structure, and this is the max feeRate. when validator change his feeRate, can't over this limitation.   
else, this is not the first time invoke this api, we check if the new feerate is small than the maxFeeRate, then store it.

### when will the new feeRate take effective.
My opinion is the new feeRate will take effective at the new round after renew. because if the validator can change his feeRate next epoch, the delegator can't expect his earnings.
In the StakeOutRun(), if the current cycle is expiring, we get the newFeerate from UpdateFeeRate to stakeInfo.

### storeage change.
currently, the stakeInfo is storeed in
```
StakersInfoAddr       = common.BytesToAddress(big.NewInt(400).Bytes())
```
we add a new storage
```
StakersFeeAddr        = common.BytesToAddress(big.NewInt(402).Bytes())
```
UpdateFeeRate structure will store in this storage.
the key is ValidatorAddr convert to hash.

