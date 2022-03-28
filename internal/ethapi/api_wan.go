// Copyright 2015 The go-ethereum Authors
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

package ethapi

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pos/posconfig"
	posutil "github.com/ethereum/go-ethereum/pos/util"

	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"strings"
)

const (
	defaultGas = 90000
)

var (
	ErrInvalidWAddress                  = errors.New("Invalid Waddress, try again")
	ErrFailToGeneratePKPairFromWAddress = errors.New("Fail to generate publickey pair from WAddress")
	ErrFailToGeneratePKPairSlice        = errors.New("Fail to generate publickey pair hex slice")
	ErrInvalidPrivateKey                = errors.New("Invalid private key")
	ErrInvalidOTAMixSet                 = errors.New("Invalid OTA mix set")
	ErrInvalidOTAAddr                   = errors.New("Invalid OTA address")
	ErrReqTooManyOTAMix                 = errors.New("Require too many OTA mix address")
	ErrInvalidOTAMixNum                 = errors.New("Invalid required OTA mix address number")
	ErrInvalidInput                     = errors.New("Invalid input")
	ErrInvalidOTAImage                  = errors.New("Invalid OTA image")
)

// RingSignedData represents a ring-signed digital signature
type RingSignedData struct {
	PublicKeys []*ecdsa.PublicKey
	KeyImage   *ecdsa.PublicKey
	Ws         []*big.Int
	Qs         []*big.Int
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
//func (s *PublicEthereumAPI) ProtocolVersion() hexutil.Uint {
//	return hexutil.Uint(s.b.ProtocolVersion())
//}

func (s *PrivateAccountAPI) UpdateAccount(addr common.Address, oldPassword string, newPassword string) error {
	keystore, err := fetchKeystore(s.am)
	if keystore == nil {
		return errors.New("invalid keystore!")
	}
	if err != nil {
		return err
	}

	account := accounts.Account{Address: addr}
	return keystore.Update(account, oldPassword, newPassword)
}

// SendPrivacyCxtTransaction will create a transaction from the given arguments and
// tries to sign it with the OTA key associated with args.To.
func (s *PrivateAccountAPI) SendPrivacyCxtTransaction(ctx context.Context, args TransactionArgs, sPrivateKey string) (common.Hash, error) {

	if !hexutil.Has0xPrefix(sPrivateKey) {
		return common.Hash{}, ErrInvalidPrivateKey
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}

	if args.To == nil || len(args.data()) == 0 {
		return common.Hash{}, ErrInvalidInput
	}

	// Assemble the transaction and sign with the wallet
	tx := args.toOTATransaction()

	var chainID *big.Int
	//if config := s.b.ChainConfig(); config.IsEIP155(s.b.CurrentBlock().Number()) {
	if config := s.b.ChainConfig(); config != nil {
		chainID = config.ChainID
	}

	privateKey, err := crypto.HexToECDSA(sPrivateKey[2:])
	if err != nil {
		return common.Hash{}, err
	}

	//for fixing bug send privacy tx with a different private key
	fromAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	if !bytes.Equal(fromAddr[:], args.From[:]) {
		return common.Hash{}, errors.New("from address mismatch private key")
	}

	var signed *types.Transaction
	signed, err = types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	return submitTransaction(ctx, s.b, signed)
}

// GenRingSignData generate ring sign data
func (s *PrivateAccountAPI) GenRingSignData(ctx context.Context, hashMsg string, privateKey string, mixWanAdresses string) (string, error) {
	if !hexutil.Has0xPrefix(privateKey) {
		return "", ErrInvalidPrivateKey
	}

	hmsg, err := hexutil.Decode(hashMsg)
	if err != nil {
		return "", err
	}

	ecdsaPrivateKey, err := crypto.HexToECDSA(privateKey[2:])
	if err != nil {
		return "", err
	}

	privKey, err := hexutil.Decode(privateKey)
	if err != nil {
		return "", err
	}

	if privKey == nil {
		return "", ErrInvalidPrivateKey
	}

	wanAddresses := strings.Split(mixWanAdresses, "+")
	if len(wanAddresses) == 0 {
		return "", ErrInvalidOTAMixSet
	}

	return genRingSignData(hmsg, privKey, &ecdsaPrivateKey.PublicKey, wanAddresses)
}

func genRingSignData(hashMsg []byte, privateKey []byte, actualPub *ecdsa.PublicKey, mixWanAdress []string) (string, error) {
	otaPrivD := new(big.Int).SetBytes(privateKey)

	publicKeys := make([]*ecdsa.PublicKey, 0)
	publicKeys = append(publicKeys, actualPub)

	for _, strWanAddr := range mixWanAdress {
		pubBytes, err := hexutil.Decode(strWanAddr)
		if err != nil {
			return "", errors.New("fail to decode wan address!")
		}

		if len(pubBytes) != common.WAddressLength {
			return "", ErrInvalidWAddress
		}

		publicKeyA, _, err := keystore.GeneratePKPairFromWAddress(pubBytes)
		if err != nil {

			return "", errors.New("Fail to generate public key from wan address!")

		}

		publicKeys = append(publicKeys, publicKeyA)
	}

	retPublicKeys, keyImage, w_random, q_random, err := crypto.RingSign(hashMsg, otaPrivD, publicKeys)
	if err != nil {
		return "", err
	}

	return encodeRingSignOut(retPublicKeys, keyImage, w_random, q_random)
}

//  encode all ring sign out data to a string
func encodeRingSignOut(publicKeys []*ecdsa.PublicKey, keyimage *ecdsa.PublicKey, Ws []*big.Int, Qs []*big.Int) (string, error) {
	tmp := make([]string, 0)
	for _, pk := range publicKeys {
		tmp = append(tmp, common.ToHex(crypto.FromECDSAPub(pk)))
	}

	pkStr := strings.Join(tmp, "&")
	k := common.ToHex(crypto.FromECDSAPub(keyimage))
	wa := make([]string, 0)
	for _, wi := range Ws {
		wa = append(wa, hexutil.EncodeBig(wi))
	}

	wStr := strings.Join(wa, "&")
	qa := make([]string, 0)
	for _, qi := range Qs {
		qa = append(qa, hexutil.EncodeBig(qi))
	}
	qStr := strings.Join(qa, "&")
	outs := strings.Join([]string{pkStr, k, wStr, qStr}, "+")
	return outs, nil
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

// ChainId returns the chainID value for transaction replay protection.
func (s *PublicBlockChainAPI) ChainId() *hexutil.Big {
	if posutil.IsJupiterForkArrived() {
		fmt.Println("IsJupiterForkArrived")
		if params.JupiterChainId(s.b.ChainConfig().ChainID.Uint64()) != params.NOT_JUPITER_CHAIN_ID {
			fmt.Println("IsJupiterForkArrived2")
			return (*hexutil.Big)(big.NewInt(0).SetUint64(params.JupiterChainId(s.b.ChainConfig().ChainID.Uint64())))
		}
	}
	fmt.Println("IsJupiterForkArrived33")
	return (*hexutil.Big)(s.b.ChainConfig().ChainID)
}

// GetOTABalance returns OTA balance
func (s *PublicBlockChainAPI) GetOTABalance(ctx context.Context, otaWAddr string, blockNr rpc.BlockNumber) (*big.Int, error) {
	if !hexutil.Has0xPrefix(otaWAddr) {
		return nil, ErrInvalidOTAAddr
	}

	state, _, err := s.b.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}

	var otaAX []byte
	otaWAddrByte := common.FromHex(otaWAddr)
	switch len(otaWAddrByte) {
	case common.HashLength:
		otaAX = otaWAddrByte
	case common.WAddressLength:
		otaAX, _ = vm.GetAXFromWanAddr(otaWAddrByte)
	default:
		return nil, ErrInvalidOTAAddr
	}

	return vm.GetOtaBalanceFromAX(state, otaAX)
}

func (s *PublicBlockChainAPI) GetSupportWanCoinOTABalances(ctx context.Context) []*big.Int {
	return vm.GetSupportWanCoinOTABalances()
}

func (s *PublicBlockChainAPI) GetSupportStampOTABalances(ctx context.Context) []*big.Int {
	return vm.GetSupportStampOTABalances()
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *PublicBlockChainAPI) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	head := b.Header() // copies the header once
	fields := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             b.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"totalDifficulty":  (*hexutil.Big)(s.b.GetTd(nil, b.Hash())),
		"extraData":        hexutil.Bytes(head.Extra),
		"size":             b.Size().String(),
		"gasLimit":         (*hexutil.Big)(big.NewInt(0).SetUint64(head.GasLimit)),
		"gasUsed":          (*hexutil.Big)(big.NewInt(0).SetUint64(head.GasUsed)),
		"timestamp":        (*hexutil.Big)(big.NewInt(0).SetUint64(head.Time)),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}

		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransactionFromBlockHash(b, tx.Hash(), s.b.ChainConfig()), nil
			}
		}

		txs := b.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range b.Transactions() {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields["transactions"] = transactions
	}

	uncles := b.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes
	if head.Number.Uint64() >= s.b.ChainConfig().PosFirstBlock.Uint64() {
		epochid, slotid := posutil.CalEpSlbyTd(head.Difficulty.Uint64())
		fields["epochId"] = epochid
		fields["slotId"] = slotid
	}
	return fields, nil
}

