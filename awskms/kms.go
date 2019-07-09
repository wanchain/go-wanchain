package awskms

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/wanchain/go-wanchain/log"
	"io/ioutil"
	"os"
	"path/filepath"
)

func EncryptFile(srcFile, desFile, aKID, secretKey, region, keyId string) error {
	keyjson, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return err
	}

	ciphertext, err := Encrypt(string(keyjson), aKID, secretKey, region, keyId)
	if err != nil {
		return err
	}

	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(desFile), dirPerm); err != nil {
		return err
	}

	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(desFile), "."+filepath.Base(desFile)+".tmp")
	if err != nil {
		return err
	}

	wLen, err := f.Write(ciphertext)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}

	if wLen != len(ciphertext) {
		f.Close()
		os.Remove(f.Name())
		return errors.New("The data len writen to file is less than excepted")
	}

	f.Close()
	return os.Rename(f.Name(), desFile)
}

func DecryptFile(srcFile, desFile, aKID, secretKey, region string) error {
	plaintext, err := DecryptFileToBuffer(srcFile, aKID, secretKey, region)
	if err != nil {
		return err
	}

	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(desFile), dirPerm); err != nil {
		return err
	}

	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(desFile), "."+filepath.Base(desFile)+".tmp")
	if err != nil {
		return err
	}

	wLen, err := f.Write(plaintext)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}

	if wLen != len(plaintext) {
		f.Close()
		os.Remove(f.Name())
		return errors.New("The data len writen to file is less than excepted")
	}

	f.Close()
	return os.Rename(f.Name(), desFile)
}

func DecryptFileToBuffer(srcFile, aKID, secretKey, region string) ([]byte, error) {
	keyjson, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return nil, err
	}

	plaintext, err := Decrypt(keyjson, aKID, secretKey, region)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func Encrypt(text, aKID, secretKey, region, keyId string) ([]byte, error) {
	// Initialize a session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(aKID, secretKey, ""),
	})

	if err != nil {
		log.Error("create kms session fail", "err", err)
		return nil, err
	}

	// Create KMS service client
	svc := kms.New(sess)

	// Encrypt the data key
	result, err := svc.Encrypt(&kms.EncryptInput{
		KeyId:     aws.String(keyId),
		Plaintext: []byte(text),
	})

	if err != nil {
		log.Error("kms encrypt fail", "err", err)
		return nil, err
	}

	return result.CiphertextBlob, nil
}

func Decrypt(text []byte, aKID, secretKey, region string) ([]byte, error) {
	// Initialize a session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(aKID, secretKey, ""),
	})

	if err != nil {
		log.Error("create kms session fail", "err", err)
		return nil, err
	}

	// Create KMS service client
	svc := kms.New(sess)

	// Encrypted data
	blob := []byte(text)

	// Decrypt the data
	result, err := svc.Decrypt(&kms.DecryptInput{CiphertextBlob: blob})
	if err != nil {
		log.Error("kms decrypt fail", "err", err)
		return nil, err
	}

	return result.Plaintext, nil
}
