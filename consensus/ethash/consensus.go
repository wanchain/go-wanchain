// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethash

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/consensus"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/sha3"
	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/rlp"
	set "gopkg.in/fatih/set.v0"
)

const (
	//	checkpointInterval = 1024 // Number of blocks after which to save the signers state
	checkpointInterval = 4 // Number of blocks after which to save the signers state, using small value for testing...
)

// Ethash proof-of-work protocol constants.
var (
	blockReward *big.Int = big.NewInt(5e+18) // Block reward in wei for successfully mining a block
	maxUncles            = 2                 // Maximum number of uncles allowed in a single block

	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errLargeBlockTime    = errors.New("timestamp too big")
	errZeroBlockTime     = errors.New("timestamp equals parent's")
	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")
	errNonceOutOfRange   = errors.New("nonce out of range")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
	errMissingSignature  = errors.New("extra data 65 byte suffix signature missing")
	errUnauthorized      = errors.New("unauthorized signer")
	errAuthorTooOften    = errors.New("signer too often")
)

// INFO: copied from consensus/clique/clique.go
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// INFO: copied from consensus/clique/clique.go
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65], // Yes, this will panic if extra is too short
	})
	hasher.Sum(hash[:0])
	return hash
}

// INFO: copied from consensus/clique/clique.go
// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header) (common.Address, error) {
	functrace.Enter()
	defer functrace.Exit()

	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Ethereum address
	//log.Trace("ecrecover(): cr@zy seal")
	log.Trace(fmt.Sprintf(header.String()))

	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	//log.Trace("ecrecover(): cr@zy seal hash", "Input hash", sigHash(header).String())
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	return signer, nil
}

// INFO: copied from consensus/clique/clique.go
// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	functrace.Enter()
	defer functrace.Exit()

	return header.Coinbase, nil
}