//
//// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
//type SendTxArgs struct {
//	From     common.Address  `json:"from"`
//	To       *common.Address `json:"to"`
//	Gas      *hexutil.Big    `json:"gas"`
//	GasPrice *hexutil.Big    `json:"gasPrice"`
//	Value    *hexutil.Big    `json:"value"`
//	Data     hexutil.Bytes   `json:"data"`
//	Nonce    *hexutil.Uint64 `json:"nonce"`
//}
//
//// prepareSendTxArgs is a helper function that fills in default values for unspecified tx fields.
//func (args *SendTxArgs) setDefaults(ctx context.Context, b Backend) error {
//	if args.Gas == nil {
//		args.Gas = (*hexutil.Big)(big.NewInt(defaultGas))
//	}
//	if args.GasPrice == nil {
//		price, err := b.SuggestPrice(ctx)
//		if err != nil {
//			return err
//		}
//		args.GasPrice = (*hexutil.Big)(price)
//	}
//	if args.Value == nil {
//		args.Value = new(hexutil.Big)
//	}
//	if args.Nonce == nil {
//		nonce, err := b.GetPoolNonce(ctx, args.From)
//		if err != nil {
//			return err
//		}
//		args.Nonce = (*hexutil.Uint64)(&nonce)
//	}
//	return nil
//}
//
//func (args *SendTxArgs) String() string {
//	return fmt.Sprintf("From=%v,To=%v,Gas=%v,GasPrice=%v,Nonce=%v",
//		args.From, args.To, args.Gas, args.GasPrice, *(args.Nonce))
//}
//
//func (args *SendTxArgs) toTransaction(txType uint64) *types.Transaction {
//	data := &types.WanLegacyTx{
//		Txtype:   txType,
//		To:       args.To,
//		Nonce:    uint64(*args.Nonce),
//		Gas:      (*big.Int)(args.Gas).Uint64(),
//		GasPrice: (*big.Int)(args.GasPrice),
//		Value:    (*big.Int)(args.Value),
//		Data:     args.Data,
//	}
//	return types.NewTx(data)
//}

