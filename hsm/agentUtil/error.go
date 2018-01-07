// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package agentUtil

import (
	"fmt"
	"github.com/wanchain/go-wanchain/hsm/syncFile"
)

type DiaError struct {
	Address string
}

type DecryptError struct {
}

type SyncFileError struct {
	ErrNo int32
}

func (de *DiaError) Error() string {
	strFormat := "Cannot connect address: %s"
	return fmt.Sprintf(strFormat, de.Address)
}

func (dee *DecryptError) Error() string {
	return "File Decrypt Error"
}

func (sfe *SyncFileError) Error() string {
	switch sfe.ErrNo {
		case syncFile.SYNCFILE_ERROR_PIN:
			return "Incorrect PIN"
		case syncFile.SYNCFILE_ERROR_FILEDATA:
			return "Incorrect File Data"
		default:
			return "Unknown Error"	
	}
}