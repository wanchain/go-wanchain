//package uleaderselection
//Algorithms for unique leader selection
//including Random proposer + Epoch leader selection, Secret Message Array Construction, Slot Leader Selection and DLEQ Proof Generation
package uleaderselection

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/wanchain/go-wanchain/log"
	"io"
	"math/big"

	"github.com/wanchain/go-wanchain/crypto"
)

//Parameters
var (
	Big1                                   = big.NewInt(1)
	Big0                                   = big.NewInt(0)
	ErrInvalidRandomProposerSelection      = errors.New("Invalid Random Proposer Selection")                  //Invalid Random Proposer Selection
	ErrInvalidProbabilityfloat2big         = errors.New("Invalid Transform Probability From Float To Bigint") //Invalid Transform Probability From Float To Bigint
	ErrInvalidGenerateCommitment           = errors.New("Invalid Commitment Generation")                      //Invalid Commitment Generation
	ErrInvalidArrayPieceGeneration         = errors.New("Invalid ArrayPiece Generation")                      //Invalid ArrayPiece Generation
	ErrInvalidDleqProofGeneration          = errors.New("Invalid DLEQ Proof Generation")                      //Invalid DLEQ Proof Generation
	ErrInvalidSecretMessageArrayGeneration = errors.New("Invalid Secret Message Array Generation")            //Invalid Secret Message Array Generation
	ErrInvalidSortPublicKeys               = errors.New("Invalid PublicKeys Sort Operation")                  //Invalid PublicKeys Sort Operation
	ErrInvalidSlotLeaderSequenceGeneration = errors.New("Invalid Slot Leader Sequence Generation")            //Invalid Slot Leader Sequence Generation
	ErrInvalidSlotLeaderLocation           = errors.New("Invalid Slot Leader Location")                       //Invalid Slot Leader Location
	ErrInvalidSlotLeaderProofGeneration    = errors.New("Invalid Slot Leader Proof Generation")               //Invalid Slot Leader Proof Generation

	ErrInvalidPrivateKey = errors.New("private key is nil")
	ErrInvalidSMA        = errors.New("SMA is nil")
	ErrInvalidPublicKey  = errors.New("public key is nil")
	ErrSortPublicKey     = errors.New("sort public key error")
	ErrPublicKeyNotEqual = errors.New("public key is not equal")
)

const Accuracy float64 = 1024.0 //accuracy to magnificate

/*_________________________________________________Random proposer + Epoch leader selection___________________________________________________*/

//Transform Probabilities from float to bigint
func ProbabilityFloat2big(Probabilities []*float64) ([]*big.Int, error) {
	if len(Probabilities) == 0 {
		return nil, ErrInvalidProbabilityfloat2big
	}
	for _, probability := range Probabilities {
		if probability == nil {
			return nil, ErrInvalidProbabilityfloat2big
		}
	}

	n := len(Probabilities)
	var temp int64
	var probabilitiesBig = make([]*big.Int, n) //probabilities_big as new probability array

	for i := 0; i < n; i++ {
		temp = int64(*Probabilities[i] * Accuracy)
		probabilitiesBig[i] = big.NewInt(temp)
	}

	return probabilitiesBig, nil
}

//samples nr random proposers by random number r（Random Beacon) from PublicKeys based on proportion of Probabilities
func RandomProposerSelection(r []byte, nr int, PublicKeys []*ecdsa.PublicKey, Probabilities []*float64) ([]*ecdsa.PublicKey, error) {
	if r == nil || nr <= 0 || len(PublicKeys) == 0 || len(Probabilities) == 0 || len(PublicKeys) != len(Probabilities) {
		return nil, ErrInvalidRandomProposerSelection
	}
	for _, publicKey := range PublicKeys {
		if publicKey == nil || publicKey.X == nil || publicKey.Y == nil {
			return nil, ErrInvalidRandomProposerSelection
		}
	}
	for _, probability := range Probabilities {
		if probability == nil {
			return nil, ErrInvalidRandomProposerSelection
		}
	}

	probabilitiesBig, _ := ProbabilityFloat2big(Probabilities) //transform probabilities from float64 to bigint
	tp := new(big.Int).SetInt64(0)                             //total probability of probabilities_big
	randomProposerPublicKeys := make([]*ecdsa.PublicKey, 0)    //store the selected publickeys
	n := len(probabilitiesBig)

	for _, probabilityBig := range probabilitiesBig {
		tp.Add(tp, probabilityBig)
	}
	var Byte0 = []byte{byte(0)}
	var buffer bytes.Buffer
	buffer.Write(Byte0)
	buffer.Write(r)
	r0 := buffer.Bytes()       //r0 = 0||r
	cr := crypto.Keccak256(r0) //cr = hash(r0)

	for i := 0; i < nr; i++ {

		crBig := new(big.Int).SetBytes(cr)
		crBig.Mod(crBig, tp) //cr_big = cr mod tp

		//select pki whose probability bigger than cr_big left
		sumtemp := new(big.Int).SetInt64(0)
		for j := 0; j < n; j++ {
			sumtemp.Add(sumtemp, probabilitiesBig[j])
			if sumtemp.Cmp(crBig) == 1 {
				pkselected := new(ecdsa.PublicKey) //new publickey to store the selected one
				pkselected.Curve = crypto.S256()
				pkselected.X = new(big.Int).Set(PublicKeys[j].X)
				pkselected.Y = new(big.Int).Set(PublicKeys[j].Y)
				randomProposerPublicKeys = append(randomProposerPublicKeys, pkselected)
				break
			}
		}
		cr = crypto.Keccak256(cr)
	}

	return randomProposerPublicKeys, nil
}

