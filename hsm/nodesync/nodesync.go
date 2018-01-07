// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package nodesync

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"github.com/wanchain/go-wanchain/hsm/nodeUtil"
	"github.com/wanchain/go-wanchain/hsm/syncFile"
	"github.com/wanchain/go-wanchain/accounts/keystore"
	"crypto/ecdsa"
	"os/user"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"io/ioutil"
)

var
(
	  NodeSignKey *ecdsa.PrivateKey
)

func Nodesync(ctx *cli.Context)(error) {

	nc, err := ParseNodeContext(ctx)
	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return err
	}

	usr,err :=  user.Current()
	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return err
	}

	//Support TLS with pem and key file.
	keyjson, err := nodeUtil.StartSyncFile(nc.Address, nc.NodePIN,  usr.HomeDir +syncFile.SYNC_NODE_CERT, usr.HomeDir + syncFile.SYNC_NODE_KEY)
	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return err
	}

	keyPwd, err := ioutil.ReadFile(nc.pwd)
	if err != nil {
		return err
	}

	strpwd := string(keyPwd)

	ks,err := keystore.DecryptKey(keyjson,strpwd)
	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return err
	}

	NodeSignKey = ks.PrivateKey


	return nil
}

func GetSinger() common.Address {

	var signer common.Address
	pubkey := crypto.FromECDSAPub(&NodeSignKey.PublicKey)
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	fmt.Println(common.ToHex(signer[:]))

	return signer
}