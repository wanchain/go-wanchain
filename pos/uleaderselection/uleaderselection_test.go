package uleaderselection

import (
	"bytes"
	"crypto/ecdsa"
	Rand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/wanchain/go-wanchain/crypto"
)

var (
	Ns                         = 100 //num of publickey samples
	Nr                         = 10  //num of random proposers
	Ne                         = 10  //num of epoch leaders, limited <= 256 now
	EL                         = 100 //num of slots in an epoch
	ErrInvalidgenPublicKeys    = errors.New("Invalid PublicKey Sample Generation")
	ErrInvalidgenProbabilities = errors.New("Invalid Probabilitiy Sample Generation")
	ErrPublickeyGeneration     = errors.New("Invalid Publickey Generation")
	ErrInvalidgenPrivateKeys   = errors.New("Invalid PrivateKey Sample Generation")
	ErrInvalidSortPrivateKeys  = errors.New("Invalid PrivateKey Sort Operation")
)

func TestProbabilityFloat2big(t *testing.T) {
	probabilities, err := genProbabilities(Ns)
	if err != nil {
		t.Fatal(err)
	}
	probabilitiesBig, err := ProbabilityFloat2big(probabilities)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(probabilitiesBig)
}

func TestRandomProposerSelection(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(pksamples)
	probabilities, err := genProbabilities(Ns)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(probabilities)
	var r = []byte{byte(1)}
	randomProposerPublickeys, err := RandomProposerSelection(r, Nr, pksamples, probabilities)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(randomProposerPublickeys)
}

func TestDleqProofGeneration(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(pksamples)
	fmt.Println(alpha)
	ArrayPiece := make([]*ecdsa.PublicKey, 0)
	n := len(pksamples)
	for i := 0; i < n; i++ {
		piece := new(ecdsa.PublicKey)
		piece.Curve = crypto.S256()
		piece.X, piece.Y = crypto.S256().ScalarMult(pksamples[i].X, pksamples[i].Y, alpha.Bytes()) //piece = alpha * PublicKey
		ArrayPiece = append(ArrayPiece, piece)
	}
	fmt.Println(ArrayPiece)
	proof, err := DleqProofGeneration(pksamples, ArrayPiece, alpha)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(proof)
}

func TestVerifyDleqProof(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ArrayPiece := make([]*ecdsa.PublicKey, 0)
	n := len(pksamples)
	for i := 0; i < n; i++ {
		piece := new(ecdsa.PublicKey)
		piece.Curve = crypto.S256()
		piece.X, piece.Y = crypto.S256().ScalarMult(pksamples[i].X, pksamples[i].Y, alpha.Bytes()) //piece = alpha * PublicKey
		ArrayPiece = append(ArrayPiece, piece)
	}
	proof, err := DleqProofGeneration(pksamples, ArrayPiece, alpha)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyDleqProof(pksamples, ArrayPiece, proof) {
		fmt.Println("Verification succeed!")
	}
}

func TestGenerateArrayPiece(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	PublicKeys, alphaPublicKeys, Proof, err := GenerateArrayPiece(pksamples, alpha)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(PublicKeys)
	fmt.Println(alphaPublicKeys)
	fmt.Println(Proof)
}

func TestPublicKeyEqual(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	pkselected := new(ecdsa.PublicKey) //new publickey to store the selected one
	pkselected.Curve = crypto.S256()
	pkselected.X = new(big.Int).Set(pksamples[0].X)
	pkselected.Y = new(big.Int).Set(pksamples[0].Y)
	pkselected1 := new(ecdsa.PublicKey) //new publickey to store the selected one
	pkselected1.Curve = crypto.S256()
	pkselected1.X = new(big.Int).Set(pksamples[0].X)
	pkselected1.Y = new(big.Int).Set(pksamples[0].Y)
	if PublicKeyEqual(pkselected, pkselected1) {
		fmt.Println("Verify Equal Succeed!")
	} else {
		fmt.Println("Failed!")
	}
}