/*_________________________________________________Secret Message Array Construction___________________________________________________*/

//GenerateCommitment compute the commitment of PublicKey, Commitment = PublicKey || alpha * PublicKey
func GenerateCommitment(PublicKey *ecdsa.PublicKey, alpha *big.Int) ([]*ecdsa.PublicKey, error) {
	if PublicKey == nil || PublicKey.X == nil || PublicKey.Y == nil || PublicKey.Curve == nil || alpha.Cmp(Big0) == 0 || alpha.Cmp(Big1) == 0 {
		return nil, ErrInvalidGenerateCommitment
	}
	Commitment := make([]*ecdsa.PublicKey, 0)

	publickey := new(ecdsa.PublicKey) //copy publickey from PublicKey
	publickey.Curve = crypto.S256()
	publickey.X = new(big.Int).Set(PublicKey.X)
	publickey.Y = new(big.Int).Set(PublicKey.Y)
	Commitment = append(Commitment, publickey)

	commit := new(ecdsa.PublicKey)
	commit.Curve = crypto.S256()
	commit.X, commit.Y = crypto.S256().ScalarMult(PublicKey.X, PublicKey.Y, alpha.Bytes()) //commit = alpha * PublicKey
	Commitment = append(Commitment, commit)                                                //Commitment = PublicKey || alpha * PublicKey
	return Commitment, nil
}

//GenerateArrayPiece compute message sent out, where ArrayPiece = (alpha * Pk1, alpha * Pk2, ..., alpha * Pkn)
//Additional DLEQ proof needs to be added
func GenerateArrayPiece(PublicKeys []*ecdsa.PublicKey, alpha *big.Int) ([]*ecdsa.PublicKey, []*ecdsa.PublicKey, []*big.Int, error) {
	if len(PublicKeys) == 0 || alpha.Cmp(Big0) == 0 || alpha.Cmp(Big1) == 0 {
		return nil, nil, nil, ErrInvalidArrayPieceGeneration
	}
	ArrayPiece := make([]*ecdsa.PublicKey, 0)
	n := len(PublicKeys)
	for i := 0; i < n; i++ {
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
	proof, err := DleqProofGeneration(PublicKeys, ArrayPiece, alpha)
	if err != nil {
		return nil, nil, nil, ErrInvalidArrayPieceGeneration
	}
	return PublicKeys, ArrayPiece, proof, nil

}

//VerifyArrayPiece validates the encrypted message array
func VerifyArrayPiece(Commitment []*ecdsa.PublicKey, PublicKeys []*ecdsa.PublicKey, ArrayPiece []*ecdsa.PublicKey, Proof []*big.Int) bool {
	if len(Commitment) != 2 || len(PublicKeys) == 0 || len(ArrayPiece) == 0 || len(PublicKeys) != len(ArrayPiece) || len(Proof) != 2 {
		return false
	}
	n := len(PublicKeys)
	//verify the commitment coordinates with ArrayPiece
	temp := 0
	for i := 0; i < n; i++ {
		//if PublicKey is the same, commitment must equals to alphaPublicKey
		if PublicKeyEqual(Commitment[0], PublicKeys[i]) {
			temp = temp + 1
			if !PublicKeyEqual(Commitment[1], ArrayPiece[i]) {
				return false
			}
		}
	}
	//if no commitment before, ArrayPiece is not valid
	if temp == 0 {
		return false
	}
	//Verify the DLEQ Proof
	return VerifyDleqProof(PublicKeys, ArrayPiece, Proof)
}

//PublicKeyEqual test the equavalance of two publickey
func PublicKeyEqual(PublicKey1 *ecdsa.PublicKey, PublicKey2 *ecdsa.PublicKey) bool {
	if PublicKey1 == nil || PublicKey2 == nil {
		return false
	}
	return PublicKey1.Curve == PublicKey2.Curve && PublicKey1.X.Cmp(PublicKey2.X) == 0 && PublicKey1.Y.Cmp(PublicKey2.Y) == 0
}

//GenerateSMA compute the Secret Message Array from the array piece received
//need to sort the array received based on PublicKeys in advance
func GenerateSMA(PrivateKey *ecdsa.PrivateKey, ArrayPiece []*ecdsa.PublicKey) ([]*ecdsa.PublicKey, error) {
	if PrivateKey == nil || PrivateKey.D == nil || len(ArrayPiece) == 0 {
		log.SyslogErr("uleaderselection", "GenerateSMA error", ErrInvalidSecretMessageArrayGeneration.Error())
		return nil, ErrInvalidSecretMessageArrayGeneration
	}
	for _, piece := range ArrayPiece {
		if piece == nil || piece.X == nil || piece.Y == nil {
			log.SyslogErr("uleaderselection", "GenerateSMA pieces error", ErrInvalidSecretMessageArrayGeneration.Error())
			return nil, ErrInvalidSecretMessageArrayGeneration
		}
	}
	//calculate the inverse of privatekey
	skInverse := new(big.Int).ModInverse(PrivateKey.D, crypto.S256().Params().N)
	//SMA = skInverse * ArrayPiece
	SMA := make([]*ecdsa.PublicKey, 0)
	n := len(ArrayPiece)
	for i := 0; i < n; i++ {
		spiece := new(ecdsa.PublicKey)
		spiece.Curve = crypto.S256()
		spiece.X, spiece.Y = crypto.S256().ScalarMult(ArrayPiece[i].X, ArrayPiece[i].Y, skInverse.Bytes())
		SMA = append(SMA, spiece)
	}

	return SMA, nil
}

/*_____________________________________________________Slot Leader Selection_______________________________________________________*/

//SortPublicKeys sort the publickeys by random beacon to produce a publickey sequence
func SortPublicKeys(PublicKeys []*ecdsa.PublicKey, RB []byte) ([]*ecdsa.PublicKey, error) {
	if len(PublicKeys) == 0 || RB == nil {
		return nil, ErrInvalidSortPublicKeys
	}
	for _, publickey := range PublicKeys {
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return nil, ErrInvalidSortPublicKeys
		}
	}
	hasharray := make([]*big.Int, 0)
	n := len(PublicKeys)
	for i := 0; i < n; i++ {
		var buffer bytes.Buffer
		buffer.Write(RB)
		buffer.Write(crypto.FromECDSAPub(PublicKeys[i]))
		temp := buffer.Bytes()
		tempbyte := crypto.Keccak256(temp)
		tempBig := new(big.Int).SetBytes(tempbyte)
		hasharray = append(hasharray, tempBig)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if hasharray[i].Cmp(hasharray[j]) == 1 {
				hasharray[i], hasharray[j] = hasharray[j], hasharray[i]
				PublicKeys[i], PublicKeys[j] = PublicKeys[j], PublicKeys[i]
			}
		}
	}
	return PublicKeys, nil
}