func (self *Ethash) Authorize(signer common.Address, signFn SignerFn) {
	self.lock.Lock()
	defer self.lock.Unlock()

	functrace.Enter(signer.String())
	defer functrace.Exit()

	self.signFn = signFn
	self.signer = signer
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (ethash *Ethash) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.fakeFull {
		log.Error("fakeFull")
		return nil
	}

	functrace.Enter()
	defer functrace.Exit()

	// Short circuit if the header is known, or it's parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	parents := []*types.Header{parent}
	return ethash.verifyHeader(chain, header, parents, false, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (ethash *Ethash) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	functrace.Enter()
	defer functrace.Exit()

	// If we're running a full engine faking, accept any input as valid
	if ethash.fakeFull || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = ethash.verifyHeaderWorker(chain, headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (ethash *Ethash) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header, seals []bool, index int) error {
	functrace.Enter()
	defer functrace.Exit()

	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}

	var parents []*types.Header
	if index == 0 {
		parents = []*types.Header{parent}
	} else {
		parents = headers[:index]
	}
	return ethash.verifyHeader(chain, headers[index], parents, false, seals[index])
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum ethash engine.
func (ethash *Ethash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	functrace.Enter()
	defer functrace.Exit()

	// If we're running a full engine faking, accept any input as valid
	if ethash.fakeFull {
		return nil
	}
	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := set.New(), make(map[common.Hash]*types.Header)

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestor := chain.GetBlock(parent, number)
		if ancestor == nil {
			break
		}
		ancestors[ancestor.Hash()] = ancestor.Header()
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
		parent, number = ancestor.ParentHash(), number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Has(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		//if err := ethash.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true); err != nil {
		//	return err
		//}
	}
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
//
// See YP section 4.3.4. "Block Header Validity"
func (ethash *Ethash) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header, uncle bool, seal bool) error {
	functrace.Enter()
	defer functrace.Exit()

	// Ensure that the header's extra-data section is of a reasonable size
	parent := parents[len(parents)-1]
	//	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
	//		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	//	}
	// ^^^ superseded by the check below

	if len(header.Extra) != extraVanity+extraSeal {
		return fmt.Errorf("extra-data size invalid: %d", len(header.Extra))
	}
	// Verify the header's timestamp
	if uncle {
		if header.Time.Cmp(math.MaxBig256) > 0 {
			return errLargeBlockTime
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time.Cmp(parent.Time) <= 0 {
		return errZeroBlockTime
	}
	// Verify the block's difficulty based in it's timestamp and parent's difficulty
	expected := CalcDifficulty(chain.Config(), header.Time.Uint64(), parent.Time.Uint64(), parent.Number, parent.Difficulty)
	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}
	// Verify that the gas limit is <= 2^63-1
	if header.GasLimit.Cmp(math.MaxBig63) > 0 {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, math.MaxBig63)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed.Cmp(header.GasLimit) > 0 {
		return fmt.Errorf("invalid gasUsed: have %v, gasLimit %v", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := new(big.Int).Set(parent.GasLimit)
	diff = diff.Sub(diff, header.GasLimit)
	diff.Abs(diff)

	limit := new(big.Int).Set(parent.GasLimit)
	limit = limit.Div(limit, params.GasLimitBoundDivisor)

	if diff.Cmp(limit) >= 0 || header.GasLimit.Cmp(params.MinGasLimit) < 0 {
		return fmt.Errorf("invalid gas limit: have %v, want %v += %v", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	if seal {
		if err := ethash.VerifySeal(chain, header); err != nil {
			return err
		}
	}

	if seal {
		if err := ethash.verifySignerIdentity(chain, header, parents); err != nil {
			return err
		}
	}

	return nil
}

// TODO: its's different from POA, need consider thread-security
func (self *Ethash) verifySignerIdentity(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	number := header.Number.Uint64()

	functrace.Enter(fmt.Sprintf("number= %d", number))
	defer functrace.Exit()

	if number == 0 {
		return nil
	}

	_, err := self.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

    return nil
}

// INFO: copied from consensus/clique/clique.go , mostly
// snapshot retrieves the signer status at a given point
func (self *Ethash) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	var (
		headers []*types.Header
		snap    *Snapshot
	)

	functrace.Enter(fmt.Sprintf("number= %d", number))
	defer functrace.Exit()

	for snap == nil {
		if s, ok := self.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}

		if number%checkpointInterval == 0 {
			if s, err := LoadSnapShot(self.db, hash); err == nil {
				snap = s
				break
			}
		}

		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			// TODO: cr@zy think maybe useful add this
			//if err := self.VerifyHeader(chain,genesis, false); err != nil {
			//	return nil, err
			//}
			// TODO: does we need
			signers := make([]common.Address, len(genesis.Extra)/common.AddressLength)
			for i := 0; i < len(signers); i++ {
				copy(signers[i][:], genesis.Extra[i*common.AddressLength:])
			}
			snap = NewSnapshot(0, genesis.Hash(), signers)
			if err := snap.store(self.db); err != nil {
				return nil, err
			}
			break
		}

		var header *types.Header
		if len(parents) > 0 {
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}

	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}

	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}

	self.recents.Add(snap.Hash, snap)

	if snap.Number%checkpointInterval == 0 && len(headers) > 0{
		if err = snap.store(self.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}

	return snap, nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have when created at time given the parent block's time
// and difficulty.
//
// TODO (karalabe): Move the chain maker into this package and make this private!
func CalcDifficulty(config *params.ChainConfig, time, parentTime uint64, parentNumber, parentDiff *big.Int) *big.Int {
	functrace.Enter()
	defer functrace.Exit()

	if config.IsHomestead(new(big.Int).Add(parentNumber, common.Big1)) {
		return calcDifficultyHomestead(time, parentTime, parentNumber, parentDiff)
	}
	return calcDifficultyFrontier(time, parentTime, parentNumber, parentDiff)
}

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)
)

// calcDifficultyHomestead is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyHomestead(time, parentTime uint64, parentNumber, parentDiff *big.Int) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.mediawiki
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).SetUint64(parentTime)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp -parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(common.Big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)))
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parentDiff, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parentDiff, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parentNumber, common.Big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(common.Big1) > 0 {
		y.Sub(periodCount, common.Big2)
		y.Exp(common.Big2, y, nil)
		x.Add(x, y)
	}
	return x
}

