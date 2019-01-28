package vm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/pos/postools"

	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
	"github.com/wanchain/go-wanchain/pos/posdb"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/pos/wanpos_crypto"
)

const (
	_ int = iota
	RB_DKG_STAGE
	RB_DKG_CONFIRM_STAGE
	RB_SIGN_STAGE
	RB_AFTER_SIGH_STAGE
)

var (
	rbscDefinition       = `[{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"epochId","type":"uint256"},{"name":"r","type":"uint256"}],"name":"genR","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"sigshare","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	rbscAbi, errRbscInit = abi.JSON(strings.NewReader(rbscDefinition))

	dkgId      [4]byte
	sigshareId [4]byte
	genRId     [4]byte

	kind_cij   = []byte{100}
	kind_ens   = []byte{200}
	// Generator of G1
	//gbase = new(bn256.G1).ScalarBaseMult(big.NewInt(int64(1)))
	// Generator of G2
	hbase = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))

	dkgBeginId = 0
	dkgEndId = pos.Cfg().DkgEnd
	signBeginId = pos.Cfg().SignBegin
	signEndId = pos.Cfg().SignEnd

	errDkgParse  = errors.New("dkg payload parse failed")
	errSigParse  = errors.New("sig payload parse failed")
	errDkgUnpack = errors.New("dkg param unpack failed")
	errRScode = errors.New("rscode verify failed")
	errDleq = errors.New("dleq verify failed")
	errRlpCij = errors.New("rlp encode cij failed")
	errRlpEns = errors.New("rlp encode ens failed")
	errUnRlpCij = errors.New("rlp decode cij failed")
	errUnRlpEns = errors.New("rlp decode ens failed")
)

func GetRBStage(slotId uint64) int {
	if slotId < dkgEndId {
		return RB_DKG_STAGE
	} else if slotId < signBeginId {
		return RB_DKG_CONFIRM_STAGE
	} else if slotId < signEndId {
		return RB_SIGN_STAGE
	} else {
		return RB_AFTER_SIGH_STAGE
	}
}

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

func GetDkgId() []byte {
	return dkgId[:]
}
func GetSigshareId() []byte {
	return sigshareId[:]
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
	} else {
		log.Debug("No match id found")
	}

	return nil, nil
}

func (c *RandomBeaconContract) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	return nil
}

func GetRBKeyHash(kind []byte, epochId uint64, proposerId uint32) *common.Hash {
	keyBytes := make([]byte, 12 + len(kind))
	copy(keyBytes, kind)
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
		log.Warn("***Can not found random r just use epoch 0 R", "epochID", epochId)
		r = GetStateR(db, 0)
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

func GetSig(db StateDB, epochId uint64, proposerId uint32) (*RbSIGTxPayload, error) {
	hash := GetRBKeyHash(sigshareId[:], epochId, proposerId)
	log.Debug("vm.GetSig", "len(sigshareId)", len(sigshareId), "epochID", epochId, "proposerId", proposerId, "hash", hash.Hex())
	payloadBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	// if missing
	if len(payloadBytes) == 0 {
		return nil, nil
	}

	var sigParam RbSIGTxPayload
	err := rlp.DecodeBytes(payloadBytes, &sigParam)
	// if in wrong format
	if err != nil {
		return nil, errSigParse
	}

	return &sigParam, nil
}

func GetRBM(db StateDB, epochId uint64) ([]byte, error) {
	epochIdBigInt := big.NewInt(int64(epochId + 1))
	preRandom := GetR(db, epochId)

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

func ByteSliceToUInt(bs []byte) uint64 {
	return binary.LittleEndian.Uint64(bs)
}
func ByteSliceToUInt32(bs []byte) uint32 {
	return binary.LittleEndian.Uint32(bs)
}

type RbDKGTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Enshare    []*bn256.G1
	Commit     []*bn256.G2
	Proof      []wanpos.DLEQproof
}
type RbDKGTxPayload1 struct {
	EpochId    uint64
	ProposerId uint32
	// 64
	Enshare    [][]byte
	// 128 --
	Commit     [][]byte
	Proof      []wanpos.DLEQproof1
}

type RbSIGTxPayload struct {
	EpochId    uint64
	ProposerId uint32
	Gsigshare  *bn256.G1
}

type RbDKGTxPayloadPure struct {
	EpochId    uint64
	ProposerId uint32
	Enshare    []*bn256.G1
	Commit     []*bn256.G2
}
func DkgToDkg1(d *RbDKGTxPayload) *RbDKGTxPayload1 {
	var dkgParam RbDKGTxPayload1
	dkgParam.EpochId = d.EpochId
	dkgParam.ProposerId = d.ProposerId

	l := len(d.Commit)
	dkgParam.Commit = make([][]byte, l, l)
	for i := 0; i<l; i++ {
		dkgParam.Commit[i] = d.Commit[i].Marshal()
	}

	l = len(d.Enshare)
	dkgParam.Enshare = make([][]byte, l, l)
	for i := 0; i<l; i++ {
		dkgParam.Enshare[i] = d.Enshare[i].Marshal()
	}

	l = len(d.Proof)
	dkgParam.Proof = make([]wanpos.DLEQproof1, l, l)
	for i := 0; i<l; i++ {
		dkgParam.Proof[i] = wanpos.ProofToProof1(&d.Proof[i])
	}

	return &dkgParam
}

func BytesToCij(d *[][]byte) []*bn256.G2 {
	l := len(*d)
	cijs := make([]*bn256.G2, l, l)
	g2s := make([]bn256.G2, l, l)
	for i := 0; i<l; i++ {
		//var g2 bn256.G2
		g2s[i].UnmarshalPure((*d)[i])
		cijs[i] = &g2s[i]
	}

	return cijs
}

func BytesToEns(d *[][]byte) []*bn256.G1 {
	l := len(*d)
	enss := make([]*bn256.G1, l, l)
	g1s := make([]bn256.G1, l, l)
	for i := 0; i<l; i++ {
		g1s[i].UnmarshalPure((*d)[i])
		enss[i] = &g1s[i]
	}

	return enss
}

func Dkg1ToDkg(d * RbDKGTxPayload1) *RbDKGTxPayload {
	var dkgParam RbDKGTxPayload
	dkgParam.EpochId = d.EpochId
	dkgParam.ProposerId = d.ProposerId

	l := len(d.Commit)
	dkgParam.Commit = make([]*bn256.G2, l, l)
	g2s := make([]bn256.G2, l, l)
	for i := 0; i<l; i++ {
		//var g2 bn256.G2
		g2s[i].Unmarshal(d.Commit[i])
		dkgParam.Commit[i] = &g2s[i]
	}

	l = len(d.Enshare)
	dkgParam.Enshare = make([]*bn256.G1, l, l)
	g1s := make([]bn256.G1, l, l)
	for i := 0; i<l; i++ {
		g1s[i].Unmarshal(d.Enshare[i])
		dkgParam.Enshare[i] = &g1s[i]
	}

	l = len(d.Proof)
	dkgParam.Proof = make([]wanpos.DLEQproof, l, l)
	for i := 0; i<l; i++ {
		//dkgParam.Proof[i] = wanpos.Proof1ToProof(&d.Proof[i])
		(&dkgParam.Proof[i]).Proof1ToProof(&d.Proof[i])
	}
	return &dkgParam
}

// TODO: evm.EpochId evm.SlotId, Cfg.K---dkg:0 ~ 4k -1, sig: 5k ~ 8k -1
// stage 0, 1 dkg sign
func isValidEpochStage(epochId uint64, stage int, evm *EVM) bool {
	eid, sid := postools.CalEpochSlotID(evm.Time.Uint64())
	if epochId != eid {
		return false
	}
	ss := GetRBStage(sid)
	if ss != stage {
		return false
	}
	return true
}

func isInRandomGroup(pks *[]bn256.G1, proposerId uint32) bool {
	if len(*pks) <= int(proposerId) {
		return false
	}
	return true
}

var getRBProposerGroupVar func(epochId uint64) []bn256.G1 = posdb.GetRBProposerGroup
var getRBMVar func(db StateDB, epochId uint64) ([]byte, error) = GetRBM
var isValidEpochStageVar func(epochId uint64, stage int, evm *EVM) bool = isValidEpochStage
var isInRandomGroupVar func(pks *[]bn256.G1, proposerId uint32) bool = isInRandomGroup

func buildError(err string, epochId uint64, proposerId uint32) error {
	return errors.New(fmt.Sprintf("%v epochId = %v, proposerId = %v ", err, epochId, proposerId))
	//return errors.New(err + ". epochId " + strconv.FormatUint(epochId, 10) + ", proposerId " + strconv.FormatUint(uint64(proposerId), 10))
}

func logError(err error) error {
	log.Error(err.Error())
	return err
}

func GetPolynomialX(pk *bn256.G1, proposerId uint32) []byte {
	return crypto.Keccak256(pk.Marshal(), big.NewInt(int64(proposerId)).Bytes())
}

func GetCji(db StateDB, epochId uint64, proposerId uint32) ([]*bn256.G2, error) {
	hash := GetRBKeyHash(kind_cij, epochId, proposerId)
	dkgBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(dkgBytes) == 0 {
		return nil, nil
	}
	cijs := make([][]byte, 0)
	err := rlp.DecodeBytes(dkgBytes, &cijs)
	if err != nil {
		return nil, errUnRlpCij
	}

	rt := BytesToCij(&cijs)

	return rt, nil
}

func GetEns(db StateDB, epochId uint64, proposerId uint32) ([]*bn256.G1, error) {
	hash := GetRBKeyHash(kind_ens, epochId, proposerId)
	dkgBytes := db.GetStateByteArray(randomBeaconPrecompileAddr, *hash)
	if len(dkgBytes) == 0 {
		return nil, nil
	}
	enss := make([][]byte, 0)
	err := rlp.DecodeBytes(dkgBytes, &enss)
	if err != nil {
		return nil, errUnRlpEns
	}

	rt := BytesToEns(&enss)

	return rt, nil
}

func (c *RandomBeaconContract) dkg(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	var dkgParam1 RbDKGTxPayload1
	t1 := time.Now()
	err := rlp.DecodeBytes(payload, &dkgParam1)
	elapsed2 := time.Since(t1)
	fmt.Println("***dkg1: ", elapsed2)
	dkgParam := Dkg1ToDkg(&dkgParam1)
	elapsed3 := time.Since(t1)
	fmt.Println("***dkg2: ", elapsed3)
	if err != nil {
		return nil, logError(errDkgParse)
	}
	eid := dkgParam.EpochId
	pid := dkgParam.ProposerId
	log.Debug("contract do dkg begin", "epochId", eid, "proposerId", pid)

	pks := getRBProposerGroupVar(eid)
	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RB_DKG_STAGE, evm) {
		return nil, logError(errors.New(" error epochId " + strconv.FormatUint(eid, 10)))
	}
	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(&pks, pid) {
		return nil, logError(errors.New(" error proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// 3. Enshare, Commit, Proof has the same size
	// check same size
	nr := len(dkgParam.Proof)
	if nr != len(dkgParam.Enshare) || nr != len(dkgParam.Commit) {
		return nil, logError(buildError("error in dkg params have different length", eid, pid))
	}

	// 4. proof verification
	for j := 0; j < nr; j++ {
		// get send public Key
		if !wanpos.VerifyDLEQ(dkgParam.Proof[j], pks[j], *hbase, *dkgParam.Enshare[j], *dkgParam.Commit[j]) {
			return nil, logError(errDleq)
		}
	}

	// 5. Reed-Solomon code verification
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(GetPolynomialX(&pks[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}
	temp := make([]bn256.G2, nr)
	for j := 0; j < nr; j++ {
		temp[j] = *dkgParam.Commit[j]
	}
	if !wanpos.RScodeVerify(temp, x, int(pos.Cfg().PolymDegree)) {
		return nil, logError(errRScode)
	}

	// save cij
	hash := GetRBKeyHash(kind_cij, eid, pid)
	cijBytes, err := rlp.EncodeToBytes(dkgParam1.Commit)
	if err != nil {
		return nil, logError(errRlpCij)
	}
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, cijBytes)

	// save ens
	hash = GetRBKeyHash(kind_ens, eid, pid)
	ensBytes, err := rlp.EncodeToBytes(dkgParam1.Enshare)
	if err != nil {
		return nil, logError(errRlpEns)
	}
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, ensBytes)
	// TODO: add an dkg event
	// add event
	log.Debug("vm.dkg", "len(dkgId)", len(dkgId), "epochID", eid, "proposerId", pid, "hash", hash.Hex())
	return nil, nil
}

func getSigsNum(epochId uint64, evm *EVM) uint32 {
	tmpKey := common.Hash{0}
	bytes := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, tmpKey)
	if bytes != nil {
		eid := ByteSliceToUInt(bytes)
		if eid == epochId {
			num := ByteSliceToUInt32(bytes[8:12])
			return num
		}
	}
	return 0
}

func setSigsNum(epochId uint64, num uint32, evm *EVM) {
	tmpKey := common.Hash{0}
	dataBytes := make([]byte, 12)
	copy(dataBytes[0:], UIntToByteSlice(epochId))
	copy(dataBytes[8:], UInt32ToByteSlice(num))
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, tmpKey, dataBytes)
}

func (c *RandomBeaconContract) sigshare(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
	t1 := time.Now()
	var sigshareParam RbSIGTxPayload
	err := rlp.DecodeBytes(payload, &sigshareParam)
	if err != nil {
		return nil, logError(errors.New("error in dkg param has a wrong struct"))
	}
	eid := sigshareParam.EpochId
	pid := sigshareParam.ProposerId
	log.Info("contract do sig begin", "epochId", eid, "proposerId", pid)

	pks := getRBProposerGroupVar(eid)
	// 1. EpochId: weather in a wrong time
	if !isValidEpochStageVar(eid, RB_SIGN_STAGE, evm) {
		return nil, logError(errors.New(" error epochId " + strconv.FormatUint(eid, 10)))
	}
	// 2. ProposerId: weather in the random commit
	if !isInRandomGroupVar(&pks, pid) {
		return nil, logError(errors.New(" error proposerId " + strconv.FormatUint(uint64(pid), 10)))
	}

	// 3. Verification
	M, err := getRBMVar(evm.StateDB, eid)
	if err != nil {
		return nil, logError(buildError("getRBM error", eid, pid))
	}
	m := new(big.Int).SetBytes(M)

	var gpkshare bn256.G2

	dkgDatas := make([]RbCijDataCollector, 0)
	for id, _ := range pks {
		dkgData, err := GetCji(evm.StateDB, eid, uint32(id))
		if err == nil && dkgData != nil {
			dkgDatas = append(dkgDatas, RbCijDataCollector{dkgData, &pks[id]})
			gpkshare.Add(&gpkshare, dkgData[pid])
		}
	}
	if uint(len(dkgDatas)) < pos.Cfg().RBThres {
		//return nil, logError(buildError(" insufficient proposer ", eid, pid))
		logError(buildError(" insufficient proposer ", eid, pid))
		return nil, nil
	}

	mG := new(bn256.G1).ScalarBaseMult(m)
	pair1 := bn256.Pair(sigshareParam.Gsigshare, hbase)
	pair2 := bn256.Pair(mG, &gpkshare)
	if pair1.String() != pair2.String() {
		return nil, logError(buildError(" unequal sigi", eid, pid))
	}

	// save
	hash := GetRBKeyHash(sigshareId[:], eid, pid)
	// TODO: maybe we can use tx hash to replace payloadBytes, a tx saved in a chain block
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hash, payload)

	/////////////////
	// calc r if not exist
	signum := getSigsNum(eid, evm) + 1
	setSigsNum(eid, signum, evm)
	if uint(signum) >= pos.Cfg().RBThres {
		r, err := computeRandom(evm.StateDB, eid, dkgDatas, pks)
		if r != nil && err == nil {
			hashR := GetRBRKeyHash(eid + 1)
			evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, *hashR, r.Bytes())
			log.Info("generate", "r", r, "epochid", eid+1)
		}
	}

	// TODO: add an dkg event
	log.Info("contract do sig end", "epochId", eid, "proposerId", pid)

	elapsed := time.Since(t1)
	fmt.Println("***App elapsed: ", elapsed)
	return nil, nil
}

type RbCijDataCollector struct {
	cij []*bn256.G2
	pk   *bn256.G1
}

type RbSIGDataCollector struct {
	data *RbSIGTxPayload
	pk   *bn256.G1
}

// compute random[epochid+1] by data of epoch[epochid]
func computeRandom(statedb StateDB, epochId uint64, dkgDatas []RbCijDataCollector, pks []bn256.G1) (*big.Int, error) {
	log.Info("do compute random", "epochId", epochId)
	randomInt := GetStateR(statedb, epochId+1)
	if randomInt != nil && randomInt.Cmp(big.NewInt(0)) != 0 {
		// exist already
		log.Info("random exist already", "epochId", epochId+1, "random", randomInt.String())
		return randomInt, errors.New("random exist already")
	}

	if len(pks) == 0 {
		log.Error("can't find random beacon proposer group")
		return nil, errors.New("can't find random beacon proposer group")
	}

	// collact DKG SIG
	sigDatas := make([]RbSIGDataCollector, 0)
	for id, _ := range pks {
		sigData, err := GetSig(statedb, epochId, uint32(id))
		if err == nil && sigData != nil {
			sigDatas = append(sigDatas, RbSIGDataCollector{sigData, &pks[id]})
		}
	}

	log.Info("dkgDatas and sigDatas length", "len(dkgDatas)", len(dkgDatas), "len(sigDatas)", len(sigDatas))
	if uint(len(sigDatas)) < pos.Cfg().RBThres {
		log.Warn("compute random fail, insufficient proposer", "epochId", epochId, "min", pos.Cfg().RBThres, "acture", len(sigDatas))
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
			c[i].Add(&c[i], dkgDatas[j].cij[i])
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
