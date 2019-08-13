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
	SlotLeaderSeq, cr, _, err := GenerateSlotLeaderSeqAndIndex(SMA, pksamples, RB, uint64(EL),uint64(0))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(cr)
	fmt.Println(SlotLeaderSeq)
	fmt.Println("Slot Leader Sequence Generation Succeed!")
}

func TestGenerateSlotLeaderOne(t *testing.T) {
	pksamples, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	SMA, err := genPublicKeys(Ns)
	if err != nil {
		t.Fatal(err)
	}
	var RB = []byte{byte(1)}
	SlotLeaderPtr, err := GenerateSlotLeaderSeqOne(SMA, pksamples, RB, 0,uint64(0))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(SlotLeaderPtr)
	fmt.Println("Slot Leader Sequence Generation Succeed!")
}

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
	SlotLeaderSeq, cr, _, err := GenerateSlotLeaderSeqAndIndex(SMA, PublicKeys, RB, uint64(EL),uint64(0))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Slot Leader Sequence Generation Succeed!")
	//-------------------------------------------------------------

	for i := 0; i < EL; i++ {
		if PublicKeyEqual(&PrivateKeys[0].PublicKey, SlotLeaderSeq[i]) {
			fmt.Println(i)
			ProofMeg, Proof, err := GenerateSlotLeaderProof(PrivateKeys[0], SMA, PublicKeys, RB, uint64(i), uint64(0))
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
// length of Bytes of big.Int
func TestTemp(t *testing.T){
	const uintSize = 32 << (^uint(0) >> 32 & 1)
	fmt.Printf("%v\n",uintSize)

	bgTemp1 := big.NewInt(0)
	bgTemp2 := big.NewInt(100000000)
	bgTemp1.Add(bgTemp1,bgTemp2)

	for i:=2; i<20;i++{
		fmt.Printf("%v\n",len(bgTemp1.Bytes()))
		bgTemp1 = bgTemp1.Mul(bgTemp1,bgTemp2)
	}
}

func TestProofWithZero(t *testing.T) {
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

	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	PublicKeys, alphaPublicKeys, Proof, err := GenerateArrayPiece(PublicKeys, alpha)
	if err != nil {
		t.Fatal(err)
	}

	Proof[0] = big.NewInt(0).SetUint64(uint64(0))
	Proof[1] = big.NewInt(0).SetUint64(uint64(0))

	ret := VerifyDleqProof(PublicKeys, alphaPublicKeys, Proof)
	if ret {
		t.Errorf("VerifyDleqProof should return false,but true")
	}

	Proof[0] = nil
	Proof[1] = nil
	ret = VerifyDleqProof(PublicKeys, alphaPublicKeys, Proof)
	if ret {
		t.Errorf("VerifyDleqProof should return false,but true")
	}

}


func TestProofWithLongBytesLen(t *testing.T) {
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

	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	PublicKeys, alphaPublicKeys, Proof, err := GenerateArrayPiece(PublicKeys, alpha)
	if err != nil {
		t.Fatal(err)
	}

	Proof[0] = big.NewInt(0).SetUint64(uint64(^(uint64(0))))
	Proof[1] = big.NewInt(0).SetUint64(uint64(^(uint64(0))))

	Proof[0] = Proof[0].Exp(Proof[0],big.NewInt(0).SetUint64(uint64(5)),nil)
	Proof[1] = Proof[1].Exp(Proof[1],big.NewInt(0).SetUint64(uint64(5)),nil)

	fmt.Printf("Len of bytes of Proof[0]= %v\n",len(Proof[0].Bytes()))
	fmt.Printf("Len of bytes of Proof[1]= %v\n",len(Proof[1].Bytes()))

	ret := VerifyDleqProof(PublicKeys, alphaPublicKeys, Proof)
	if ret {
		t.Errorf("VerifyDleqProof should return false,but true")
	}
}

func TestDleqWithDiffAlpha(t *testing.T) {
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

	alpha, err := randFieldElement(Rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	PublicKeys, alphaPublicKeys, Proof, err := GenerateArrayPieceWithDiffAlpha(PublicKeys, alpha)
	//PublicKeys, alphaPublicKeys, Proof, err := GenerateArrayPiece(PublicKeys, alpha)
	if err != nil {
		t.Fatal(err)
	}

	ret := VerifyDleqProof(PublicKeys, alphaPublicKeys, Proof)
	if ret {
		t.Errorf("VerifyDleqProof should return false,but true")
	}
}

func GenerateArrayPieceWithDiffAlpha(PublicKeys []*ecdsa.PublicKey,
	alpha *big.Int) ([]*ecdsa.PublicKey, []*ecdsa.PublicKey, []*big.Int, error) {
	if len(PublicKeys) == 0 || alpha.Cmp(Big0) == 0 || alpha.Cmp(Big1) == 0 {
		return nil, nil, nil, ErrInvalidArrayPieceGeneration
	}
	ArrayPiece := make([]*ecdsa.PublicKey, 0)
	n := len(PublicKeys)
	for i := 0; i < n-1; i++ {
		piece := new(ecdsa.PublicKey)
		piece.Curve = crypto.S256()
		if PublicKeys[i] == nil {
			fmt.Println("------ERROR----PublicKey == nil")
			fmt.Println(PublicKeys)
			return nil, nil, nil, ErrInvalidArrayPieceGeneration
		}
		piece.X, piece.Y = crypto.S256().ScalarMult(PublicKeys[i].X, PublicKeys[i].Y, alpha.Bytes()) //piece = alpha * PublicKey
		ArrayPiece = append(ArrayPiece, piece)                                                       //ArrayPiece = (alpha * Pk1, alpha * Pk2, ..., alpha * Pkn)
	}

	// the last one element with different random.
	alphaOne := alpha.Add(alpha,Big1)
	piece := new(ecdsa.PublicKey)
	piece.Curve = crypto.S256()
	piece.X, piece.Y = crypto.S256().ScalarMult(PublicKeys[n-1].X, PublicKeys[n-1].Y, alphaOne.Bytes()) //piece = alpha * PublicKey
	ArrayPiece = append(ArrayPiece, piece)                                                       //ArrayPiece = (alpha * Pk1, alpha * Pk2, ..., alpha * Pkn)

	proof, err := DleqProofGeneration(PublicKeys, ArrayPiece, alpha)
	if err != nil {
		return nil, nil, nil, ErrInvalidArrayPieceGeneration
	}
	return PublicKeys, ArrayPiece, proof, nil

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
