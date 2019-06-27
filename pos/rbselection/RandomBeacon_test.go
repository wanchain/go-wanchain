// This file is wanchain pos random beacon part. It simulates the process of Nr parties working together
// to generate a random which will be used in unique leader selection.
package rbselection

import (
	"crypto/rand"
	"fmt"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/crypto/bn256/cloudflare"
	"math/big"
	"testing"
)

func TestRandomBeacon(t *testing.T) {

	// The number of random proposer selected from participants pool.
	const Nr = 10

	// Threshold: As long as over Thres random proposers take part in the process, the output of random beacon
	// will be generated correctly.
	const Thres = Nr / 2

	const Degree = Thres - 1

	// Generate Nr random propoers' public key and secret key which is used in random beacon process.
	// Attention! Public keys and secret keys here have nothing to do with user's account. They use a
	// different parameters. When a user participate in the pos process, he has to generate such a pair
	// of keys and register in the smart contract.
	Pubkey := make([]bn256.G1, Nr)
	Prikey := make([]big.Int, Nr)

	for i := 0; i < Nr; i++ {
		Pri, Pub, err := bn256.RandomG1(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		Prikey[i] = *Pri
		Pubkey[i] = *Pub
	}

	// Fix the evaluation point: Hash(Pub[1]), Hash(Pub[2]), ..., Hash(Pub[Nr])
	x := make([]big.Int, Nr)
	for i := 0; i < Nr; i++ {
		x[i].SetBytes(crypto.Keccak256(Pubkey[i].Marshal()))
		x[i].Mod(&x[i], bn256.Order)
	}

	//----------------------------------------------  DKG STAGE ----------------------------------------------//
	// in this stage, the random proposers work together non-interactively through the blockchain to generate a
	// group public key and get their group secret key shares at the same time.

	// Each of random propoer generates a random si
	s := make([]*big.Int, Nr)

	for i := 0; i < Nr; i++ {
		s[i], _ = rand.Int(rand.Reader, bn256.Order)
	}

	// Each random propoer conducts the shamir secret sharing process
	poly := make([]Polynomial, Nr)

	var sshare [Nr][Nr]big.Int

	for i := 0; i < Nr; i++ {
		poly[i],_ = RandPoly(Degree, *s[i]) // fi(x), set si as its constant term
		for j := 0; j < Nr; j++ {
			sshare[i][j], _ = EvaluatePoly(poly[i], &x[j], Degree) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
		}
	}

	// Encrypt the secret share, i.e. mutiply with the receiver's public key
	var enshare [Nr][Nr]bn256.G1
	for i := 0; i < Nr; i++ {
		for j := 0; j < Nr; j++ { // enshare[j] = sshare[j]*Pub[j], it is a point on ECC
			enshare[i][j] = *new(bn256.G1).ScalarMult(&Pubkey[j], &sshare[i][j])
		}
	}

	// Make commitment for the secret share, i.e. mutiply with the generator of G2
	var commit [Nr][Nr]bn256.G2
	for i := 0; i < Nr; i++ {
		for j := 0; j < Nr; j++ { // commit[j] = sshare[j] * G2
			commit[i][j] = *new(bn256.G2).ScalarBaseMult(&sshare[i][j])
		}
	}

	// generate DLEQ proof
	var proof [Nr][Nr]DLEQproof
	for i := 0; i < Nr; i++ {
		for j := 0; j < Nr; j++ { // proof = (a1, a2, z)
			proof[i][j], _ = DLEQ(Pubkey[j], *hbase, &sshare[i][j])
		}
	}

	// At this time, all the enshare, commit, proof will be pushed into the blockchain
	// through a special transaction. DKG stage finishes.

	//---------------------------------------------- Verification Logic for Tx in DKG Stage ----------------------------------------------//
	// DLEQ proof verification
	for i := 0; i < Nr; i++ {
		for j := 0; j < Nr; j++ {
			if !VerifyDLEQ(proof[i][j], Pubkey[j], *hbase, enshare[i][j], commit[i][j]) {
				t.Fatal("DLEQ Verification Failed!")
			}
		}
	}

	//RScode Verification
	temp := make([]bn256.G2, Nr)
	for i := 0; i < Nr; i++ {
		for j := 0; j < Nr; j++ {
			temp[j] = commit[i][j]
		}
		if !RScodeVerify(temp, x, Degree) {
			t.Fatal("RScode Verification Failed ")
		}
	}

	//---------------------------------------------- Compute Group Secret Key Share ----------------------------------------------//
	// Random proposers get information from the blockchain and compute its group secret share.

	gskshare := make([]bn256.G1, Nr)

	for i := 0; i < Nr; i++ {

		gskshare[i].ScalarBaseMult(big.NewInt(int64(0))) //set zero

		skinver := new(big.Int).ModInverse(&Prikey[i], bn256.Order) // sk^-1

		for j := 0; j < Nr; j++ {
			temp := new(bn256.G1).ScalarMult(&enshare[j][i], skinver)
			gskshare[i].Add(&gskshare[i], temp) // gskshare[i] = (sk^-1)*(enshare[1][i]+...+enshare[Nr][i])
		}
	}

	//---------------------------------------------- Signing Stage ----------------------------------------------//
	// In this stage, each random proposer computes its signature share and sends it on chain.

	// Fix M = Hase("wanchain")
	// Attention: in our realization, M should be set to Hash(r||τ_r-1)
	M := crypto.Keccak256([]byte("wanchain"))
	m := new(big.Int).SetBytes(M)

	// Compute signature share
	gsigshare := make([]bn256.G1, Nr)

	for i := 0; i < Nr; i++ { // signature share = M * secret key share
		gsigshare[i] = *new(bn256.G1).ScalarMult(&gskshare[i], m)
	}

	//---------------------------------- Verification Logic for Tx in Signing Stage ------------------------------------//
	// Compute the group public key share to verify the signature share

	gpkshare := make([]bn256.G2, Nr)

	// Computation of group public key share
	for i := 0; i < Nr; i++ {
		gpkshare[i].ScalarBaseMult(big.NewInt(int64(0))) //set zero
		for j := 0; j < Nr; j++ {
			gpkshare[i].Add(&gpkshare[i], &commit[j][i])
		}
	}

	// Verify using pairing
	mG := new(bn256.G1).ScalarBaseMult(m)
	for i := 0; i < Nr; i++ {
		pair1 := bn256.Pair(&gsigshare[i], hbase)
		pair2 := bn256.Pair(mG, &gpkshare[i])
		if pair1.String() != pair2.String() {
			t.Fatal("Pairing Check Failed")
		}
	}

	//---------------------------------- Compute the Output of Random Beacon ------------------------------------//
	Outpoint := LagrangeSig(gsigshare, x, Degree)

	Output := crypto.Keccak256(Outpoint.Marshal())

	fmt.Println("The output of random beacon is  ", Output)

	//---------------------------------- Verification Logic for the Output of Random Beacon ------------------------------------//
	// Compute the group public key to verify the group signature

	// Computation of group public key
	C := make([]bn256.G2, Nr)

	for i := 0; i < Nr; i++ {
		C[i].ScalarBaseMult(big.NewInt(int64(0)))
		for j := 0; j < Nr; j++ {
			C[i].Add(&C[i], &commit[j][i])
		}
	}
	GPub := LagrangePub(C, x, Degree)

	// Verify using pairing
	pair1 := bn256.Pair(&Outpoint, hbase)
	pair2 := bn256.Pair(mG, &GPub)
	if pair1.String() != pair2.String() {
		t.Fatal("Final Pairing Check Failed")
	}

}
