package vm

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/rbselection"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
	mrand "math/rand"
	"testing"
	"time"
)

type CTStateDB struct {
}

func (CTStateDB) CreateAccount(common.Address) {}

func (CTStateDB) SubBalance(common.Address, *big.Int) {}
func (CTStateDB) AddBalance(addr common.Address, pval *big.Int) {

}
func (CTStateDB) GetBalance(addr common.Address) *big.Int {
	defaulVal, _ := new(big.Int).SetString("10000000000000000000", 10)
	return defaulVal
}
func (CTStateDB) GetNonce(common.Address) uint64                                         { return 0 }
func (CTStateDB) SetNonce(common.Address, uint64)                                        {}
func (CTStateDB) GetCodeHash(common.Address) common.Hash                                 { return common.Hash{} }
func (CTStateDB) GetCode(common.Address) []byte                                          { return nil }
func (CTStateDB) SetCode(common.Address, []byte)                                         {}
func (CTStateDB) GetCodeSize(common.Address) int                                         { return 0 }
func (CTStateDB) AddRefund(*big.Int)                                                     {}
func (CTStateDB) GetRefund() *big.Int                                                    { return nil }
func (CTStateDB) GetState(common.Address, common.Hash) common.Hash                       { return common.Hash{} }
func (CTStateDB) SetState(common.Address, common.Hash, common.Hash)                      {}
func (CTStateDB) Suicide(common.Address) bool                                            { return false }
func (CTStateDB) HasSuicided(common.Address) bool                                        { return false }
func (CTStateDB) Exist(common.Address) bool                                              { return false }
func (CTStateDB) Empty(common.Address) bool                                              { return false }
func (CTStateDB) RevertToSnapshot(int)                                                   {}
func (CTStateDB) Snapshot() int                                                          { return 0 }
func (CTStateDB) AddLog(*types.Log)                                                      {}
func (CTStateDB) AddPreimage(common.Hash, []byte)                                        {}
func (CTStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool)     {}
func (CTStateDB) ForEachStorageByteArray(common.Address, func(common.Hash, []byte) bool) {}

var (
	rbepochId = uint64(0)
	rbdb = make(map[common.Hash][]byte)
	rbgroupdb = make(map[uint64][]bn256.G1)
	rbranddb = make(map[uint64]*big.Int)
)

func clearDB() {
	rbepochId = uint64(0)
	rbdb = make(map[common.Hash][]byte)
	rbgroupdb = make(map[uint64][]bn256.G1)
	rbranddb = make(map[uint64]*big.Int)
}

func (CTStateDB) GetStateByteArray(addr common.Address, hs common.Hash) []byte {
	return rbdb[hs]
}

func (CTStateDB) SetStateByteArray(addr common.Address, hs common.Hash, data []byte) {
	rbdb[hs] = data
}

type dummyCtRef struct {
	calledForEach bool
}

func (dummyCtRef) ReturnGas(*big.Int)          {}
func (dummyCtRef) Address() common.Address     { return common.Address{} }
func (dummyCtRef) Value() *big.Int             { return new(big.Int) }
func (dummyCtRef) SetCode(common.Hash, []byte) {}
func (d *dummyCtRef) ForEachStorage(callback func(key, value common.Hash) bool) {
	d.calledForEach = true
}
func (d *dummyCtRef) SubBalance(amount *big.Int) {}
func (d *dummyCtRef) AddBalance(amount *big.Int) {}
func (d *dummyCtRef) SetBalance(*big.Int)        {}
func (d *dummyCtRef) SetNonce(uint64)            {}
func (d *dummyCtRef) Balance() *big.Int          { return new(big.Int) }

type dummyCtDB struct {
	CTStateDB
	ref *dummyCtRef
}

