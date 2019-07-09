// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/wanchain/go-wanchain/awskms"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"io/ioutil"
	"strings"

	"github.com/wanchain/go-wanchain/accounts"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/cmd/utils"
	"github.com/wanchain/go-wanchain/console"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/log"
	"gopkg.in/urfave/cli.v1"
)

var (
	walletCommand = cli.Command{
		Name:      "wallet",
		Usage:     "Manage Ethereum presale wallets",
		ArgsUsage: "",
		Category:  "ACCOUNT COMMANDS",
		Description: `
    geth wallet import /path/to/my/presale.wallet

will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.`,
		Subcommands: []cli.Command{
			{

				Name:      "import",
				Usage:     "Import Ethereum presale wallet",
				ArgsUsage: "<keyFile>",
				Action:    utils.MigrateFlags(importWallet),
				Category:  "ACCOUNT COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.PasswordFileFlag,
					utils.LightKDFFlag,
				},
				Description: `
	geth wallet [options] /path/to/my/presale.wallet

will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.`,
			},
		},
	}

	accountCommand = cli.Command{
		Name:     "account",
		Usage:    "Manage accounts",
		Category: "ACCOUNT COMMANDS",
		Description: `

Manage accounts, list all existing accounts, import a private key into a new
account, create a new account or update an existing account.

It supports interactive mode, when you are prompted for password as well as
non-interactive mode where passwords are supplied via a given password file.
Non-interactive mode is only meant for scripted use on test networks or known
safe environments.

Make sure you remember the password you gave when creating a new account (with
either new or import). Without it you are not able to unlock your account.

Note that exporting your key in unencrypted format is NOT supported.

Keys are stored under <DATADIR>/keystore.
It is safe to transfer the entire directory or the individual keys therein
between ethereum nodes by simply copying.

Make sure you backup your keys regularly.`,
		Subcommands: []cli.Command{
			{
				Name:   "list",
				Usage:  "Print summary of existing accounts",
				Action: utils.MigrateFlags(accountList),
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
				},
				Description: `
Print a short summary of all accounts`,
			},
			{
				Name:   "new",
				Usage:  "Create a new account",
				Action: utils.MigrateFlags(accountCreate),
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.PasswordFileFlag,
					utils.LightKDFFlag,
				},
				Description: `
    geth account new

Creates a new account and prints the address.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the --password flag:

Note, this is meant to be used for testing only, it is a bad idea to save your
password to file or expose in any other way.
`,
			},
			{
				Name:      "update",
				Usage:     "Update an existing account",
				Action:    utils.MigrateFlags(accountUpdate),
				ArgsUsage: "<address>",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.LightKDFFlag,
				},
				Description: `
    geth account update <address>

Update an existing account.

The account is saved in the newest version in encrypted format, you are prompted
for a passphrase to unlock the account and another to save the updated file.

This same command can therefore be used to migrate an account of a deprecated
format to the newest format or change the password for an account.

For non-interactive use the passphrase can be specified with the --password flag:

    geth account update [options] <address>

Since only one password can be given, only format update can be performed,
changing your password is only possible interactively.
`,
			},
			{
				Name:   "import",
				Usage:  "Import a private key into a new account",
				Action: utils.MigrateFlags(accountImport),
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
					utils.PasswordFileFlag,
					utils.LightKDFFlag,
				},
				ArgsUsage: "<keyFile>",
				Description: `
    geth account import <keyfile>

Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.

The keyfile is assumed to be a json file which contains an unencrypted private key pair in hexencoded string format.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the -password flag:

    geth account import [options] <keyfile>

Note:
As you can directly copy your encrypted accounts to another ethereum instance,
this import mechanism is not needed when you transfer an account between
nodes.
`,
			},
			{
				Name:   "pubkeys",
				Usage:  "Print public keys",
				Action: utils.MigrateFlags(showPublicKey),
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
				},
				Description: `
Print public key of an address`,
			},
			{
				Name:      "encrypt",
				Usage:     "Encrypt an existing account with AWS KMS",
				Action:    utils.MigrateFlags(accountEncrypt),
				ArgsUsage: "<address>",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
				},
				Description: `
    gwan account encrypt <address>

Encrypt an existing account.

The account will be encrypted by AWS KMS, and ciphertext will be saved into new file named as "<original-name>-cipher"
`,
			},
			{
				Name:      "decrypt",
				Usage:     "Decrypt an existing AWS KMS encrypted account",
				Action:    utils.MigrateFlags(accountDecrypt),
				ArgsUsage: "<address>",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
				},
				Description: `
    gwan account decrypt <address>

Decrypt an existing account.

The account will be decrypted by AWS KMS, and plaintext will be saved into new file named as "<original-name>"
`,
			},
		},
	}
)

