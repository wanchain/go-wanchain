package vm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/pos/wanpos_crypto"
	"math/big"
	"strconv"
	"strings"
)

var (
	rbscDefinition = `[{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg","outputs":[{"name":"Info","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"sigshare","outputs":[{"name":"Info","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	rbscAbi, errRbscInit = abi.JSON(strings.NewReader(rbscDefinition))

	dkgId [4]byte
	sigshareId [4]byte
	genRId [4]byte
	// Generator of G1
	//gbase = new(bn256.G1).ScalarBaseMult(big.NewInt(int64(1)))
	// Generator of G2
	hbase = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))

	RANDOMBEACON_DB_KEY = "PosRandomBeacon"

)

type RandomBeaconContract struct {
}


func init()  {
	if errRbscInit != nil {
		panic("err in rbsc abi initialize")
	}

	copy(dkgId[:], 	rbscAbi.Methods["dkg"].Id())
	copy(sigshareId[:], rbscAbi.Methods["sigshare"].Id())
	copy(genRId[:], rbscAbi.Methods["genR"].Id())
}

func (c *RandomBeaconContract) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *RandomBeaconContract) Run(input []byte, contract *Contract, evm *EVM) ([]byte, error) {
	// check data
	if len(input) < 4 {
		return nil, errParameters
	}

	var methodId [4]byte
	copy(methodId[:], input[:4])

	if methodId == dkgId {
		return c.dkg(input[4:], contract, evm)
	} else if methodId == sigshareId {
		return c.sigshare(input[4:], contract, evm)
	}

	return nil, nil
}

func (c *RandomBeaconContract) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

func GetRBKeyHash(funId []byte, epochId uint64, proposerId uint32) (*common.Hash) {
	keyBytes := make([]byte, 20)
	keyBytes = append(keyBytes, funId ...)
	keyBytes = append(keyBytes, UIntToByteSlice(epochId) ...)
	keyBytes = append(keyBytes, UIntToByteSlice(uint64(proposerId)) ...)
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	return &hash
}

func GetDkg(db StateDB, epochId uint64, proposerId uint32) (*RbDKGTxPayload, error) {
	hash := GetRBKeyHash(dkgId[:], epochId, proposerId)
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	var dkgParam RbDKGTxPayload
	err := rlp.DecodeBytes(payloadBytes, &dkgParam)
	if err != nil {
		return nil, buildError("load dkg error", dkgParam.EpochId, dkgParam.ProposerId)
	}

	return &dkgParam, nil
}

func GetSig(db StateDB, epochId uint64, proposerId uint32) (*RbSIGTxPayload, error) {
	hash := GetRBKeyHash(dkgId[:], epochId, proposerId)
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	var sigParam RbSIGTxPayload
	err := rlp.DecodeBytes(payloadBytes, &sigParam)
	if err != nil {
		return nil, buildError("load sig error", epochId, proposerId)
	}

	return &sigParam, nil
}
func GetGenesisRandon() *big.Int {
	return big.NewInt(1)
}
func GetRandom(epochId uint64) (*big.Int, error) {
	bt, err := posdb.GetDb().Get(epochId, RANDOMBEACON_DB_KEY)
	if err != nil {
		if epochId == 0 {
			return GetGenesisRandon(), nil
		}

		return nil, err
	}

	return new(big.Int).SetBytes(bt), nil
}

func GetRBM(epochId uint64) ([]byte, error) {
	epochIdBigInt := big.NewInt(int64(epochId + 1))
	preRandom, err := GetRandom(epochId)
	if err != nil {
		return nil, err
	}

	buf := epochIdBigInt.Bytes()
	buf = append(buf, preRandom.Bytes()...)
	return crypto.Keccak256(buf), nil
}

func GetRBAbiDefinition() (string) {
	return rbscDefinition
}

func GetRBAddress() (common.Address) {
	return randomBeaconPrecompileAddr
}

func GetRBProposerGroup(epochId uint64) []bn256.G1 {
	db, b := posdb.GetDbByName("rblocaldb")
	if !b {
		return nil
	}
	pks := db.GetStorageByteArray(epochId)
	length := len(pks)
	if length == 0 {
		return nil
	}
	g1s := make([]bn256.G1, length, length)

	for i := 0; i < length; i++ {
		g1s[i] = *new(bn256.G1)
		g1s[i].Unmarshal(pks[i])
	}
	return g1s
}

func GetProposerPubkey(pks *[]bn256.G1, proposerId uint32) (*bn256.G1) {
	return &(*pks)[proposerId]
}

func UIntToByteSlice(num uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, num)
	return b
}

type RbDKGTxPayload struct {
	EpochId uint64
	ProposerId uint32
	Enshare []*bn256.G1
	Commit []*bn256.G2
	Proof []wanpos.DLEQproof
}

type RbSIGTxPayload struct {
	EpochId uint64
	ProposerId uint32
	Gsigshare *bn256.G1
}

// TODO: evm.EpochId evm.SlotId, Cfg.K---dkg:0 ~ 4k -1, sig: 5k ~ 8k -1
func (c *RandomBeaconContract) isValidEpoch(epochId uint64) (bool) {
	//Cfg
	// evm
	return true
}

func (c *RandomBeaconContract) isInRandomGroup(pks *[]bn256.G1, proposerId uint32) (bool) {
	if len(*pks) <= int(proposerId) {
		return false
	}
	return true
}

func buildError(err string, epochId uint64, proposerId uint32) (error) {
	return errors.New(fmt.Sprintf("%v epochId = %v, proposerId = %v ", err, epochId, proposerId))
	//return errors.New(err + ". epochId " + strconv.FormatUint(epochId, 10) + ", proposerId " + strconv.FormatUint(uint64(proposerId), 10))
}

func (c *RandomBeaconContract) dkg(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	// TODO: next line is just for test, and will be removed later
	functrace.Enter("dkg")
	var payloadHex string
	err := rbscAbi.Unpack(&payloadHex, "dkg", payload)
	if err != nil {
		return nil, errors.New("error in dkg abi parse ")
	}

	payloadBytes := common.FromHex(payloadHex)

	var dkgParam RbDKGTxPayload
	err = rlp.DecodeBytes(payloadBytes, &dkgParam)
	if err != nil {
		return nil, errors.New("error in dkg param has a wrong struct")
	}
	
	pks := GetRBProposerGroup(dkgParam.EpochId)
	// TODO: check
	// 1. EpochId: weather in a wrong time
	if !c.isValidEpoch(dkgParam.EpochId) {
		return nil, errors.New(" error epochId " + strconv.FormatUint(dkgParam.EpochId, 10))
	}
	// 2. ProposerId: weather in the random commit
	if !c.isInRandomGroup(&pks, dkgParam.ProposerId) {
		return nil, errors.New(" error proposerId " + strconv.FormatUint(uint64(dkgParam.ProposerId), 10))
	}

	// 3. Enshare, Commit, Proof has the same size
	// check same size
	nr := len(dkgParam.Proof)
	thres := nr / 2
	if nr != len(dkgParam.Enshare) || nr != len(dkgParam. Commit) {
		return nil, buildError("error in dkg params have different length", dkgParam.EpochId, dkgParam.ProposerId)
	}

	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(crypto.Keccak256(pks[i].Marshal()))
		x[i].Mod(&x[i], bn256.Order)
	}

	// get send public Key
	pubkey := GetProposerPubkey(&pks, dkgParam.ProposerId)
	// 4. proof verification
	for j := 0; j < nr; j++ {
		if !wanpos.VerifyDLEQ(dkgParam.Proof[j], *pubkey, *hbase, *dkgParam.Enshare[j], *dkgParam.Commit[j]) {
			return nil, buildError("dkg verify dleq error", dkgParam.EpochId, dkgParam.ProposerId)
		}
	}
	temp := make([]bn256.G2, nr)
	// 5. Reed-Solomon code verification
	for j := 0; j < nr; j++ {
		temp[j] = *dkgParam.Commit[j]
	}
	if !wanpos.RScodeVerify(temp, x, thres - 1) {
		return nil, buildError("rscode check error", dkgParam.EpochId, dkgParam.ProposerId)
	}

	// save epochId*2^64 + proposerId
	hash := GetRBKeyHash(dkgId[:], dkgParam.EpochId, dkgParam.ProposerId)
	// TODO: maybe we can use tx hash to replace payloadBytes, a tx saved in a chain block
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, payloadBytes)
	// TODO: add an dkg event
	// add event

	return nil, nil
}

func (c *RandomBeaconContract) sigshare(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var payloadHex string
	err := rbscAbi.Unpack(&payloadHex, "sigshare", payload)
	if err != nil {
		return nil, errors.New("error in sigshare abi parse")
	}

	payloadBytes := common.FromHex(payloadHex)

	var sigshareParam RbSIGTxPayload
	err = rlp.DecodeBytes(payloadBytes, &sigshareParam)
	if err != nil {
		return nil, errors.New("error in dkg param has a wrong struct")
	}

	pks := GetRBProposerGroup(sigshareParam.EpochId)
	// TODO: check
	// 1. EpochId: weather in a wrong time
	if !c.isValidEpoch(sigshareParam.EpochId) {
		return nil, errors.New(" error epochId " + strconv.FormatUint(sigshareParam.EpochId, 10))
	}
	// 2. ProposerId: weather in the random commit
	if !c.isInRandomGroup(&pks, sigshareParam.ProposerId) {
		return nil, errors.New(" error proposerId " + strconv.FormatUint(uint64(sigshareParam.ProposerId), 10))
	}
	// TODO: check weather dkg stage has been finished

	// 3. Verification
	M, err := GetRBM(sigshareParam.EpochId)
	if err != nil {
		return nil, buildError("getRBM error", sigshareParam.EpochId, sigshareParam.ProposerId)
	}
	m := new(big.Int).SetBytes(M)

	cj0, err := c.getCji(evm, sigshareParam.EpochId, 0)
	if err != nil {
		return nil, buildError(" can't get cj0 ", sigshareParam.EpochId, sigshareParam.ProposerId)
	}
	nr := len(cj0)
	var gpkshare bn256.G2
	gpkshare.Add(&gpkshare, cj0[sigshareParam.ProposerId])
	for i := 1; i < nr; i++ {
		cji, err := c.getCji(evm, sigshareParam.EpochId, 0)
		if err != nil {
			return nil, buildError(" can't get cji ", sigshareParam.EpochId, sigshareParam.ProposerId)
		}
		gpkshare.Add(&gpkshare, cji[sigshareParam.ProposerId])
	}

	mG := new(bn256.G1).ScalarBaseMult(m)
	pair1 := bn256.Pair(sigshareParam.Gsigshare, hbase)
	pair2 := bn256.Pair(mG, &gpkshare)
	if pair1.String() != pair2.String() {
		return nil, buildError(" unequal sigi", sigshareParam.EpochId, sigshareParam.ProposerId)
	}

	// save
	hash := GetRBKeyHash(sigshareId[:], sigshareParam.EpochId, sigshareParam.ProposerId)
	// TODO: maybe we can use tx hash to replace payloadBytes, a tx saved in a chain block
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, payloadBytes)
	// TODO: add an dkg event
	return nil, nil
}

func (c *RandomBeaconContract) getCji(evm *EVM, epochId uint64, proposerId uint32) ([]*bn256.G2, error) {
	keyBytes := make([]byte, 16)
	keyBytes = append(UIntToByteSlice(epochId), UIntToByteSlice(uint64(proposerId)) ...)
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	dkgBytes := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, hash)

	var dkgParam RbDKGTxPayload
	err := rlp.DecodeBytes(dkgBytes, &dkgParam)
	if err != nil {
		return nil, buildError("error in sigshare, decode dkg rlp error", epochId, proposerId)
	}
	return dkgParam.Commit, nil
}