func TestVerifyArrayPiece(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	PublicKeys, alphaPublicKeys, Proof, err := GenerateArrayPiece(pksamples, alpha)
	if err != nil {
		t.Fatal(err)
	}
	Commitment, err := GenerateCommitment(pksamples[Ns-1], alpha)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyArrayPiece(Commitment, PublicKeys, alphaPublicKeys, Proof) {
		fmt.Println("Verification Succeed!")
	} else {
		fmt.Println("Failed!")
	}
}

func TestGenerateSMA(t *testing.T) {
	PrivateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	ArrayPiece := make([]*ecdsa.PublicKey, 0)
	oriSMA := make([]*ecdsa.PublicKey, 0)
	for i := 0; i < Ne; i++ {
		alpha, err := randFieldElement(Rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		piece := new(ecdsa.PublicKey)
		piece.Curve = crypto.S256()
		piece.X, piece.Y = crypto.S256().ScalarMult(PrivateKey.PublicKey.X, PrivateKey.PublicKey.Y, alpha.Bytes())
		ArrayPiece = append(ArrayPiece, piece)
		oripiece := new(ecdsa.PublicKey)
		oripiece.Curve = crypto.S256()
		oripiece.X, oripiece.Y = crypto.S256().ScalarBaseMult(alpha.Bytes())
		oriSMA = append(oriSMA, oripiece)
	}
	SMA, err := GenerateSMA(PrivateKey, ArrayPiece)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < Ne; i++ {
		if PublicKeyEqual(SMA[i], oriSMA[i]) {
			fmt.Println("Yeah!")
		} else {
			fmt.Println("No!")
		}
	}
}

func TestSortPublicKeys(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	var RB = []byte{byte(1)}
	pksamples, err = SortPublicKeys(pksamples, RB)
	if err != nil {
		t.Fatal(err)
	} else {
		n := len(pksamples)
		for i := 0; i < n; i++ {
			var buffer bytes.Buffer
			buffer.Write(RB)
			buffer.Write(crypto.FromECDSAPub(pksamples[i]))
			temp := buffer.Bytes()
			tempbyte := crypto.Keccak256(temp)
			tempBig := new(big.Int).SetBytes(tempbyte)
			fmt.Println(tempBig)
		}
	}
}

func TestSort(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	pksamples, err = Sort(pksamples)
	if err != nil {
		t.Fatal(err)
	}
	n := len(pksamples)
	for i := 0; i < n; i++ {
		tempbyte := crypto.Keccak256(crypto.FromECDSAPub(pksamples[i]))
		tempBig := new(big.Int).SetBytes(tempbyte)
		fmt.Println(tempBig)
	}
}

func TestGenerateSlotLeaderSeq(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	SMA, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	var RB = []byte{byte(1)}
	SlotLeaderSeq, cr, err := GenerateSlotLeaderSeq(SMA, pksamples, RB, EL)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(cr)
	fmt.Println(SlotLeaderSeq)
	fmt.Println("Slot Leader Sequence Generation Succeed!")
}

// //Partial Test For GenerateSlotLeaderProof and Verication
// func TestGenerateSlotLeaderProof(t *testing.T) {
// 	PrivateKey, err := crypto.GenerateKey()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	pksamples, err := genPublicKeys(Ne)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	var RB = []byte{byte(1)}
// 	pksamples[0] = &PrivateKey.PublicKey
// 	SMA, err := genPublicKeys(Ne)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	SlotLeaderSeq, cr, err := GenerateSlotLeaderSeq(SMA, pksamples, RB, EL)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for i := 0; i < EL ; i++ {
// 		if PublicKeyEqual(&PrivateKey.PublicKey, SlotLeaderSeq[i]) {
// 			fmt.Println(i)
// 			ProofMeg, Proof, err := GenerateSlotLeaderProof(PrivateKey, SMA, pksamples, RB, cr , i)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			// fmt.Println(ProofMeg)
// 			// fmt.Println(Proof)
// 			if VerifySlotLeaderProof(Proof, ProofMeg ,pksamples , RB) {
// 				fmt.Println("Verification Succeed!")
// 			}
// 		}
// 	}
// }

// Whole Flow Test
func TestGenerateSlotLeaderProof(t *testing.T) {
	PrivateKeys, err := genPrivateKeys(Ne)
	if err != nil {
		t.Fatal(err)
	}
	var RB = []byte{byte(1)}
	//Sort PrivateKeys
	PrivateKeys, err = SortPrivateKey(PrivateKeys, RB)
	if err != nil {
		t.Fatal(err)
	}
	PublicKeys := make([]*ecdsa.PublicKey, 0)
	for i := 0; i < Ne; i++ {
		PublicKeys = append(PublicKeys, &PrivateKeys[i].PublicKey)
	}

	//Secret Message Array Generation (e.g. PK[0])
	//-------step 1-------------------------------------------
	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	//Commitment = PublicKey || alpha * PublicKey
	Commitment, err := GenerateCommitment(PublicKeys[0], alpha)
	if err != nil {
		t.Fatal(err)
	}

	//tx send Commitment
	//--------------------------------------------------------

	//-------step 2-------------------------------------------
	PublicKeys, alphaPublicKeys, Proof, err := GenerateArrayPiece(PublicKeys, alpha)
	if err != nil {
		t.Fatal(err)
	}

	//tx send PublicKeys, alphaPublicKeys, Proof
	//--------------------------------------------------------

	//--------box 3-------------------------------------------
	if VerifyArrayPiece(Commitment, PublicKeys, alphaPublicKeys, Proof) {
		fmt.Println("PK[0] Generate Message Array Verification Succeed!")
	} else {
		fmt.Println("PK[0] Generate Message Array Verification Failed!")
	}
	//--------------------------------------------------------

	//Contruct SMA
	//--------simualte of receive stage 2's tx ArrayPiece. Got only pk of local----Box 4.1 and Box 4.2-------
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
		piece.X, piece.Y = crypto.S256().ScalarMult(PrivateKeys[0].PublicKey.X, PrivateKeys[0].PublicKey.Y, alphas[i].Bytes())
		ArrayReceived = append(ArrayReceived, piece)
	}
	//ArrayReceived tx send?
	//---------------------------------------------------------------

	//----------Box 4.3---------------------------------------------------
	SMA, err := GenerateSMA(PrivateKeys[0], ArrayReceived)
	if err != nil {
		t.Fatal(err)
	} else {
		fmt.Println("Secret Message Array Contruction Succeed!")
	}
	//This epoch over
	//SMA tx send?
	//--------------------------------------------------------------

	//Next epoch start
	//Leader Selection and Verification
	//---------Box 5-----------------------------------------------
	SlotLeaderSeq, cr, err := GenerateSlotLeaderSeq(SMA, PublicKeys, RB, EL)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Slot Leader Sequence Generation Succeed!")
	//-------------------------------------------------------------

	for i := 0; i < EL; i++ {
		if PublicKeyEqual(&PrivateKeys[0].PublicKey, SlotLeaderSeq[i]) {
			fmt.Println(i)
			ProofMeg, Proof, err := GenerateSlotLeaderProof(PrivateKeys[0], SMA, PublicKeys, RB, cr, i)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println("Slot Leader Proof Generation Succeed!")
			//Verify the sum of alphaPK[0] equals to ProofMeg[2]
			na := len(ArrayReceived)
			skGt := new(ecdsa.PublicKey)
			skGt.Curve = crypto.S256()
			if i == 0 {
				skGt.X = new(big.Int).Set(ArrayReceived[0].X)
				skGt.Y = new(big.Int).Set(ArrayReceived[0].Y)
				for j := 1; j < na; j++ {
					skGt.X, skGt.Y = crypto.S256().Add(skGt.X, skGt.Y, ArrayReceived[j].X, ArrayReceived[j].Y)
				}
			} else {
				que := 0
				for t := 0; t < na; t++ {
					if cr[i-1].Bit(t) == 1 {
						if que == 1 {
							skGt.X, skGt.Y = crypto.S256().Add(skGt.X, skGt.Y, ArrayReceived[t].X, ArrayReceived[t].Y)
						} else if que == 0 {
							skGt.X = new(big.Int).Set(ArrayReceived[t].X)
							skGt.Y = new(big.Int).Set(ArrayReceived[t].Y)
							que = 1
						}

					}
				}
			}
			if PublicKeyEqual(skGt, ProofMeg[2]) {
				if VerifySlotLeaderProof(Proof, ProofMeg, PublicKeys, RB) {
					fmt.Println("Slot Leader Proof Verification Succeed!")
				}
			} else {
				fmt.Println("Slot Leader Proof Verification Failed!")
			}
		}
	}
}