//Sort PublicKeys sequence by hash from min to max
func Sort(PublicKeys []*ecdsa.PublicKey) ([]*ecdsa.PublicKey, error) {
	if len(PublicKeys) == 0 || PublicKeys == nil {
		return nil, ErrInvalidSortPublicKeys
	}
	for _, publickey := range PublicKeys {
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return nil, ErrInvalidSortPublicKeys
		}
	}
	hasharray := make([]*big.Int, 0)
	n := len(PublicKeys)
	for i := 0; i < n; i++ {
		tempBig := new(big.Int).SetBytes(crypto.Keccak256(crypto.FromECDSAPub(PublicKeys[i])))
		hasharray = append(hasharray, tempBig)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if hasharray[i].Cmp(hasharray[j]) == 1 {
				hasharray[i], hasharray[j] = hasharray[j], hasharray[i]
				PublicKeys[i], PublicKeys[j] = PublicKeys[j], PublicKeys[i]
			}
		}
	}
	return PublicKeys, nil
}

//GenerateSlotLeaderSeq produce the sequence of the slot leaders, SMA length limited < 256
func GenerateSlotLeaderSeq(SMA []*ecdsa.PublicKey, PublicKeys []*ecdsa.PublicKey, RB []byte, epochlen int) ([]*ecdsa.PublicKey, []*big.Int, error) {
	if len(SMA) == 0 || len(PublicKeys) == 0 || RB == nil || epochlen <= 0 {
		return nil, nil, ErrInvalidSlotLeaderSequenceGeneration
	}
	for _, piece := range SMA {
		if piece == nil || piece.X == nil || piece.Y == nil {
			return nil, nil, ErrInvalidSlotLeaderSequenceGeneration
		}
	}
	for _, publickey := range PublicKeys {
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return nil, nil, ErrInvalidSlotLeaderSequenceGeneration
		}
	}
	//make the sequence of epoch leaders by Random Beacon

	var err error
	PublicKeys, err = SortPublicKeys(PublicKeys, RB)
	if err != nil {
		return nil, nil, ErrInvalidSortPublicKeys
	}
	//calculate the cr sequence
	cr := make([]*big.Int, 0)
	na := len(SMA)
	//cr[0] = hash(alpha1*G+alpha2*G+...+alphan*G)
	sumpub := new(ecdsa.PublicKey)
	sumpub.Curve = crypto.S256()
	sumpub.X = new(big.Int).Set(SMA[0].X)
	sumpub.Y = new(big.Int).Set(SMA[0].Y)
	for i := 1; i < na; i++ {
		sumpub.X, sumpub.Y = Wadd(sumpub.X, sumpub.Y, SMA[i].X, SMA[i].Y)
	}
	cr0 := new(big.Int).SetBytes(crypto.Keccak256(crypto.FromECDSAPub(sumpub)))
	cr = append(cr, cr0)
	//cr[i] = hash(bi-1,0*alpha1*G+bi-1,1*alpha2*G+...+bi-1,n-1*alphan*G)
	//cr[i-1]=bi-1,0||bi-1,1||...||bi-1,n-1
	for i := 1; i < epochlen; i++ {
		temp := new(ecdsa.PublicKey)
		temp.Curve = crypto.S256()
		que := 0
		for j := 0; j < na; j++ {
			if cr[i-1].Bit(j) == 1 {
				if que == 1 {
					temp.X, temp.Y = Wadd(temp.X, temp.Y, SMA[j].X, SMA[j].Y)
				} else if que == 0 {
					temp.X = new(big.Int).Set(SMA[j].X)
					temp.Y = new(big.Int).Set(SMA[j].Y)
					que = 1
				}

			}
		}
		if temp.X == nil { //if bi-1,x are all 0, set cr[i] = hash(RB)
			//cri := new(big.Int).SetBytes(crypto.Keccak256(RB))
			cri := cr0
			cr = append(cr, cri)
		} else {
			cri := new(big.Int).SetBytes(crypto.Keccak256(crypto.FromECDSAPub(temp)))
			cr = append(cr, cri)
		}
	}
	//calculate the slot leader sequence
	SlotLeaderSeq := make([]*ecdsa.PublicKey, 0)
	choicelen := new(big.Int).SetInt64(int64(len(PublicKeys)))
	//cs[i] = cr[i] mod n, n is the number of PublicKeys
	for i := 0; i < epochlen; i++ {
		cstemp := new(big.Int).Mod(cr[i], choicelen).Int64()
		tempsl := new(ecdsa.PublicKey)
		tempsl.Curve = crypto.S256()
		tempsl.X = new(big.Int).Set(PublicKeys[cstemp].X)
		tempsl.Y = new(big.Int).Set(PublicKeys[cstemp].Y)
		SlotLeaderSeq = append(SlotLeaderSeq, tempsl)
	}
	//return cr to calculate slot leader proof
	return SlotLeaderSeq, cr, nil
}

