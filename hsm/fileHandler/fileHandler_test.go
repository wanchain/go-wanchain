// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package fileHandler

import (
	"testing"
)

func TestFileHandler(t *testing.T) {
    err := EncryptWriteZip([]byte("aaaaaaaaaaaaaaaabbbbbbbbbb"), "./mytestkey.txt", "111111")
	if err != nil {
		t.Errorf("Write Error: %s\n", err)
	}
	
	
	buf, err := DecryptReadZip("./mytestkey.txt.zip", "111111")
	if err != nil {
		t.Errorf("Read Error: %s\n", err)
	}
	
	if string(buf) != "aaaaaaaaaaaaaaaabbbbbbbbbb" {
		t.Errorf("Encrypted data is not correctly decrypted\n")
	}
}
