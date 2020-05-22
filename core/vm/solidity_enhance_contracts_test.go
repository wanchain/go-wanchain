package vm

import (
	"crypto/ecdsa"
	Rand "crypto/rand"
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"io"
	"math/big"
	"testing"
)
var (
	Big1                       = big.NewInt(1)
	Big0                        = big.NewInt(0)
	Ne                         = 10
)

func randFieldElement(rand io.Reader) (k *big.Int, err error) {
	params := crypto.S256().Params()
	b := make([]byte, params.BitSize/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}
	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, Big1)
	k.Mod(k, n)
	k.Add(k, Big1)
	return
}


func TestPloy(t *testing.T) {

	// Number of storeman nodes
	const Nstm = 21

	// Threshold for storeman signature
	const Thres = 18

	// Polynomial degree for shamir secret sharing
	const Degree = Thres - 1

	// Generate storeman's public key and private key
	Pubkey := make([]*ecdsa.PublicKey, Nstm)
	Prikey := make([]*ecdsa.PrivateKey, Nstm)

	for i := 0; i < Nstm; i++ {
		Prikey[i], _ = ecdsa.GenerateKey(crypto.S256(), Rand.Reader)
		Pubkey[i] = &Prikey[i].PublicKey
	}

	x := make([]big.Int, Nstm)
	for i := 0; i < Nstm; i++ {
		x[i].SetBytes(crypto.Keccak256(crypto.FromECDSAPub(Pubkey[i])))
		x[i].Mod(&x[i], crypto.S256().Params().N)
	}

	// Each of storeman node generates a random si
	s := make([]*big.Int, Nstm)

	for i := 0; i < Nstm; i++ {
		s[i], _ = Rand.Int(Rand.Reader, crypto.S256().Params().N)
	}

	poly := make([]Polynomial, Nstm)

	poly[0] = RandPoly(Degree, *s[0])


	pb := crypto.FromECDSAPub(Pubkey[0])
	pbs := common.Bytes2Hex(pb);


	fmt.Println("pubkey= " + pbs)

	ws := ""
	for i := 0; i < len(poly[0]); i++ {
		ba := poly[0][i].Bytes();
		baHex := common.Bytes2Hex(ba)
		ws = ws + "||"   + baHex
	}


	fmt.Println("ws= " +ws)

	var sshare [Nstm][Nstm]big.Int

	sshare[0][0] = EvaluatePoly(poly[0], &x[0], Degree)




	//for i := 0; i < Nstm; i++ {
	//
	//	poly[i] = RandPoly(Degree, *s[i]) // fi(x), set si as its constant term
	//
	//	for j := 0; j < Nstm; j++ {
	//		// share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
	//		sshare[i][j] = EvaluatePoly(poly[i], &x[j], Degree)
	//	}
	//}

}

const PkLength = 65

// Generate a random polynomial, its constant item is nominated
func RandPoly(degree int, constant big.Int) Polynomial {

	poly := make(Polynomial, degree+1)

	poly[0].Mod(&constant, crypto.S256().Params().N)

	for i := 1; i < degree+1; i++ {

		temp, _ := Rand.Int(Rand.Reader, crypto.S256().Params().N)

		// in case of polynomial degenerating
		poly[i] = *temp.Add(temp, Big1)
	}
	return poly
}


// Whole Flow Test
func TestAadd(t *testing.T) {

	Pubkey := make([]*ecdsa.PublicKey, Ne)
	Prikey := make([]*ecdsa.PrivateKey, Ne)

	for i := 0; i < Ne; i++ {
		Prikey[i], _ = ecdsa.GenerateKey(crypto.S256(), Rand.Reader)
		Pubkey[i] = &Prikey[i].PublicKey
	}

	alphas := make([]*big.Int, 0)
	for i := 0; i < Ne; i++ {
		x, err := randFieldElement(Rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		alphas = append(alphas, x)
	}

	ArrayReceived := make([]*ecdsa.PublicKey, 0) //need to make the order of received message the same among different users
	for i := 0; i < Ne; i++ {
		piece := new(ecdsa.PublicKey)
		piece.Curve = crypto.S256()
		piece.X, piece.Y = crypto.S256().ScalarMult(Prikey[i].PublicKey.X, Prikey[i].PublicKey.Y, alphas[i].Bytes())
		ArrayReceived = append(ArrayReceived, piece)


	}

	skGt := new(ecdsa.PublicKey)
	skGt.Curve = crypto.S256()

	skGt.X = new(big.Int).Set(ArrayReceived[0].X)
	skGt.Y = new(big.Int).Set(ArrayReceived[0].Y)

	fmt.Println(common.ToHex(skGt.X.Bytes()))
	fmt.Println(common.ToHex(skGt.Y.Bytes()))


	skGt1 := new(ecdsa.PublicKey)
	skGt1.Curve = crypto.S256()

	skGt1.X = new(big.Int).Set(ArrayReceived[1].X)
	skGt1.Y = new(big.Int).Set(ArrayReceived[1].Y)

	fmt.Println(common.ToHex(skGt1.X.Bytes()))
	fmt.Println(common.ToHex(skGt1.Y.Bytes()))

	res := crypto.S256().IsOnCurve(skGt.X,skGt.Y)

	res1 := crypto.S256().IsOnCurve(skGt1.X,skGt1.Y)

	if res == res1 {
		fmt.Println("")
	}



	//for j := 1; j < Ne; j++ {
	//	skGt.X, skGt.Y = crypto.S256().Add(skGt.X, skGt.Y, ArrayReceived[j].X, ArrayReceived[j].Y)
	//}


}