var (
	nr = 21
	thres = posconfig.Cfg().PolymDegree + 1

	db, _      = ethdb.NewMemDatabase()
	statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	ref = &dummyCtRef{}
	evm = NewEVM(Context{Time:big.NewInt(time.Now().Unix())}, dummyCtDB{ref: ref}, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})


	rbcontract = &RandomBeaconContract{}
	rbcontractParam = &Contract{self:&dummyCtRef{false}}


	pubs, pris, hpubs = generateKeyPairs()
	_, _, enshareA, commitA, proofA = prepareDkg(pubs, hpubs)
)

// pubs,pris,hashPubs
func generateKeyPairs() ([]bn256.G1, []big.Int, []big.Int) {
	Pubkey := make([]bn256.G1, nr)
	Prikey := make([]big.Int, nr)

	for i := 0; i < nr; i++ {
		Pri, Pub, err := bn256.RandomG1(rand.Reader)
		if err != nil {
			println(err)
		}
		Prikey[i] = *Pri
		Pubkey[i] = *Pub
	}
	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(GetPolynomialX(&Pubkey[i], uint32(i)))
		x[i].Mod(&x[i], bn256.Order)
	}

	return Pubkey, Prikey, x
}

func prepareDkg(Pubkey []bn256.G1, x []big.Int) ([]*big.Int, [][]big.Int, [][]*bn256.G1, [][]*bn256.G2, [][]rbselection.DLEQproof) {
	// Each of random propoer generates a random si
	s := make([]*big.Int, nr)

	source := mrand.NewSource(int64(nr))
	r := mrand.New(source)

	for i := 0; i < nr; i++ {
		s[i], _ = rand.Int(r, bn256.Order)
	}

	// Each random propoer conducts the shamir secret sharing process
	poly := make([]rbselection.Polynomial, nr)

	sshare := make([][]big.Int, nr, nr)

	for i := 0; i < nr; i++ {
		sshare[i] = make([]big.Int, nr, nr)
		poly[i],_ = rbselection.RandPoly(int(thres-1), *s[i])	// fi(x), set si as its constant term
		for j := 0; j < nr; j++ {
			sshare[i][j], _ = rbselection.EvaluatePoly(poly[i], &x[j], int(thres-1)) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
		}
	}

	// Encrypt the secret share, i.e. mutiply with the receiver's public key
	enshare := make([][]*bn256.G1, nr, nr)
	for i := 0; i < nr; i++ {
		enshare[i] = make([]*bn256.G1, nr, nr)
		for j := 0; j < nr; j++ { // enshare[j] = sshare[j]*Pub[j], it is a point on ECC
			enshare[i][j] = new(bn256.G1).ScalarMult(&Pubkey[j], &sshare[i][j])
		}
	}

	// Make commitment for the secret share, i.e. multiply with the generator of G2
	commit := make([][]*bn256.G2, nr, nr)
	for i := 0; i < nr; i++ {
		commit[i] = make([]*bn256.G2, nr, nr)
		for j := 0; j < nr; j++ { // commit[j] = sshare[j] * G2
			commit[i][j] = new(bn256.G2).ScalarBaseMult(&sshare[i][j])
		}
	}

	// generate DLEQ proof
	proof := make([][]rbselection.DLEQproof, nr, nr)
	for i := 0; i < nr; i++ {
		proof[i] = make([]rbselection.DLEQproof, nr, nr)
		for j := 0; j < nr; j++ { // proof = (a1, a2, z)
			proof[i][j], _ = rbselection.DLEQ(Pubkey[j], *hBase, &sshare[i][j])
		}
	}

	return s, sshare, enshare, commit, proof
}

func prepareSig(Prikey []big.Int, enshare [][]*bn256.G1) ([]*bn256.G1)  {
	gskshare := make([]bn256.G1, nr)

	for i := 0; i < nr; i++ {

		gskshare[i].ScalarBaseMult(big.NewInt(int64(0))) //set zero

		skinver := new(big.Int).ModInverse(&Prikey[i], bn256.Order) // sk^-1

		for j := 0; j < nr; j++ {
			temp := new(bn256.G1).ScalarMult(enshare[j][i], skinver)
			gskshare[i].Add(&gskshare[i], temp) // gskshare[i] = (sk^-1)*(enshare[1][i]+...+enshare[Nr][i])
		}
	}

	M, err := getRBMVar(statedb, rbepochId)
	if err != nil {
		fmt.Printf("get rbm error id:%v\n", rbepochId)
	}
	m := new(big.Int).SetBytes(M)

	// Compute signature share
	gSigShare := make([]*bn256.G1, nr)
	for i := 0; i < nr; i++ { // signature share = M * secret key share
		gSigShare[i] = new(bn256.G1).ScalarMult(&gskshare[i], m)
	}
	return gSigShare
}

