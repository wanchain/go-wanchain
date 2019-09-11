package step

import (
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/crypto"
	mpccrypto "github.com/wanchain/go-wanchain/schnorr/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/schnorr/storemanmpc/protocol"
	"github.com/wanchain/go-wanchain/log"
	"math/big"
)

type mpcPointGenerator struct {
	seed        [2]big.Int
	message     map[uint64][2]big.Int
	result      [2]big.Int
	preValueKey string
}

func createPointGenerator(preValueKey string) *mpcPointGenerator {
	return &mpcPointGenerator{message: make(map[uint64][2]big.Int), preValueKey: preValueKey}
}

func (point *mpcPointGenerator) initialize(peers *[]mpcprotocol.PeerInfo, result mpcprotocol.MpcResultInterface) error {
	log.SyslogInfo("mpcPointGenerator.initialize begin")

	value, err := result.GetValue(point.preValueKey)
	if err != nil {
		log.SyslogErr("mpcPointGenerator.initialize get preValueKey fail")
		return err
	}

	curve := crypto.S256()
	x, y := curve.ScalarBaseMult(value[0].Bytes())
	if x == nil || y == nil {
		log.SyslogErr("mpcPointGenerator.ScalarBaseMult fail. err:%s", mpcprotocol.ErrPointZero.Error())
		return mpcprotocol.ErrPointZero
	}

	point.seed = [2]big.Int{*x, *y}

	log.SyslogInfo("mpcPointGenerator.initialize succeed")
	return nil
}

func (point *mpcPointGenerator) calculateResult() error {
	log.SyslogInfo("mpcPointGenerator.calculateResult begin")

	result := new(ecdsa.PublicKey)
	result.Curve = crypto.S256()
	var i = 0
	for _, value := range point.message {
		if i == 0 {
			result.X = new(big.Int).Set(&value[0])
			result.Y = new(big.Int).Set(&value[1])
			i++
		} else {
			result.X, result.Y = crypto.S256().Add(result.X, result.Y, &value[0], &value[1])
		}
	}

	if !mpccrypto.ValidatePublicKey(result) {
		log.SyslogErr("mpcPointGenerator.ValidatePublicKey fail. err:%s", mpcprotocol.ErrPointZero.Error())
		return mpcprotocol.ErrPointZero
	}

	point.result = [2]big.Int{*result.X, *result.Y}

	log.SyslogInfo("mpcPointGenerator.calculateResult succeed")
	return nil
}
