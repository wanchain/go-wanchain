// Copyright 2017 The go-ethereum Authors
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
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"unicode"
	"bytes"
	"strings"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/naoina/toml"
	"github.com/wanchain/go-wanchain/cmd/utils"
	"github.com/wanchain/go-wanchain/contracts/release"
	"github.com/wanchain/go-wanchain/eth"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/node"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/storeman"
	"github.com/wanchain/go-wanchain/storeman/storemanmpc"
	whisper "github.com/wanchain/go-wanchain/whisper/whisperv5"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/common"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(append(nodeFlags, rpcFlags...), whisperFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type ethstatsConfig struct {
	URL string `toml:",omitempty"`
}

type gethConfig struct {
	Eth      eth.Config
	Shh      whisper.Config
	Sm       storeman.Config
	Node     node.Config
	Ethstats ethstatsConfig
}

func loadConfig(file string, cfg *gethConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "eth", "shh")
	cfg.HTTPModules = append(cfg.HTTPModules, "wan", "shh")
	cfg.HTTPModules = append(cfg.HTTPModules, "storeman")
	cfg.WSModules = append(cfg.WSModules, "eth", "shh")
	cfg.WSModules = append(cfg.WSModules, "wan", "shh")
	cfg.IPCPath = "gwan.ipc"
	return cfg
}

func makeConfigNode(ctx *cli.Context) (*node.Node, gethConfig) {
	// Load defaults.
	cfg := gethConfig{
		Eth:  eth.DefaultConfig,
		Shh:  whisper.DefaultConfig,
		Node: defaultNodeConfig(),
		Sm:   storeman.DefaultConfig,
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	cfg.Sm.StoremanNodes = cfg.Node.P2P.StoremanNodes
	if ctx.GlobalIsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.GlobalString(utils.EthStatsURLFlag.Name)
	}

	utils.SetShhConfig(ctx, stack, &cfg.Shh)

	return stack, cfg
}

// enableWhisper returns true in case one of the whisper flags is set.
func enableWhisper(ctx *cli.Context) bool {
	for _, flag := range whisperFlags {
		if ctx.GlobalIsSet(flag.GetName()) {
			return true
		}
	}
	return false
}

func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)

	utils.RegisterEthService(stack, &cfg.Eth)

	// Whisper must be explicitly enabled by specifying at least 1 whisper flag or in dev mode
	shhEnabled := enableWhisper(ctx)
	shhAutoEnabled := !ctx.GlobalIsSet(utils.WhisperEnabledFlag.Name) && ctx.GlobalIsSet(utils.DevModeFlag.Name)
	if shhEnabled || shhAutoEnabled {
		if ctx.GlobalIsSet(utils.WhisperMaxMessageSizeFlag.Name) {
			cfg.Shh.MaxMessageSize = uint32(ctx.Int(utils.WhisperMaxMessageSizeFlag.Name))
		}
		if ctx.GlobalIsSet(utils.WhisperMinPOWFlag.Name) {
			cfg.Shh.MinimumAcceptedPOW = ctx.Float64(utils.WhisperMinPOWFlag.Name)
		}
		utils.RegisterShhService(stack, &cfg.Shh)
	}

	// Add the Ethereum Stats daemon if requested.
	if cfg.Ethstats.URL != "" {
		utils.RegisterEthStatsService(stack, cfg.Ethstats.URL)
	}

	if ctx.GlobalIsSet(utils.StoremanFlag.Name) {
		cfg.Sm.DataPath = cfg.Node.DataDir
		enableKms := ctx.GlobalIsSet(utils.AwsKmsFlag.Name)

		var status int
		var suc bool

		kmsInfo := &storemanmpc.KmsInfo{}
		if enableKms {
			kmsInfo = getKmsInfo()
		}

		password := getPassword(ctx, false)
		verify, accounts := getVerifyAccounts()
		if verify {
			suc, status = verifySecurityInfo(stack, enableKms, kmsInfo, password, accounts)
			for !suc  {
				log.Error("verify security info fail, please input again")
				if status == 0x00 || (status == 0x01 && !enableKms) {
					verify, accounts = getVerifyAccounts()
				} else if status == 0x01 {
					kmsInfo = getKmsInfo()
				} else {
					password = getPassword(ctx, true)
				}

				if verify {
					suc, status = verifySecurityInfo(stack, enableKms, kmsInfo, password, accounts)
				} else {
					suc = true
				}
			}
		}

		cfg.Sm.Password = password
		utils.RegisterSmService(stack, &cfg.Sm, kmsInfo.AKID, kmsInfo.SecretKey, kmsInfo.Region)
	}

	// Add the release oracle service so it boots along with node.
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		config := release.Config{
			Oracle: relOracle,
			Major:  uint32(params.VersionMajor),
			Minor:  uint32(params.VersionMinor),
			Patch:  uint32(params.VersionPatch),
		}
		commit, _ := hex.DecodeString(gitCommit)
		copy(config.Commit[:], commit)
		return release.NewReleaseService(ctx, config)
	}); err != nil {
		utils.Fatalf("Failed to register the Geth release oracle service: %v", err)
	}

	return stack
}


