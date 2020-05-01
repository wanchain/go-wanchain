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

// Package pluto implements the proof-of-authority consensus engine.
package pluto

import (
	"errors"
	"math/big"
	"math"

	//"math/rand"
	"sync"
	"time"

	"github.com/wanchain/go-wanchain/pos/epochLeader"
	"github.com/wanchain/go-wanchain/pos/util"

	lru "github.com/hashicorp/golang-lru"
	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/consensus"

	"encoding/hex"

	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/sha3"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/incentive"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/slotleader"
	posUtil "github.com/wanchain/go-wanchain/pos/util"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/go-wanchain/rpc"
)

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

	wiggleTime = 500 * time.Millisecond // Random delay (per signer) to allow concurrent signers
)

// Pluto proof-of-authority protocol constants.
var (
	epochLength = uint64(30000) // Default number of blocks after which to checkpoint and reset the pending votes
	blockPeriod = uint64(15)    // Default minimum difference between two consecutive block's timestamps

	extraVanity = 0  // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.

	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn      = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn      = big.NewInt(1) // Block difficulty for out-of-turn signatures
	lastEpochSlotId = uint64(0)
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errInvalidCheckpointVote is returned if a checkpoint/epoch transition block
	// has a vote nonce set to non-zeroes.
	errInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// errExtraSigners is returned if non-checkpoint block contain signer data in
	// their extra-data fields.
	errExtraSigners = errors.New("non-checkpoint block contains extra signer list")

	// errInvalidCheckpointSigners is returned if a checkpoint block contains an
	// invalid list of signers (i.e. non divisible by 20 bytes, or not the correct
	// ones).
	errInvalidCheckpointSigners = errors.New("invalid signer list on checkpoint block")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	errInvalidDifficulty = errors.New("invalid difficulty")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")

	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errWaitTransactions is returned if an empty block is attempted to be sealed
	// on an instant chain (0 second period). It's important to refuse these as the
	// block reward is zero, so an empty block just bloats the chain... fast.
	errWaitTransactions = errors.New("waiting for transactions")
)

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// sigHash returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
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
		header.Extra[:len(header.Extra)-extraSeal], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	})
	hasher.Sum(hash[:0])
	return hash
}

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	log.Debug("signature", "hex", hex.EncodeToString(signature))

	log.Debug("sigHash(header)", "Bytes", hex.EncodeToString(sigHash(header).Bytes()))

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}

	log.Debug("pubkey in ecrecover", "pk", hex.EncodeToString(pubkey))
	// pubkey := signature
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	log.Debug("signer in ecrecover", "signer", signer.Hex())

	sigcache.Add(hash, signer)
	return signer, nil
}

// Pluto is the proof-of-authority consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type Pluto struct {
	config *params.PlutoConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	proposals map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields

	key *keystore.Key // Unlocked key
}

