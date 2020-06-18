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

/*
 *test case,normal operation, should work well
 */
func TestMulG_1(t *testing.T)  {

	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81ae"
	exp := "979425111f1b36b6e0426988d3a0f4724aaa57db4ef14720667cd42b9f5f456a854f45e729f2c1ed6f2051b295b80c5729857385794a469f3436385c67d7a021"

	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)


	seh := &SolEnhance{}
	res,err := seh.mulG(input,nil,nil)

	fmt.Println(common.Bytes2Hex(res))
	if err != nil {
		t.Fatalf("test failed,no error happens")
	}

	if exp != common.Bytes2Hex(res) {
		t.Fatalf("test failed,result is not match with expected value")
	}

}

/*
 *test case,short scalar data, should failed
 */
func TestMulG_2(t *testing.T)  {

	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81"

	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)

	seh := &SolEnhance{}
	_,err := seh.mulG(input,nil,nil)

	if err != nil {
		t.Fatalf("test failed,no error happens")
	}
}


/*
 *test case,input wrong scalar data,should failed
 */
func TestMulG_3(t *testing.T)  {

	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa8100"

	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)

	seh := &SolEnhance{}
	_,err := seh.mulG(input,nil,nil)

	if err != nil {
		t.Fatalf("test failed,no error happens")
	}
}


/*
 *test case,normal operation, should work well
 */
func TestCalPolyCommit_1(t *testing.T)  {
	pk := "042bda949acb1f1d5e6a2952c928a0524ee088e79bb71be990274ad0d3884230544b0f95d167eef4f76962a5cf569dabc018d025d7494986f7f0b11af7f0bdcbf4";
	poly := "0477947c2048cefbeb637ca46d98a1992c8f0a832e288be5adb36bce9ffb7965deef0024de93f1c30255a6b7deec2ba09d14f0c2f457416098b8266bb16a67e52004e84e2ab12f974cea11c948d276ce38b75638907f3259e8c60db07cf80b492d7da5a4c6e915ab16ba695a9825e6e4441cc843016100534fbce9a7d947d290afc904d665dd602ca1bc43245843dd4721dc7e4509b89c0b94e4744366c4ec491e9aad6efde662ab34bc836724db7f8613ff9131986fc21338e0f2352134b7f915f3d80425e027d24a8c65c0264ae8afbc4218cdd72266f8f245017b8725ef730ad4e80884dd77fbac60297ff6cf5cf6cb130b03b4551605cb5fc85f23ad98a9c6ea24d204367763779f7857ff97a304042885516f70e215ba57852d2763692ea8c6be93a7af3551a2014f7d2a1174335ce69808c57b8dc3c8b2f4ae948696052d8b81034304f6c5c039d2dc4d70aad4baefec8e31a5cc9ebd628cda32da8ed770189cf0dee3d5d5688618ff76e46bd3d40b1aa68b122c5c73af09060c065900790c68ee535304eff4a83c31442c94afd04414d7d4a41ecc20dfd6c587b94fd6a0398555c5dacf350411dab79965e9ef184b443b711b666aa290cfb0e2c263a317be9d0d3ec79a049eb4a277716d47fb868daab644eb66f0fff79a931b483af19a11fb2d097d59c09e73d02d7de04f099f463f10a368334e5b94a618eb6dfd80cfa29f6d9c5832e4047f33a451cb89f81d03823b73bbcc3e3efcaddc015c5e2907d2d4a9535eb6ecf23790c8451554319cec0848b1043281fde3d656e4d89f4041718221ad91cbd71a04e6b755737ccb1afcf5a839869a6d6dab529d263796a06e839190b25a45b31c8696659dade33df0be779a2d3aa987810bcf85d45a7e4d905c3ecf0b977a5dfc9f044c9c5be87bd1f4b334b4a34eac2fac1fb45a248eb071a077fb65e725670fa2367a9ffdb79233769859d44511f01f17a8eb3ae5092c739f2f37d07d656c440cd4043c188a61cdf98bc160935134a039acf3bf1a76d5389841fe93e93317fae34bc15d26c76d926650944c1d8c696212d48691540b04a362ff9e710f8fba967fb58004e919ca4d9a9f59b925579c17fd27fddbf144259a64562051cd93f1672729c3cb24ef17632d7538aa0f49c44b591f26685d3e0edba529e8f868f091839802c037043680e14d808cb3d9f34243204b16f6cdaf172253100526b3a774bc5cb1cbd70d2f9f5f52793b5aeb8b2e22861be26f71ee762aed65b983910fcfe6cab00d4f1704e03eee5f2f37368d687350ee6088d5255263c145ac7c65d630a2d3d7f81452a7d474e5f92e76f0fafddec74e4b0cc65499a34965e6485e3474166a21d6262cbc0444ca736fcd0476b316701d4c636f4abe69bca60e9f66f80293d821fdf3549d604c45dabc802c75c68ff9de8dff63e946d62a44c99c108558addd4568f63cdc66047021ed3d4f2d75ec7dbbdb4fffd429f9784cd4781481b6bb03f80673190751f0cb5f4d690ded3c1cecd9181fab90ed34bec67c1af519caa36e8c24bdd6430901";
	exp := "e6a38df3bb7511cbf03dfd738f0d2086f5e4088d17ec3bbce352880341ddf663f265073ba814058800cb26038c622d54573849a65830711908ba7776660a6348"
	input := make([]byte,0)

	for i:=0;i<len(poly);i+=130 {
		subStr := poly[i+2:i+130]
		input = append(input, common.FromHex(subStr)...)
	}

	input = append(input,common.FromHex(pk[2:])...)

	seh := &SolEnhance{}
	res,err := seh.calPolyCommit(input,nil,nil)
	if err != nil {
		t.Fatalf("errors happend during caculating")
	}

	fmt.Println(common.Bytes2Hex(res))

	if common.Bytes2Hex(res) != exp {
		t.Fatalf("the result do not match with expected value")
	}

}

