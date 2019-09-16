package shcnorrmpc

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"github.com/wanchain/go-wanchain/crypto"
	"math/big"
	"testing"
)

func TestSchnorr(t *testing.T) {

	// Number of storeman nodes
	const Nstm = 50

	// Threshold for schnorr signature
	const Thres = 26

	// Polynomial degree for shamir secret sharing
	const Degree = Thres - 1

	// Generate storeman's public key and private key
	Pubkey := make([]*ecdsa.PublicKey, Nstm)
	Prikey := make([]*ecdsa.PrivateKey, Nstm)

	for i := 0; i < Nstm; i++ {
		Prikey[i], _ = ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		Pubkey[i] = &Prikey[i].PublicKey
	}

	// Fix the evaluation point: Hash(Pub[1]), Hash(Pub[2]), ..., Hash(Pub[Nr])
	x := make([]big.Int, Nstm)
	for i := 0; i < Nstm; i++ {
		x[i].SetBytes(crypto.Keccak256(crypto.FromECDSAPub(Pubkey[i])))
		x[i].Mod(&x[i], crypto.S256().Params().N)
	}

	//----------------------------------------------  Setup  ----------------------------------------------//
	// In this stage, the storeman nodes work together to generate the group public keys and get its own
	// group private key share

	// Each of storeman node generates a random si
	s := make([]*big.Int, Nstm)

	for i := 0; i < Nstm; i++ {
		s[i], _ = rand.Int(rand.Reader, crypto.S256().Params().N)
	}

	// Each storeman node conducts the shamir secret sharing process
	poly := make([]Polynomial, Nstm)

	var sshare [Nstm][Nstm]big.Int

	for i := 0; i < Nstm; i++ {
		poly[i] = RandPoly(Degree, *s[i]) // fi(x), set si as its constant term
		for j := 0; j < Nstm; j++ {
			sshare[i][j] = EvaluatePoly(poly[i], &x[j], Degree) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
		}
	}

	// every storeman node sends the secret shares to other nodes in secret!
	// Attention! IN SECRET!

	// After reveiving the secret shares, each node computes its group private key share
	gskshare := make([]*big.Int, Nstm)

	for i := 0; i < Nstm; i++ {
		gskshare[i] = big.NewInt(0)
		for j := 0; j < Nstm; j++ {
			gskshare[i].Add(gskshare[i], &sshare[j][i])
		}
		gskshare[i].Mod(gskshare[i], crypto.S256().Params().N)
	}

	// Each storeman node publishs the scalar point of its group private key share
	gpkshare := make([]ecdsa.PublicKey, Nstm)

	for i := 0; i < Nstm; i++ {
		gpkshare[i].X, gpkshare[i].Y = crypto.S256().ScalarBaseMult(gskshare[i].Bytes())
	}

	// Each storeman node computes the group public key by Lagrange's polynomial interpolation
	gpk := LagrangeECC(gpkshare, x, Degree)

	//----------------------------------------------  Signing ----------------------------------------------//

	// 1st step: each storeman node decides a random number r using shamir secret sharing

	rr := make([]*big.Int, Nstm)

	for i := 0; i < Nstm; i++ {
		rr[i], _ = rand.Int(rand.Reader, crypto.S256().Params().N)
	}

	poly1 := make([]Polynomial, Nstm)

	var rrshare [Nstm][Nstm]big.Int

	for i := 0; i < Nstm; i++ {
		poly1[i] = RandPoly(Degree, *s[i]) // fi(x), set si as its constant term
		for j := 0; j < Nstm; j++ {
			rrshare[i][j] = EvaluatePoly(poly1[i], &x[j], Degree) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
		}
	}

	// every storeman node sends the secret shares to other nodes in secret!
	// Attention! IN SECRET!

	rshare := make([]*big.Int, Nstm)

	for i := 0; i < Nstm; i++ {
		rshare[i] = big.NewInt(0)
		for j := 0; j < Nstm; j++ {
			rshare[i].Add(rshare[i], &rrshare[j][i])
		}
		rshare[i].Mod(rshare[i], crypto.S256().Params().N)
	}

	// Compute the scalar point of r
	rpkshare := make([]ecdsa.PublicKey, Nstm)

	for i := 0; i < Nstm; i++ {
		rpkshare[i].X, rpkshare[i].Y = crypto.S256().ScalarBaseMult(rshare[i].Bytes())
	}

	rpk := LagrangeECC(rpkshare, x, Degree)

	// Forming the m: hash(message||rpk)
	var buffer bytes.Buffer
	buffer.Write([]byte("wanchain"))
	buffer.Write(crypto.FromECDSAPub(rpk))

	M := crypto.Keccak256(buffer.Bytes())
	m := new(big.Int).SetBytes(M)

	// Each storeman node computes the signature share
	sigshare := make([]big.Int, Nstm)

	for i := 0; i < Nstm; i++ {
		sigshare[i] = SchnorrSign(*gskshare[i], *rshare[i], *m)
	}

	// Compute the signature using Lagrange's polynomial interpolation

	ss := Lagrange(sigshare, x, Degree)

	// the final signature = (rpk,ss)

	//----------------------------------------------  Verification ----------------------------------------------//
	// check ssG = rpk + m*gpk

	ssG := new(ecdsa.PublicKey)
	ssG.X, ssG.Y = crypto.S256().ScalarBaseMult(ss.Bytes())

	mgpk := new(ecdsa.PublicKey)
	mgpk.X, mgpk.Y = crypto.S256().ScalarMult(gpk.X, gpk.Y, m.Bytes())

	temp := new(ecdsa.PublicKey)
	temp.X, temp.Y = crypto.S256().Add(mgpk.X, mgpk.Y, rpk.X, rpk.Y)

	if ssG.X.Cmp(temp.X) == 0 && ssG.Y.Cmp(temp.Y) == 0 {
		fmt.Println("Verification Succeeded")
		fmt.Println(" ", ssG.X)
		fmt.Println(" ", ssG.Y)
		fmt.Println(" ", temp.X)
		fmt.Println(" ", temp.Y)
	} else {
		t.Fatal("Verification Failed")
	}
}
