package ethash

import (
	"github.com/wanchain/go-wanchain/common"
	"testing"
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
	"fmt"
	"strings"
	"github.com/wanchain/go-wanchain/core/types"
	"math/big"
)

type SignerInfo struct {
	private *ecdsa.PrivateKey
	addr    common.Address
	str     string
	index   int
}

var (
	totalSigner  = 20
	signerSet    = make(map[string]*SignerInfo)
	addrStrArray = make([]string, 0)
	addrArray = make([]common.Address,0)
	indexAddrStrMap = make(map[int]string)
)


func init(){
	// generate
	for i := 0; i < totalSigner; i++ {
		private, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(private.PublicKey)
		str := addr.String()
		signerSet[str] = &SignerInfo{private:private, addr:addr, str:str, index:i}
		addrStrArray = append(addrStrArray, str)
		addrArray = append(addrArray, addr)
		indexAddrStrMap[i] = str
	}
}

//store and retrieve permission pow
func TestStoreAndLoadEmptySnapshot(t *testing.T) {
	//create a initial snapshot with only one permission signer
	addr := addrStrArray[0]
	genesisAddr := signerSet[addr].addr
	hash := crypto.Keccak256Hash([]byte{0})
	s := newSnapshot(0, hash,[]common.Address{genesisAddr})

	db, _ := ethdb.NewMemDatabase()
	s.store(db)

	sload, _ := loadSnapShot(db, hash)
	if len(sload.PermissionSigners) != 1 || sload.Number != 0 ||
		len(sload.UsedSigners) != 0 || sload.RecentSignersWindow.Len() != 0  {
		t.Error("load snapshot failed")
	}

	if _, ok := sload.PermissionSigners[genesisAddr]; !ok{
		t.Error("load snapshot failed")
	}
}

//store and retrieve permission pow
func TestStoreAndLoadRunningSnapshot(t *testing.T) {
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := uint64(88)
	s := newSnapshot(blockNumber, hash,addrArray)

	usedCount := totalSigner / 2
	for i := 0; i < usedCount; i++ {
		s.UsedSigners[addrArray[i]] = struct{}{}
	}

	windowLen := (usedCount-1)/2
	for i := windowLen-1; i >=0 ; i--{
		s.RecentSignersWindow.PushFront(addrArray[i])
	}

	db, _ := ethdb.NewMemDatabase()
	s.store(db)

	sload, _ := loadSnapShot(db, hash)
	if len(sload.PermissionSigners) != totalSigner || sload.Number != blockNumber ||
		len(sload.UsedSigners) != usedCount || sload.RecentSignersWindow.Len() != windowLen  {
		t.Error("load snapshot failed")
	}

	for i := 0; i < usedCount; i++ {
		if _, ok := sload.PermissionSigners[addrArray[i]]; !ok{
			t.Error("load snapshot failed")
		}
	}

	i := 0
	for e := s.RecentSignersWindow.Front(); e != nil; e = e.Next() {
		addr := e.Value.(common.Address)
		if strings.Compare(addr.String(), addrStrArray[i]) != 0 {
			t.Error("error in recent window store or retrieve")
		}
		i++
	}
	if i != windowLen {
		t.Error("window length is not right")
	}
}

func internalApply(s *Snapshot,headers[]*types.Header, testUsedSigners map[common.Address]struct{}, testWindow []common.Address){
	//for _, header := range headers {
	//
	//}
}

func internalUpdateWindow(testUsedSigners map[common.Address]struct{}, testWindow[]common.Address, signer common.Address, isExist bool) {

}

func sign(header *types.Header, signer common.Address) {
	si := signerSet[signer.String()]
	sig, _ := crypto.Sign(sigHash(header).Bytes(), si.private)
	copy(header.Extra[len(header.Extra)-65:], sig)
}

// indexes indicated use which signer to sign header
func prepareHeaders(indexes []int, blockNumbers []int) []*types.Header{
	headers := make([]*types.Header, 0)
	for i, n := range indexes{
		signer := addrArray[n]
		h := &types.Header{
			Coinbase: signer,
			Time:     big.NewInt(int64(blockNumbers[i]) * int64(1000)),
			Number:   big.NewInt(int64(blockNumbers[i])),
			Extra:    make([]byte, extraSeal+ extraVanity),
		}
		sign(h, signer)
		headers = append(headers, h)
	}
	return headers
}

func TestPPOWApplyingErrBlockNumberHeaders(t *testing.T){
	fmt.Println("")
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := 0
	s := newSnapshot(uint64(blockNumber), hash,addrArray)
	blockNumber++

	//invalid block headers number order
	invalidNumberHeaders := prepareHeaders([]int{0,1}, []int{1,0})
	_, err := s.apply(invalidNumberHeaders)
	if err == nil{
		t.Error("apply error invalid order block number")
	}
}

func TestPPOWApplyingNotCotinueNumberHeaders(t *testing.T){
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := 8
	s := newSnapshot(uint64(blockNumber), hash,addrArray)

	//invalid block headers number order
	invalidNumberHeaders := prepareHeaders([]int{0}, []int{blockNumber})
	_, err := s.apply(invalidNumberHeaders)
	if err == nil{
		t.Error("apply error invalid order block number")
	}
	invalidNumberHeaders = prepareHeaders([]int{0}, []int{blockNumber-1})
	_, err = s.apply(invalidNumberHeaders)
	if err == nil{
		t.Error("apply error invalid order block number")
	}
	invalidNumberHeaders = prepareHeaders([]int{0}, []int{blockNumber+2})
	_, err = s.apply(invalidNumberHeaders)
	if err == nil{
		t.Error("apply error invalid order block number")
	}
}

func TestPPOWApplyingRandom(t *testing.T){
    fmt.Printf("")
}

func TestIsSignerLegal(t *testing.T){

}