/*
 *test case,input wrong data, should fail
 */
func TestCalPolyCommit_2(t *testing.T)  {
	pk := "042bda949acb1f1d5e6a2952c928a0524ee088e79bb71be990274ad0d3884230544b0f95d167eef4f76962a5cf569dabc018d025d7494986f7f0b11af7f0bdcb";
	poly := "0477947c2048cefbeb637ca46d98a1992c8f0a832e288be5adb36bce9ffb7965deef0024de93f1c30255a6b7deec2ba09d14f0c2f457416098b8266bb16a67e52004e84e2ab12f974cea11c948d276ce38b75638907f3259e8c60db07cf80b492d7da5a4c6e915ab16ba695a9825e6e4441cc843016100534fbce9a7d947d290afc904d665dd602ca1bc43245843dd4721dc7e4509b89c0b94e4744366c4ec491e9aad6efde662ab34bc836724db7f8613ff9131986fc21338e0f2352134b7f915f3d80425e027d24a8c65c0264ae8afbc4218cdd72266f8f245017b8725ef730ad4e80884dd77fbac60297ff6cf5cf6cb130b03b4551605cb5fc85f23ad98a9c6ea24d204367763779f7857ff97a304042885516f70e215ba57852d2763692ea8c6be93a7af3551a2014f7d2a1174335ce69808c57b8dc3c8b2f4ae948696052d8b81034304f6c5c039d2dc4d70aad4baefec8e31a5cc9ebd628cda32da8ed770189cf0dee3d5d5688618ff76e46bd3d40b1aa68b122c5c73af09060c065900790c68ee535304eff4a83c31442c94afd04414d7d4a41ecc20dfd6c587b94fd6a0398555c5dacf350411dab79965e9ef184b443b711b666aa290cfb0e2c263a317be9d0d3ec79a049eb4a277716d47fb868daab644eb66f0fff79a931b483af19a11fb2d097d59c09e73d02d7de04f099f463f10a368334e5b94a618eb6dfd80cfa29f6d9c5832e4047f33a451cb89f81d03823b73bbcc3e3efcaddc015c5e2907d2d4a9535eb6ecf23790c8451554319cec0848b1043281fde3d656e4d89f4041718221ad91cbd71a04e6b755737ccb1afcf5a839869a6d6dab529d263796a06e839190b25a45b31c8696659dade33df0be779a2d3aa987810bcf85d45a7e4d905c3ecf0b977a5dfc9f044c9c5be87bd1f4b334b4a34eac2fac1fb45a248eb071a077fb65e725670fa2367a9ffdb79233769859d44511f01f17a8eb3ae5092c739f2f37d07d656c440cd4043c188a61cdf98bc160935134a039acf3bf1a76d5389841fe93e93317fae34bc15d26c76d926650944c1d8c696212d48691540b04a362ff9e710f8fba967fb58004e919ca4d9a9f59b925579c17fd27fddbf144259a64562051cd93f1672729c3cb24ef17632d7538aa0f49c44b591f26685d3e0edba529e8f868f091839802c037043680e14d808cb3d9f34243204b16f6cdaf172253100526b3a774bc5cb1cbd70d2f9f5f52793b5aeb8b2e22861be26f71ee762aed65b983910fcfe6cab00d4f1704e03eee5f2f37368d687350ee6088d5255263c145ac7c65d630a2d3d7f81452a7d474e5f92e76f0fafddec74e4b0cc65499a34965e6485e3474166a21d6262cbc0444ca736fcd0476b316701d4c636f4abe69bca60e9f66f80293d821fdf3549d604c45dabc802c75c68ff9de8dff63e946d62a44c99c108558addd4568f63cdc66047021ed3d4f2d75ec7dbbdb4fffd429f9784cd4781481b6bb03f80673190751f0cb5f4d690ded3c1cecd9181fab90ed34bec67c1af519caa36e8c24bdd6430901";
	input := make([]byte,0)

	for i:=0;i<len(poly);i+=130 {
		subStr := poly[i+2:i+130]
		input = append(input, common.FromHex(subStr)...)
	}

	input = append(input,common.FromHex(pk[2:])...)

	seh := &SolEnhance{}
	_,err := seh.calPolyCommit(input,nil,nil)
	if err == nil {
		t.Fatalf("errors happend during caculating")
	}

}

