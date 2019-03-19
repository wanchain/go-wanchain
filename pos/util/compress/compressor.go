package compress

import (
	"bytes"
	"compress/gzip"
	"time"
	"log"
	"fmt"
)

func Compress(data []byte)([]byte,error)  {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	// Setting the Header fields is optional.
	zw.Name = "compress"
	zw.Comment = "pos"
	zw.ModTime = time.Now()

	_, err := zw.Write(data)
	if err != nil {
		return nil,err
	}

	if err := zw.Close(); err != nil {
		return nil,err
	}

	return  buf.Bytes(),nil
}

func Uncompress(data []byte)([]byte,error)  {
	var buf bytes.Buffer
	buf.Write(data)

	zr, err := gzip.NewReader(&buf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\nComment: %s\nModTime: %s\n\n", zr.Name, zr.Comment, zr.ModTime.UTC())

	var ucd bytes.Buffer
	if _,err = ucd.ReadFrom(zr);err != nil {
		return nil,err
	}

	if err := zr.Close(); err != nil {
		return nil,err
	}

	return ucd.Bytes(),nil
}