//GenerateSlotLeaderProof produce the proof of being the slt slot leader
func GenerateSlotLeaderProof(PrivateKey *ecdsa.PrivateKey, SMA []*ecdsa.PublicKey, PublicKeys []*ecdsa.PublicKey, RB []byte, cr []*big.Int, slt int) ([]*ecdsa.PublicKey, []*big.Int, error) {
	if PrivateKey == nil || PrivateKey.D == nil || &PrivateKey.PublicKey == nil || PrivateKey.PublicKey.X == nil || PrivateKey.PublicKey.Y == nil || len(SMA) == 0 || len(PublicKeys) == 0 || RB == nil || len(cr) == 0 || len(cr) <= slt {
		return nil, nil, ErrInvalidPrivateKey
	}
	for _, piece := range SMA {
		if piece == nil || piece.X == nil || piece.Y == nil {
			return nil, nil, ErrInvalidSMA
		}
	}
	for _, publickey := range PublicKeys {
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return nil, nil, ErrInvalidPublicKey
		}
	}
	//make the sequence of epoch leaders by Random Beacon
	var err error
	PublicKeys, err = SortPublicKeys(PublicKeys, RB)
	if err != nil {
		return nil, nil, ErrSortPublicKey
	}
	//if it is the leader of slt slot, then calculate ProofMeg = [PK , Gt , skGt] and Proof = [e ,z]
	choicelen := new(big.Int).SetInt64(int64(len(PublicKeys)))
	csbigtemp := new(big.Int).Mod(cr[slt], choicelen)
	tempint := csbigtemp.Int64()
	if PublicKeyEqual(&PrivateKey.PublicKey, PublicKeys[tempint]) {
		ProofMeg := make([]*ecdsa.PublicKey, 0)
		//Copy PK to ProofMeg[0]
		pk := new(ecdsa.PublicKey)
		pk.Curve = crypto.S256()
		pk.X = new(big.Int).Set(PrivateKey.PublicKey.X)
		pk.Y = new(big.Int).Set(PrivateKey.PublicKey.Y)
		ProofMeg = append(ProofMeg, pk)
		//calculate Gt = bi-1,0*alpha1*G+bi-1,1*alpha2*G+...+bi-1,n-1*alphan*G
		na := len(SMA)
		Gt := new(ecdsa.PublicKey)
		Gt.Curve = crypto.S256()
		if slt == 0 {
			Gt.X = new(big.Int).Set(SMA[0].X)
			Gt.Y = new(big.Int).Set(SMA[0].Y)
			for i := 1; i < na; i++ {
				Gt.X, Gt.Y = Wadd(Gt.X, Gt.Y, SMA[i].X, SMA[i].Y)
			}
		} else {
			que := 0
			for j := 0; j < na; j++ {
				if cr[slt-1].Bit(j) == 1 {
					if que == 1 {
						Gt.X, Gt.Y = Wadd(Gt.X, Gt.Y, SMA[j].X, SMA[j].Y)
					} else if que == 0 {
						Gt.X = new(big.Int).Set(SMA[j].X)
						Gt.Y = new(big.Int).Set(SMA[j].Y)
						que = 1
					}
				}
			}
			if Gt == nil || Gt.X == nil || Gt.Y == nil {
				Gt.X = new(big.Int).Set(SMA[0].X)
				Gt.Y = new(big.Int).Set(SMA[0].Y)
				for i := 1; i < na; i++ {
					Gt.X, Gt.Y = Wadd(Gt.X, Gt.Y, SMA[i].X, SMA[i].Y)
				}
			}
		}
		//set Gt to ProofMeg[1]
		ProofMeg = append(ProofMeg, Gt)
		//calculate skGt = bi-1,0*alpha1*PK+bi-1,1*alpha2*PK+...+bi-1,n-1*alphan*PK
		skGt := new(ecdsa.PublicKey)
		skGt.Curve = crypto.S256()
		if Gt == nil || Gt.X == nil || Gt.Y == nil || PrivateKey == nil || PrivateKey.D == nil {
			fmt.Printf("Gt:%v, Gt.X:%v, Gt.Y:%v, PrivateKey:%v, PrivateKey.D:%v\n", Gt, Gt.X, Gt.Y, PrivateKey, PrivateKey.D)
			fmt.Printf("len(SMA):%d, slt:%d, cr[slt-1]:%s, na:%d \n", len(SMA), slt, cr[slt-1].String(), na)
			return nil, nil, ErrInvalidSlotLeaderProofGeneration
		}
		skGt.X, skGt.Y = crypto.S256().ScalarMult(Gt.X, Gt.Y, PrivateKey.D.Bytes())
		//set skGt to ProofMeg[2]
		ProofMeg = append(ProofMeg, skGt)
		//Generate DLEQ Proof (G, PK, Gt, skGt) = [e,z]
		Pks := make([]*ecdsa.PublicKey, 0)
		skPks := make([]*ecdsa.PublicKey, 0)
		BasePoint := new(ecdsa.PublicKey)
		BasePoint.Curve = crypto.S256()
		BasePoint.X, BasePoint.Y = crypto.S256().ScalarBaseMult(Big1.Bytes())
		Pks = append(Pks, BasePoint)
		Pks = append(Pks, Gt)
		skPks = append(skPks, &PrivateKey.PublicKey)
		skPks = append(skPks, skGt)
		Proof, err := DleqProofGeneration(Pks, skPks, PrivateKey.D)
		if err != nil {
			return nil, nil, ErrInvalidSlotLeaderProofGeneration
		}
		return ProofMeg, Proof, nil
	}
	return nil, nil, ErrPublicKeyNotEqual
}

