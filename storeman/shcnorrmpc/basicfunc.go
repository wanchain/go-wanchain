package shcnorrmpc

import (
	Rand "crypto/rand"
	"math/big"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
)

// Generate a random polynomial, its constant item is nominated
func RandPoly(degree int, constant big.Int) Polynomial {

	poly := make(Polynomial, degree+1)

	poly[0].Mod(&constant, crypto.S256().Params().N)

	for i := 1; i < degree+1; i++ {

		temp, _ := Rand.Int(Rand.Reader, crypto.S256().Params().N)
		
		// in case of polynomial degenerating
		poly[i] = *temp.Add(temp, bigOne)
	}
	return poly
}


// Calculate polynomial's evaluation at some point
func EvaluatePoly(f Polynomial, x *big.Int, degree int) big.Int {

	sum := big.NewInt(0)

	for i := 0; i < degree+1; i++ {

		temp1 := new(big.Int).Exp(x, big.NewInt(int64(i)), crypto.S256().Params().N)

		temp1.Mod(temp1, crypto.S256().Params().N)

		temp2 := new(big.Int).Mul(&f[i], temp1)

		temp2.Mod(temp2, crypto.S256().Params().N)

		sum.Add(sum, temp2)

		sum.Mod(sum, crypto.S256().Params().N)
	}
	return *sum
}



// Calculate the b coefficient in Lagrange's polynomial interpolation algorithm

func evaluateB(x []big.Int, degree int) []*big.Int {

	//k := len(x)

	k := degree + 1

	b := make([]*big.Int, k)

	for i := 0; i < k; i++ {
		b[i] = evaluateb(x, i, degree)
	}
	return b
}



// sub-function for evaluateB

func evaluateb(x []big.Int, i int, degree int) *big.Int {

	//k := len(x)

	k := degree + 1

	sum := big.NewInt(1)

	for j := 0; j < k; j++ {

		if j != i {

			temp1 := new(big.Int).Sub(&x[j], &x[i])

			temp1.ModInverse(temp1, crypto.S256().Params().N)

			temp2 := new(big.Int).Mul(&x[j], temp1)

			sum.Mul(sum, temp2)

			sum.Mod(sum, crypto.S256().Params().N)

		} else {
			continue
		}
	}
	return sum
}



// Lagrange's polynomial interpolation algorithm: working in ECC points
func LagrangeECC(sig []ecdsa.PublicKey, x []big.Int, degree int) *ecdsa.PublicKey {

	b := evaluateB(x, degree)

	sum := new(ecdsa.PublicKey)

	//累加过程中，第一个点需设为无穷远点，但是这样会生成空指针，后面点加函数报错
	//因此，只能退而求其次，先将累加的第一个点设入到sum中.然后循环的起点为1
	sum.X, sum.Y = crypto.S256().ScalarMult(sig[0].X, sig[0].Y, b[0].Bytes()) 
	
	for i := 1; i < degree+1; i++ {
		temp := new(ecdsa.PublicKey)
		temp.X, temp.Y = crypto.S256().ScalarMult(sig[i].X, sig[i].Y, b[i].Bytes()) 
		sum.X, sum.Y = crypto.S256().Add(sum.X, sum.Y, temp.X, temp.Y)
	}
	return sum
}

func SchnorrSign(psk big.Int, r big.Int, m big.Int) big.Int{
	sum := big.NewInt(1)
	sum.Mul(&psk, &m)
	sum.Mod(sum, crypto.S256().Params().N)
	sum.Add(sum, &r)
	sum.Mod(sum, crypto.S256().Params().N)
	return *sum
}

// Lagrange's polynomial interpolation algorithm
func Lagrange(f []big.Int, x []big.Int, degree int) big.Int {

	b := evaluateB(x, degree)

	s := big.NewInt(0)

	for i := 0; i < degree+1; i++ {

		temp1 := new(big.Int).Mul(&f[i], b[i])

		s.Add(s, temp1)

		s.Mod(s, crypto.S256().Params().N)
	}
	return *s
}








