// Package bn256 implements a particular bilinear group at the 128-bit security
// level.
//
// Bilinear groups are the basis of many of the new cryptographic protocols that
// have been proposed over the past decade. They consist of a triplet of groups
// (G₁, G₂ and GT) such that there exists a function e(g₁ˣ,g₂ʸ)=gTˣʸ (where gₓ
// is a generator of the respective group). That function is called a pairing
// function.
//
// This package specifically implements the Optimal Ate pairing over a 256-bit
// Barreto-Naehrig curve as described in
// http://cryptojedi.org/papers/dclxvi-20100714.pdf. Its output is not
// compatible with the implementation described in that paper, as different
// parameters are chosen.
//
// (This package previously claimed to operate at a 128-bit security level.
// However, recent improvements in attacks mean that is no longer true. See
// https://moderncrypto.org/mail-archive/curves/2016/000740.html.)
package bn256

import (
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"github.com/ethereum/go-ethereum/rlp"
)

// add by demmon
func (e *G2) IsInfinity() bool {
	return e.p.IsInfinity()
}


func (e *G1) UnmarshalPure(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) < 2*numBytes {
		return nil, errors.New("bn256: not enough data")
	}
	// Unmarshal the points and check their caps
	if e.p == nil {
		e.p = &curvePoint{}
	} else {
		e.p.x, e.p.y = gfP{0}, gfP{0}
	}
	var err error
	if err = e.p.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = e.p.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}
	// Encode into Montgomery form and ensure it's on the curve
	montEncode(&e.p.x, &e.p.x)
	montEncode(&e.p.y, &e.p.y)

	zero := gfP{0}
	if e.p.x == zero && e.p.y == zero {
		// This is the point at infinity.
		e.p.y = *newGFp(1)
		e.p.z = gfP{0}
		e.p.t = gfP{0}
	} else {
		e.p.z = *newGFp(1)
		e.p.t = *newGFp(1)
	}
	return m[2*numBytes:], nil
}

func (e *G2) UnmarshalPure(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) < 4*numBytes {
		return nil, errors.New("bn256: not enough data")
	}
	// Unmarshal the points and check their caps
	if e.p == nil {
		e.p = &twistPoint{}
	}
	var err error
	if err = e.p.x.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = e.p.x.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}
	if err = e.p.y.x.Unmarshal(m[2*numBytes:]); err != nil {
		return nil, err
	}
	if err = e.p.y.y.Unmarshal(m[3*numBytes:]); err != nil {
		return nil, err
	}
	// Encode into Montgomery form and ensure it's on the curve
	montEncode(&e.p.x.x, &e.p.x.x)
	montEncode(&e.p.x.y, &e.p.x.y)
	montEncode(&e.p.y.x, &e.p.y.x)
	montEncode(&e.p.y.y, &e.p.y.y)

	if e.p.x.IsZero() && e.p.y.IsZero() {
		// This is the point at infinity.
		e.p.y.SetOne()
		e.p.z.SetZero()
		e.p.t.SetZero()
	} else {
		e.p.z.SetOne()
		e.p.t.SetOne()
	}
	return m[4*numBytes:], nil
}


var (
	bn256_B        = big.NewInt(3)
	bn256_q        = new(big.Int).Div(new(big.Int).Add(P, big.NewInt(1)), big.NewInt(4))
	err_ep_nil     = errors.New("ep is zero")
	err_compress   = errors.New("compress failed")
	err_decompress = errors.New("decompress failed")
	err_decode     = errors.New("decode failed")
	err_infinity   = errors.New("infinity point")
)

func GfpToBytes(p *gfP) []byte {
	bs := make([]byte, 32)
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(bs[i*8:], p[i])
	}

	return bs
}
func BytesToGfp(b []byte) *gfP {
	var g gfP
	for i := 0; i < 4; i++ {
		g[i] = binary.LittleEndian.Uint64(b[8*i:])
	}

	return &g
}

func (e *G1) EncodeRLP(w io.Writer) error {
	if e.p == nil {
		return err_ep_nil
	}
	b := e.Marshal()
	return rlp.Encode(w, b)
}

// DecodeRLP implements rlp.Decoder
func (e *G1) DecodeRLP(s *rlp.Stream) error {
	if e.p == nil {
		e.p = new(curvePoint)
	}
	var b = make([]byte, 64)
	err := s.Decode(&b)
	if err != nil {
		return err
	}
	_, err = e.Unmarshal(b)
	return err
}


// add by jacob begin
func (e *G2) EncodeRLP(w io.Writer) error {
	if e.p == nil {
		return err_ep_nil
	}
	if e.p.IsInfinity() {
		return err_infinity
	}

	b := e.Marshal()
	return rlp.Encode(w, b)
}

// DecodeRLP implements rlp.Decoder
func (e *G2) DecodeRLP(s *rlp.Stream) error {
	if e.p == nil {
		e.p = new(twistPoint)
	}
	var b = make([]byte, 128)
	err := s.Decode(&b)
	if err != nil {
		return err
	}
	_, err = e.Unmarshal(b)
	return err
}

// add by jacob end.