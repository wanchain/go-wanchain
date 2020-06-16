package vm

import (
	"fmt"
	"github.com/wanchain/go-wanchain/common"
	"testing"
)

/*
 *test case,normal operation, should work well
 */
func TestAdd_1(t *testing.T) {

	x1 := "0x69088a1c79a78b5e66859a5e6594d70c8f12a1ff882d84a05ffdbbcff5a4abcb"
	y1 := "0x5d4c67c05b0a693fb72b47abf7e0d6381fc722ca45c8bb076e6cb4f9f0912906"

	x2 := "0xfb4a50e7008341df6390ad3dcd758b1498959bf18369edc335435367088910c6"
	y2 := "0xe55f58908701c932768c2fd16932f694acd30e21a5f2a4f6242b5f0567696240"

	exp := "3e758e5b2af18254a885210d63c573facc2bd85edb27fdb98e3d0b0ab2dfcd1b7e14602a338ed7011b8f22b4752234619011482fe8b6dcee0e2eeb96c721318c";

	input := make([]byte,0)
	input = append(input,common.FromHex(x1)...)
	input = append(input,common.FromHex(y1)...)
	input = append(input,common.FromHex(x2)...)
	input = append(input,common.FromHex(y2)...)

	seh := &SolEnhance{}

	res,err :=seh.add(input,nil,nil)

	if err != nil {
		t.Fatalf("error happens")
	}


	if exp != common.Bytes2Hex(res) {
		t.Fatalf("the result is not match")
	}
}

/*
 *test case,data length is not enough,failed
 */
func TestAadd_2(t *testing.T) {

	x1 := "0x69088a1c79a78b5e66859a5e6594d70c8f12a1ff882d84a05ffdbbcff5a4a"
	y1 := "0x5d4c67c05b0a693fb72b47abf7e0d6381fc722ca45c8bb076e6cb4f9f0912"

	x2 := "0xfb4a50e7008341df6390ad3dcd758b1498959bf18369edc335435367088910c6"
	y2 := "0xe55f58908701c932768c2fd16932f694acd30e21a5f2a4f6242b5f0567696240"

	input := make([]byte,0)
	input = append(input,common.FromHex(x1)...)
	input = append(input,common.FromHex(y1)...)
	input = append(input,common.FromHex(x2)...)
	input = append(input,common.FromHex(y2)...)

	seh := &SolEnhance{}

	_,err :=seh.add(input,nil,nil)

	if err == nil {
		t.Fatalf("error happens")
	}

}

/*
 *test case,point is not on curve,failed
 */
func TestAadd_3(t *testing.T) {

	x1 := "0x69088a1c79a78b5e66859a5e6594d70c8f12a1ff882d84a05ffdbbcff5a4a11"
	y1 := "0x5d4c67c05b0a693fb72b47abf7e0d6381fc722ca45c8bb076e6cb4f9f091211"

	x2 := "0xfb4a50e7008341df6390ad3dcd758b1498959bf18369edc335435367088910c6"
	y2 := "0xe55f58908701c932768c2fd16932f694acd30e21a5f2a4f6242b5f0567696240"

	input := make([]byte,0)
	input = append(input,common.FromHex(x1)...)
	input = append(input,common.FromHex(y1)...)
	input = append(input,common.FromHex(x2)...)
	input = append(input,common.FromHex(y2)...)

	seh := &SolEnhance{}

	_,err :=seh.add(input,nil,nil)

	if err == nil {
		t.Fatalf("error happens")
	}

}


/*
 *test case,normal operation, should work well
 */
func TestMulPk_1(t *testing.T)  {
	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81ae"
	xPk := "0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	yPk := "0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"
	exp := "979425111f1b36b6e0426988d3a0f4724aaa57db4ef14720667cd42b9f5f456a854f45e729f2c1ed6f2051b295b80c5729857385794a469f3436385c67d7a021"
	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)
	input = append(input,common.FromHex(xPk)...)
	input = append(input,common.FromHex(yPk)...)


	seh := &SolEnhance{}
	res,err := seh.mulPk(input,nil,nil)

	if err != nil {
		t.Fatalf("test failed,error happens")
	}

	fmt.Println(common.Bytes2Hex(res))
	if common.Bytes2Hex(res) != exp {
		t.Fatalf("test failed, the result do not mathc with expect")
	}

}

/*
 *test case,data length is not enough
 */
func TestMulPk_2(t *testing.T)  {
	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa"
	xPk := "0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	yPk := "0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"

	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)
	input = append(input,common.FromHex(xPk)...)
	input = append(input,common.FromHex(yPk)...)


	seh := &SolEnhance{}
	_,err := seh.mulPk(input,nil,nil)

	if err == nil {
		t.Fatalf("test failed,no error happens")
	}

}

/*
 *test case,data length is not enough
 */
func TestMulPk_3(t *testing.T)  {
	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81ae"
	xPk := "0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81700"
	yPk := "0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d499"

	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)
	input = append(input,common.FromHex(xPk)...)
	input = append(input,common.FromHex(yPk)...)


	seh := &SolEnhance{}
	_,err := seh.mulPk(input,nil,nil)

	if err == nil {
		t.Fatalf("test failed,no error happens")
	}

}