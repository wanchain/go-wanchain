package vm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	bn256 "github.com/wanchain/pos/cloudflare"
	wanpos "github.com/wanchain/pos/wanpos_crypto"
)

var (
	rbscDefinition       = `[{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"epochId","type":"uint256"},{"name":"r","type":"uint256"}],"name":"genR","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"sigshare","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	rbscAbi, errRbscInit = abi.JSON(strings.NewReader(rbscDefinition))

	dkgId      [4]byte
	sigshareId [4]byte
	genRId     [4]byte
	// Generator of G1
	//gbase = new(bn256.G1).ScalarBaseMult(big.NewInt(int64(1)))
	// Generator of G2
	hbase = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))
)

type RandomBeaconContract struct {
}

func init() {
	if errRbscInit != nil {
		panic("err in rbsc abi initialize")
	}

	copy(dkgId[:], rbscAbi.Methods["dkg"].Id())
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

	log.Debug("RandomBeaconContract is called", "inputLen", len(input), "methodId", methodId, "dkgId", dkgId, "sigshareId", sigshareId, "genRId", genRId)

	if methodId == dkgId {
		return c.dkg(input[4:], contract, evm)
	} else if methodId == sigshareId {
		return c.sigshare(input[4:], contract, evm)
	} else if methodId == genRId {
		return c.genR(input[4:], contract, evm)
	} else {
		log.Error("No match id found")
	}

	return nil, nil
}

func (c *RandomBeaconContract) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

func GetRBKeyHash(funId []byte, epochId uint64, proposerId uint32) *common.Hash {
	keyBytes := make([]byte, 16)
	copy(keyBytes, funId)
	copy(keyBytes[4:], UIntToByteSlice(epochId))
	copy(keyBytes[12:], UInt32ToByteSlice(proposerId))
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	return &hash
}

func GetRBRKeyHash(epochId uint64) *common.Hash {
	keyBytes := make([]byte, 12)
	copy(keyBytes, genRId[:])
	copy(keyBytes[4:], UIntToByteSlice(epochId))
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	return &hash
}

func GetR(db StateDB, epochId uint64) *big.Int {
	r := GetStateR(db, epochId)
	if r == nil {
		i := uint64(1)
		for ; r == nil; i++ {
			r = GetStateR(db, epochId - i)
		}
		log.Warn("***fake random r", "epochId", epochId, "i", epochId - i + 1, "r", r)
		bytes := r.Bytes()
		bytes = append(bytes, UIntToByteSlice(epochId) ...)
		random := crypto.Keccak256(bytes)
		r = new(big.Int).SetBytes(random)
	}
	return r
}
// get r in statedb
func GetStateR(db StateDB, epochId uint64) *big.Int {
	if epochId == 0 {
		return new(big.Int).SetBytes(crypto.Keccak256(big.NewInt(1).Bytes()))
	}
	hash := GetRBRKeyHash(epochId)
	rBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(rBytes) != 0 {
		r := new(big.Int).SetBytes(rBytes)
		return r
	}
	return nil
}

func GetDkg(db StateDB, epochId uint64, proposerId uint32) (*RbDKGTxPayload, error) {
	hash := GetRBKeyHash(dkgId[:], epochId, proposerId)
	log.Debug("vm.GetDkg", "len(dkgId)", len(dkgId), "epochID", epochId, "proposerId", proposerId, "hash", hash.Hex())
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	var dkgParam RbDKGTxPayload
	err := rlp.DecodeBytes(payloadBytes, &dkgParam)
	if err != nil {
		return nil, buildError("load dkg error", epochId, proposerId)
	}

	return &dkgParam, nil
}

func GetSig(db StateDB, epochId uint64, proposerId uint32) (*RbSIGTxPayload, error) {
	hash := GetRBKeyHash(sigshareId[:], epochId, proposerId)
	log.Debug("vm.GetSig", "len(sigshareId)", len(sigshareId), "epochID", epochId, "proposerId", proposerId, "hash", hash.Hex())
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	var sigParam RbSIGTxPayload
	err := rlp.DecodeBytes(payloadBytes, &sigParam)
	if err != nil {
		return nil, buildError("load sig error", epochId, proposerId)
	}

	return &sigParam, nil
}

func GetRBM(db StateDB, epochId uint64) ([]byte, error) {
	epochIdBigInt := big.NewInt(int64(epochId + 1))
	preRandom := GetR(db, epochId)
	if preRandom == nil {
		return nil, errors.New("Get RBM error")
	}

	buf := epochIdBigInt.Bytes()
	buf = append(buf, preRandom.Bytes()...)
	return crypto.Keccak256(buf), nil
}

func GetRBAbiDefinition() string {
	return rbscDefinition
}

func GetRBAddress() common.Address {
	return randomBeaconPrecompileAddr
}

func getRBProposerGroup(epochId uint64) []bn256.G1 {
	db := posdb.GetDbByName("rblocaldb")
	if db == nil {
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

var getRBProposerGroupVar func(epochId uint64) []bn256.G1 = posdb.GetRBProposerGroup
var getRBMVar func(db StateDB, epochId uint64) ([]byte, error) = GetRBM

func UIntToByteSlice(num uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, num)
	return b
}
func UInt32ToByteSlice(num uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, num)
	return b
}

type RbDKGTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Enshare    []*bn256.G1
	Commit     []*bn256.G2
	Proof      []wanpos.DLEQproof
}

type RbSIGTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Gsigshare  *bn256.G1
}

// TODO: evm.EpochId evm.SlotId, Cfg.K---dkg:0 ~ 4k -1, sig: 5k ~ 8k -1
func (c *RandomBeaconContract) isValidEpoch(epochId uint64) bool {
	//Cfg
	// evm
	return true
}

func (c *RandomBeaconContract) isInRandomGroup(pks *[]bn256.G1, proposerId uint32) bool {
	if len(*pks) <= int(proposerId) {
		return false
	}
	return true
}

func buildError(err string, epochId uint64, proposerId uint32) error {
	return errors.New(fmt.Sprintf("%v epochId = %v, proposerId = %v ", err, epochId, proposerId))
	//return errors.New(err + ". epochId " + strconv.FormatUint(epochId, 10) + ", proposerId " + strconv.FormatUint(uint64(proposerId), 10))
}

func GetPolynomialX(pk *bn256.G1, proposerId uint32) []byte {
	return crypto.Keccak256(pk.Marshal(), big.NewInt(int64(proposerId)).Bytes())
}

func (c *RandomBeaconContract) getCji(evm *EVM, epochId uint64, proposerId uint32) ([]*bn256.G2, error) {
	hash := GetRBKeyHash(dkgId[:], epochId, proposerId)
	dkgBytes := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	var dkgParam RbDKGTxPayload
	err := rlp.DecodeBytes(dkgBytes, &dkgParam)
	if err != nil {
		log.Debug("rlp decode dkg fail", "err", err)
		return nil, buildError("error in sigshare, decode dkg rlp error", epochId, proposerId)
	}

	log.Info("getCji success")
	return dkgParam.Commit, nil
}

func (c *RandomBeaconContract) dkg(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	// TODO: next line is just for test, and will be removed later
	functrace.Enter("dkg")
	var payloadHex string
	err := rbscAbi.UnpackInput(&payloadHex, "dkg", payload)
	if err != nil {
		return nil, errors.New("error in dkg abi parse ")
	}

	payloadBytes := common.FromHex(payloadHex)

	var dkgParam RbDKGTxPayload
	err = rlp.DecodeBytes(payloadBytes, &dkgParam)
	if err != nil {
		return nil, errors.New("error in dkg param has a wrong struct")
	}
	log.Info("contract do dkg begin", "epochId", dkgParam.EpochId, "proposerId", dkgParam.ProposerId)

	pks := getRBProposerGroupVar(dkgParam.EpochId)
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
	thres := pos.Cfg().PolymDegree + 1
	if nr != len(dkgParam.Enshare) || nr != len(dkgParam.Commit) {
		return nil, buildError("error in dkg params have different length", dkgParam.EpochId, dkgParam.ProposerId)
	}

	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(GetPolynomialX(&pks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	// 4. proof verification
	for j := 0; j < nr; j++ {
		// get send public Key
		if !wanpos.VerifyDLEQ(dkgParam.Proof[j], pks[j], *hbase, *dkgParam.Enshare[j], *dkgParam.Commit[j]) {
			return nil, buildError("dkg verify dleq error", dkgParam.EpochId, dkgParam.ProposerId)
		}
	}
	temp := make([]bn256.G2, nr)
	// 5. Reed-Solomon code verification
	for j := 0; j < nr; j++ {
		temp[j] = *dkgParam.Commit[j]
	}
	if !wanpos.RScodeVerify(temp, x, int(thres-1)) {
		return nil, buildError("rscode check error", dkgParam.EpochId, dkgParam.ProposerId)
	}

	// save epochId*2^64 + proposerId
	hash := GetRBKeyHash(dkgId[:], dkgParam.EpochId, dkgParam.ProposerId)
	log.Debug("vm.dkg", "len(dkgId)", len(dkgId), "epochID", dkgParam.EpochId, "proposerId", dkgParam.ProposerId, "hash", hash.Hex())
	// TODO: maybe we can use tx hash to replace payloadBytes, a tx saved in a chain block
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, payloadBytes)
	// TODO: add an dkg event
	// add event

	log.Info("contract do dkg end", "epochId", dkgParam.EpochId, "proposerId", dkgParam.ProposerId)
	return nil, nil
}

func (c *RandomBeaconContract) sigshare(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var payloadHex string
	err := rbscAbi.UnpackInput(&payloadHex, "sigshare", payload)
	if err != nil {
		return nil, errors.New("error in sigshare abi parse")
	}

	payloadBytes := common.FromHex(payloadHex)

	var sigshareParam RbSIGTxPayload
	err = rlp.DecodeBytes(payloadBytes, &sigshareParam)
	if err != nil {
		return nil, errors.New("error in dkg param has a wrong struct")
	}

	log.Info("contract do sig begin", "epochId", sigshareParam.EpochId, "proposerId", sigshareParam.ProposerId)
	pks := getRBProposerGroupVar(sigshareParam.EpochId)
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
	M, err := getRBMVar(evm.StateDB, sigshareParam.EpochId)
	if err != nil {
		return nil, buildError("getRBM error", sigshareParam.EpochId, sigshareParam.ProposerId)
	}
	m := new(big.Int).SetBytes(M)

	var gpkshare bn256.G2

	j := uint(0)
	for i := 0; i < len(pks); i++ {
		ci, _ := c.getCji(evm, sigshareParam.EpochId, uint32(i))
		if ci == nil {
			continue
		}
		j++
		gpkshare.Add(&gpkshare, ci[sigshareParam.ProposerId])
	}
	if j < pos.Cfg().MinRBProposerCnt {
		return nil, buildError(" insufficient proposer ", sigshareParam.EpochId, sigshareParam.ProposerId)
	}

	mG := new(bn256.G1).ScalarBaseMult(m)
	pair1 := bn256.Pair(sigshareParam.Gsigshare, hbase)
	pair2 := bn256.Pair(mG, &gpkshare)
	if pair1.String() != pair2.String() {
		return nil, buildError(" unequal sigi", sigshareParam.EpochId, sigshareParam.ProposerId)
	}

	// save
	hash := GetRBKeyHash(sigshareId[:], sigshareParam.EpochId, sigshareParam.ProposerId)
	log.Debug("vm.sigshare", "len(sigshareId)", len(sigshareId), "epochID", sigshareParam.EpochId, "proposerId", sigshareParam.ProposerId, "hash", hash.Hex())
	// TODO: maybe we can use tx hash to replace payloadBytes, a tx saved in a chain block
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, payloadBytes)

	/////////////////
	// calc r if not exist
	r, err := computeRandom(evm.StateDB, sigshareParam.EpochId)
	if r != nil && err == nil {
		hashR := GetRBRKeyHash(sigshareParam.EpochId + 1)
		evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hashR, r.Bytes())
		log.Info("generate","r", r, "epochid", sigshareParam.EpochId + 1)
	}

	// TODO: add an dkg event
	log.Info("contract do sig end", "epochId", sigshareParam.EpochId, "proposerId", sigshareParam.ProposerId)


	return nil, nil
}

// TODO: delete
func (c *RandomBeaconContract) genR(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var (
		epochId = big.NewInt(0)
		r       = big.NewInt(0)
	)
	out := []interface{}{&epochId, &r}
	err := rbscAbi.UnpackInput(&out, "genR", payload)
	if err != nil {
		return nil, errors.New("error in genR abi parse")
	}

	// save
	hash := GetRBRKeyHash(epochId.Uint64())
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, r.Bytes())
	log.Info("contract do genR end", "epochId=", epochId.Uint64())

	return nil, nil
}


type RbDKGDataCollector struct {
	data *RbDKGTxPayload
	pk   *bn256.G1
}

type RbSIGDataCollector struct {
	data *RbSIGTxPayload
	pk   *bn256.G1
}

// compute random[epochid+1] by data of epoch[epochid]
func computeRandom(statedb StateDB, epochId uint64) (*big.Int, error) {
	log.Info("do compute random", "epochId", epochId)
	randomInt := GetStateR(statedb, epochId+1)
	if randomInt != nil && randomInt.Cmp(big.NewInt(0)) != 0 {
		// exist already
		log.Info("random exist already", "epochId", epochId+1, "random", randomInt.String())
		return randomInt, errors.New("random exist already")
	}

	pks := getRBProposerGroupVar(epochId)
	if len(pks) == 0 {
		log.Error("can't find random beacon proposer group")
		return nil, errors.New("can't find random beacon proposer group")
	}

	// collact DKG SIG
	dkgDatas := make([]RbDKGDataCollector, 0)
	sigDatas := make([]RbSIGDataCollector, 0)
	for id, _ := range pks {
		dkgData, err := GetDkg(statedb, epochId, uint32(id))
		if err == nil && dkgData != nil {
			dkgDatas = append(dkgDatas, RbDKGDataCollector{dkgData, &pks[id]})
		}

		sigData, err := GetSig(statedb, epochId, uint32(id))
		if err == nil && sigData != nil {
			sigDatas = append(sigDatas, RbSIGDataCollector{sigData, &pks[id]})
		}
	}

	log.Info("dkgDatas and sigDatas length", "len(dkgDatas)", len(dkgDatas), "len(sigDatas)", len(sigDatas))
	if uint(len(sigDatas)) < pos.Cfg().MinRBProposerCnt {
		log.Warn("compute random fail, insufficient proposer", "epochId", epochId, "min", pos.Cfg().MinRBProposerCnt, "acture", len(sigDatas))
		return nil, errors.New("insufficient proposer")
	}

	gsigshare := make([]bn256.G1, len(sigDatas))
	xSig := make([]big.Int, len(sigDatas))
	for i, data := range sigDatas {
		gsigshare[i] = *data.data.Gsigshare
		xSig[i].SetBytes(GetPolynomialX(data.pk, data.data.ProposerId))
	}

	// Compute the Output of Random Beacon
	gsig := wanpos.LagrangeSig(gsigshare, xSig, int(pos.Cfg().PolymDegree))
	random := crypto.Keccak256(gsig.Marshal())
	log.Info("sig lagrange", "gsig", gsig, "gsigshare", gsigshare)

	// Verification Logic for the Output of Random Beacon
	// Computation of group public key
	nr := len(pks)
	c := make([]bn256.G2, nr)
	for i := 0; i < nr; i++ {
		c[i].ScalarBaseMult(big.NewInt(int64(0)))
		for j := 0; j < len(dkgDatas); j++ {
			c[i].Add(&c[i], dkgDatas[j].data.Commit[i])
		}
	}

	xAll := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		xAll[i].SetBytes(GetPolynomialX(&pks[i], uint32(i)))
		xAll[i].Mod(&xAll[i], bn256.Order)
	}
	gPub := wanpos.LagrangePub(c, xAll, int(pos.Cfg().PolymDegree))

	// mG
	mBuf, err := getRBMVar(statedb, epochId)
	if err != nil {
		log.Error("get m fail", "err", err)
		return nil, err
	}

	m := new(big.Int).SetBytes(mBuf)
	mG := new(bn256.G1).ScalarBaseMult(m)

	// Verify using pairing
	pair1 := bn256.Pair(&gsig, wanpos.Hbase)
	pair2 := bn256.Pair(mG, &gPub)
	log.Info("verify random", "pair1", pair1.String(), "pair2", pair2.String())
	if pair1.String() != pair2.String() {
		return nil, errors.New("final pairing check failed")
	}

	log.Info("compute random success", "epochId", epochId+1, "random", common.Bytes2Hex(random))
	return big.NewInt(0).SetBytes(random), nil
}