func (args *TransactionArgs) String() string {
	return fmt.Sprintf("From=%v,To=%v,Gas=%v,GasPrice=%v,Nonce=%v",
		args.from(), args.To, args.Gas, args.GasPrice, *(args.Nonce))
}

// submitTransaction is a helper function that submits tx to txPool and logs a message.
func submitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}

	if tx.To() == nil {
		signer := types.MakeSigner(b.ChainConfig(), b.CurrentBlock().Number())
		from, err := types.Sender(signer, tx)
		if err != nil {
			return common.Hash{}, err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Info("Submitted contract creation", "fullhash", tx.Hash().Hex(), "contract", addr.Hex())
	} else {
		log.Debug("Submitted transaction", "fullhash", tx.Hash().Hex(), "recipient", tx.To())
	}
	return tx.Hash(), nil
}

func (s *PublicTransactionPoolAPI) SendPosTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.from()}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.from())
		defer s.nonceLock.UnlockAddr(args.from())
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b); err != nil {
		return common.Hash{}, err
	}
	log.Info("SendPosTransaction", "args", args.String())
	// Assemble the transaction and sign with the wallet
	tx := args.ToTransaction()

	var chainID *big.Int

	//if config := s.b.ChainConfig(); config.IsEIP155(s.b.CurrentBlock().Number()) {
	if config := s.b.ChainConfig(); config != nil {
		chainID = config.ChainID
	}

	signed, err := wallet.SignTx(account, tx, chainID)
	if err != nil {
		return common.Hash{}, err
	}
	return submitTransaction(ctx, s.b, signed)
}