/*
 *test case,input wrong data, should fail
 */
func TestCalPolyCommit_3(t *testing.T)  {
	pk := "042bda949acb1f1d5e6a2952c928a0524ee088e79bb71be990274ad0d3884230544b0f95d167eef4f76962a5cf569dabc018d025d7494986f7f0b11af7f0bdcbf4";
	poly := "047788882048cefbeb637ca46d98a1992c8f0a832e288be5adb36bce9ffb7965deef0024de93f1c30255a6b7deec2ba09d14f0c2f457416098b8266bb16a67e52004e84e2ab12f974cea11c948d276ce38b75638907f3259e8c60db07cf80b492d7da5a4c6e915ab16ba695a9825e6e4441cc843016100534fbce9a7d947d290afc904d665dd602ca1bc43245843dd4721dc7e4509b89c0b94e4744366c4ec491e9aad6efde662ab34bc836724db7f8613ff9131986fc21338e0f2352134b7f915f3d80425e027d24a8c65c0264ae8afbc4218cdd72266f8f245017b8725ef730ad4e80884dd77fbac60297ff6cf5cf6cb130b03b4551605cb5fc85f23ad98a9c6ea24d204367763779f7857ff97a304042885516f70e215ba57852d2763692ea8c6be93a7af3551a2014f7d2a1174335ce69808c57b8dc3c8b2f4ae948696052d8b81034304f6c5c039d2dc4d70aad4baefec8e31a5cc9ebd628cda32da8ed770189cf0dee3d5d5688618ff76e46bd3d40b1aa68b122c5c73af09060c065900790c68ee535304eff4a83c31442c94afd04414d7d4a41ecc20dfd6c587b94fd6a0398555c5dacf350411dab79965e9ef184b443b711b666aa290cfb0e2c263a317be9d0d3ec79a049eb4a277716d47fb868daab644eb66f0fff79a931b483af19a11fb2d097d59c09e73d02d7de04f099f463f10a368334e5b94a618eb6dfd80cfa29f6d9c5832e4047f33a451cb89f81d03823b73bbcc3e3efcaddc015c5e2907d2d4a9535eb6ecf23790c8451554319cec0848b1043281fde3d656e4d89f4041718221ad91cbd71a04e6b755737ccb1afcf5a839869a6d6dab529d263796a06e839190b25a45b31c8696659dade33df0be779a2d3aa987810bcf85d45a7e4d905c3ecf0b977a5dfc9f044c9c5be87bd1f4b334b4a34eac2fac1fb45a248eb071a077fb65e725670fa2367a9ffdb79233769859d44511f01f17a8eb3ae5092c739f2f37d07d656c440cd4043c188a61cdf98bc160935134a039acf3bf1a76d5389841fe93e93317fae34bc15d26c76d926650944c1d8c696212d48691540b04a362ff9e710f8fba967fb58004e919ca4d9a9f59b925579c17fd27fddbf144259a64562051cd93f1672729c3cb24ef17632d7538aa0f49c44b591f26685d3e0edba529e8f868f091839802c037043680e14d808cb3d9f34243204b16f6cdaf172253100526b3a774bc5cb1cbd70d2f9f5f52793b5aeb8b2e22861be26f71ee762aed65b983910fcfe6cab00d4f1704e03eee5f2f37368d687350ee6088d5255263c145ac7c65d630a2d3d7f81452a7d474e5f92e76f0fafddec74e4b0cc65499a34965e6485e3474166a21d6262cbc0444ca736fcd0476b316701d4c636f4abe69bca60e9f66f80293d821fdf3549d604c45dabc802c75c68ff9de8dff63e946d62a44c99c108558addd4568f63cdc66047021ed3d4f2d75ec7dbbdb4fffd429f9784cd4781481b6bb03f80673190751f0cb5f4d690ded3c1cecd9181fab90ed34bec67c1af519caa36e8c24bdd6430901";
	input := make([]byte,0)

	for i:=0;i<len(poly);i+=130 {
		subStr := poly[i+2:i+130]
		input = append(input, common.FromHex(subStr)...)
	}

	input = append(input,common.FromHex(pk[2:])...)

	seh := &SolEnhance{}
	_,err := seh.calPolyCommit(input,nil,nil)
	if err == nil {
		t.Fatalf("errors happend during caculating")
	}
}


