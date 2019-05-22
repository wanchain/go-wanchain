package rbselection

import (
	"github.com/wanchain/go-wanchain/crypto/bn256"
	"github.com/wanchain/go-wanchain/rlp"
	"io"
	"math/big"
)

var bigZero = big.NewInt(0)

var bigOne = big.NewInt(1)

// Generator of G1
var gbase = new(bn256.G1).ScalarBaseMult(big.NewInt(int64(1)))

// Generator of G2
var hbase = new(bn256.G2).ScalarBaseMult(big.NewInt(int64(1)))

const numBytes = 256 / 8

var BigZero = bigZero
var BigOne = bigOne
var Gbase = gbase
var Hbase = hbase

// Structure defination for polynomial
type Polynomial []big.Int

// Structure defination for DLEQ proof: a zero knowledge proof for index
type DLEQproof struct {
	a1 *bn256.G1
	a2 *bn256.G2
	z  *big.Int
}

type DLEQproofFlat struct {
	A1 []byte
	A2 []byte
	Z  *big.Int
}

func ProofToProofFlat(d *DLEQproof) DLEQproofFlat {
	var d1 DLEQproofFlat
	d1.A1 = d.a1.Marshal()
	d1.A2 = d.a2.Marshal()
	d1.Z = d.z
	return d1
}
func (p1 *DLEQproof) ProofFlatToProof(d *DLEQproofFlat) {
	//var d1 DLEQproof
	var g1 bn256.G1
	var g2 bn256.G2
	_, _ = g1.Unmarshal(d.A1)
	_, _ = g2.Unmarshal(d.A2)
	p1.a1 = &g1
	p1.a2 = &g2
	p1.z = d.Z
}

// DecodeRLP implements rlp.Encoder
func (proof *DLEQproof) EncodeRLP(w io.Writer) error {
	err := rlp.Encode(w, proof.a1)
	if err != nil {
		return err
	}
	err = rlp.Encode(w, proof.a2)
	if err != nil {
		return err
	}
	return rlp.Encode(w, proof.z)
}

// DecodeRLP implements rlp.Decoder
func (proof *DLEQproof) DecodeRLP(s *rlp.Stream) error {
	proof.a1 = new(bn256.G1)
	proof.a2 = new(bn256.G2)
	proof.z = new(big.Int)
	err := s.Decode(proof.a1)
	if err != nil {
		return err
	}
	err = s.Decode(proof.a2)
	if err != nil {
		return err
	}
	return s.Decode(proof.z)
}