func getRBProposerGroupMock(epochId uint64) ([]bn256.G1,error){
	return rbgroupdb[epochId],nil
}


func getRBMMock(_ StateDB, epochId uint64) ([]byte, error) {
	nextEpochId := big.NewInt(int64(epochId + 1))

	preRandom, exisit := rbranddb[epochId]

	if preRandom == nil || !exisit {
		preRandom = new(big.Int).SetBytes(crypto.Keccak256(big.NewInt(1).Bytes()))
	}

	//buf := make([]byte, len(nextEpochId.Bytes()) + len(preRandom.Bytes()))
	buf := nextEpochId.Bytes()
	buf = append(buf, preRandom.Bytes()...)
	rt := crypto.Keccak256(buf)

	rbranddb[epochId + 1] = new(big.Int).SetBytes(rt)

	return rt, nil
}


func isValidEpochStageMock(_ uint64, _ int, _ uint64) bool {
	return true
}
func isInRandomGroupMock(_ []bn256.G1, _ uint64, _ uint32, _ common.Address) bool {
	return true
}

// test cases runs in testMain
func init() {
	rbranddb[0] = big.NewInt(1)
	getRBProposerGroupVar = getRBProposerGroupMock
	getRBMVar = getRBMMock
	isValidEpochStageVar = isValidEpochStageMock
	isInRandomGroupVar = isInRandomGroupMock
}

func buildDkg1(payloadBytes [] byte) []byte {
	payload := make([]byte, 4+len(payloadBytes))
	copy(payload, GetDkg1Id())
	copy(payload[4:], payloadBytes)
	return payload
}
func buildDkg2(payloadBytes [] byte) []byte {
	payload := make([]byte, 4+len(payloadBytes))
	copy(payload, GetDkg2Id())
	copy(payload[4:], payloadBytes)
	return payload
}
func buildSig(payloadBytes [] byte) []byte {
	payload := make([]byte, 4+len(payloadBytes))
	copy(payload, GetSigShareId())
	copy(payload[4:], payloadBytes)
	return payload
}

func TestRBDkg1(t *testing.T) {

	rbepochId = 0
	epochTimeSpan := uint64(posconfig.SlotTime * posconfig.SlotCount)*rbepochId
	evm.Time = big.NewInt(0).SetUint64(epochTimeSpan+posconfig.SlotTime*(posconfig.Cfg().Dkg1End-2))
	evm.BlockNumber = big.NewInt(0).SetUint64(uint64(100))

	rbgroupdb[rbepochId] = pubs

	for i := 0; i < nr; i++ {
		var dkgParam RbDKG1TxPayload
		dkgParam.EpochId = rbepochId
		dkgParam.ProposerId = uint32(i)
		dkgParam.Commit = commitA[i]

		dkg1 := Dkg1ToDkg1Flat(&dkgParam)
		cijBytes1, _ := rlp.EncodeToBytes(dkg1.Commit)

		payloadBytes, _ := rlp.EncodeToBytes(dkg1)


		payload := buildDkg1(payloadBytes)

		hashCij := GetRBKeyHash(kindCij, dkgParam.EpochId, dkgParam.ProposerId)

		_, err := rbcontract.Run(payload, rbcontractParam, evm)
		if err != nil {
			t.Error(err)
		}

		cijBytes2 := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, *hashCij)

		if !bytes.Equal(cijBytes1, cijBytes2) {
			println("cij error")
		}
	}
}

