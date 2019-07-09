package awskms

//import (
//	"testing"
//)

//func TestKms(t *testing.T) {
//
//	var aKID = "**************"
//	var secretKey = "***********************"
//	var region = "******************"
//	var keyId = "*************************"
//	var plaintextOrg = "Hello, AWS KMS"
//
//	ciphertext, err := Encrypt(plaintextOrg, aKID, secretKey, region, keyId)
//	if err != nil {
//		t.Errorf("encrypt fail. err:%s", err.Error())
//	}
//
//	t.Logf("get ciphertext:%s", string(ciphertext))
//	plaintext, err := Decrypt(ciphertext, aKID, secretKey, region)
//	if err != nil {
//		t.Errorf("decrypt fail. err:%s", err.Error())
//	}
//
//	t.Logf("get plaintext:%s", plaintext)
//	if plaintextOrg != string(plaintext) {
//		t.Errorf("decrypt fail. expected plaintext:%s, getting plaintext:%s", plaintextOrg, plaintext)
//	}
//}
