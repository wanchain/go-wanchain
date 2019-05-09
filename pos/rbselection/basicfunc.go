package rbselection

import (
	"bytes"
	"crypto/rand"
	"errors"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
)

// Generate a random polynomial, its constant item is nominated
func RandPoly(degree int, constant big.Int) (Polynomial, error) {
	if degree < 0 {
		err := errors.New("invalid poly degree")
		log.SyslogErr("get rand poly fail", "err", err.Error())
		return nil, err
	}

	poly := make(Polynomial, degree+1)
	poly[0].Mod(&constant, bn256.Order)
	for i := 1; i < degree+1; i++ {
		temp, err := rand.Int(rand.Reader, bn256.Order)
		if err != nil {
			log.SyslogErr("get rand poly fail", "err", err.Error())
			return nil, err
		}

		// in case of polynomial degenerating
		poly[i] = *temp.Add(temp, bigOne)
	}

	return poly, nil
}

// Calculate polynomial's evaluation at some point
func EvaluatePoly(f Polynomial, x *big.Int, degree int) (big.Int, error) {
	if len(f) < 1 || degree != len(f)-1 {
		err := errors.New("invalid polynomial len")
		log.SyslogErr("evaluate poly fail", "err", err.Error())
		return *new(big.Int), err
	}

	sum := big.NewInt(0)
	for i := 0; i < degree+1; i++ {
		temp1 := new(big.Int).Exp(x, big.NewInt(int64(i)), bn256.Order)
		temp1.Mod(temp1, bn256.Order)
		temp2 := new(big.Int).Mul(&f[i], temp1)
		temp2.Mod(temp2, bn256.Order)
		sum.Add(sum, temp2)
		sum.Mod(sum, bn256.Order)
	}

	return *sum, nil
}

// Calculate the b coefficient in Lagrange's polynomial interpolation algorithm
func evaluateB(x []big.Int, degree int) []big.Int {
	k := degree + 1
	b := make([]big.Int, k)
	for i := 0; i < k; i++ {
		b[i] = evaluateb(x, i, degree)
	}

	return b
}

// sub-function for evaluateB
func evaluateb(x []big.Int, i int, degree int) big.Int {
	k := degree + 1
	sum := big.NewInt(1)
	for j := 0; j < k; j++ {
		if j != i {
			temp1 := new(big.Int).Sub(&x[j], &x[i])
			temp1.ModInverse(temp1, bn256.Order)
			temp2 := new(big.Int).Mul(&x[j], temp1)
			sum.Mul(sum, temp2)
			sum.Mod(sum, bn256.Order)
		}
	}

	return *sum
}

// Lagrange's polynomial interpolation algorithm
func Lagrange(f []big.Int, x []big.Int, degree int) big.Int {
	b := evaluateB(x, degree)
	s := big.NewInt(0)
	for i := 0; i < degree+1; i++ {
		temp1 := new(big.Int).Mul(&f[i], &b[i])
		s.Add(s, temp1)
		s.Mod(s, bn256.Order)
	}

	return *s
}

// Lagrange's polynomial interpolation algorithm: group signature share --> group signature
func LagrangeSig(sig []bn256.G1, x []big.Int, degree int) bn256.G1 {
	b := evaluateB(x, degree)
	sum := new(bn256.G1).ScalarBaseMult(big.NewInt(int64(0)))
	for i := 0; i < degree+1; i++ {
		temp := new(bn256.G1).ScalarMult(&sig[i], &b[i])
		sum.Add(sum, temp)
	}

	return *sum
}

// Lagrange's polynomial interpolation algorithm: group public key share --> group public key
func LagrangePub(cmt []bn256.G2, x []big.Int, degree int) bn256.G2 {
	b := evaluateB(x, degree)
	sum := new(bn256.G2).ScalarBaseMult(big.NewInt(int64(0)))
	for i := 0; i < degree+1; i++ {
		temp := new(bn256.G2).ScalarMult(&cmt[i], &b[i])
		sum.Add(sum, temp)
	}

	return *sum
}