func TestRBDkg2(t *testing.T) {
	clearDB()
	rbepochId = 0
	epochTimeSpan := uint64(posconfig.SlotTime * posconfig.SlotCount)*rbepochId
	evm.Time = big.NewInt(0).SetUint64(epochTimeSpan+posconfig.SlotTime*(posconfig.Cfg().Dkg1End-2))

	rbgroupdb[rbepochId] = pubs
	TestRBDkg1(t)

	evm.Time = big.NewInt(0).SetUint64(epochTimeSpan+posconfig.SlotTime*(posconfig.Cfg().Dkg2End-2))

	for i := 0; i < nr; i++ {
		var dkgParam RbDKG2TxPayload
		dkgParam.EpochId = rbepochId
		dkgParam.ProposerId = uint32(i)
		dkgParam.EnShare = enshareA[i]
		dkgParam.Proof = proofA[i]

		dkg1 := Dkg2ToDkg2Flat(&dkgParam)
		ensBytes1, _ := rlp.EncodeToBytes(dkg1.EnShare)

		payloadBytes, _ := rlp.EncodeToBytes(dkg1)


		payload := buildDkg2(payloadBytes)

		hashEns := GetRBKeyHash(kindEns, dkgParam.EpochId, dkgParam.ProposerId)

		_, err := rbcontract.Run(payload, rbcontractParam, evm)
		if err != nil {
			t.Error(err)
		}

		ensBytes2 := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, *hashEns)

		if !bytes.Equal(ensBytes1, ensBytes2) {
			println("cij error")
		}
	}
}

func TestRBSig(t *testing.T)  {
	clearDB()

	rbepochId = 0
	epochTimeSpan := uint64(posconfig.SlotTime * posconfig.SlotCount)*rbepochId
	evm.Time = big.NewInt(0).SetUint64(epochTimeSpan+posconfig.SlotTime*(posconfig.Cfg().Dkg2End-2))

	TestRBDkg2(t)

	evm.Time = big.NewInt(0).SetUint64(epochTimeSpan+posconfig.SlotTime*(posconfig.Cfg().SignBegin+2))

	gsigshareA := prepareSig(pris, enshareA)
	for i := 0; i < nr; i++ {
		var sigShareParam RbSIGTxPayload
		sigShareParam.EpochId = rbepochId
		sigShareParam.ProposerId = uint32(i)
		sigShareParam.GSignShare = gsigshareA[i]

		payloadBytes, _ := rlp.EncodeToBytes(sigShareParam)
		payload := buildSig(payloadBytes)
		hash := GetRBKeyHash(sigShareId[:], sigShareParam.EpochId, sigShareParam.ProposerId)

		_, err := rbcontract.Run(payload, rbcontractParam, evm)
		if err != nil {
			t.Error(err)
		}
		payloadBytes2 := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, *hash)

		if !bytes.Equal(payloadBytes, payloadBytes2) {
			println("error")
		}
		sigshareParam2, err := GetSig(evm.StateDB, rbepochId, uint32(i))
		if err != nil {
			t.Error(err)
		}
		if sigshareParam2.EpochId != sigShareParam.EpochId ||
			sigshareParam2.ProposerId != sigShareParam.ProposerId ||
			!bytes.Equal(sigshareParam2.GSignShare.Marshal(), sigShareParam.GSignShare.Marshal()) {
			println("rb sign error")
		}


	}
}

