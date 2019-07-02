package slotleader

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/wanchain/go-wanchain/pos/posconfig"
	"github.com/wanchain/go-wanchain/pos/uleaderselection"
	"math/big"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/crypto"
)

func Wadd(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	if x1.Cmp(x2) == 0 && y1.Cmp(y2) == 0 {
		return crypto.S256().Double(x1, y1)
	} else {
		return crypto.S256().Add(x1, y1, x2, y2)
	}
}

func VerifyDleqProof(PublicKeys []*ecdsa.PublicKey, AlphaPublicKeys []*ecdsa.PublicKey, Proof []*big.Int) bool {
	t1 := time.Now()

	if len(PublicKeys) == 0 || len(AlphaPublicKeys) == 0 || len(PublicKeys) != len(AlphaPublicKeys) || len(Proof) != 2 {
		return false
	}
	n := len(PublicKeys)
	var ebuffer bytes.Buffer
	for i := 0; i < n; i++ {
		ebuffer.Write(crypto.FromECDSAPub(PublicKeys[i]))
		ebuffer.Write(crypto.FromECDSAPub(AlphaPublicKeys[i]))
	}

	wLpublickey := new(ecdsa.PublicKey)
	wLpublickey.Curve = crypto.S256()
	wRpublickey := new(ecdsa.PublicKey)
	wRpublickey.Curve = crypto.S256()

	fmt.Println("VerifyDleqProof time 001:", time.Since(t1))

	for i := 0; i < n; i++ {

		t3 := time.Now()
		wLpublickey.X, wLpublickey.Y = crypto.S256().ScalarMult(PublicKeys[i].X, PublicKeys[i].Y, Proof[1].Bytes())
		fmt.Println("1:", time.Since(t3))

		wRpublickey.X, wRpublickey.Y = crypto.S256().ScalarMult(AlphaPublicKeys[i].X, AlphaPublicKeys[i].Y, Proof[0].Bytes())
		fmt.Println("2:", time.Since(t3))

		wLpublickey.X, wLpublickey.Y = Wadd(wLpublickey.X, wLpublickey.Y, wRpublickey.X, wRpublickey.Y)
		fmt.Println("3:", time.Since(t3))
		ebuffer.Write(crypto.FromECDSAPub(wLpublickey))
	}
	fmt.Println("VerifyDleqProof time 002:", time.Since(t1))

	ebyte := crypto.Keccak256(ebuffer.Bytes())
	e := new(big.Int).SetInt64(0)
	e.SetBytes(ebyte)
	fmt.Println("VerifyDleqProof time 003:", time.Since(t1))
	return e.Cmp(Proof[0]) == 0
}

func TestVerifyDleqProof(t *testing.T) {
	t0 := time.Now()

	pks := make([]*ecdsa.PublicKey, 50)
	alphaPks := make([]*ecdsa.PublicKey, 50)
	proof := make([]*big.Int, 2)

	for i := 0; i < len(pks); i++ {
		key, _ := crypto.GenerateKey()
		pks[i] = &key.PublicKey
		key, _ = crypto.GenerateKey()
		alphaPks[i] = &key.PublicKey

		if i < 2 {
			proof[i] = key.D
		}
	}

	t1 := time.Now()
	VerifyDleqProof(pks, alphaPks, proof)
	fmt.Println("VerifyDleqProof time:", time.Since(t1))

	fmt.Println("TestVerifyDleqProof total:", time.Since(t0))
}