// Generate a DLEQ proof, i.e. exit sk, s.t x = sk*G1 and y = sk*G2
func DLEQ(gbase bn256.G1, hbase bn256.G2, sk *big.Int) (DLEQproof, error) {
	if sk == nil {
		err := errors.New("invalid DLEQ input param")
		log.SyslogErr("generate DKEQ proof fail", "err", err.Error())
		return DLEQproof{nil, nil, nil}, err
	}

	w, err := rand.Int(rand.Reader, bn256.Order)
	if err != nil {
		log.SyslogErr("generate DKEQ proof fail", "err", err.Error())
		return DLEQproof{nil, nil, nil}, err
	}

	var a1 bn256.G1
	var a2 bn256.G2

	a1.ScalarMult(&gbase, w) //a1=w*gbase
	a2.ScalarMult(&hbase, w) //a2=w*hbase

	var buffer bytes.Buffer
	ret1 := a1.Marshal()
	ret2 := a2.Marshal()
	buffer.Write(ret1)
	buffer.Write(ret2)
	ret := buffer.Bytes()

	var temp big.Int
	temp.SetBytes(crypto.Keccak256(ret)) //e=Hash(a1||a2)
	e := temp.Mod(&temp, bn256.Order)

	var z big.Int
	z.Mul(sk, e)
	z.Sub(w, &z)
	z.Mod(&z, bn256.Order) // z = w - sk*e

	var proof DLEQproof
	proof.a1 = &a1
	proof.a2 = &a2
	proof.z = &z

	return proof, nil
}

// to verify DLEQ proof
func VerifyDLEQ(proof DLEQproof, gbase bn256.G1, hbase bn256.G2, x bn256.G1, y bn256.G2) bool {
	if proof.a1 == nil || proof.a2 == nil || proof.z == nil {
		return false
	}

	a1 := *proof.a1
	a2 := *proof.a2
	z := *proof.z

	var buffer bytes.Buffer
	ret1 := a1.Marshal()
	ret2 := a2.Marshal()
	buffer.Write(ret1)
	buffer.Write(ret2)
	ret := buffer.Bytes()

	var temp big.Int
	temp.SetBytes(crypto.Keccak256(ret))
	e := temp.Mod(&temp, bn256.Order)

	var G1z bn256.G1
	var G2z bn256.G2

	G1z.ScalarMult(&gbase, &z)
	G2z.ScalarMult(&hbase, &z)

	var xe bn256.G1
	var ye bn256.G2

	xe.Set(&x)
	ye.Set(&y)
	xe.ScalarMult(&xe, e)
	ye.ScalarMult(&ye, e)

	aa1 := G1z.Add(&G1z, &xe)
	aa2 := G2z.Add(&G2z, &ye)

	return (CompareG1(a1, *aa1) && CompareG2(a2, *aa2))
}

// The comparison function of G1
func CompareG1(a bn256.G1, b bn256.G1) bool {
	return a.String() == b.String()
}

// The comparison function of G2
func CompareG2(a bn256.G2, b bn256.G2) bool {
	return a.String() == b.String()
}

// Subfunction for rscoffGenerate to compute vi: vi = (1-i)^-1 * (2-i)^-1 * ... * (Nr-i)^-1
func rscoffGeneratei(x []big.Int, i int) big.Int {
	k := len(x)
	sum := big.NewInt(1)
	for j := 0; j < k; j++ {
		if j != i {
			temp1 := new(big.Int).Sub(&x[i], &x[j])
			temp1.Mod(temp1, bn256.Order)
			temp1.ModInverse(temp1, bn256.Order)
			sum.Mul(sum, temp1)
			sum.Mod(sum, bn256.Order)
		}
	}

	return *sum
}

// Subfunction for rscodeGenerate to compute coff of RScode
func rscoffGenerate(x []big.Int) []big.Int {
	k := len(x)
	if k < 2 {
		return nil
	}

	b := make([]big.Int, k)
	for i := 0; i < k; i++ {
		b[i] = rscoffGeneratei(x, i)
	}

	return b
}

// Genetate RScode
func rscodeGenerate(x []big.Int, degree int) []big.Int {
	if len(x) < 2 || degree < 1 {
		return nil
	}

	k := len(x)
	constant, _ := rand.Int(rand.Reader, bn256.Order)
	poly, err := RandPoly(degree, *constant)
	if err != nil {
		return nil
	}

	coff := rscoffGenerate(x)
	if coff == nil {
		return nil
	}

	C := make([]big.Int, k)
	for i := 0; i < k; i++ {
		temp, _ := EvaluatePoly(poly, &x[i], degree)
		C[i].Mul(&temp, &coff[i])
		C[i].Mod(&C[i], bn256.Order)
	}

	return C
}

// RScode Verification function
func RScodeVerify(P []bn256.G2, x []big.Int, deg int) bool {
	if P == nil || x == nil || len(P) != len(x) || len(P) <= deg+2 {
		return false
	}

	k := len(x)
	degree := k - deg - 2
	C := rscodeGenerate(x, degree)
	if C == nil {
		return false
	}

	sum := new(bn256.G2).ScalarBaseMult(big.NewInt(int64(0)))
	for i := 0; i < k; i++ {
		temp := new(bn256.G2).ScalarMult(&P[i], &C[i])
		sum.Add(sum, temp)
	}

	return sum.IsInfinity()
}