/*
 *test case,normal operation, should work well
 */
func TestCheckSig_1(t *testing.T)  {

	r := "0xba1d75823c0f4c07be3e07723e54c3d503829d3c9d0599a78426ac4995096a17"
	s := "0x9a3b16eac39592d14e53b030e0275d087b9e6b38dc9d47a7383df40b4c7aec90"
	hash := "0xb536ad7724251502d75380d774ecb5c015fd8a191dd6ceb05abf677e281b81e1"
	pk :="0xd9482a01dd8bb0fb997561e734823d6cf341557ab117b7f0de72530c5e2f0913ef74ac187589ed90a2b9b69f736af4b9f87c68ae34c550a60f4499e2559cbfa5"

	expected := "0000000000000000000000000000000000000000000000000000000000000001"

	input := make([]byte,0)
	input = append(input,common.FromHex(hash)...)
	input = append(input,common.FromHex(r)...)
	input = append(input,common.FromHex(s)...)
	input = append(input,common.FromHex(pk)...)

	seh := &SolEnhance{}
	ret,err := seh.checkSig(input,nil,nil)
	if err != nil {
		t.Fatalf("errors happend during caculating")
	}
	fmt.Println(common.Bytes2Hex(ret))
	if expected != common.Bytes2Hex(ret) {
		t.Fatalf("the result is not match with expected value")
	}

}



/*
 *test case,wrong data length,should failed
 */
func TestCheckSig_2(t *testing.T)  {

	r := "0xba1d75823c0f4c07be3e07723e54c3d503829d3c9d0599a78426ac499509"
	s := "0x9a3b16eac39592d14e53b030e0275d087b9e6b38dc9d47a7383df40b4c7aec90"
	hash := "0xb536ad7724251502d75380d774ecb5c015fd8a191dd6ceb05abf677e281b81e1"
	pk :="0xd9482a01dd8bb0fb997561e734823d6cf341557ab117b7f0de72530c5e2f0913ef74ac187589ed90a2b9b69f736af4b9f87c68ae34c550a60f4499e2559cbfa5"


	input := make([]byte,0)
	input = append(input,common.FromHex(hash)...)
	input = append(input,common.FromHex(r)...)
	input = append(input,common.FromHex(s)...)
	input = append(input,common.FromHex(pk)...)

	seh := &SolEnhance{}
	_,err := seh.checkSig(input,nil,nil)
	if err != nil {
		t.Fatalf("errors happend during caculating")
	}

}


/*
 *test case,wrong r,s,should return 0
 */