//VerifySlotLeaderProof validates the proof of being the slot leader
//need a verification before that the message array received by PublicKey(ProofMeg[0]) equals to ProofMeg[2]
func VerifySlotLeaderProof(Proof []*big.Int, ProofMeg []*ecdsa.PublicKey, PublicKeys []*ecdsa.PublicKey, RB []byte) bool {
	if len(Proof) != 2 || len(ProofMeg) != 3 || len(PublicKeys) == 0 || RB == nil {
		return false
	}
	for _, piece := range ProofMeg {
		if piece == nil || piece.X == nil || piece.Y == nil {
			return false
		}
	}
	for _, piece := range PublicKeys {
		if piece == nil || piece.X == nil || piece.Y == nil {
			return false
		}
	}
	for _, piece := range Proof {
		if piece == nil {
			return false
		}
	}
	//sort the PublicKeys
	var err error
	PublicKeys, err = SortPublicKeys(PublicKeys, RB)
	if err != nil {
		return false
	}
	//calculate cr = hash(ProofMeg[1]) and cs = cr mod n, n is the number of PublicKeys
	cr := new(big.Int).SetBytes(crypto.Keccak256(crypto.FromECDSAPub(ProofMeg[1])))
	choicelen := new(big.Int).SetInt64(int64(len(PublicKeys)))
	cs := new(big.Int).Mod(cr, choicelen).Int64()
	//Verify the chosen PublicKey is ProofMeg[0]
	if !PublicKeyEqual(ProofMeg[0], PublicKeys[cs]) {
		return false
	} else {
		//Verify DLEQ Proof
		Pks := make([]*ecdsa.PublicKey, 0)
		skPks := make([]*ecdsa.PublicKey, 0)
		BasePoint := new(ecdsa.PublicKey)
		BasePoint.Curve = crypto.S256()
		BasePoint.X, BasePoint.Y = crypto.S256().ScalarBaseMult(Big1.Bytes())
		Pks = append(Pks, BasePoint)
		Pks = append(Pks, ProofMeg[1])
		skPks = append(skPks, ProofMeg[0])
		skPks = append(skPks, ProofMeg[2])
		return VerifyDleqProof(Pks, skPks, Proof)
	}
}