func TestGetSlotLeaderProof(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()

	pks, isGenesis, err := s.getSMAPieces(0)
	if err != nil {
		t.Error(err.Error())
	}
	if !isGenesis {
		t.Fail()
	}

	if len(pks) != posconfig.EpochLeaderCount {
		t.Fail()
	}

	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		s.epochLeadersPtrArrayGenesis[i] = &prvKey.PublicKey
	}

	s.randomGenesis = big.NewInt(1)

	alphas := make([]*big.Int, 0)
	for _, value := range s.epochLeadersPtrArrayGenesis {
		tempInt := new(big.Int).SetInt64(0)
		tempInt.SetBytes(crypto.Keccak256(crypto.FromECDSAPub(value)))
		alphas = append(alphas, tempInt)
	}

	for i := 0; i < posconfig.EpochLeaderCount; i++ {

		// G
		BasePoint := new(ecdsa.PublicKey)
		BasePoint.Curve = crypto.S256()
		BasePoint.X, BasePoint.Y = crypto.S256().ScalarBaseMult(big.NewInt(1).Bytes())

		// alphaG SMAGenesis
		smaPiece := new(ecdsa.PublicKey)
		smaPiece.Curve = crypto.S256()
		smaPiece.X, smaPiece.Y = crypto.S256().ScalarMult(BasePoint.X, BasePoint.Y, alphas[i].Bytes())
		s.smaGenesis[i] = smaPiece

	}

	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof(prvKey,
		s.smaGenesis[:],
		s.epochLeadersPtrArrayGenesis[:],
		s.randomGenesis.Bytes(), 0, 0)

	if len(profMeg) != 3 || len(proof) != 2 || err != nil {
		t.Fail()
	}
}

func TestVerifySlotProofByGenesis(t *testing.T) {
	SlsInit()
	s := GetSlotLeaderSelection()
	pks, isGenesis, err := s.getSMAPieces(0)
	if err != nil {
		t.Error(err.Error())
	}
	if !isGenesis {
		t.Fail()
	}

	if len(pks) != posconfig.EpochLeaderCount {
		t.Fail()
	}

	prvKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fail()
	}

	s.randomGenesis = big.NewInt(1)

	for i := 0; i < posconfig.EpochLeaderCount; i++ {
		s.epochLeadersPtrArrayGenesis[i] = &prvKey.PublicKey
	}

	alphas := make([]*big.Int, 0)
	for _, value := range s.epochLeadersPtrArrayGenesis {
		tempInt := new(big.Int).SetInt64(0)
		tempInt.SetBytes(crypto.Keccak256(crypto.FromECDSAPub(value)))
		alphas = append(alphas, tempInt)
	}

	for i := 0; i < posconfig.EpochLeaderCount; i++ {

		// AlphaPK  stage1Genesis
		mi0 := new(ecdsa.PublicKey)
		mi0.Curve = crypto.S256()
		mi0.X, mi0.Y = crypto.S256().ScalarMult(s.epochLeadersPtrArrayGenesis[i].X, s.epochLeadersPtrArrayGenesis[i].Y,
			alphas[i].Bytes())
		s.stageOneMiGenesis[i] = mi0

		// G
		BasePoint := new(ecdsa.PublicKey)
		BasePoint.Curve = crypto.S256()
		BasePoint.X, BasePoint.Y = crypto.S256().ScalarBaseMult(big.NewInt(1).Bytes())

		// alphaG SMAGenesis
		smaPiece := new(ecdsa.PublicKey)
		smaPiece.Curve = crypto.S256()
		smaPiece.X, smaPiece.Y = crypto.S256().ScalarMult(BasePoint.X, BasePoint.Y, alphas[i].Bytes())
		s.smaGenesis[i] = smaPiece

		for j := 0; j < posconfig.EpochLeaderCount; j++ {
			// AlphaIPki stage2Genesis, used to verify genesis proof
			alphaIPkj := new(ecdsa.PublicKey)
			alphaIPkj.Curve = crypto.S256()
			alphaIPkj.X, alphaIPkj.Y = crypto.S256().ScalarMult(s.epochLeadersPtrArrayGenesis[j].X,
				s.epochLeadersPtrArrayGenesis[j].Y, alphas[i].Bytes())

			s.stageTwoAlphaPKiGenesis[i][j] = alphaIPkj
		}

	}

	profMeg, proof, err := uleaderselection.GenerateSlotLeaderProof(prvKey,
		s.smaGenesis[:],
		s.epochLeadersPtrArrayGenesis[:],
		s.randomGenesis.Bytes(), 0, 0)

	if len(profMeg) != 3 || len(proof) != 2 || err != nil {
		t.Fail()
	}

	if !s.verifySlotProofByGenesis(0, 0, proof, profMeg) {
		t.Fail()
	}
}