// New creates a Pluto proof-of-authority consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.PlutoConfig, db ethdb.Database) *Pluto {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySignatures)

	return &Pluto{
		config:     &conf,
		db:         db,
		recents:    recents,
		signatures: signatures,
		proposals:  make(map[common.Address]bool),
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *Pluto) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Pluto) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return c.verifyHeader(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Pluto) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := c.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (c *Pluto) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {

	if header.Number == nil {
		return errUnknownBlock
	}
	//number := header.Number.Uint64()

	// Don't waste time checking blocks from the future
	if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}

	// Checkpoint blocks need to enforce zero beneficiary
	// checkpoint := (number % c.config.Epoch) == 0
	// if checkpoint && header.Coinbase != (common.Address{}) {
	// 	return errInvalidCheckpointBeneficiary
	// }
	// Nonces must be 0x00..0 or 0xff..f, zeroes enforced on checkpoints
	// if !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
	// 	return errInvalidVote
	// }
	// if checkpoint && !bytes.Equal(header.Nonce[:], nonceDropVote) {
	// 	return errInvalidCheckpointVote
	// }
	// Check that the extra-data contains both the vanity and signature
	// if len(header.Extra) < extraVanity {
	// 	return errMissingVanity
	// }
	// if len(header.Extra) < extraVanity+extraSeal {
	// 	return errMissingSignature
	// }
	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	// signersBytes := len(header.Extra) - extraVanity - extraSeal
	// if !checkpoint && signersBytes != 0 {
	// 	return errExtraSigners
	// }
	// if checkpoint && signersBytes%common.AddressLength != 0 {
	// 	return errInvalidCheckpointSigners
	// }
	// Ensure that the mix digest is zero as we don't have fork protection currently
	// if header.MixDigest != (common.Hash{}) {
	// 	return errInvalidMixDigest
	// }
	// Ensure that the block doesn't contain any uncles which are meaningless in PoA
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	// if number > 0 {
	// 	if header.Difficulty == nil || (header.Difficulty.Cmp(diffInTurn) != 0 && header.Difficulty.Cmp(diffNoTurn) != 0) {
	// 		return errInvalidDifficulty
	// 	}
	// }
	// If all checks passed, validate any special fields for hard forks
	//if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
	//	return err
	//}
	// All basic checks passed, verify cascading fields
	// caculate leader
	//epochidSlotid := header.Difficulty.Uint64()
	//epochId := epochidSlotid >> 32
	//fmt.Println("verifyheader epochid: ", epochId)
	//if epochId != 0 {
	//	randombeacon.GetRandonBeaconInst().DoComputeRandom(epochId-1)
	//}

	return c.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Pluto) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	// if parent.Time.Uint64()+c.config.Period > header.Time.Uint64() {
	// 	return ErrInvalidTimestamp
	// }
	// Retrieve the snapshot needed to verify this header and cache it
	// snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	// if err != nil {
	// 	return err
	// }
	// If the block is a checkpoint block, verify the signer list
	// if number%c.config.Epoch == 0 {
	// 	signers := make([]byte, len(snap.Signers)*common.AddressLength)
	// 	for i, signer := range snap.signers() {
	// 		copy(signers[i*common.AddressLength:], signer[:])
	// 	}
	// 	extraSuffix := len(header.Extra) - extraSeal
	// 	if !bytes.Equal(header.Extra[extraVanity:extraSuffix], signers) {
	// 		return errInvalidCheckpointSigners
	// 	}
	// }
	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents, false)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (c *Pluto) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := c.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(c.config, c.signatures, c.db, hash); err == nil {
				log.Trace("Loaded voting snapshot form disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at block zero, make a snapshot
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := c.VerifyHeader(chain, genesis, false); err != nil {
				return nil, err
			}
			signers := make([]common.Address, (len(genesis.Extra))/common.AddressLength)
			for i := 0; i < len(signers); i++ {
				copy(signers[i][:], genesis.Extra[i*common.AddressLength:])
			}
			snap = newSnapshot(c.config, c.signatures, 0, genesis.Hash(), signers)
			if err := snap.store(c.db); err != nil {
				return nil, err
			}
			log.Trace("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	c.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(c.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *Pluto) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

func (c *Pluto) VerifyGenesisBlocks(chain consensus.ChainReader, block *types.Block) error {
	//passed verify default,need to remove if open verify
	return nil

	//epochId, _ := posUtil.CalEpochSlotID(block.Header().Time.Uint64())
	//hc, ok := chain.(*core.HeaderChain)
	//if !ok {
	//	bc,ok := chain.(*core.BlockChain)
	//	if !ok {
	//		log.Error("un support chain type")
	//		return errors.New("un support chain type")
	//	}
	//
	//	hc = bc.GetHc()
	//}
	////if hc.IsEpochFirstBlkNumber(epochId, block.Header().Number.Uint64(), nil) {
	////	extraType := block.Header().Extra[0]
	////	if extraType == 'g' {
	////		if len(block.Header().Extra) > extraSeal + 33 {
	////			egHash := common.BytesToHash(block.Header().Extra[1:33])
	////			if err := hc.VerifyEpochGenesisHash(epochID - 1, egHash, true); err != nil {
	////				return err
	////			}
	////
	////		} else {
	////			return fmt.Errorf("header extra info length is too short for epochGenesisHeadHash")
	////		}
	////	}
	////}
	//if block.Header().Number.Uint64() == posconfig.Pow2PosUpgradeBlockNumber{
	//	posconfig.FirstEpochId, _ = posUtil.CalEpSlbyTd(block.Header().Difficulty.Uint64())
	//}
	//
	//bGenerate := false
	//if hc.IsEpochFirstBlkNumber(epochId, block.Header().Number.Uint64(), nil) {
	//	bGenerate = true
	//}
	//
	//egHashPre, err := hc.GetEgHash(epochId - 1, block.Header().Number.Uint64(), bGenerate)
	//if err != nil {
	//	return err
	//}
	//egHashHeader := common.BytesToHash(block.Header().Extra[0:32])
	//if egHashPre != egHashHeader {
	//	log.Error("VerifyGenesisBlocks failed, epoch id="+ strconv.FormatUint(epochId, 10))
	//	return errors.New("VerifyGenesisBlocks failed, epoch id="+ strconv.FormatUint(epochId, 10))
	//}
	//return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (c *Pluto) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return c.verifySeal(chain, header, nil, true)
}

func (c *Pluto) verifyProof(block *types.Block, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	epochID, slotID := util.GetEpochSlotIDFromDifficulty(header.Difficulty)

	s := slotleader.GetSlotLeaderSelection()

	proof, proofMeg, err := s.GetInfoFromHeadExtra(epochID, header.Extra[:len(header.Extra)-extraSeal])

	if err != nil {
		log.Error("Can not GetInfoFromHeadExtra, verify failed", "error", err.Error())
		return err
	} else {
		log.Debug("verifyProof GetInfoFromHeadExtra", "pk", hex.EncodeToString(crypto.FromECDSAPub(proofMeg[0])))

		if !s.VerifySlotProof(block, epochID, slotID, proof, proofMeg) {
			log.Error("verifyProof failed", "number", number, "epochID", epochID, "slotID", slotID)
			return errUnauthorized
		} else {
			//log.Info("VerifyPackedSlotProof success", "number", number, "epochID", epochID, "slotID", slotID)
		}
	}

	return nil
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.

func (c *Pluto) verifySeal(chain consensus.ChainReader, header *types.Header, parents []*types.Header, isSlotVerify bool) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	epidTime, slIdTime := posUtil.CalEpochSlotID(header.Time.Uint64())

	epochID, slotID := util.GetEpochSlotIDFromDifficulty(header.Difficulty)

	if epidTime != epochID || slIdTime != slotID || header.Difficulty.Cmp(new(big.Int).SetUint64(math.MaxUint64))>0 {
		log.SyslogErr("epochId or slotid do not match", "epidTime=", epidTime, "slIdTime=", slIdTime, "epidFromDiffulty=", epochID, "slotIDFromDifficulty=", slotID)
		return errors.New("epochId or slotid do not match")
	}

	s := slotleader.GetSlotLeaderSelection()

	if len(header.Extra) > 512 { // proof,proofmsg,sign
		log.SyslogErr("Header extra info length is too long")
		return errUnauthorized
	} else if len(header.Extra) <= extraSeal {
		log.SyslogErr("Header extra info length is too short")
		return errUnauthorized

	} else {
		_, proofMeg, err := s.GetInfoFromHeadExtra(epochID, header.Extra[:len(header.Extra)-extraSeal])

		if err != nil {
			log.SyslogErr("Can not GetInfoFromHeadExtra, verify failed", "error", err.Error())
			return errUnauthorized
		} else {

			log.Debug("verifySeal GetInfoFromHeadExtra", "pk", hex.EncodeToString(crypto.FromECDSAPub(proofMeg[0])))

			pk := proofMeg[0]
			log.Debug("ecrecover(header, c.signatures)")
			signer, err := ecrecover(header, c.signatures)
			if err != nil {
				log.SyslogErr(err.Error())
				return errUnauthorized
			}

			if signer.Hex() != crypto.PubkeyToAddress(*pk).Hex() {
				log.SyslogErr("Pk signer verify failed in verifySeal", "number", number,
					"epochID", epochID, "slotID", slotID, "signer", signer.Hex(), "PkAddress", crypto.PubkeyToAddress(*pk).Hex())
				return errUnauthorized
			}

			if isSlotVerify {

				err := s.ValidateBody(types.NewBlockWithHeader(header))
				if err != nil {
					return err
				}
			}

			log.Debug("end c *Pluto ValidateBody")
		}
	}

	// Retrieve the snapshot needed to verify this header and cache it
	// snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	// if err != nil {
	// 	return err
	// }

	// Resolve the authorization key and check against signers
	// signer, err := ecrecover(header, c.signatures)
	// if err != nil {
	// 	return err
	// }

	// if _, ok := snap.Signers[signer]; !ok {
	// 	return errUnauthorized
	// }
	// for seen, recent := range snap.Recents {
	// 	if recent == signer {
	// 		// Signer is among recents, only fail if the current block doesn't shift it out
	// 		if limit := uint64(len(snap.Signers)/2 + 1); seen > number-limit {
	// 			return errUnauthorized
	// 		}
	// 	}
	// }
	// Ensure that the difficulty corresponds to the turn-ness of the signer
	// inturn := snap.inturn(header.Number.Uint64(), signer)
	// if inturn && header.Difficulty.Cmp(diffInTurn) != 0 {
	// 	return errInvalidDifficulty
	// }
	// if !inturn && header.Difficulty.Cmp(diffNoTurn) != 0 {
	// 	return errInvalidDifficulty
	// }
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Pluto) Prepare(chain consensus.ChainReader, header *types.Header, mining bool) error {
	// If the block isn't a checkpoint, cast a random vote (good enough for now)
	header.Coinbase = common.Address{}
	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()

	// Assemble the voting snapshot to check which votes make sense
	//snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	//if err != nil {
	//	return err
	//}
	//if number%c.config.Epoch != 0 {
	//	c.lock.RLock()
	//
	//	// Gather all the proposals that make sense voting on
	//	addresses := make([]common.Address, 0, len(c.proposals))
	//	for address, authorize := range c.proposals {
	//		if snap.validVote(address, authorize) {
	//			addresses = append(addresses, address)
	//		}
	//	}
	//	// If there's pending proposals, cast a vote on them
	//	if len(addresses) > 0 {
	//		header.Coinbase = addresses[rand.Intn(len(addresses))]
	//		if c.proposals[header.Coinbase] {
	//			copy(header.Nonce[:], nonceAuthVote)
	//		} else {
	//			copy(header.Nonce[:], nonceDropVote)
	//		}
	//	}
	//	c.lock.RUnlock()
	//}
	// Set the correct difficulty
	// header.Difficulty = CalcDifficulty(snap, c.signer)
	header.Difficulty = big.NewInt(1)

	// Ensure the extra data has all it's components
	//if len(header.Extra) < extraVanity {
	//	header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	//}
	//header.Extra = header.Extra[:extraVanity]

	//if number%c.config.Epoch == 0 {
	//	for _, signer := range snap.signers() {
	//		header.Extra = append(header.Extra, signer[:]...)
	//	}
	//}
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	//header.Time = new(big.Int).Add(parent.Time, new(big.Int).SetUint64(c.config.Period))
	//if header.Time.Int64() < time.Now().Unix() {
	//	header.Time = big.NewInt(time.Now().Unix())
	//}
	curEpochId, curSlotId := util.CalEpochSlotID(header.Time.Uint64())

	//if posconfig.EpochBaseTime == 0 {
	//	cur := time.Now().Unix()
	//	hcur := cur - (cur % posconfig.SlotTime) + posconfig.SlotTime
	//	header.Time = big.NewInt(hcur)
	//} else {
	//	//if curEpochId != 0 || curSlotId != 0 {
	//		header.Time = big.NewInt(int64(posconfig.EpochBaseTime + (curEpochId*posconfig.SlotCount+curSlotId)*posconfig.SlotTime))
	//	//}
	//}

	epochSlotId := uint64(1)
	epochSlotId += curSlotId << 8
	epochSlotId += curEpochId << 32

	// TODO: the Difficulty is duplicated with time. should delete it?
	header.Difficulty.SetUint64(epochSlotId)
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (c *Pluto) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	epochID, slotID := util.GetEpochSlotIDFromDifficulty(header.Difficulty)
	if posconfig.FirstEpochId != 0 && epochID > posconfig.FirstEpochId+2 && epochID >= posconfig.IncentiveDelayEpochs && slotID > posconfig.IncentiveStartStage {
		log.Debug("--------Incentive Start--------", "number", header.Number.String(), "epochID", epochID)
		snap := state.Snapshot()
		if !incentive.Run(chain, state, epochID-posconfig.IncentiveDelayEpochs) {
			log.SyslogAlert("********Incentive Failed********", "number", header.Number.String(), "epochID", epochID)
			state.RevertToSnapshot(snap)
		} else {
			log.Debug("--------Incentive Finish--------", "number", header.Number.String(), "epochID", epochID)
		}

		snap = state.Snapshot()
		if !epochLeader.StakeOutRun(state, epochID) {
			log.SyslogErr("Stake Out failed.")
			state.RevertToSnapshot(snap)
		}
	}

	if chain.Config().ChainId.Int64() == params.TestnetChainId && header.Number.Uint64() == posconfig.TestnetAdditionalBlock   {
		log.Info("Finalize testnet", "blockNumber", posconfig.TestnetAdditionalBlock)
		state.AddBalance(posconfig.PosOwnerAddrTestnet, posconfig.TestnetAdditionalValue)
		epochLeader.CleanInactiveValidator(state, epochID)
		//epochLeader.ListValidator(state)
	}

	// No block rewards in PoA, so the state remains as is and uncles are dropped
	state.Finalise(true)
	header.Root = state.IntermediateRoot(true /*chain.Config().IsEIP158(header.Number)*/)

	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *Pluto) Authorize(signer common.Address, signFn SignerFn, key *keystore.Key) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
	c.key = key
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Pluto) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return nil, errUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if c.config.Period == 0 && len(block.Transactions()) == 0 {
		return nil, errWaitTransactions
	}
	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn, key := c.signer, c.signFn, c.key
	c.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	// snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	// if err != nil {
	// 	return nil, err
	// }
	// if _, authorized := snap.Signers[signer]; !authorized {
	// 	return nil, errUnauthorized
	// }
	// check if our trun
	epochSlotId := uint64(1)
	epochId, slotId := util.CalEpochSlotID(header.Time.Uint64())
	epochSlotId += slotId << 8
	epochSlotId += epochId << 32
	if epochSlotId <= lastEpochSlotId {
		return nil, nil
	}
	localPublicKey := hex.EncodeToString(crypto.FromECDSAPub(&c.key.PrivateKey.PublicKey))
	leaderPub, err := slotleader.GetSlotLeaderSelection().GetSlotLeader(epochId, slotId)
	if err != nil {
		return nil, err
	}
	leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
	if leader != localPublicKey {
		return nil, nil
	}

	log.Info("Generate a new block", "number", number, "epochID", epochId, "slotId", slotId, "curTime", time.Now(),
		"header.Time", header.Time)

	//leaderPub, err := slotleader.GetSlotLeaderSelection().GetSlotLeader(epochId, slotId)
	//if err != nil {
	//	return nil, err
	//}
	//leader := hex.EncodeToString(crypto.FromECDSAPub(leaderPub))
	//localPublicKey := hex.EncodeToString(crypto.FromECDSAPub(&c.key.PrivateKey.PublicKey))
	//if leader == localPublicKey {
	//	log.Info("Generate a new block", "number", number, "epochID", epochId, "slotId", slotId, "curTime", time.Now(),
	//		"header.Time", header.Time)
	//} else {
	//	return nil, nil
	//}

	header.Difficulty.SetUint64(epochSlotId)
	header.Coinbase = signer

	s := slotleader.GetSlotLeaderSelection()
	buf, err := s.PackSlotProof(epochId, slotId, key.PrivateKey)
	if err != nil {
		log.Warn("PackSlotProof failed in Seal", "epochID", epochId, "slotID", slotId, "error", err.Error())
		return nil, err
	}

	extra := make([]byte, len(buf)+extraSeal)
	header.Extra = extra

	copy(header.Extra[:len(buf)], buf)
	header.Difficulty.SetUint64(epochSlotId)

	sighash, err := signFn(accounts.Account{Address: signer}, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sighash)

	log.Debug("signature", "hex", hex.EncodeToString(sighash))
	log.Debug("sigHash(header)", "Bytes", hex.EncodeToString(sigHash(header).Bytes()))
	log.Debug("Packed slotleader proof info success", "epochID", epochId, "slotID", slotId, "len", len(header.Extra), "pk", hex.EncodeToString(crypto.FromECDSAPub(&key.PrivateKey.PublicKey)))

	err = c.verifySeal(nil, header, nil, false)
	if err != nil {
		log.Warn("Seal error", "error", err.Error())
		return nil, err
	}

	err = c.verifyProof(block.WithSeal(header), header, nil)
	if err != nil {
		log.Warn("Seal error", "error", err.Error())
		return nil, err
	}
	lastEpochSlotId = epochSlotId
	return block.WithSeal(header), nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *Pluto) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	snap, err := c.snapshot(chain, parent.Number.Uint64(), parent.Hash(), nil)
	if err != nil {
		return nil
	}
	return CalcDifficulty(snap, c.signer)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func CalcDifficulty(snap *Snapshot, signer common.Address) *big.Int {
	if snap.inturn(snap.Number+1, signer) {
		return new(big.Int).Set(diffInTurn)
	}
	return new(big.Int).Set(diffNoTurn)
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Pluto) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "pluto",
		Version:   "1.0",
		Service:   &API{chain: chain, pluto: c},
		Public:    false,
	}}
}
