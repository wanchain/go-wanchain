package compress
//
//import (
//	"github.com/ulikunitz/xz"
//	"log"
//	"bytes"
//)
//
//func LZMACompress(data []byte)([]byte,error)  {
//	var buf bytes.Buffer
//	// compress text
//	w, err := xz.NewWriter(&buf)
//	if err != nil {
//		log.Fatalf("xz.NewWriter error %s", err)
//		return nil,err
//	}
//
//	if _, err := w.Write(data); err != nil {
//		log.Fatalf("WriteString error %s", err)
//		return nil,err
//	}
//
//	if err := w.Close(); err != nil {
//		log.Fatalf("w.Close error %s", err)
//		return nil,err
//	}
//
//	return buf.Bytes(),nil
//}
//
//func LZMAUncompress(data []byte)([]byte,error)  {
//
//	var buf bytes.Buffer
//	buf.Write(data)
//
//	lzmzRd, err := xz.NewReader(&buf)
//	if err != nil {
//		log.Fatalf("NewReader error %s", err)
//		return nil,err
//	}
//
//	var ucd bytes.Buffer
//	if _,err = ucd.ReadFrom(lzmzRd);err != nil {
//		return nil,err
//	}
//
//	return ucd.Bytes(),nil
//
//}