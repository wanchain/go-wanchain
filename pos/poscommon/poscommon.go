package poscommon

import (
	"crypto/ecdsa"
	"math/big"
)

type PkWithStake struct {
	PK    *ecdsa.PublicKey
	Stake *big.Int
}
type PkWithStakeArr []PkWithStake

//Len()
func (s PkWithStakeArr) Len() int {
	return len(s)
}

// Less
func (s PkWithStakeArr) Less(i, j int) bool {
	return s[i].Stake.Cmp(s[j].Stake) < 0
}

//Swap()
func (s PkWithStakeArr) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// type Leader struct {
// 	PubSec256     []byte
// 	PubBn256      []byte
// 	SecAddr       common.Address
// 	FromAddr      common.Address
// 	Probabilities *big.Int
// }
