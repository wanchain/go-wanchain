// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package agentUtil

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"github.com/wanchain/go-wanchain/hsm/syncFile"
	"google.golang.org/grpc/credentials"
)

func SyncFile(address string, nodePIN string, fileData []byte) error {

	//Support TLS for gRPC communication.
	
	creds, err := credentials.NewClientTLSFromFile("./server.pem", "")
	if err != nil {
		return err
	}
	conn, err := grpc.Dial(address + syncFile.SYNC_PORT, grpc.WithTransportCredentials(creds))
	
	//conn, err := grpc.Dial(address, grpc.WithInsecure())
	
	if err != nil {
		return &DiaError{address + syncFile.SYNC_PORT}
	}
	defer conn.Close()
	c := syncFile.NewHsmSyncClient(conn)

	// Contact the server and print out its response.
	r, err := c.SyncFile(context.Background(), &syncFile.SyncFileRequest{NodeAddr: address, NodePIN: nodePIN, FileData: fileData})

	if err != nil {
		return err
	}

	if r.ResCode == syncFile.SYNCFILE_ERROR_NONE {
		return nil
	} else {
		return &SyncFileError{r.ResCode}
	}
}
