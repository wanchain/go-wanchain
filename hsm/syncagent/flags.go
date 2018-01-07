// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package main

import (
	cli "gopkg.in/urfave/cli.v1"
	"errors"
)

type AgentContext struct {
	Address string
	NodePIN string
	EnvPWD string
	File string
}

var (
	AddressFlag = cli.StringFlag{
		Name:  "address",
		Usage: "Node address to connect",
	}
	
	NodePINFlag = cli.StringFlag{
		Name:  "nodePIN",
		Usage: "NodePIN to be checked in Node",
	}
	
	EnvPWDFlag = cli.StringFlag{
		Name:  "envPWD",
		Usage: "Envelope password to decrypt",
	}
	
	FileFlag = cli.StringFlag{
		Name:  "file",
		Usage: "File path of the envelope keystore",
	}
)

func ParseAgentContext(c *cli.Context) (*AgentContext, error) {
	ac := new (AgentContext)
	ac.Address = c.String(AddressFlag.GetName())
	ac.NodePIN = c.String(NodePINFlag.GetName())
	ac.EnvPWD = c.String(EnvPWDFlag.GetName())
	ac.File = c.String(FileFlag.GetName())
	
	if ac.Address == "" || ac.NodePIN == "" || ac.File == "" || ac.EnvPWD == "" {
		return nil, errors.New("Invalid Arguments!")
	}
	
	return ac, nil
}