// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package main

import (
	"gopkg.in/urfave/cli.v1"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "syncagent"
	app.Usage = "use syncagent to sync keystore file to node!"
	app.Flags = []cli.Flag{
		AddressFlag,
		NodePINFlag,
		EnvPWDFlag,
		FileFlag,
	}
	
	app.Action = syncagent

	app.Run(os.Args)

}