func (s *PublicTransactionPoolAPI) GetOTAMixSet(ctx context.Context, otaAddr string, setLen int) ([]string, error) {
	if setLen <= 0 {
		return []string{}, ErrInvalidOTAMixNum
	}

	if uint64(setLen) > params.GetOTAMixSetMaxSize {
		return []string{}, ErrReqTooManyOTAMix
	}

	if !hexutil.Has0xPrefix(otaAddr) {
		return []string{}, ErrInvalidOTAAddr
	}

	orgOtaAddr := common.FromHex(otaAddr)
	if len(orgOtaAddr) < common.HashLength {
		return []string{}, ErrInvalidOTAAddr
	}

	state, _, err := s.b.StateAndHeaderByNumber(ctx, rpc.BlockNumber(-1))
	if state == nil || err != nil {
		return nil, err
	}

	otaAX := orgOtaAddr[:common.HashLength]
	if len(orgOtaAddr) == common.WAddressLength {
		otaAX, _ = vm.GetAXFromWanAddr(orgOtaAddr)
	}

	otaByteSet, _, err := vm.GetOTASet(state, otaAX, setLen)
	if err != nil {
		return nil, err
	}

	ret := make([]string, 0, setLen)
	for _, otaByte := range otaByteSet {
		ret = append(ret, common.ToHex(otaByte))

	}

	return ret, nil
}

func (s *PublicTransactionPoolAPI) CheckOTAUsed(ctx context.Context, OTAImage string) (bool, error) {
	if !hexutil.Has0xPrefix(OTAImage) {
		return false, ErrInvalidOTAImage
	}

	imageByte := common.FromHex(OTAImage)
	if len(imageByte) == 0 {
		return false, ErrInvalidOTAImage
	}

	state, _, err := s.b.StateAndHeaderByNumber(ctx, rpc.BlockNumber(-1))
	if state == nil || err != nil {
		return false, err
	}

	exist, _, err := vm.CheckOTAImageExist(state, imageByte)
	return exist, err
}

// ComputeOTAPPKeys compute ota private key, public key and short address
// from account address and ota full address.
func (s *PublicTransactionPoolAPI) ComputeOTAPPKeys(ctx context.Context, address common.Address, inOtaAddr string) (string, error) {
	account := accounts.Account{Address: address}
	//wallet, err := s.b.AccountManager().Find(account)
	//if err != nil {
	//	return "", err
	//}

	wanBytes, err := hexutil.Decode(inOtaAddr)
	if err != nil {
		return "", err
	}

	otaBytes, err := keystore.WaddrToUncompressedRawBytes(wanBytes)
	if err != nil {
		return "", err
	}

	otaAddr := hexutil.Encode(otaBytes)

	//AX string, AY string, BX string, BY string
	otaAddr = strings.Replace(otaAddr, "0x", "", -1)
	AX := "0x" + otaAddr[0:64]
	AY := "0x" + otaAddr[64:128]

	BX := "0x" + otaAddr[128:192]
	BY := "0x" + otaAddr[192:256]
	bd := s.b.AccountManager().Backends(keystore.KeyStoreType)

	sS, err := bd[0].(accounts.WanWallet).ComputeOTAPPKeys(account, AX, AY, BX, BY)
	if err != nil {
		return "", err
	}

	otaPub := sS[0] + sS[1][2:]
	otaPriv := sS[2]

	privateKey, err := crypto.HexToECDSA(otaPriv[2:])
	if err != nil {
		return "", err
	}

	var addr common.Address
	pubkey := crypto.FromECDSAPub(&privateKey.PublicKey)
	//caculate the address for replaced pub
	copy(addr[:], crypto.Keccak256(pubkey[1:])[12:])

	return otaPriv + "+" + otaPub + "+" + hexutil.Encode(addr[:]), nil

}