//GetSlotLeader find the slot leader of slt slot
func GetSlotLeader(SMA []*ecdsa.PublicKey, PublicKeys []*ecdsa.PublicKey, RB []byte, epochlen int, slt int) (*ecdsa.PublicKey, error) {
	if len(SMA) == 0 || len(PublicKeys) == 0 || RB == nil || epochlen <= 0 || slt <= 0 || slt >= epochlen {
		return nil, ErrInvalidSlotLeaderLocation
	}
	SlotLeaderSeq, _, err := GenerateSlotLeaderSeq(SMA, PublicKeys, RB, epochlen)
	if err != nil {
		return nil, ErrInvalidSlotLeaderLocation
	}
	return SlotLeaderSeq[slt], nil
}

/*_________________________________________________________DLEQ Proof_________________________________________________________________*/
func RandFieldElement(rand io.Reader) (k *big.Int, err error) {
	return randFieldElement(rand)
}

//randFieldElement generate a random number in the order of the group generated by basepoint G
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

//DlequProofGeneration generate the DLEQ Proof
//PublicKeys = [PK1, PK2, ...,Pkn]
//AlphaPublicKeys = [alpha*PK1, alpha*PK2,...,alpha*PKn]
//return Proof = [e,z]
func DleqProofGeneration(PublicKeys []*ecdsa.PublicKey, AlphaPublicKeys []*ecdsa.PublicKey, alpha *big.Int) ([]*big.Int, error) {
	if len(PublicKeys) == 0 || len(AlphaPublicKeys) == 0 || len(PublicKeys) != len(AlphaPublicKeys) || alpha.Cmp(Big0) == 0 || alpha.Cmp(Big1) == 0 {
		return nil, ErrInvalidDleqProofGeneration
	}
	n := len(PublicKeys)
	var ebuffer bytes.Buffer
	for i := 0; i < n; i++ {
		ebuffer.Write(crypto.FromECDSAPub(PublicKeys[i]))
		ebuffer.Write(crypto.FromECDSAPub(AlphaPublicKeys[i]))
	}
	w := make([]*big.Int, 2)
	var err error
	w[1], err = randFieldElement(rand.Reader)
	if err != nil {
		return nil, err
	}
	for i := 0; i < n; i++ {
		wpublickey := new(ecdsa.PublicKey)
		wpublickey.Curve = crypto.S256()
		wpublickey.X, wpublickey.Y = crypto.S256().ScalarMult(PublicKeys[i].X, PublicKeys[i].Y, w[1].Bytes()) //wPi = w * Pi
		ebuffer.Write(crypto.FromECDSAPub(wpublickey))
	}
	ebyte := crypto.Keccak256(ebuffer.Bytes())
	e := new(big.Int).SetInt64(0)
	e.SetBytes(ebyte) //e = hash(P1,alphaP1,P2,alphaP2,...,wP1,...,wPn) mod N

	w[0] = e
	alphae := new(big.Int).Mul(alpha, e)
	alphae.Mod(alphae, crypto.S256().Params().N)
	w[1].Sub(w[1], alphae)
	w[1].Mod(w[1], crypto.S256().Params().N)

	return w, nil
}

func VerifyDleqProof(PublicKeys []*ecdsa.PublicKey, AlphaPublicKeys []*ecdsa.PublicKey, Proof []*big.Int) bool {
	if len(PublicKeys) == 0 || len(AlphaPublicKeys) == 0 || len(PublicKeys) != len(AlphaPublicKeys) || len(Proof) != 2 {
		return false
	}
	n := len(PublicKeys)
	var ebuffer bytes.Buffer
	for i := 0; i < n; i++ {
		ebuffer.Write(crypto.FromECDSAPub(PublicKeys[i]))
		ebuffer.Write(crypto.FromECDSAPub(AlphaPublicKeys[i]))
	}
	for i := 0; i < n; i++ {
		wLpublickey := new(ecdsa.PublicKey)
		wLpublickey.Curve = crypto.S256()
		wLpublickey.X, wLpublickey.Y = crypto.S256().ScalarMult(PublicKeys[i].X, PublicKeys[i].Y, Proof[1].Bytes())

		wRpublickey := new(ecdsa.PublicKey)
		wRpublickey.Curve = crypto.S256()
		wRpublickey.X, wRpublickey.Y = crypto.S256().ScalarMult(AlphaPublicKeys[i].X, AlphaPublicKeys[i].Y, Proof[0].Bytes())

		wLpublickey.X, wLpublickey.Y = Wadd(wLpublickey.X, wLpublickey.Y, wRpublickey.X, wRpublickey.Y)

		ebuffer.Write(crypto.FromECDSAPub(wLpublickey))
	}
	ebyte := crypto.Keccak256(ebuffer.Bytes())
	e := new(big.Int).SetInt64(0)
	e.SetBytes(ebyte)
	return e.Cmp(Proof[0]) == 0
}

