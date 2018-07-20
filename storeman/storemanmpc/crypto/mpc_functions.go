package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
)

func UintRand(MaxValue uint64) (uint64, error) {
	num, err := rand.Int(rand.Reader, new(big.Int).SetUint64(MaxValue))
	if err != nil {
		return 0, err
	}

	return num.Uint64(), nil
}

func GetRandCoefficients(num int) ([]big.Int, error) {
	randCoefficient := make([]big.Int, num)
	for i := 0; i < num; i++ {
		co, err := crypto.RandFieldElement2528(rand.Reader)
		if err != nil {
			return nil, err
		}
		randCoefficient[i] = *co
	}

	return randCoefficient, nil
}

func EvaluatePoly(coefficient []big.Int, x *big.Int) big.Int {
	degree := len(coefficient) - 1
	sumx := make([]big.Int, degree+1)
	for i := 0; i <= degree; i++ {
		sumx[i].Set(&coefficient[i])
	}

	for i := 1; i <= degree; i++ {
		for j := i; j <= degree; j++ {
			sumx[j].Mul(&sumx[j], x)
			sumx[j].Mod(&sumx[j], crypto.Secp256k1_N)
		}
	}

	sum := big.NewInt(0)
	for i := 0; i < len(sumx); i++ {
		sum.Add(sum, &sumx[i])
		sum.Mod(sum, crypto.Secp256k1_N)
	}

	return *sum
}

func evaluateB(x []big.Int) []big.Int {
	k := len(x)
	b := make([]big.Int, k)
	for i := 0; i < k; i++ {
		b[i] = evaluateb(x, i)
	}

	return b
}

func evaluateb(x []big.Int, i int) big.Int {
	k := len(x)
	sum := big.NewInt(1)
	temp1 := big.NewInt(1)
	temp2 := big.NewInt(1)
	for j := 0; j < k; j++ {
		if j != i {
			temp1.Sub(&x[j], &x[i])
			temp1.ModInverse(temp1, crypto.Secp256k1_N)
			temp2.Mul(&x[j], temp1)
			sum.Mul(sum, temp2)
			sum.Mod(sum, crypto.Secp256k1_N)
		} else {
			continue
		}
	}

	return *sum
}

// Lagrange's polynomial interpolation algorithm
func Lagrange(f []big.Int, x []big.Int) big.Int {
	degree := len(x) - 1
	b := evaluateB(x)
	s := big.NewInt(0)
	temp1 := big.NewInt(1)

	for i := 0; i < degree+1; i++ {
		temp1.Mul(&f[i], &b[i])
		s.Add(s, temp1)
		s.Mod(s, crypto.Secp256k1_N)
	}

	return *s
}

func TransSignature(R *big.Int, S *big.Int, V *big.Int) ([]byte, error) {
	if S.Cmp(new(big.Int).Div(crypto.Secp256k1_N, big.NewInt(2))) > 0 {
		S.Sub(crypto.Secp256k1_N, S)
		v := V.Int64()
		v ^= 1
		V.SetInt64(v)
	}

	log.Debug("trans signature", "S", S.String())
	sig := make([]byte, 65)
	copy(sig[:], math.PaddedBigBytes(R, 32))
	copy(sig[32:], math.PaddedBigBytes(S, 32))
	sig[64] = byte(V.Int64())
	return sig, nil
}

func SenderEcrecover(sighash, sig []byte) (common.Address, error) {
	pub, err := crypto.Ecrecover(sighash, sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func ValidatePublicKey(k *ecdsa.PublicKey) bool {
	return k != nil && k.X != nil && k.Y != nil && k.X.Sign() != 0 && k.Y.Sign() != 0
}

func ValidatePrivateKey(k *big.Int) bool {
	if k == nil || k.Sign() == 0 {
		return false
	}

	return true
}