////////////////////added for privacy tx ////////////////////////////////////////
// GetWanAddress returns corresponding WAddress of an ordinary account
func (s *PublicTransactionPoolAPI) GetWanAddress(ctx context.Context, a common.Address) (string, error) {
	account := accounts.Account{Address: a}
	bd := s.b.AccountManager().Backends(keystore.KeyStoreType)
	// first fetch the wallet/keystore, and then retrieve the wanaddress
	//wallet, err := s.b.AccountManager().Find(account)
	//if err != nil {
	//	return "", err
	//}
	wanAddr, err := bd[0].(accounts.WanWallet).GetWanAddress(account)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(wanAddr[:]), nil
}

// GenerateOneTimeAddress returns corresponding One-Time-Address for a given WanAddress
func (s *PublicTransactionPoolAPI) GenerateOneTimeAddress(ctx context.Context, wAddr string) (string, error) {
	strlen := len(wAddr)
	if strlen != (common.WAddressLength<<1)+2 {
		return "", ErrInvalidWAddress
	}

	PKBytesSlice, err := hexutil.Decode(wAddr)
	if err != nil {
		return "", err
	}

	PK1, PK2, err := keystore.GeneratePKPairFromWAddress(PKBytesSlice)
	if err != nil {
		return "", ErrFailToGeneratePKPairFromWAddress
	}

	PKPairSlice := hexutil.PKPair2HexSlice(PK1, PK2)

	SKOTA, err := crypto.GenerateOneTimeKey(PKPairSlice[0], PKPairSlice[1], PKPairSlice[2], PKPairSlice[3])
	if err != nil {
		return "", err
	}

	otaStr := strings.Replace(strings.Join(SKOTA, ""), "0x", "", -1)
	raw, err := hexutil.Decode("0x" + otaStr)
	if err != nil {
		return "", err
	}

	rawWanAddr, err := keystore.WaddrFromUncompressedRawBytes(raw)
	if err != nil || rawWanAddr == nil {
		return "", err
	}

	return hexutil.Encode(rawWanAddr[:]), nil
}

func (args *TransactionArgs) toOTATransaction() *types.Transaction {
	return types.NewOTATransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), (*big.Int)(args.Value), (*big.Int)(args.GasPrice), args.data())
}

func (s *PrivateAccountAPI) GetOTABalance(ctx context.Context, blockNr rpc.BlockNumber) (*big.Int, error) {
	state, _, err := s.b.StateAndHeaderByNumber(ctx, blockNr)
	if state == nil || err != nil {
		return nil, err
	}

	otaB, err := vm.GetUnspendOTATotalBalance(state)
	if err != nil {
		return common.Big0, err
	}

	return otaB, state.Error()

}

func (s *PrivateAccountAPI) ShowPublicKey(addr common.Address, passwd string) ([]string, error) {

	if len(addr) == 0 {
		return nil, errors.New("address must be given as argument")
	}
	if len(passwd) == 0 {
		return nil, errors.New("passwd must be given as argument")
	}

	ks := s.am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	all := ks.Accounts()
	lenth := len(all)

	pubs := make([]string, 0)
	var exisit bool
	for i := 0; i < lenth; i++ {
		if all[i].Address == addr {
			key, err := ks.GetKey(all[i], passwd)
			if err != nil {
				return nil, err
			}

			if key.PrivateKey != nil {
				pubs = append(pubs, common.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey)))
			}
			D3 := posconfig.GenerateD3byKey2(key.PrivateKey2)
			G1 := new(bn256.G1).ScalarBaseMult(D3)
			pubs = append(pubs, common.ToHex(G1.Marshal()))
			exisit = true
			break
		}
	}
	if !exisit {
		return nil, errors.New("invalid address")
	}
	return pubs, nil
}
func (s *PrivateAccountAPI) ImportRawKey(privkey0, privkey1 string, password string) (common.Address, error) {
	key0, err := crypto.HexToECDSA(privkey0)
	if err != nil {
		return common.Address{}, err
	}
	key1, err := crypto.HexToECDSA(privkey1)
	if err != nil {
		return common.Address{}, err
	}
	ks, err := fetchKeystore(s.am)
	if err != nil {
		return common.Address{}, err
	}
	acc, err := ks.ImportECDSA(key0, key1, password)
	return acc.Address, err
}