func SortPublicKeysAndIndex(PublicKeys []*ecdsa.PublicKey, RB []byte) ([]*ecdsa.PublicKey, []uint64, error) {
	if len(PublicKeys) == 0 || RB == nil {
		return nil, nil, ErrInvalidSortPublicKeys
	}
	for _, publickey := range PublicKeys {
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return nil, nil, ErrInvalidSortPublicKeys
		}
	}
	hasharray := make([]*big.Int, 0)
	publicKeyIndex := make([]uint64, len(PublicKeys))
	n := len(PublicKeys)
	for i := 0; i < n; i++ {
		var buffer bytes.Buffer
		buffer.Write(RB)
		buffer.Write(crypto.FromECDSAPub(PublicKeys[i]))
		temp := buffer.Bytes()
		tempbyte := crypto.Keccak256(temp)
		tempBig := new(big.Int).SetBytes(tempbyte)
		hasharray = append(hasharray, tempBig)

		publicKeyIndex[i] = uint64(i)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if hasharray[i].Cmp(hasharray[j]) == 1 {
				hasharray[i], hasharray[j] = hasharray[j], hasharray[i]
				PublicKeys[i], PublicKeys[j] = PublicKeys[j], PublicKeys[i]

				publicKeyIndex[i], publicKeyIndex[j] = publicKeyIndex[j], publicKeyIndex[i]
			}
		}
	}
	return PublicKeys, publicKeyIndex, nil
}

func Wadd(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	if x1.Cmp(x2) == 0 && y1.Cmp(y2) == 0 {
		return crypto.S256().Double(x1, y1)
	} else {
		return crypto.S256().Add(x1, y1, x2, y2)
	}
}

func GenerateSlotLeaderSeqAndIndex(SMA []*ecdsa.PublicKey, PublicKeys []*ecdsa.PublicKey, RB []byte, epochlen uint64, epochID uint64) ([]*ecdsa.PublicKey, []*big.Int, []uint64, error) {
	if len(SMA) == 0 || len(PublicKeys) == 0 || RB == nil || epochlen <= 0 {
		return nil, nil, nil, ErrInvalidSlotLeaderSequenceGeneration
	}
	for _, piece := range SMA {
		if piece == nil || piece.X == nil || piece.Y == nil {
			return nil, nil, nil, ErrInvalidSlotLeaderSequenceGeneration
		}
	}
	for _, publickey := range PublicKeys {
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return nil, nil, nil, ErrInvalidSlotLeaderSequenceGeneration
		}
	}
	//make the sequence of epoch leaders by Random Beacon

	var err error
	PublicKeysIndex := make([]uint64, len(PublicKeys))
	PublicKeys, PublicKeysIndex, err = SortPublicKeysAndIndex(PublicKeys, RB)
	if err != nil {
		return nil, nil, nil, ErrInvalidSortPublicKeys
	}

	SlotLeaderSeqIndex := make([]uint64, epochlen)
	//calculate the cr sequence
	cr := make([]*big.Int, 0)
	gts := make([]*ecdsa.PublicKey, 0)
	//na := len(SMA)
	smaLen := new(big.Int).SetInt64(int64(len(SMA)))
	var i uint64
	for i = 0; i < epochlen; i++ {
		Gt := new(ecdsa.PublicKey)
		Gt.Curve = crypto.S256()

		var buffer bytes.Buffer
		buffer.Write(RB)
		buffer.Write(Uint64ToBytes(epochID))
		buffer.Write(Uint64ToBytes(uint64(i)))
		temp := buffer.Bytes()

		for i := 0; i < len(PublicKeys); i++ {
			tempHash := crypto.Keccak256(temp)
			tempBig := new(big.Int).SetBytes(tempHash)
			cstemp := new(big.Int).Mod(tempBig, smaLen)
			if i == 0 {
				Gt.X = new(big.Int).Set(SMA[cstemp.Int64()].X)
				Gt.Y = new(big.Int).Set(SMA[cstemp.Int64()].Y)
			} else {
				Gt.X, Gt.Y = Wadd(Gt.X, Gt.Y, SMA[cstemp.Int64()].X, SMA[cstemp.Int64()].Y)
			}
			temp = tempHash
		}
		crTemp := new(big.Int).SetBytes(crypto.Keccak256(crypto.FromECDSAPub(Gt)))

		gts = append(gts, Gt)
		cr = append(cr, crTemp)
	}

	//calculate the slot leader sequence
	SlotLeaderSeq := make([]*ecdsa.PublicKey, 0)
	choicelen := new(big.Int).SetInt64(int64(len(PublicKeys)))
	//cs[i] = cr[i] mod n, n is the number of PublicKeys
	for i = 0; i < epochlen; i++ {
		cstemp := new(big.Int).Mod(cr[i], choicelen).Int64()
		tempsl := new(ecdsa.PublicKey)
		tempsl.Curve = crypto.S256()
		tempsl.X = new(big.Int).Set(PublicKeys[cstemp].X)
		tempsl.Y = new(big.Int).Set(PublicKeys[cstemp].Y)
		SlotLeaderSeq = append(SlotLeaderSeq, tempsl)

		SlotLeaderSeqIndex[i] = PublicKeysIndex[cstemp]
	}
	//return cr to calculate slot leader proof
	return SlotLeaderSeq, cr, SlotLeaderSeqIndex, nil
}

func Uint64ToBytes(input uint64) []byte {
	if input == 0 {
		return []byte{0}
	}
	return big.NewInt(0).SetUint64(input).Bytes()
}