//Test Sample Functions below

func genPublicKeys(x int) ([]*ecdsa.PublicKey, error) {
	if x <= 0 {
		return nil, ErrInvalidgenPublicKeys
	}
	PublicKeys := make([]*ecdsa.PublicKey, 0) //PublicKey Samples
	for i := 0; i < x; i++ {
		privateksample, err := crypto.GenerateKey()
		if err != nil {
			return nil, ErrPublickeyGeneration
		}
		PublicKeys = append(PublicKeys, &privateksample.PublicKey)
	}
	return PublicKeys, nil
}

func genProbabilities(x int) ([]*float64, error) {
	if x <= 0 {
		return nil, ErrInvalidgenProbabilities
	}
	Probabilities := make([]*float64, 0)
	for i := 0; i < x; i++ {
		probability := rand.New(rand.NewSource(int64(i))).Float64()
		Probabilities = append(Probabilities, &probability)
	}
	return Probabilities, nil
}

func genPrivateKeys(x int) ([]*ecdsa.PrivateKey, error) {
	if x <= 0 {
		return nil, ErrInvalidgenPrivateKeys
	}
	PrivateKeys := make([]*ecdsa.PrivateKey, 0)
	for i := 0; i < x; i++ {
		privateksample, err := crypto.GenerateKey()
		if err != nil {
			return nil, ErrInvalidgenPrivateKeys
		}
		PrivateKeys = append(PrivateKeys, privateksample)
	}
	return PrivateKeys, nil
}

