package bn256

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/rlp"
	"math/big"
	"testing"
)

func TestG1Marshal(t *testing.T) {
	_, Ga, err := RandomG1(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ma := Ga.Marshal()

	Gb := new(G1)
	_, err = Gb.Unmarshal(ma)
	if err != nil {
		t.Fatal(err)
	}
	mb := Gb.Marshal()

	if !bytes.Equal(ma, mb) {
		t.Fatal("bytes are different")
	}
}


func UIntToBigEndBytes(num uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, num)
	return b
}
func gfpTOBytes(g gfP) []byte {
	b := make([]byte, 32)
	for i:=0; i<4; i++ {
		copy(b[24 - i*8:], UIntToBigEndBytes(g[i]))
	}
	return b
}
func bytesToGfp(b []byte)  *gfP {
	var g gfP
	for i:=0; i<4; i++ {
		g[i] = binary.BigEndian.Uint64(b[24 - i*8:])
	}
	return &g
}

func TestBigIntConvertBase(t *testing.T) {
	var num10 = bigFromBase10("21888242871839275222246405745257275088696311157297823662689037894645226208583")
	var p2 = [4]uint64{0x3c208c16d87cfd47, 0x97816a916871ca8d, 0xb85045b68181585d, 0x30644e72e131a029}
	var p3 = gfP{0x3c208c16d87cfd47, 0x97816a916871ca8d, 0xb85045b68181585d, 0x30644e72e131a029}
	//var p3 = [4]uint64{0x30644e72e131a029, 0xb85045b68181585d, 0x97816a916871ca8d, 0x3c208c16d87cfd47}
	b3 := make([]byte, 32)
	// marshal is real x!!
	p3.Marshal(b3)
	b := gfpTOBytes(p2)
	println("b:", common.Bytes2Hex(b))
	println("b3:", common.Bytes2Hex(b3))
	//c := gfpTOBytes(p3)
	var num16 = new(big.Int).SetBytes(b)
	//var num162 = new(big.Int).SetBytes(c)
	println(num16.String())
	//println(num162.String())
	println(num10.String())
	//var base = big.NewInt(16)
	//for i:=0; i<64; i++ {
	//	var m = new(big.Int)
	//	num10.DivMod(num10, base, m)
	//	println(m.String())
	//	println(num10.String())
	//}
}

func TestG1EncodeRLP(t *testing.T) {
	for i:=0; i<1000; i++ {
		_, Ga, _ := RandomG1(rand.Reader)
		tmp := Ga.Marshal()
		//println("tmp:", common.Bytes2Hex(tmp))
		c := compress(tmp)
		//println("c  :", common.Bytes2Hex(c))
		out,_ := rlp.EncodeToBytes(c)

		////////////////////////
		var cc = make([]byte, 33)
		rlp.DecodeBytes(out, &cc)
		//println("cc :", common.Bytes2Hex(cc))

		b, _ := decompress(cc)
		//println("bb :", common.Bytes2Hex(b))

		e := new(G1)
		e.Unmarshal(b)

		//Gb := new(G1)
		//Gb.Unmarshal(tmp)

		if Ga.String() != e.String() {
			println("g1 encode failed")
		}
		//if Ga.String() != Gb.String() {
		//	t.Fatal("ga encode failed")
		//}
	}
}
func TestG2EncodeRLP(t *testing.T) {
	//_, Ga, _ := RandomG2(rand.Reader)
	//b := Ga.Marshal()
	//x := compress(b[0:64])
	//y := compress(b[64:128])
	//xy := append(x, y...)
	//out,_ := rlp.EncodeToBytes(xy)
	//
	//var cc = make([]byte, 66)
	//rlp.DecodeBytes(out, &cc)
	//
	//xx, _ := decompress(cc[0:33])
	//yy, _ := decompress(cc[33:66])
	//
	//bb := append(xx, yy...)
	//
	//e := new(G2)
	//e.Unmarshal(bb)
	//
	//if Ga.String() != e.String() {
	//	t.Fatal("g2 encode failed")
	//}
}

// big---->bit  should eq with bit ----> big
func TestBigInt2GfP(t *testing.T) {
	_, Ga, _ := RandomG1(rand.Reader)
	b := GfpToBytes(&Ga.p.x)
	var xi = new(big.Int).SetBytes(b)
	var x = BytesToGfp(xi.Bytes())

	if Ga.p.x != *x {
		t.Fatal("g2 failed failed")
	}
}

func TestG2Marshal(t *testing.T) {
	_, Ga, err := RandomG2(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ma := Ga.Marshal()

	Gb := new(G2)
	_, err = Gb.Unmarshal(ma)
	if err != nil {
		t.Fatal(err)
	}
	mb := Gb.Marshal()

	if !bytes.Equal(ma, mb) {
		t.Fatal("bytes are different")
	}
}

func TestBilinearity(t *testing.T) {
	for i := 0; i < 2; i++ {
		a, p1, _ := RandomG1(rand.Reader)
		b, p2, _ := RandomG2(rand.Reader)
		e1 := Pair(p1, p2)

		e2 := Pair(&G1{curveGen}, &G2{twistGen})
		e2.ScalarMult(e2, a)
		e2.ScalarMult(e2, b)

		if *e1.p != *e2.p {
			t.Fatalf("bad pairing result: %s", e1)
		}
	}
}

func TestTripartiteDiffieHellman(t *testing.T) {
	a, _ := rand.Int(rand.Reader, Order)
	b, _ := rand.Int(rand.Reader, Order)
	c, _ := rand.Int(rand.Reader, Order)

	pa, pb, pc := new(G1), new(G1), new(G1)
	qa, qb, qc := new(G2), new(G2), new(G2)

	pa.Unmarshal(new(G1).ScalarBaseMult(a).Marshal())
	qa.Unmarshal(new(G2).ScalarBaseMult(a).Marshal())
	pb.Unmarshal(new(G1).ScalarBaseMult(b).Marshal())
	qb.Unmarshal(new(G2).ScalarBaseMult(b).Marshal())
	pc.Unmarshal(new(G1).ScalarBaseMult(c).Marshal())
	qc.Unmarshal(new(G2).ScalarBaseMult(c).Marshal())

	k1 := Pair(pb, qc)
	k1.ScalarMult(k1, a)
	k1Bytes := k1.Marshal()

	k2 := Pair(pc, qa)
	k2.ScalarMult(k2, b)
	k2Bytes := k2.Marshal()

	k3 := Pair(pa, qb)
	k3.ScalarMult(k3, c)
	k3Bytes := k3.Marshal()

	if !bytes.Equal(k1Bytes, k2Bytes) || !bytes.Equal(k2Bytes, k3Bytes) {
		t.Errorf("keys didn't agree")
	}
}

func BenchmarkG1(b *testing.B) {
	x, _ := rand.Int(rand.Reader, Order)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		new(G1).ScalarBaseMult(x)
	}
}

func BenchmarkG2(b *testing.B) {
	x, _ := rand.Int(rand.Reader, Order)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		new(G2).ScalarBaseMult(x)
	}
}
func BenchmarkPairing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Pair(&G1{curveGen}, &G2{twistGen})
	}
}
