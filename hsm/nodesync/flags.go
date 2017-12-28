// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package nodesync

import (
	cli "gopkg.in/urfave/cli.v1"
	"errors"
)

type NodeContext struct {
	Address string
	NodePIN string
	pwd 	string
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

	PwdFlag = cli.StringFlag{
		Name:  "pwd",
		Usage: "password to decrypt keystore",
	}

	mineFlag = cli.StringFlag{
		Name:  "minerthreads",
		Usage: "password to decrypt keystore",
	}

	HsmFlags = []cli.Flag{
		AddressFlag,
		NodePINFlag,
		PwdFlag,
	}
)

func ParseNodeContext(c *cli.Context) (*NodeContext, error) {
	nc := new (NodeContext)
	nc.Address = c.String(AddressFlag.GetName())
	nc.NodePIN = c.String(NodePINFlag.GetName())
	nc.pwd 	   = c.String(PwdFlag.GetName())

	mine 	   := c.String(mineFlag.GetName())

	//not mine node
	if mine == "" {

		return nil, nil

	} else {

		if nc.Address == "" || nc.NodePIN == "" || nc.pwd == "" {
			return nil, errors.New("Invalid arguments!")
		}
	}
	
	return nc, nil
}