package vm

import (
	"encoding/binary"
	"errors"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/go-wanchain/functrace"
	"github.com/wanchain/go-wanchain/rlp"
	"github.com/wanchain/pos/wanpos_crypto"
	"math/big"
	"strconv"
	"strings"
)

var (
	rbscDefinition = `[{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"dkg","outputs":[],"payable":false,"type":"function","stateMutability":"nonpayable"},{"constant":false,"inputs":[{"name":"info","type":"string"}],"name":"sigshare","outputs":[],"payable":false,"type":"function","stateMutability":"nonpayable"}]`
	rbscAbi, errRbscInit = abi.JSON(strings.NewReader(rbscDefinition))

	dkgId [4]byte
	sigshareId [4]byte
	genRId [4]byte
	// Generator of G1
	//gbase = new(bn256.G1).ScalarBaseMult(big.NewInt(int64(1)))
	// Generator of G2
	hbase = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))
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

func GetRBProposerGroup(epochId uint64) []bn256.G1 {
	return nil
}

func GetProposerPubkey(proposerId uint32) *bn256.G1 {

	return nil
}

func (c *RandomBeaconContract) isValidEpoch(epochId uint64) (bool) {
	return true
}

func (c *RandomBeaconContract) isInRandomGroup(proposerId uint32) (bool) {
	return true
}

func UIntToByteSlice(num uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, num)
	return b
}

type RbDKGTxPayload struct {
	EpochId uint64
	ProposerId uint32
	Enshare []bn256.G1
	Commit []bn256.G2
	Proof []wanpos.DLEQproof
}
func (c *RandomBeaconContract) dkg(payload []byte, contract *Contract, evm *EVM) ([]byte, error) {
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
	if !c.isInRandomGroup(dkgParam.ProposerId) {
		return nil, errors.New(" error proposerId " + strconv.FormatUint(uint64(dkgParam.ProposerId), 10))
	}

	// 3. Enshare, Commit, Proof has the same size
	// check same size
	nr := len(dkgParam.Proof)
	thres := nr / 2
	if nr != len(dkgParam.Enshare) || nr != len(dkgParam. Commit) {
		return nil, errors.New("error in dkg params have different length")
	}

	x := make([]big.Int, nr)
	for i := 0; i < nr; i++ {
		x[i].SetBytes(crypto.Keccak256(pks[i].Marshal()))
		x[i].Mod(&x[i], bn256.Order)
	}

	// get send public Key
	pubkey := GetProposerPubkey(dkgParam.ProposerId)
	// 4. proof verification
	for j := 0; j < nr; j++ {
		if !wanpos.VerifyDLEQ(dkgParam.Proof[j], *pubkey, *hbase, dkgParam.Enshare[j], dkgParam.Commit[j]) {
			return nil, errors.New("dkg verify dleq error")
		}
	}
	temp := make([]bn256.G2, nr)
	// 5. Reed-Solomon code verification
	for j := 0; j < nr; j++ {
		temp[j] = dkgParam.Commit[j]
	}
	if !wanpos.RScodeVerify(temp, x, thres - 1) {
		return nil, errors.New("rscode check error")
	}

	// save epochId*2^64 + proposerId
	keyBytes := make([]byte, 16)
	keyBytes = append(UIntToByteSlice(dkgParam.EpochId), UIntToByteSlice(uint64(dkgParam.ProposerId)) ...)
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	// TODO: maybe we can use tx hash to replace payloadBytes, a tx saved in a chain block
	evm.StateDB.SetStateByteArray(randomBeaconPrecompileAddr, hash, payloadBytes)
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

	type RbSIGTxPayload struct {
		EpochId uint64
		ProposerId uint32
		Gsigshare *bn256.G1
	}
	var sigshareParam RbSIGTxPayload
	err = rlp.DecodeBytes(payloadBytes, &sigshareParam)
	if err != nil {
		return nil, errors.New("error in dkg param has a wrong struct")
	}

	// TODO: check
	// 1. EpochId: weather in a wrong time
	if !c.isValidEpoch(sigshareParam.EpochId) {
		return nil, errors.New(" error epochId " + strconv.FormatUint(sigshareParam.EpochId, 10))
	}
	// 2. ProposerId: weather in the random commit
	if !c.isInRandomGroup(sigshareParam.ProposerId) {
		return nil, errors.New(" error proposerId " + strconv.FormatUint(uint64(sigshareParam.ProposerId), 10))
	}
	// TODO: check weather dkg stage has been finished

	// 3. Verification
	M := crypto.Keccak256([]byte("wanchain"))
	m := new(big.Int).SetBytes(M)

	cj0, err := c.getCji(evm, sigshareParam.EpochId, 0)
	if err != nil {
		return nil, errors.New(" can't get cj0 ")
	}
	nr := len(cj0)
	var gpkshare bn256.G2
	gpkshare.Add(&gpkshare, &cj0[sigshareParam.ProposerId])
	for i := 1; i < nr; i++ {
		cji, err := c.getCji(evm, sigshareParam.EpochId, 0)
		if err != nil {
			return nil, errors.New(" can't get cji ")
		}
		gpkshare.Add(&gpkshare, &cji[sigshareParam.ProposerId])
	}

	mG := new(bn256.G1).ScalarBaseMult(m)
	pair1 := bn256.Pair(sigshareParam.Gsigshare, hbase)
	pair2 := bn256.Pair(mG, &gpkshare)
	if pair1.String() != pair2.String() {
		return nil, errors.New(" unequal sigi")
	}
	return nil, nil
}

func (c *RandomBeaconContract) getCji(evm *EVM, sloterId uint64, proposerId uint32) ([]bn256.G2, error) {
	keyBytes := make([]byte, 16)
	keyBytes = append(UIntToByteSlice(sloterId), UIntToByteSlice(uint64(proposerId)) ...)
	hash := common.BytesToHash(crypto.Keccak256(keyBytes))
	dkgBytes := evm.StateDB.GetStateByteArray(randomBeaconPrecompileAddr, hash)

	var dkgParam RbDKGTxPayload
	err := rlp.DecodeBytes(dkgBytes, &dkgParam)
	if err != nil {
		return nil, errors.New("error in sigshare, decode dkg rlp error")
	}
	return dkgParam.Commit, nil
}