// calcDifficultyFrontier is the difficulty adjustment algorithm. It returns the
// difficulty that a new block should have when created at time given the parent
// block's time and difficulty. The calculation uses the Frontier rules.
func calcDifficultyFrontier(time, parentTime uint64, parentNumber, parentDiff *big.Int) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parentDiff, params.DifficultyBoundDivisor)
	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.SetUint64(parentTime)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
		diff.Add(parentDiff, adjust)
	} else {
		diff.Sub(parentDiff, adjust)
	}
	if diff.Cmp(params.MinimumDifficulty) < 0 {
		diff.Set(params.MinimumDifficulty)
	}

	periodCount := new(big.Int).Add(parentNumber, common.Big1)
	periodCount.Div(periodCount, expDiffPeriod)
	if periodCount.Cmp(common.Big1) > 0 {
		// diff = diff + 2^(periodCount - 2)
		expDiff := periodCount.Sub(periodCount, common.Big2)
		expDiff.Exp(common.Big2, expDiff, nil)
		diff.Add(diff, expDiff)
		diff = math.BigMax(diff, params.MinimumDifficulty)
	}
	return diff
}

// VerifySeal implements consensus.Engine, checking whether the given block satisfies
// the PoW difficulty requirements.
func (ethash *Ethash) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// If we're running a fake PoW, accept any seal as valid
	if ethash.fakeMode {
		time.Sleep(ethash.fakeDelay)
		if ethash.fakeFail == header.Number.Uint64() {
			return errInvalidPoW
		}
		return nil
	}

	functrace.Enter()
	defer functrace.Exit()

	// If we're running a shared PoW, delegate verification to it
	if ethash.shared != nil {
		return ethash.shared.VerifySeal(chain, header)
	}
	// Sanity check that the block number is below the lookup table size (60M blocks)
	number := header.Number.Uint64()
	if number/epochLength >= uint64(len(cacheSizes)) {
		// Go < 1.7 cannot calculate new cache/dataset sizes (no fast prime check)
		return errNonceOutOfRange
	}
	// Ensure that we have a valid difficulty for the block
	if header.Difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}
	// Recompute the digest and PoW value and verify against the header
	cache := ethash.cache(number)

	size := datasetSize(number)
	if ethash.tester {
		size = 32 * 1024
	}
	digest, result := hashimotoLight(size, cache, header.HashNoNonce().Bytes(), header.Nonce.Uint64())
	if !bytes.Equal(header.MixDigest[:], digest) {
		return errInvalidMixDigest
	}
	target := new(big.Int).Div(maxUint256, header.Difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) > 0 {
		return errInvalidPoW
	}
	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (ethash *Ethash) Prepare(chain consensus.ChainReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	functrace.Enter(fmt.Sprintf("header number= %d", header.Number.Uint64()))
	defer functrace.Exit()

	snap, err := ethash.snapshot(chain, parent.Number.Uint64(), parent.Hash(), nil)
	if err != nil {
		return err
	}

	err = snap.isLegal4Sign(header.Coinbase)
	if err != nil{
		return err
	}

	header.Difficulty = CalcDifficulty(chain.Config(), header.Time.Uint64(),
		parent.Time.Uint64(), parent.Number, parent.Difficulty)

	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, make([]byte, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state and assembling the block.
func (ethash *Ethash) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	functrace.Enter()
	defer functrace.Exit()

	// Accumulate any block and uncle rewards and commit the final state root
	AccumulateRewards(state, header, uncles)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, txs, uncles, receipts), nil
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
//
// TODO (karalabe): Move the chain maker into this package and make this private!
func AccumulateRewards(state *state.StateDB, header *types.Header, uncles []*types.Header) {
	functrace.Enter()
	defer functrace.Exit()

	/*reward := new(big.Int).Set(blockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, blockReward)
		r.Div(r, big8)
		state.AddBalance(uncle.Coinbase, r)

		r.Div(blockReward, big32)
		reward.Add(reward, r)
	}
	state.AddBalance(header.Coinbase, reward)*/
}
