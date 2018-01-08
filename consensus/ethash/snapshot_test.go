// Copyright 2018 Wanchain Foundation Ltd

package ethash

import (
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/ethdb"
)

type SignerInfo struct {
	private *ecdsa.PrivateKey
	addr    common.Address
	str     string
	index   int
}

var (
	// assert never be lower than 6
	totalSigner                              = 20
	signerSet                                = make(map[string]*SignerInfo)
	addrStrArray                             = make([]string, 0)
	addrArray                                = make([]common.Address, 0)
	indexAddrStrMap                          = make(map[int]string)
	unAuthorizedSigner                       = common.Address{}
	unAuthorizedPrivateKey *ecdsa.PrivateKey = nil
)

func init() {
	// generate
	for i := 0; i < totalSigner; i++ {
		private, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(private.PublicKey)
		str := addr.String()
		signerSet[str] = &SignerInfo{private: private, addr: addr, str: str, index: i}
		addrStrArray = append(addrStrArray, str)
		addrArray = append(addrArray, addr)
		indexAddrStrMap[i] = str
	}
	unAuthorizedPrivateKey, _ = crypto.GenerateKey()
	unAuthorizedSigner.Set(crypto.PubkeyToAddress(unAuthorizedPrivateKey.PublicKey))
}

//store and retrieve permission pow
func TestStoreAndLoadEmptySnapshot(t *testing.T) {
	//create a initial snapshot with only one permission signer
	addr := addrStrArray[0]
	genesisAddr := signerSet[addr].addr
	hash := crypto.Keccak256Hash([]byte{0})
	s := newSnapshot(0, hash, []common.Address{genesisAddr})

	db, _ := ethdb.NewMemDatabase()
	s.store(db)

	sload, _ := loadSnapShot(db, hash)
	if len(sload.PermissionSigners) != 1 || sload.Number != 0 ||
		len(sload.UsedSigners) != 0 || sload.RecentSignersWindow.Len() != 0 {
		t.Error("load snapshot failed")
	}

	if _, ok := sload.PermissionSigners[genesisAddr]; !ok {
		t.Error("load snapshot failed")
	}
}

