package bn256

import (
	"crypto/rand"
	"errors"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"math/big"
)

type PublicKeyBn256 struct {
	// g1, g2
	G1 *bn256.G1
}

type PrivateKeyBn256 struct {
	PublicKeyBn256
	D *big.Int
}

func ToBn256(d []byte) (*PrivateKeyBn256, error) {
	// TODO: use S256
	if d == nil || len(d) * 8 != 256 {
		return nil, errors.New("invalid d param")
	}
	privateKeyBn256 := new(PrivateKeyBn256)
	privateKeyBn256.D = new(big.Int).SetBytes(d)
	privateKeyBn256.PublicKeyBn256.G1 = new(bn256.G1).ScalarBaseMult(privateKeyBn256.D)
	return privateKeyBn256, nil
}


// TODO: ask zhongzhong which random method should choose
func GenerateBn256() (*PrivateKeyBn256, error) {
	pri, g1, err := bn256.RandomG1(rand.Reader)
	if err != nil {
		return nil, err
	}
	privateKeyBn256 := new(PrivateKeyBn256)
	privateKeyBn256.D = pri
	privateKeyBn256.G1 = g1
	return privateKeyBn256, nil
}
