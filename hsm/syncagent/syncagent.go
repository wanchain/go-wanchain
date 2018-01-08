// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package main

import (
	"fmt"
	"github.com/wanchain/go-wanchain/hsm/agentUtil"
	"github.com/wanchain/go-wanchain/hsm/fileHandler"
	"gopkg.in/urfave/cli.v1"
//	"io/ioutil"
)

func handleSyncFile(ac *AgentContext) error {
	buf, err := fileHandler.DecryptReadZip(ac.File, ac.EnvPWD)
	if err != nil {
		return err
	}
	
	err1 := agentUtil.SyncFile(ac.Address, ac.NodePIN, buf)
	if err1 != nil {
		return err1
	}
	
	return nil
}

func syncagent(ctx *cli.Context) error {
	ac, err := ParseAgentContext(ctx)
	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return err
	}


	err = handleSyncFile(ac)
	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return err
	}
	
	fmt.Printf("Complete successfully!\n\n")
	return nil
}