//store and retrieve permission pow
func TestStoreAndLoadRunningSnapshot(t *testing.T) {
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := uint64(88)
	s := newSnapshot(blockNumber, hash, addrArray)

	usedCount := totalSigner / 2
	for i := 0; i < usedCount; i++ {
		s.UsedSigners[addrArray[i]] = struct{}{}
	}

	windowLen := (usedCount - 1) / 2
	for i := windowLen - 1; i >= 0; i-- {
		s.RecentSignersWindow.PushFront(addrArray[i])
	}

	db, _ := ethdb.NewMemDatabase()
	s.store(db)

	sload, _ := loadSnapShot(db, hash)
	if len(sload.PermissionSigners) != totalSigner || sload.Number != blockNumber ||
		len(sload.UsedSigners) != usedCount || sload.RecentSignersWindow.Len() != windowLen {
		t.Error("load snapshot failed")
	}

	for i := 0; i < usedCount; i++ {
		if _, ok := sload.PermissionSigners[addrArray[i]]; !ok {
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

func sign(header *types.Header, signer common.Address) {
	si := signerSet[signer.String()]
	sig, _ := crypto.Sign(sigHash(header).Bytes(), si.private)
	copy(header.Extra[len(header.Extra)-65:], sig)
}

// indexes indicated use which signer to sign header
func prepareHeaders(indexes []int, blockNumbers []int) []*types.Header {
	headers := make([]*types.Header, 0)
	for i, n := range indexes {
		signer := addrArray[n]
		h := &types.Header{
			Coinbase: signer,
			Time:     big.NewInt(int64(blockNumbers[i]) * int64(1000)),
			Number:   big.NewInt(int64(blockNumbers[i])),
			Extra:    make([]byte, extraSeal+extraVanity),
		}
		sign(h, signer)
		headers = append(headers, h)
	}
	return headers
}

func prepareUnAuthorizedSignerHeader(blockNumber int) []*types.Header {
	headers := make([]*types.Header, 0)
	h := &types.Header{
		Coinbase: unAuthorizedSigner,
		Time:     big.NewInt(int64(blockNumber) * int64(1000)),
		Number:   big.NewInt(int64(blockNumber)),
		Extra:    make([]byte, extraSeal+extraVanity),
	}
	sig, _ := crypto.Sign(sigHash(h).Bytes(), unAuthorizedPrivateKey)
	copy(h.Extra[len(h.Extra)-65:], sig)
	headers = append(headers, h)
	return headers
}

func TestPPOWApplyingFixedCorrectHeaders(t *testing.T) {
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := 0
	s := newSnapshot(uint64(blockNumber), hash, addrArray)
	blockNumber++

	usingSigners := totalSigner - 3
	signerIndexes := make([]int, 0)
	blockNumbers := make([]int, 0)
	expectWindowLen := (usingSigners - 1) / 2
	for i := 0; i < usingSigners; i++ {
		signerIndexes = append(signerIndexes, i)
		blockNumbers = append(blockNumbers, blockNumber)
		blockNumber++
	}
	for i := 0; i < expectWindowLen; i++ {
		signerIndexes = append(signerIndexes, i)
		blockNumbers = append(blockNumbers, blockNumber)
		blockNumber++
	}
	headers := prepareHeaders(signerIndexes, blockNumbers)
	_, err := s.apply(headers)
	if err != nil {
		t.Error("apply shouldn't be failed ")
	}

	for i := 0; i < usingSigners; i++ {
		if _, ok := s.PermissionSigners[addrArray[i]]; !ok {
			t.Error("used signer didn't record")
		}
	}

	expectedIndex := expectWindowLen - 1
	for e := s.RecentSignersWindow.Front(); e != nil; e = e.Next() {
		addr := e.Value.(common.Address)
		if strings.Compare(addr.String(), addrStrArray[expectedIndex]) != 0 {
			t.Error("error in recent window store or retrieve")
		}
		expectedIndex--
	}
}

func TestPPOWApplyingErrBlockNumberHeaders(t *testing.T) {
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := 0
	s := newSnapshot(uint64(blockNumber), hash, addrArray)
	blockNumber++

	//invalid block headers number order
	invalidNumberHeaders := prepareHeaders([]int{0, 1}, []int{blockNumber + 1, blockNumber})
	_, err := s.apply(invalidNumberHeaders)
	if err == nil {
		t.Error("apply error invalid order block number")
	}
}

func TestPPOWApplyingUnAuthorizedHeader(t *testing.T) {
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := 0
	s := newSnapshot(uint64(blockNumber), hash, addrArray)
	blockNumber++

	invalidNumberHeaders := prepareUnAuthorizedSignerHeader(blockNumber)
	_, err := s.apply(invalidNumberHeaders)
	if err == nil {
		t.Error("apply error invalid order block number")
	}
}

func TestPPOWApplyingNotContinueNumberHeaders(t *testing.T) {
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := 8
	s := newSnapshot(uint64(blockNumber), hash, addrArray)

	emptyHeaders := prepareHeaders([]int{}, []int{})
	_, err := s.apply(emptyHeaders)
	if err != nil {
		t.Error("process empty header list failed")
	}
	//invalid block headers number order
	invalidNumberHeaders := prepareHeaders([]int{0}, []int{blockNumber})
	_, err = s.apply(invalidNumberHeaders)
	if err == nil {
		t.Error("apply error invalid order block number")
	}
	invalidNumberHeaders = prepareHeaders([]int{0}, []int{blockNumber - 1})
	_, err = s.apply(invalidNumberHeaders)
	if err == nil {
		t.Error("apply error invalid order block number")
	}
	invalidNumberHeaders = prepareHeaders([]int{0}, []int{blockNumber + 2})
	_, err = s.apply(invalidNumberHeaders)
	if err == nil {
		t.Error("apply error invalid order block number")
	}
}

//if have error, then
func internalApply(snap *Snapshot, snapNumber uint64, headers []*types.Header, state *internalState) (uint64, *internalState) {
	backupState := state.copy()
	if len(headers) == 0 {
		return snapNumber, backupState
	}

	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return snapNumber, backupState
		}
	}

	if headers[0].Number.Uint64() != snapNumber+1 {
		return snapNumber, backupState
	}

	for _, header := range headers {
		signer, err := ecrecover(header)
		if err != nil || 0 != strings.Compare(signer.String(), header.Coinbase.String()) {
			return snapNumber, backupState
		}

		if _, ok := snap.PermissionSigners[signer]; !ok {
			return snapNumber, backupState
		}

		for _, wSigner := range state.testWindow {
			if signer == wSigner {
				return snapNumber, backupState
			}
		}

		_, ok := state.testUsedSigners[signer]
		internalUpdateWindow(state, signer, ok)
	}
	return headers[len(headers)-1].Number.Uint64(), state
}

func internalUpdateWindow(state *internalState, signer common.Address, isExist bool) {
	if isExist {
		//if signer already presence
		if (len(state.testWindow)) > 0 {
			backup := make([]common.Address, 0)
			backup = append(backup, signer)
			backup = append(backup, state.testWindow...)
			state.testWindow = make([]common.Address, len(backup)-1)
			copy(state.testWindow, backup[:len(backup)-1])
		}
	} else {
		// This is the first time the signer appear
		state.testUsedSigners[signer] = struct{}{}
		preWindowLen := len(state.testWindow)
		newWindowLen := (len(state.testUsedSigners) - 1) / windowRatio

		if newWindowLen > preWindowLen {
			backup := make([]common.Address, 0)
			backup = append(backup, signer)
			backup = append(backup, state.testWindow...)
			state.testWindow = make([]common.Address, len(backup))
			copy(state.testWindow, backup)
		} else {
			//windowLen unchanged
			if newWindowLen > 0 {
				backup := make([]common.Address, 0)
				backup = append(backup, signer)
				backup = append(backup, state.testWindow...)
				state.testWindow = make([]common.Address, len(backup)-1)
				copy(state.testWindow, backup[:len(backup)-1])
			}
		}
	}
}

type internalState struct {
	testUsedSigners map[common.Address]struct{}
	testWindow      []common.Address
}

func (self *internalState) copy() *internalState {
	cpy := &internalState{
		testUsedSigners: make(map[common.Address]struct{}),
		testWindow:      make([]common.Address, 0),
	}

	for k := range self.testUsedSigners {
		cpy.testUsedSigners[k] = struct{}{}
	}

	for _, e := range self.testWindow {
		cpy.testWindow = append(cpy.testWindow, e)
	}
	return cpy
}

// generate x times random headers to apply
func TestPPOWApplyingRandom(t *testing.T) {
	tests := []struct {
		headersLen int
		loopTimes  int
	}{
		{
			headersLen: 1,
			loopTimes:  100,
		},
		{
			headersLen: 2,
			loopTimes:  80,
		},
		{
			headersLen: 3,
			loopTimes:  60,
		},
		{
			headersLen: 5,
			loopTimes:  50,
		},
	}
	for _, loopInfo := range tests {
		loops := totalSigner * loopInfo.loopTimes
		hash := crypto.Keccak256Hash([]byte{0})
		blockNumber := 0
		s := newSnapshot(uint64(blockNumber), hash, addrArray)
		blockNumber++

		usingSigners := totalSigner - 5

		state := &internalState{
			testUsedSigners: make(map[common.Address]struct{}),
			testWindow:      make([]common.Address, 0),
		}

		for ; loops > 0; loops-- {
			number := s.Number + 1
			indexes := make([]int, 0)
			numbers := make([]int, 0)
			for ri := 0; ri < loopInfo.headersLen; ri++ {
				selectNumber := rand.Intn(usingSigners)
				indexes = append(indexes, selectNumber)
				numbers = append(numbers, int(number))
				number++
			}
			headers := prepareHeaders(indexes, numbers)
			reservedNumber := s.Number
			ns, err := s.apply(headers)
			if err == nil {
				s = ns
			}
			var afterApplyNum uint64
			afterApplyNum, state = internalApply(s, reservedNumber, headers, state)
			if afterApplyNum != s.Number {
				t.Errorf("invalid status of snapshot :snapshot num: %d compared num: %d\n", int(s.Number), int(afterApplyNum))
			}
		}
		//compare used signers and window equal
		if len(state.testUsedSigners) != len(s.UsedSigners) {
			t.Error("usedSigners length incorrect")
		}

		for signer := range state.testUsedSigners {
			if _, ok := s.UsedSigners[signer]; !ok {
				t.Error("used signers element invalid")
			}
		}

		if s.RecentSignersWindow.Len() != len(state.testWindow) {
			t.Error("recent signer window len incorrect")
		}

		windowIndex := 0
		for e := s.RecentSignersWindow.Front(); e != nil; e = e.Next() {
			addr := e.Value.(common.Address)
			if strings.Compare(addr.String(), state.testWindow[windowIndex].String()) != 0 {
				t.Error("error in recent window signer")
			}
			windowIndex++
		}
	}
}

func TestIsSignerLegal(t *testing.T) {
	hash := crypto.Keccak256Hash([]byte{0})
	blockNumber := 0
	s := newSnapshot(uint64(blockNumber), hash, addrArray)
	blockNumber++

	usingSigners := totalSigner - 3
	signerIndexes := make([]int, 0)
	blockNumbers := make([]int, 0)
	for i := 0; i < usingSigners; i++ {
		signerIndexes = append(signerIndexes, i)
		blockNumbers = append(blockNumbers, blockNumber)
		blockNumber++
	}
	headers := prepareHeaders(signerIndexes, blockNumbers)
	s, _ = s.apply(headers)

	if nil == s.isLegal4Sign(unAuthorizedSigner) {
		t.Error("invalid process unauthorized signer")
	}

	if nil == s.isLegal4Sign(addrArray[usingSigners-1]) {
		t.Error("invalid process in window signer")
	}

	if nil != s.isLegal4Sign(addrArray[usingSigners]) {
		t.Error("invalid process valid signer")
	}
}