func verifySecurityInfo(node *node.Node, enableKms bool, info *storemanmpc.KmsInfo, password string, accounts []string) (bool, int) {

	fmt.Println("")
	fmt.Println("should verify ", len(accounts), " accounts")
	fmt.Println("begin verify keystore file security info...")
	ks := node.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	for _, acc := range accounts {
		fmt.Println("begin verify ", acc)

		_, status, err := storemanmpc.GetPrivateShare(ks, common.HexToAddress(acc), enableKms, info, password)
		if err == nil {
			fmt.Println("verify keystore file security info succeed")
			continue
		} else {
			fmt.Println("verify keystore file security info fail. err:", err.Error())
			return false, status
		}
	}

	return true, 0
}


func getKmsInfo() *storemanmpc.KmsInfo {
	ret := storemanmpc.KmsInfo{}
	fmt.Println("")

	// aws kms info
	var keyVal [3]string
	keyName := [3]string{"aKID", "secretKey", "region"}
	fmt.Println("begin collect AWS KMS info and keystore file password...")
	for i, name := range keyName {
		fmt.Println("please input aws kms ", name, ":")
		pswd, err := terminal.ReadPassword(0)
		if err != nil {
			log.Error("get AWS KMS info fail", "err", err)
			return &ret
		}

		keyVal[i] = string(pswd)
	}

	ret.AKID = keyVal[0]
	ret.SecretKey = keyVal[1]
	ret.Region = keyVal[2]
	return &ret
}

func getPassword(ctx *cli.Context, retry bool) string {
	// keystore file password : read config password file or manual input
	fmt.Println("")
	var confirmed string
	cfgPasswords := utils.MakePasswordList(ctx)
	if len(cfgPasswords) > 0 && len(cfgPasswords[0]) > 0 && !retry {
		fmt.Println("get password from file")
		confirmed = cfgPasswords[0]
	} else {
		fmt.Println("begin collect keystore file password...")

		for ; ;  {
			fmt.Println("please input keystore file  password :")
			pswd, err := terminal.ReadPassword(0)
			if err != nil {
				log.Error("get AWS KMS info fail", "err", err)
				return ""
			}

			fmt.Println("confirm  password :")
			pswdCfm, err := terminal.ReadPassword(0)
			if err != nil {
				log.Error("get AWS KMS info fail", "err", err)
				return ""
			}

			if !bytes.Equal(pswd, pswdCfm) {
				fmt.Println("those passwords didn't match. please try again...")
				continue
			} else if len(pswd) == 0 {
				fmt.Println("the password can't be empty. please try again...")
				continue
			} else {
				confirmed = string(pswd)
				break
			}
		}
	}

	return confirmed
}

func getVerifyAccounts() (bool, []string) {
	fmt.Println("")
	reader := bufio.NewReader(os.Stdin)
	var accounts []string

	for ; ;  {
		fmt.Println("please input accounts to verify (like: addr1 addr2. noinput means don't verify):")
		read, _, _ := reader.ReadLine()
		if len(read) == 0 {
			return false, accounts
		}

		splits := strings.Split(string(read), " ")
		for _, addr := range splits {
			if len(addr) == (common.AddressLength*2+2) {
				accounts = append(accounts, addr)
			}
		}

		if len(accounts) == 0 {
			fmt.Println("invalid account info. please try again")
			continue
		} else {
			break
		}
	}

	return true, accounts
}


// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.Eth.Genesis != nil {
		cfg.Eth.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}
	io.WriteString(os.Stdout, comment)
	os.Stdout.Write(out)
	return nil
}