func SortPrivateKey(PrivateKeys []*ecdsa.PrivateKey, RB []byte) ([]*ecdsa.PrivateKey, error) {
	if len(PrivateKeys) == 0 || RB == nil {
		return nil, ErrInvalidSortPrivateKeys
	}
	for _, privatekey := range PrivateKeys {
		if privatekey == nil || &privatekey.PublicKey == nil || privatekey.D == nil || privatekey.PublicKey.X == nil || privatekey.PublicKey.Y == nil {
			return nil, ErrInvalidSortPrivateKeys
		}
	}
	hasharray := make([]*big.Int, 0)
	n := len(PrivateKeys)
	for i := 0; i < n; i++ {
		var buffer bytes.Buffer
		buffer.Write(RB)
		buffer.Write(crypto.FromECDSAPub(&PrivateKeys[i].PublicKey))
		temp := buffer.Bytes()
		tempbyte := crypto.Keccak256(temp)
		tempBig := new(big.Int).SetBytes(tempbyte)
		hasharray = append(hasharray, tempBig)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if hasharray[i].Cmp(hasharray[j]) == 1 {
				hasharray[i], hasharray[j] = hasharray[j], hasharray[i]
				PrivateKeys[i], PrivateKeys[j] = PrivateKeys[j], PrivateKeys[i]
			}
		}
	}
	return PrivateKeys, nil
}
