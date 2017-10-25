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

// This file contains the code snippets from the developer's guide embedded into
// Go tests. This ensures that any code published in out guides will not break
// accidentally via some code update. If some API changes nonetheless that needs
// modifying this file, please port any modification over into the developer's
// guide wiki pages too!

package guide

import (
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/core/types"
	"time"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	"strings"
)

// Tests that the account management snippets work correctly.
func TestAccountManagement(t *testing.T) {
	// Create a temporary folder to work with
	workdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to create temporary work dir: %v", err)
	}
	defer os.RemoveAll(workdir)

	// Create an encrypted keystore with standard crypto parameters
	ks := keystore.NewKeyStore(filepath.Join(workdir, "keystore"), keystore.StandardScryptN, keystore.StandardScryptP)

	// Create a new account with the specified encryption passphrase
	newAcc, err := ks.NewAccount("Creation password")
	if err != nil {
		t.Fatalf("Failed to create new account: %v", err)
	}

	// Export the newly created account with a different passphrase. The returned
	// data from this method invocation is a JSON encoded, encrypted key-file
	jsonAcc, err := ks.Export(newAcc, "Creation password", "Export password")
	if err != nil {
		t.Fatalf("Failed to export account: %v", err)
	}

	// Update the passphrase on the account created above inside the local keystore
	if err := ks.Update(newAcc, "Creation password", "Update password"); err != nil {
		t.Fatalf("Failed to update account: %v", err)
	}

	// Delete the account updated above from the local keystore
	if err := ks.Delete(newAcc, "Update password"); err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Import back the account we've exported (and then deleted) above with yet
	// again a fresh passphrase
	if _, err := ks.Import(jsonAcc, "Export password", "Import password"); err != nil {
		t.Fatalf("Failed to import account: %v", err)
	}

	// Create a new account to sign transactions with
	signer, err := ks.NewAccount("Signer password")
	if err != nil {
		t.Fatalf("Failed to create signer account: %v", err)
	}

	tx, chain := new(types.Transaction), big.NewInt(1)


	// Sign a transaction with a single authorization
	if _, err := ks.SignTxWithPassphrase(signer, "Signer password", tx, chain); err != nil {
		t.Fatalf("Failed to sign with passphrase: %v", err)
	}

	// Sign a transaction with multiple manually cancelled authorizations
	if err := ks.Unlock(signer, "Signer password"); err != nil {
		t.Fatalf("Failed to unlock account: %v", err)
	}
	if _, err := ks.SignTx(signer, tx, chain, nil); err != nil {
		t.Fatalf("Failed to sign with unlocked account: %v", err)
	}
	if err := ks.Lock(signer.Address); err != nil {
		t.Fatalf("Failed to lock account: %v", err)
	}

	// Sign a transaction with multiple automatically cancelled authorizations
	if err := ks.TimedUnlock(signer, "Signer password", time.Second); err != nil {
		t.Fatalf("Failed to time unlock account: %v", err)
	}
	if _, err := ks.SignTx(signer, tx, chain, nil); err != nil {
		t.Fatalf("Failed to sign with time unlocked account: %v", err)
	}

}

func TestGenerateOneTimeAddress(t *testing.T) {
	// Create a temporary folder to work with
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to create temporary folder: %v\n", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an encrypted keystore with standard crypto parameters
	ks := keystore.NewKeyStore(filepath.Join(tmpDir, "keystore"), keystore.StandardScryptN, keystore.StandardScryptP)

	// Create a new account with the specified encryption passphrase
	testAcc, err := ks.NewAccount("Wanchain_One_Time_Address_Test")
	if err != nil {
		t.Fatalf("Fail to create new account: %v\n", err)
	}

	// Unlock the new account
	ks.Unlock(testAcc, "Wanchain_One_Time_Address_Test")

	// Retrieve the Wanaddress for the account
	wanAddr, err := ks.GetWanAddress(testAcc)
	if err != nil {
		t.Fatalf("Fail to get WanAddress: %v\n", err)
	}

	PKA, PKB, err := keystore.GeneratePublicKeyFromWadress(wanAddr[:])
	if err != nil {
		t.Fatalf("Fail to generate public key from wan address: %v\n", err)
	}

	strArr := hexutil.TwoPublicKeyToHexSlice(PKA, PKB)

	SK, err := crypto.GenerateOneTimeKey(strArr[0], strArr[1], strArr[2], strArr[3])
	if err != nil {
		t.Fatalf("Fail to generate One Time Key: %v\n", err)
	}

	strCombined := strings.Join(SK, "")
	strCombined = strings.Replace(strCombined, "0x", "", -1)

	rawBytes, _ := hexutil.Decode("0x" + strCombined)
	wbytes, _ := keystore.ToWaddr(rawBytes)

	if len(wbytes) != 66 {
		t.Fatal("Failed to generate One Time Key from Secret Key")
	}

}

func TestSendOTATransaction(t *testing.T) {
	// create a temporary folder to work with
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Fail to create temporary folder to work with: %v", err)
	}

	defer os.RemoveAll(tmpDir)

	// create a keystore with standard crypto parameters
	ks := keystore.NewKeyStore(filepath.Join(tmpDir, "keystore"), keystore.StandardScryptN, keystore.StandardScryptP)

	// create a new account with the specific passphrase
	testAcc, err := ks.NewAccount("Wanchain_OTATransaction_Test")

	// unlock new account
	ks.Unlock(testAcc, "Wanchain_OTATransaction_Test")

	// retireve the wanaddress
	wanAddr, err := ks.GetWanAddress(testAcc)
	if err != nil {
		t.Fatalf("Fail to get wan address")
	}

	PKA, PKB, err := keystore.GeneratePublicKeyFromWadress(wanAddr[:])
	if err != nil {
		t.Fatalf("Fail to recover public keys from wan address: %v", err)
	}

	pubKeyArr := hexutil.TwoPublicKeyToHexSlice(PKA, PKB)
	SK, err := crypto.GenerateOneTimeKey(pubKeyArr[0], pubKeyArr[1], pubKeyArr[2], pubKeyArr[3])
	if err != nil {
		t.Fatalf("Fail to generate one time key: %v", err)
	}

	combined := strings.Join(SK, "")
	combined = strings.Replace(combined, "0x", "", -1)

	rawBytes, err := hexutil.Decode("0x" + combined)
	wBytes, _ := keystore.ToWaddr(rawBytes)

	otaAddr := hexutil.Encode(wBytes)

	if len(otaAddr) != 134 {
		t.Fatalf("Fail to generate one time address, expected address length is: 134; actual length is: %d", len(otaAddr))
	}

	//chain := big.NewInt(1)

	//tx :=

	//ks.SignTx(testAcc, tx, chain, nil)

}