func TestCheckSig_3(t *testing.T)  {

	r := "0xba1d75823c0f4c07be3e07723e54c3d503829d3c9d0599a78426ac4995096a19"
	s := "0x9a3b16eac39592d14e53b030e0275d087b9e6b38dc9d47a7383df40b4c7aec90"
	hash := "0xb536ad7724251502d75380d774ecb5c015fd8a191dd6ceb05abf677e281b81e1"
	pk :="0xd9482a01dd8bb0fb997561e734823d6cf341557ab117b7f0de72530c5e2f0913ef74ac187589ed90a2b9b69f736af4b9f87c68ae34c550a60f4499e2559cbfa5"

	expected := "0000000000000000000000000000000000000000000000000000000000000000"

	input := make([]byte,0)
	input = append(input,common.FromHex(hash)...)
	input = append(input,common.FromHex(r)...)
	input = append(input,common.FromHex(s)...)
	input = append(input,common.FromHex(pk)...)

	seh := &SolEnhance{}
	ret,err := seh.checkSig(input,nil,nil)
	if err != nil {
		t.Fatalf("errors happend during caculating")
	}
	fmt.Println(common.Bytes2Hex(ret))
	if expected != common.Bytes2Hex(ret) {
		t.Fatalf("the result is not match with expected value")
	}
}


/*
 *test case,normal operation, should work well
 */
func TestS256Add_1(t *testing.T) {

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

	s := &s256Add{}
	res,err :=s.Run(input,nil,nil)

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
func TestS256Add_2(t *testing.T) {

	x1 := "0x69088a1c79a78b5e66859a5e6594d70c8f12a1ff882d84a05ffdbbcff5a4a"
	y1 := "0x5d4c67c05b0a693fb72b47abf7e0d6381fc722ca45c8bb076e6cb4f9f0912"

	x2 := "0xfb4a50e7008341df6390ad3dcd758b1498959bf18369edc335435367088910c6"
	y2 := "0xe55f58908701c932768c2fd16932f694acd30e21a5f2a4f6242b5f0567696240"

	input := make([]byte,0)
	input = append(input,common.FromHex(x1)...)
	input = append(input,common.FromHex(y1)...)
	input = append(input,common.FromHex(x2)...)
	input = append(input,common.FromHex(y2)...)

	s := &s256Add{}
	_,err :=s.Run(input,nil,nil)

	if err == nil {
		t.Fatalf("error happens")
	}

}

/*
 *test case,point is not on curve,failed
 */
func TestS256Add_3(t *testing.T) {

	x1 := "0x69088a1c79a78b5e66859a5e6594d70c8f12a1ff882d84a05ffdbbcff5a4a11"
	y1 := "0x5d4c67c05b0a693fb72b47abf7e0d6381fc722ca45c8bb076e6cb4f9f091211"

	x2 := "0xfb4a50e7008341df6390ad3dcd758b1498959bf18369edc335435367088910c6"
	y2 := "0xe55f58908701c932768c2fd16932f694acd30e21a5f2a4f6242b5f0567696240"

	input := make([]byte,0)
	input = append(input,common.FromHex(x1)...)
	input = append(input,common.FromHex(y1)...)
	input = append(input,common.FromHex(x2)...)
	input = append(input,common.FromHex(y2)...)

	s := &s256Add{}
	_,err :=s.Run(input,nil,nil)

	if err == nil {
		t.Fatalf("error happens")
	}

}


/*
 *test case,normal operation, should work well
 */
func TestS256ScalarMul_1(t *testing.T)  {
	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81ae"
	xPk := "0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	yPk := "0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"
	exp := "979425111f1b36b6e0426988d3a0f4724aaa57db4ef14720667cd42b9f5f456a854f45e729f2c1ed6f2051b295b80c5729857385794a469f3436385c67d7a021"
	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)
	input = append(input,common.FromHex(xPk)...)
	input = append(input,common.FromHex(yPk)...)


	s := &s256ScalarMul{}
	res,err := s.Run(input,nil,nil)

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
func TestS256ScalarMul_2(t *testing.T)  {
	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa"
	xPk := "0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	yPk := "0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8"

	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)
	input = append(input,common.FromHex(xPk)...)
	input = append(input,common.FromHex(yPk)...)

	s := &s256ScalarMul{}
	_,err := s.Run(input,nil,nil)
	if err == nil {
		t.Fatalf("test failed,no error happens")
	}

}

/*
 *test case,data length is not enough
 */
func TestS256ScalarMul_3(t *testing.T)  {
	scalar := "0xeb94edbc8d5113cc505ebb489d47ade76bc3f02fd02445f7f47fea454faa81ae"
	xPk := "0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81700"
	yPk := "0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d491"

	input := make([]byte,0)
	input = append(input,common.FromHex(scalar)...)
	input = append(input,common.FromHex(xPk)...)
	input = append(input,common.FromHex(yPk)...)

	s := &s256ScalarMul{}
	_,err := s.Run(input,nil,nil)

	if err == nil {
		t.Fatalf("test failed,no error happens")
	}

}