func accountList(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	var index int
	for _, wallet := range stack.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			fmt.Printf("Account #%d: {%x} %s\n", index, account.Address, &account.URL)
			index++
		}
	}
	return nil
}

// tries unlocking the specified account a few times.
func unlockAccount(ctx *cli.Context, ks *keystore.KeyStore, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := utils.MakeAddress(ks, address)
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

func unlockAccountFromAwsKmsFile(ctx *cli.Context, ks *keystore.KeyStore, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := utils.MakeAddress(ks, address)
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}

	a, err := ks.Find(account)
	if err != nil {
		utils.Fatalf("Could not find the account: %v", err)
	}

	var trials int
	var keyjson []byte
	for ; trials < 3; trials++ {
		prompt := fmt.Sprintf("AWS KMS decrypting account %s | Attempt %d/%d", address, trials+1, 3)
		keyNames := [3]string{"aKID", "secretKey", "region"}
		kmsInfo, err := getAwsKmsSecretInfo(prompt, keyNames[:])
		if err != nil {
			utils.Fatalf("Failed to read input: %v", err)
		}

		keyjson, err = awskms.DecryptFileToBuffer(a.URL.Path, kmsInfo[0], kmsInfo[1], kmsInfo[2])
		if err != nil {
			fmt.Println("invalid AWS KMS info, decrypt keystore file fail: ", err)
			continue
		}

		break
	}

	if trials == 3 || len(keyjson) == 0 {
		utils.Fatalf("AWS KMS decrypt failed")
	}

	fmt.Println("AWS KMS decrypt successful")
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		err = ks.UnlockMemKey(account, keyjson, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}

	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account %s (%v)", address, err)
	return accounts.Account{}, ""
}

// getPassPhrase retrieves the password associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError, auth string) accounts.Account {
	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your passphrase against all of them...")
	var match *accounts.Account
	for _, a := range err.Matches {
		if err := ks.Unlock(a, auth); err == nil {
			match = &a
			break
		}
	}
	if match == nil {
		utils.Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your passphrase unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, you need to remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}

// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	account, err := ks.NewAccount(password)
	if err != nil {
		utils.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("Address: {%s}\n", account.Address.Hex()[2:])
	return nil
}

// accountUpdate transitions an account from a previous format to the current
// one, also providing the possibility to change the pass-phrase.
func accountUpdate(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		utils.Fatalf("No accounts specified to update")
	}
	stack, _ := makeConfigNode(ctx)
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	for _, addr := range ctx.Args() {
		account, oldPassword := unlockAccount(ctx, ks, addr, 0, nil)
		newPassword := getPassPhrase("Please give a new password. Do not forget this password.", true, 0, nil)
		if err := ks.Update(account, oldPassword, newPassword); err != nil {
			utils.Fatalf("Could not update the account: %v", err)
		}
	}
	return nil
}

func importWallet(ctx *cli.Context) error {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	keyJson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		utils.Fatalf("Could not read wallet file: %v", err)
	}

	stack, _ := makeConfigNode(ctx)
	passphrase := getPassPhrase("", false, 0, utils.MakePasswordList(ctx))

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acct, err := ks.ImportPreSaleKey(keyJson, passphrase)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	fmt.Printf("Address: {%x}\n", acct.Address)
	return nil
}

