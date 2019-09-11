package step

import (
	"github.com/wanchain/go-wanchain/crypto"
	mpccrypto "github.com/wanchain/go-wanchain/schnorr/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/schnorr/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
	//"github.com/wanchain/go-wanchain/log"
	//"github.com/wanchain/go-wanchain/common"
)

type RandomPolynomialValue struct {
	randCoefficient []big.Int          //coefficient
	message         map[uint64]big.Int //Polynomil result
	polyValue       []big.Int
	result          *big.Int
	bJRSS           bool
}

func createJRSSValue(degree int, peerNum int) *RandomPolynomialValue {
	return &RandomPolynomialValue{make([]big.Int, degree+1), make(map[uint64]big.Int), make([]big.Int, peerNum), nil, true}
}

func createJZSSValue(degree int, peerNum int) *RandomPolynomialValue {
	return &RandomPolynomialValue{make([]big.Int, degree+1), make(map[uint64]big.Int), make([]big.Int, peerNum), nil, false}
}

func (poly *RandomPolynomialValue) initialize(peers *[]mpcprotocol.PeerInfo, result mpcprotocol.MpcResultInterface) error {
	cof, err := mpccrypto.GetRandCoefficients(len(poly.randCoefficient))
	if err != nil {
		log.SyslogErr("RandomPolynomialValue, GetRandCoefficients fail. err:%s", err.Error())
		return err
	}

	copy(poly.randCoefficient, cof)
	if !poly.bJRSS {
		poly.randCoefficient[0] = *big.NewInt(0)
	}

	for i := 0; i < len(poly.polyValue); i++ {
		poly.polyValue[i] = mpccrypto.EvaluatePoly(poly.randCoefficient, new(big.Int).SetUint64((*peers)[i].Seed))
	}

	return nil
}

func (poly *RandomPolynomialValue) calculateResult() error {
	poly.result = big.NewInt(0)
	for _, value := range poly.message {
		poly.result.Add(poly.result, &value)
		poly.result.Mod(poly.result, crypto.Secp256k1_N)
	}

	return nil
}