//GenerateSlotLeaderProof produce the proof of being the slt slot leader
func GenerateSlotLeaderProof2(PrivateKey *ecdsa.PrivateKey, SMA []*ecdsa.PublicKey, PublicKeys []*ecdsa.PublicKey,
	RB []byte, slt uint64, epochID uint64) ([]*ecdsa.PublicKey, []*big.Int, error) {
	if PrivateKey == nil || PrivateKey.D == nil || &PrivateKey.PublicKey == nil || PrivateKey.PublicKey.X == nil || PrivateKey.PublicKey.Y == nil || len(SMA) == 0 || len(PublicKeys) == 0 || RB == nil {
		return nil, nil, ErrInvalidPrivateKey
	}
	for _, piece := range SMA {
		if piece == nil || piece.X == nil || piece.Y == nil {
			return nil, nil, ErrInvalidSMA
		}
	}
	for _, publickey := range PublicKeys {
		if publickey == nil || publickey.X == nil || publickey.Y == nil {
			return nil, nil, ErrInvalidPublicKey
		}
	}
	//make the sequence of epoch leaders by Random Beacon
	var err error
	PublicKeys, err = SortPublicKeys(PublicKeys, RB)
	if err != nil {
		return nil, nil, ErrSortPublicKey
	}
	//if it is the leader of slt slot, then calculate ProofMeg = [PK , Gt , skGt] and Proof = [e ,z]
	choicelen := new(big.Int).SetInt64(int64(len(PublicKeys)))
	smaLen := new(big.Int).SetInt64(int64(len(SMA)))

	Gt := new(ecdsa.PublicKey)
	Gt.Curve = crypto.S256()

	var buffer bytes.Buffer
	buffer.Write(RB)
	buffer.Write(Uint64ToBytes(epochID))
	buffer.Write(Uint64ToBytes(slt))
	temp := buffer.Bytes()

	for i := 0; i < len(PublicKeys); i++ {
		tempHash := crypto.Keccak256(temp)
		tempBig := new(big.Int).SetBytes(tempHash)
		cstemp := new(big.Int).Mod(tempBig, smaLen)

		if i == 0 {
			Gt.X = new(big.Int).Set(SMA[cstemp.Int64()].X)
			Gt.Y = new(big.Int).Set(SMA[cstemp.Int64()].Y)
		} else {
			Gt.X, Gt.Y = Wadd(Gt.X, Gt.Y, SMA[cstemp.Int64()].X, SMA[cstemp.Int64()].Y)
		}
		temp = tempHash
	}

	bigTemp := new(big.Int).SetInt64(int64(0))
	bigTemp.SetBytes(crypto.Keccak256(crypto.FromECDSAPub(Gt)))
	csbigtemp := new(big.Int).Mod(bigTemp, choicelen)
	tempint := csbigtemp.Int64()

	if PublicKeyEqual(&PrivateKey.PublicKey, PublicKeys[tempint]) {
		ProofMeg := make([]*ecdsa.PublicKey, 0)
		//Copy PK to ProofMeg[0]
		pk := new(ecdsa.PublicKey)
		pk.Curve = crypto.S256()
		pk.X = new(big.Int).Set(PrivateKey.PublicKey.X)
		pk.Y = new(big.Int).Set(PrivateKey.PublicKey.Y)
		ProofMeg = append(ProofMeg, pk)
		//set Gt to ProofMeg[1]
		ProofMeg = append(ProofMeg, Gt)
		//calculate skGt = bi-1,0*alpha1*PK+bi-1,1*alpha2*PK+...+bi-1,n-1*alphan*PK
		skGt := new(ecdsa.PublicKey)
		skGt.Curve = crypto.S256()
		if Gt == nil || Gt.X == nil || Gt.Y == nil || PrivateKey == nil || PrivateKey.D == nil {
			fmt.Printf("Gt:%v, Gt.X:%v, Gt.Y:%v, PrivateKey:%v, PrivateKey.D:%v\n", Gt, Gt.X, Gt.Y, PrivateKey, PrivateKey.D)
			return nil, nil, ErrInvalidSlotLeaderProofGeneration
		}
		skGt.X, skGt.Y = crypto.S256().ScalarMult(Gt.X, Gt.Y, PrivateKey.D.Bytes())
		//set skGt to ProofMeg[2]
		ProofMeg = append(ProofMeg, skGt)
		//Generate DLEQ Proof (G, PK, Gt, skGt) = [e,z]
		Pks := make([]*ecdsa.PublicKey, 0)
		skPks := make([]*ecdsa.PublicKey, 0)
		BasePoint := new(ecdsa.PublicKey)
		BasePoint.Curve = crypto.S256()
		BasePoint.X, BasePoint.Y = crypto.S256().ScalarBaseMult(Big1.Bytes())
		Pks = append(Pks, BasePoint)
		Pks = append(Pks, Gt)
		skPks = append(skPks, &PrivateKey.PublicKey)
		skPks = append(skPks, skGt)
		Proof, err := DleqProofGeneration(Pks, skPks, PrivateKey.D)
		if err != nil {
			return nil, nil, ErrInvalidSlotLeaderProofGeneration
		}
		return ProofMeg, Proof, nil
	}
	return nil, nil, ErrPublicKeyNotEqual
}