func accountImport(ctx *cli.Context) error {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	key, key1, err := keystore.LoadECDSAPair(keyfile)
	if err != nil {
		utils.Fatalf("Failed to load the private key: %v", err)
	}
	stack, _ := makeConfigNode(ctx)
	passphrase := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acct, err := ks.ImportECDSA(key, key1, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: {%s}\n", acct.Address.Hex()[2:])
	return nil
}

func showPublicKey(ctx *cli.Context) error {
	addrstr := ctx.Args().Get(0)
	passwd := ctx.Args().Get(1)
	if len(addrstr) == 0 {
		utils.Fatalf("address must be given as argument")
	}
	if len(passwd) == 0 {
		utils.Fatalf("passwd must be given as argument")
	}

	stack, _ := makeConfigNode(ctx)
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	addr := common.HexToAddress(addrstr)
	all := ks.Accounts()
	lenth := len(all)
	for i := 0; i < lenth; i++ {
		if all[i].Address == addr {
			key, err := ks.GetKey(all[i], passwd)
			if err != nil {
				utils.Fatalf("Error failed to load keyfile ")
			}
			if key.PrivateKey != nil {
				fmt.Println("key1:" + common.ToHex(crypto.FromECDSAPub(&key.PrivateKey.PublicKey)))
			}
			if key.PrivateKey2 != nil {
				fmt.Println("key2:" + common.ToHex(crypto.FromECDSAPub(&key.PrivateKey2.PublicKey)))
				fmt.Println("waddress:" + common.ToHex(key.WAddress[:]))
			}
			D3 := posconfig.GenerateD3byKey2(key.PrivateKey2)
			G1 := new(bn256.G1).ScalarBaseMult(D3)
			fmt.Println("key3:" + common.ToHex(G1.Marshal()))
			break
		}
	}
	return nil
}

// accountEncrypt encrypt an account using AWS KMS,
// and save ciphertext into new file named as "<original-name>-cipher"
func accountEncrypt(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		utils.Fatalf("No accounts specified to encrypt")
	}

	keyNames := [4]string{"aKID", "secretKey", "region", "keyId"}
	keyVals, err := getAwsKmsSecretInfo("", keyNames[:])
	if err != nil {
		return err
	}

	fmt.Println("begin encrypting...")
	stack, _ := makeConfigNode(ctx)
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	for _, addr := range ctx.Args() {
		exceptAddr := common.HexToAddress(addr)
		a := accounts.Account{Address:exceptAddr}
		fa, err := ks.Find(a)
		if err != nil {
			return err
		}

		desFile := fa.URL.Path + keystore.AwsKMSCiphertextFileExt
		err = awskms.EncryptFile(fa.URL.Path, desFile, keyVals[0], keyVals[1], keyVals[2], keyVals[3])
		if err != nil {
			return err
		}

		fmt.Println("encrypt account(",  addr, ") successfully into new keystore file : ", desFile)
	}

	return nil
}

// accountDecrypt decrypt an account using AWS KMS,
// and save ciphertext into new file named as "<original-name>-plain"
func accountDecrypt(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		utils.Fatalf("No accounts specified to decrypt")
	}

	keyNames := [3]string{"aKID", "secretKey", "region"}
	keyVals, err := getAwsKmsSecretInfo("", keyNames[:])
	if err != nil {
		return err
	}

	fmt.Println("begin decrypting...")
	stack, _ := makeConfigNode(ctx)
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	for _, addr := range ctx.Args() {
		exceptAddr := common.HexToAddress(addr)
		a := accounts.Account{Address:exceptAddr}
		fa, err := ks.Find(a)
		if err != nil {
			return err
		}

		desFile := ""
		pot := strings.LastIndex(fa.URL.Path, keystore.AwsKMSCiphertextFileExt)
		if pot != -1 {
			desFile = fa.URL.Path[:pot]
		} else {
			desFile = fa.URL.Path + "-plain"
		}

		err = awskms.DecryptFile(fa.URL.Path, desFile, keyVals[0], keyVals[1], keyVals[2])
		if err != nil {
			return err
		}

		fmt.Println("decrypt account(",  addr, ") successfully into new keystore file : ", desFile)
	}

	return nil
}

func getAwsKmsSecretInfo(notice string, items []string) ([]string, error) {
	inputs := make([]string, len(items))
	fmt.Println(notice)
	for i, name := range items {
		input, err := console.Stdin.PromptPassword(name + ": ")
		if err != nil {
			return nil, err
		}

		inputs[i] = input
	}

	return inputs, nil
}

