package step

import (
	"crypto/rand"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/storeman/shcnorrmpc"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	//"github.com/wanchain/go-wanchain/log"
	//"github.com/wanchain/go-wanchain/common"
)

type RandomPolynomialValue struct {
	randCoefficient []big.Int          //coefficient
	message         map[uint64]big.Int //Polynomial result
	polyValue       []big.Int
	result          *big.Int
}

func createSkPolyValue(degree int, peerNum int) *RandomPolynomialValue {
	return &RandomPolynomialValue{make([]big.Int, degree+1), make(map[uint64]big.Int), make([]big.Int, peerNum), nil}
}

func (poly *RandomPolynomialValue) initialize(peers *[]mpcprotocol.PeerInfo,
	result mpcprotocol.MpcResultInterface) error {

	log.Info("==Jacob RandomPolynomialValue::initialize ", "len of recieved message", len(poly.message))

	degree := len(poly.randCoefficient) - 1

	s, err := rand.Int(rand.Reader, crypto.S256().Params().N)
	if err != nil {
		log.SyslogErr("RandomPolynomialValue::initialize, rand.Int fail. err:%s", err.Error())
		return err
	}
	cof := shcnorrmpc.RandPoly(degree, *s)
	copy(poly.randCoefficient, cof)

	for i := 0; i < len(poly.polyValue); i++ {
		poly.polyValue[i] = shcnorrmpc.EvaluatePoly(poly.randCoefficient,
			new(big.Int).SetUint64((*peers)[i].Seed),
			degree)
		log.Info("==Jacob RandomPolynomialValue::initialize poly ",
			"poly peerId", (*peers)[i].PeerID.String(),
			"poly x seed", (*peers)[i].Seed)
	}

	return nil
}

func (poly *RandomPolynomialValue) calculateResult() error {
	poly.result = big.NewInt(0)
	log.Info("==Jacob RandomPolynomialValue::calculateResult ", "len of recieved message", len(poly.message))
	for _, value := range poly.message {
		poly.result.Add(poly.result, &value)
		poly.result.Mod(poly.result, crypto.S256().Params().N)
	}

	return nil
}
