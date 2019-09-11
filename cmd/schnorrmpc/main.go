// Copyright 2018 Wanchain Foundation Ltd
// Copyright 2014 The go-ethereum Authors
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

// geth is the official command-line client for Ethereum.
package main

import (
	"fmt"
	"github.com/wanchain/go-wanchain/internal/debug"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/wanchain/go-wanchain/accounts/keystore"
	"github.com/wanchain/go-wanchain/cmd/utils"
	"github.com/wanchain/go-wanchain/console"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/metrics"
	"github.com/wanchain/go-wanchain/node"
	"gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "schnorrmpc" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	// Ethereum address of the Geth release oracle.
	//relOracle = common.HexToAddress("0x6b4683cafa549d9f4c06815a2397cef5a540b919")
	// The app that holds all commands and flags.
	app = utils.NewApp(gitCommit, "the go-wanchain command line interface")
	// flags that configure the node
	nodeFlags = []cli.Flag{
		//utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.KeyStoreDirFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.NoDiscoverFlag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.EthStatsURLFlag,
		configFileFlag,
		utils.StoremanFlag,
		utils.AwsKmsFlag,
	}

	rpcFlags = []cli.Flag{
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.RPCApiFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
	}

	whisperFlags = []cli.Flag{
		utils.WhisperEnabledFlag,
		utils.WhisperMaxMessageSizeFlag,
		utils.WhisperMinPOWFlag,
	}

	syslogFlags = []cli.Flag{
		utils.SysLogFlag,
		utils.SyslogNetFlag,
		utils.SyslogSvrFlag,
		utils.SyslogLevelFlag,
		utils.SyslogTagFlag,
	}
)

func init() {

	// Initialize the CLI app and start Geth
	app.Action = schnorrStart
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2017 The go-ethereum Authors; Copyright 2018 Wanchain Foundation Ltd"
	app.Commands = []cli.Command{
		// See consolecmd.go:
		consoleCommand,
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		//makecacheCommand,
		//makedagCommand,
		versionCommand,
		//bugCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, syslogFlags...)

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		// Start system runtime metrics collection
		go metrics.CollectProcessMetrics(3 * time.Second)

		utils.SetupNetwork(ctx)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// schnorr is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func schnorrStart(ctx *cli.Context) error {

	node := makeFullNode(ctx)
	startNode(ctx, node)
	node.Wait()
	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node) {

	if ctx.GlobalBool(utils.SysLogFlag.Name) {
		log.InitSyslog(
			ctx.GlobalString(utils.SyslogNetFlag.Name),
			ctx.GlobalString(utils.SyslogSvrFlag.Name),
			ctx.GlobalString(utils.SyslogLevelFlag.Name),
			ctx.GlobalString(utils.SyslogTagFlag.Name))
	}

	// Unlock any account specifically requested
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	passwords := utils.MakePasswordList(ctx)
	unlocks := strings.Split(ctx.GlobalString(utils.UnlockedAccountFlag.Name), ",")
	for i, account := range unlocks {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			if ctx.IsSet(utils.AwsKmsFlag.Name) {
				unlockAccountFromAwsKmsFile(ctx, ks, trimmed, i, passwords)
			} else {
				unlockAccount(ctx, ks, trimmed, i, passwords)
			}
		}
	}

	// Send unlock account finish event
	stack.AccountManager().SendStartupUnlockFinish()

	// Start up the node itself
	utils.StartNode(stack)

}

