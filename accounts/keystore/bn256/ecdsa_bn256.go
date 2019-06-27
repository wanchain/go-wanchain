package bn256

import (
	"crypto/rand"
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
