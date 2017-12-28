// Copyright 2017 Wanglu.
//
// Author: zhu.zhengming@wanglutech.com.

package fileHandler

import (
	"bytes"
	"github.com/yeka/zip"
	"io"
	"os"
	"errors"
)

func EncryptWriteZip(contents []byte, fileName string, envPwd string) error {

	//create 
	fzip, err := os.Create(fileName+".zip")
	if err != nil {
		return err
	}

	zipw := zip.NewWriter(fzip)
	defer zipw.Close()
	
	//set zipw to use encrypt.
	w, err := zipw.Encrypt(fileName, envPwd, zip.AES256Encryption)
	if err != nil {
		return err
	}
	_, err1 := io.Copy(w, bytes.NewReader(contents))
	if err1 != nil {
		return err
	}

	zipw.Flush()
	return nil
}

func DecryptReadZip(fileName string, envPwd string) ([]byte, error) {

	var buf bytes.Buffer
	
	r, err := zip.OpenReader(fileName)
	if err != nil {
		return nil, errors.New("Fail to open file.")
	}
	defer r.Close()

	//Assume only one encrypted file inclued in the envlope.
	if len(r.File) != 1 {
		return nil, errors.New("Include not one key file in the envelope file.")
	}
		
	f := r.File[0]
	f.SetPassword(envPwd)
	
	//open the file to read and decrypt.
	rc, err := f.Open()
	if err != nil {
		return nil, errors.New("Fail to open and decrypt file.")
	}
	_, err = io.Copy(&buf, rc)
	if err != nil {
		return nil, errors.New("Fail to copy data.")
	}

	return buf.Bytes(), nil
}
