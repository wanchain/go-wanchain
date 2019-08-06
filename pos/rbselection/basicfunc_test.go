package rbselection

import (
	"crypto/rand"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"math/big"
	"testing"
)

func TestCompareG1(t *testing.T) {

	_, Ga, err := bn256.RandomG1(rand.Reader)

	if err != nil {
		t.Fatal(err)
	}

	Gb := new(bn256.G1).Set(Ga)

	if !CompareG1(*Ga, *Gb) {
		t.Fatal("CompareG1 Failed ")
	}
}

func TestCompareG2(t *testing.T) {

	_, Ga, err := bn256.RandomG2(rand.Reader)

	if err != nil {
		t.Fatal(err)
	}

	Gb := new(bn256.G2).Set(Ga)

	if !CompareG2(*Ga, *Gb) {
		t.Fatal("CompareG2 Failed ")
	}
}

func TestDLEQ1(t *testing.T) {

	Pri, Pub, err := bn256.RandomG1(rand.Reader)

	if err != nil {
		t.Fatal(err)
	}

	proof, _ := DLEQ(*gbase, *hbase, Pri)

	commit := new(bn256.G2).ScalarBaseMult(Pri)

	if !VerifyDLEQ(proof, *gbase, *hbase, *Pub, *commit) {
		t.Fatal("DLEQ process Failed ")
	}
}

func TestDLEQ2(t *testing.T) {

	_, Pub, err := bn256.RandomG1(rand.Reader)

	if err != nil {
		t.Fatal(err)
	}

	s, _ := rand.Int(rand.Reader, bn256.P)

	sPub := new(bn256.G1).ScalarMult(Pub, s)

	proof, _ := DLEQ(*Pub, *hbase, s)

	commit := new(bn256.G2).ScalarBaseMult(s)

	if !VerifyDLEQ(proof, *Pub, *hbase, *sPub, *commit) {
		t.Fatal("DLEQ process Failed ")
	}
}

func TestRScode(t *testing.T) {

	const Nr = 10

	const Thres = 6

	s, _ := rand.Int(rand.Reader, bn256.Order)

	// the value point
	x := make([]big.Int, Nr)
	for i := 0; i < Nr; i++ {
		x[i] = *big.NewInt(int64(i + 1))
	}

	sshare := make([]big.Int, Nr)

	commit := make([]bn256.G2, Nr)

	poly, _ := RandPoly(Thres-1, *s)

	for i := 0; i < Nr; i++ {
		sshare[i], _ = EvaluatePoly(poly, &x[i], Thres-1)
		sshare[i].Mod(&sshare[i], bn256.Order)
		commit[i] = *new(bn256.G2).ScalarBaseMult(&sshare[i])
	}

	if !RScodeVerify(commit, x, Thres-1) {
		t.Fatal("RScode Verification Failed ")
	}
}