func TestValidPosTx(t *testing.T) {
	clearDB()
	evm.BlockNumber = big.NewInt(0).SetUint64(uint64(100))
	rbgroupdb[rbepochId] = pubs

	for i := 0; i < nr; i++ {
		var dkgParam RbDKG1TxPayload
		dkgParam.EpochId = rbepochId
		dkgParam.ProposerId = uint32(i)
		dkgParam.Commit = commitA[i]

		dkg1 := Dkg1ToDkg1Flat(&dkgParam)
		payloadBytes, _ := rlp.EncodeToBytes(dkg1)
		payload := buildDkg1(payloadBytes)

		err := ValidPosRBTx(evm.StateDB, contract.CallerAddress, payload)
		if err != nil {
			t.Error("verify pos tx fail. err:", err)
		}

		_, err = rbcontract.Run(payload, rbcontractParam, evm)
		if err != nil {
			t.Error("rb contract run fail. err:", err)
		}
	}

	for i := 0; i < nr; i++ {
		var dkgParam RbDKG2TxPayload
		dkgParam.EpochId = rbepochId
		dkgParam.ProposerId = uint32(i)
		dkgParam.EnShare = enshareA[i]
		dkgParam.Proof = proofA[i]

		dkg1 := Dkg2ToDkg2Flat(&dkgParam)
		payloadBytes, _ := rlp.EncodeToBytes(dkg1)
		payload := buildDkg2(payloadBytes)

		err := ValidPosRBTx(evm.StateDB, contract.CallerAddress, payload)
		if err != nil {
			t.Error("verify pos tx fail. err:", err)
		}

		_, err = rbcontract.Run(payload, rbcontractParam, evm)
		if err != nil {
			t.Error("rb contract run fail. err:", err)
		}

	}

	gsigshareA := prepareSig(pris, enshareA)
	for i := 0; i < nr; i++ {
		var sigShareParam RbSIGTxPayload
		sigShareParam.EpochId = rbepochId
		sigShareParam.ProposerId = uint32(i)
		sigShareParam.GSignShare = gsigshareA[i]

		payloadBytes, _ := rlp.EncodeToBytes(sigShareParam)
		payload := buildSig(payloadBytes)

		err := ValidPosRBTx(evm.StateDB, contract.CallerAddress, payload)
		if err != nil {
			t.Error("verify pos tx fail. err:", err)
		}

		_, err = rbcontract.Run(payload, rbcontractParam, evm)
		if err != nil {
			t.Error("rb contract run fail. err:", err)
		}

	}
}

func TestGetRBStage(t *testing.T) {
	k := posconfig.K
	data := [][]int{
		{0, 		RbDkg1Stage, 			0, 		int(2*k-1)},
		{k-1, 		RbDkg1Stage, 			k-1, 	int(k)},
		{2*k-1, 	RbDkg1Stage, 			2*k-1, 	0},
		{2*k, 		RbDkg1ConfirmStage, 	0, 		int(2*k-1)},
		{3*k-1, 	RbDkg1ConfirmStage, 	k-1, 	int(k)},
		{4*k-1, 	RbDkg1ConfirmStage, 	2*k-1, 	0},
		{4*k, 		RbDkg2Stage, 			0, 		int(2*k-1)},
		{5*k-1, 	RbDkg2Stage, 			k-1, 	int(k)},
		{6*k-1, 	RbDkg2Stage, 			2*k-1, 	0},
		{6*k, 		RbDkg2ConfirmStage, 	0, 		int(2*k-1)},
		{7*k-1, 	RbDkg2ConfirmStage, 	k-1, 	int(k)},
		{8*k-1, 	RbDkg2ConfirmStage, 	2*k-1, 	0},
		{8*k, 		RbSignStage, 			0, 		int(2*k-1)},
		{9*k-1, 	RbSignStage, 			k-1, 	int(k)},
		{10*k-1, 	RbSignStage, 			2*k-1, 	0},
		{10*k, 		RbSignConfirmStage, 	0, 		int(2*k-1)},
		{11*k-1, 	RbSignConfirmStage, 	k-1, 	int(k)},
		{12*k-1, 	RbSignConfirmStage, 	2*k-1, 	0},
	}

	for i := range data {
		stage, elapsed, left := GetRBStage(uint64(data[i][0]))
		if data[i][1] != stage || data[i][2] != elapsed || data[i][3] != left {
			t.Error("expect(stage:", data[i][1], ", elapsed:", data[i][2], ", left:", data[i][3],
				")    actual(stage:", stage, ", elapsed:", elapsed, ", left:", left, ")")
		}
	}
}
