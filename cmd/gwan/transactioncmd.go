// Copyright 2018 Wanchain Foundation Ltd
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
	"bytes"
	"errors"
	"github.com/wanchain/go-wanchain/cmd/utils"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/rlp"
	"gopkg.in/urfave/cli.v1"
	"os"
)

var (
	transactionCommand = cli.Command{
		Name:      "transaction",
		Usage:     "Manage transactions in chain",
		ArgsUsage: "",
		Category:  "TRANSACTION COMMANDS",
		Description: `
    geth --datadir ./data transaction export 0x1111111111111111111111111111111111111111111111111111111111111111 ./tx.json

will export the transaction using setting as hash from datadir chain, and save as ./tx.json .`,
		Subcommands: []cli.Command{
			{
				Name:     "export",
				Usage:    "export transaction save as json file",
				Action:   utils.MigrateFlags(exportTransaction),
				Category: "TRANSACTION COMMANDS",
				Flags:    []cli.Flag{},
				Description: `
	geth --datadir ./data transaction export 0x1111111111111111111111111111111111111111111111111111111111111111 ./tx.json

will export the transaction using setting as hash from datadir chain, and save as ./tx.json .`,
			},

			{
				Name:     "exportraw",
				Usage:    "exportraw transaction save as json file",
				Action:   utils.MigrateFlags(exportTransactionRaw),
				Category: "TRANSACTION COMMANDS",
				Flags:    []cli.Flag{},
				Description: `
	geth --datadir ./data transaction export 0x1111111111111111111111111111111111111111111111111111111111111111 ./tx.dat

will export the transaction using setting as hash from datadir chain, and save as ./tx.raw .`,
			},

			{
				Name:     "exporthex",
				Usage:    "exporthex transaction save as json file",
				Action:   utils.MigrateFlags(exportTransactionHex),
				Category: "TRANSACTION COMMANDS",
				Flags:    []cli.Flag{},
				Description: `
	geth --datadir ./data transaction export 0x1111111111111111111111111111111111111111111111111111111111111111 ./tx.txt

will export the transaction using setting as hash from datadir chain, and save as ./tx.txt .`,
			},
		},
	}
)

func exportTransaction(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) < 2 {
		return errors.New("args count not enough")
	}

	hashStr := args[0]
	filePath := args[1]

	hash := common.HexToHash(hashStr)
	stack := makeFullNode(ctx)
	_, chainDb := utils.MakeChain(ctx, stack)

	// Try to return an already finalized transaction
	var tx *types.Transaction
	if tx, _, _, _ = core.GetTransaction(chainDb, hash); tx == nil {
		return errors.New("no found tx")
	}

	out, err := tx.MarshalJSON()
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = fh.Write(out)
	return err
}

func exportTransactionRaw(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) < 2 {
		return errors.New("args count not enough")
	}

	hashStr := args[0]
	filePath := args[1]

	hash := common.HexToHash(hashStr)
	stack := makeFullNode(ctx)
	_, chainDb := utils.MakeChain(ctx, stack)

	// Try to return an already finalized transaction
	var tx *types.Transaction
	if tx, _, _, _ = core.GetTransaction(chainDb, hash); tx == nil {
		return errors.New("no found tx")
	}

	buf := new(bytes.Buffer)
	if err := rlp.Encode(buf, tx); err != nil {
		return err
	}

	fh, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = fh.Write(buf.Bytes())
	return err
}

func exportTransactionHex(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) < 2 {
		return errors.New("args count not enough")
	}

	hashStr := args[0]
	filePath := args[1]

	hash := common.HexToHash(hashStr)
	stack := makeFullNode(ctx)
	_, chainDb := utils.MakeChain(ctx, stack)

	// Try to return an already finalized transaction
	var tx *types.Transaction
	if tx, _, _, _ = core.GetTransaction(chainDb, hash); tx == nil {
		return errors.New("no found tx")
	}

	buf := new(bytes.Buffer)
	if err := rlp.Encode(buf, tx); err != nil {
		return err
	}

	fh, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = fh.Write([]byte(common.ToHex(buf.Bytes())))
	return err
}
