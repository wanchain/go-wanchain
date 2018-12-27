package vm

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/pos/wanpos_crypto"
	"io"
	"math/big"
	mrand "math/rand"
	"strings"
	"testing"
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
	dbMockRetVal        *big.Int
	balanceAddr = "0x0000000000000000000000001000000000000000"
	accountAddr = "0x03b854fc72fb01a0e36ee918b085ff52280d1842eeb282b389a1fb3d3752ed7aed"
)

func (CTStateDB) GetStateByteArray(addr common.Address, hs common.Hash) []byte {

	if !bytes.Equal(addr.Bytes(), otaImageStorageAddr.Bytes()) {

		if bytes.Equal(common.FromHex(balanceAddr), addr.Bytes()) {
			return common.FromHex(accountAddr)
		} else if  dbMockRetVal!=nil {
			return dbMockRetVal.Bytes()
		} else {
			return nil
		}

	} else {
		return nil
	}
}

func (CTStateDB) SetStateByteArray(common.Address, common.Hash, []byte) {}

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
	nr = 10
	thres = nr / 2

	db, _      = ethdb.NewMemDatabase()
	statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	ref = &dummyCtRef{}
	evm = NewEVM(Context{}, dummyCtDB{ref: ref}, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})

	pubs, pris, hpubs = generateKeyPairs()
	//s, sshare, enshare, commit, proof := prepareDkg(pubs, pris, hpubs)
	sA, sshareA, enshareA, commitA, proofA = prepareDkg(pubs, pris, hpubs)
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
		x[i].SetBytes(crypto.Keccak256(Pubkey[i].Marshal()))
		x[i].Mod(&x[i], bn256.Order)
	}

	return Pubkey, Prikey, x
}

func prepareDkg(Pubkey []bn256.G1, Prikey []big.Int, x []big.Int) ([]*big.Int, [][]big.Int, [][]*bn256.G1, [][]*bn256.G2, [][]wanpos.DLEQproof) {
	// Each of random propoer generates a random si
	s := make([]*big.Int, nr)

	source := mrand.NewSource(int64(nr))
	r := mrand.New(source)

	for i := 0; i < nr; i++ {
		s[i], _ = rand.Int(r, bn256.Order)
	}

	// Each random propoer conducts the shamir secret sharing process
	poly := make([]wanpos.Polynomial, nr)

	sshare := make([][]big.Int, nr, nr)

	for i := 0; i < nr; i++ {
		sshare[i] = make([]big.Int, nr, nr)
		poly[i] = wanpos.RandPoly(thres-1, *s[i])	// fi(x), set si as its constant term
		for j := 0; j < nr; j++ {
			sshare[i][j] = wanpos.EvaluatePoly(poly[i], &x[j], thres-1) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
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
	proof := make([][]wanpos.DLEQproof, nr, nr)
	for i := 0; i < nr; i++ {
		proof[i] = make([]wanpos.DLEQproof, nr, nr)
		for j := 0; j < nr; j++ { // proof = (a1, a2, z)
			proof[i][j] = wanpos.DLEQ(Pubkey[j], *hbase, &sshare[i][j])
		}
	}

	return s, sshare, enshare, commit, proof
}

// test cases runs in testMain
func TestMain(m *testing.M) {
	println("rb test begin")
	m.Run()
	println("rb test end")
}

func show(v interface{}) {
	println(fmt.Sprintf("%v", v))
}

func TestRBDkg(t *testing.T) {
	var dkgParam RbDKGTxPayload
	dkgParam.EpochId = 0
	dkgParam.ProposerId = 2
	dkgParam.Commit = commitA[2]
	dkgParam.Enshare = enshareA[2]
	dkgParam.Proof = proofA[2]

	payloadBytes, _ := rlp.EncodeToBytes(dkgParam)
	payloadStr := common.Bytes2Hex(payloadBytes)
	rbAbi, _ := abi.JSON(strings.NewReader(GetRBAbiDefinition()))
	payload, _ := rbAbi.Pack("dkg", payloadStr)

	var strtest = "abcdefghi"
	strPayload, _ := rbAbi.Pack("dkg", strtest)
	var str string
	rbAbi.Unpack(&str, "dkg", strPayload[4:])
	var str1 string
	rbAbi.UnpackInput(&str1, "dkg", strPayload[4:])
	if strtest != str {
		println("string pack unpack error")
	}
	if str1 != str {
		println("string pack unpack Input error")
	}

	contract := &RandomBeaconContract{}
	hash := GetRBKeyHash(dkgId[:], dkgParam.EpochId, dkgParam.ProposerId)

	contract.Run(payload, nil, evm)

	payloadBytes2 := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, *hash)

	if len(payloadBytes) != len(payloadBytes2) {
		println("error")
	}
	//contract.Run(data, nil, nil)
}

func TestUtil(t *testing.T) {
	arr := [5]int{1, 2, 3, 4, 5}
	slice1 := arr[1:4]
	println(slice1)

	var randr = rand.Reader
	data := make([]byte, 4)
	data1 := make([]byte, 4)
	if randr == rand.Reader {
		println("same rand")
	}
	io.ReadFull(randr, data)
	io.ReadFull(rand.Reader, data1)
	io.ReadFull(randr, data)
	io.ReadFull(rand.Reader, data1)
}