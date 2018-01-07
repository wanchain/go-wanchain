// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package nodeUtil

import (
	"github.com/wanchain/go-wanchain/hsm/syncFile"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"time"
	"errors"
	"sync"
)

var (
	isTimeout bool
	isStopped bool
	mutex sync.Mutex
)

// server is used to implement syncFile.HsmSyncServer.
type syncServer struct {
	nodePIN string
	fileData []byte
	realServer *grpc.Server
}

func (s *syncServer) triggerServerStop(timeout bool) {
	mutex.Lock()
	if isStopped == false {
		isStopped = true
		isTimeout = timeout
		mutex.Unlock()
		s.realServer.GracefulStop()
	} else {
		mutex.Unlock()
	}
}

func (s *syncServer) triggerTimeoutMonitor() {
	time.Sleep(1000000000*60*10) //10 mins.
	s.triggerServerStop(true)
}

// SyncFile implements syncFile.HsmSyncServer.
func (s *syncServer) SyncFile(ctx context.Context, in *syncFile.SyncFileRequest) (*syncFile.SyncFileReply, error) {
	
	if s.nodePIN != in.GetNodePIN() {
		return &syncFile.SyncFileReply{ResCode: syncFile.SYNCFILE_ERROR_PIN}, nil
	}
	
	s.fileData = in.GetFileData()
	if len(s.fileData) < 1 {
		return &syncFile.SyncFileReply{ResCode: syncFile.SYNCFILE_ERROR_FILEDATA}, nil
	}

	go s.triggerServerStop(false)
	return &syncFile.SyncFileReply{ResCode: syncFile.SYNCFILE_ERROR_NONE}, nil
}



// Main entry to perform the sync file and retrieve the file data.
// TBD. 
func StartSyncFile(address string, nodePIN string, pemFile string, keyFile string) ([]byte, error) {
	
	lis, err := net.Listen("tcp", address + syncFile.SYNC_PORT)
	if err != nil {
		return nil, err
	}
	
	//Support TLS for gRPC communication.
	
	creds, err := credentials.NewServerTLSFromFile(pemFile, keyFile)
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer(grpc.Creds(creds))
	
	
	//s := grpc.NewServer()
	
	var syncserver *syncServer = new(syncServer)
	syncserver.nodePIN = nodePIN
	syncserver.fileData = nil
	syncserver.realServer = s
	
	syncFile.RegisterHsmSyncServer(s, syncserver)
	
	//Ensure the sync should be finished in 10 mins.
	isStopped = false
	go syncserver.triggerTimeoutMonitor()
	
	s.Serve(lis)
	
	if isTimeout == true {
		return nil, errors.New("sync is timeout")
	}
	return syncserver.fileData, nil
}
