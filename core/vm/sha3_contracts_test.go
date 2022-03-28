package vm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

// TODO: add unit test later

// import "testing"

// type precompiledTest struct {
// 	input    string
// 	expected string
// 	name     string
// }

// // sha3fips  test vectors
// var sha3fipsTests = []precompiledTest{
// 	{
// 		input:    "0448250ebe88d77e0a12bcf530fe6a2cf1ac176945638d309b840d631940c93b78c2bd6d16f227a8877e3f1604cd75b9c5a8ab0cac95174a8a0a0f8ea9e4c10bca",
// 		expected: "c7647f7e251bf1bd70863c8693e93a4e77dd0c9a689073e987d51254317dc704",
// 		name:     "sha3fip",
// 	},
// 	{
// 		input:    "1234",
// 		expected: "19becdc0e8d6dd4aa2c9c2983dbb9c61956a8ade69b360d3e6019f0bcd5557a9",
// 		name:     "sha3fip",
// 	},
// }

// func TestPrecompiledSHA3fips(t *testing.T) {

// 	for _, test := range sha3fipsTests {
// 		testPrecompiled("66", test, t)
// 	}

// }

// // EcRecoverPublicKey test vectors
// var ecRecoverPublicKeyTests = []precompiledTest{
// 	{
// 		input: "c5d6c454e4d7a8e8a654f5ef96e8efe41d21a65b171b298925414aa3dc061e37" +
// 			"0000000000000000000000000000000000000000000000000000000000000000" +
// 			"4011de30c04302a2352400df3d1459d6d8799580dceb259f45db1d99243a8d0c" +
// 			"64f548b7776cb93e37579b830fc3efce41e12e0958cda9f8c5fcad682c610795",
// 		expected: "0448250ebe88d77e0a12bcf530fe6a2cf1ac176945638d309b840d631940c93b78c2bd6d16f227a8877e3f1604cd75b9c5a8ab0cac95174a8a0a0f8ea9e4c10bca",
// 		name:     "Call ecrecoverPublicKey recoverable Key",
// 	},
// 	{
// 		input: "c5d6c454e4d7a8e8a654f5ef96e8efe41d21a65b171b298925414aa3dc061e37" +
// 			"000000000000000000000000000000000000000000000000000000000000001b" +
// 			"4011de30c04302a2352400df3d1459d6d8799580dceb259f45db1d99243a8d0c" +
// 			"64f548b7776cb93e37579b830fc3efce41e12e0958cda9f8c5fcad682c610795",
// 		expected: "",
// 		name:     "call ecrecoverPublicKey un-recoverable Key- Invalid 'v' parity",
// 	},
// }

// func TestPrecompiledEcrecoverPublicKey(t *testing.T) {
// 	for _, test := range ecRecoverPublicKeyTests {
// 		testPrecompiled("67", test, t)
// 	}

// }

func TestEcrecoverPublicKey(t *testing.T) {
	input := "c5d6c454e4d7a8e8a654f5ef96e8efe41d21a65b171b298925414aa3dc061e37" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"4011de30c04302a2352400df3d1459d6d8799580dceb259f45db1d99243a8d0c" +
		"64f548b7776cb93e37579b830fc3efce41e12e0958cda9f8c5fcad682c610795"

	//fmt.Printf(input)
	sc := &ecrecoverPublicKey{nil, nil}
	ret, err := sc.Run(common.Hex2Bytes(input))
	fmt.Println("ret", common.ToHex(ret))
	fmt.Println("err", err)
	if common.ToHex(ret) != "0x0448250ebe88d77e0a12bcf530fe6a2cf1ac176945638d309b840d631940c93b78c2bd6d16f227a8877e3f1604cd75b9c5a8ab0cac95174a8a0a0f8ea9e4c10bca" {
		t.Failed()
	}
}
