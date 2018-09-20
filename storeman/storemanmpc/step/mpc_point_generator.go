package step

import (
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/crypto"
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
	"github.com/wanchain/go-wanchain/log"
)

type mpcPointGenerator struct {
	seed        [2]big.Int
	message     map[uint64][2]big.Int
	result      [2]big.Int
	preValueKey string
}

func createPointGenerator(preValueKey string) *mpcPointGenerator {
	log.Warn("-----------------createPointGenerator begin", "preValueKey", preValueKey)
	return &mpcPointGenerator{message: make(map[uint64][2]big.Int), preValueKey: preValueKey}
}

func (point *mpcPointGenerator) initialize(peers *[]mpcprotocol.PeerInfo, result mpcprotocol.MpcResultInterface) error {
	log.Warn("-----------------mpcPointGenerator.initialize begin")
	value, err := result.GetValue(point.preValueKey)
	log.Warn("-----------------mpcPointGenerator.initialize", "preValueKey", point.preValueKey, "value", value[0].String())
	if err != nil {
		mpcsyslog.Err("mpcPointGenerator.initialize get preValueKey fail")
		return err
	}

	curve := crypto.S256()
	x, y := curve.ScalarBaseMult(value[0].Bytes())
	if x == nil || y == nil {
		mpcsyslog.Err("mpcPointGenerator.ScalarBaseMult fail. err:%s", mpcprotocol.ErrPointZero.Error())
		return mpcprotocol.ErrPointZero
	}

	point.seed = [2]big.Int{*x, *y}
	log.Warn("-----------------mpcPointGenerator.initialize", "seed", point.seed[0].String())
	return nil
}

func (point *mpcPointGenerator) calculateResult() error {
	log.Warn("-----------------mpcPointGenerator.calculateResult begin")
	result := new(ecdsa.PublicKey)
	result.Curve = crypto.S256()
	var i = 0
	for _, value := range point.message {
		log.Warn("-----------------mpcPointGenerator.calculateResult", "i", i, "value0", value[0].String(), "value1", value[1].String())
		if i == 0 {
			result.X = new(big.Int).Set(&value[0])
			result.Y = new(big.Int).Set(&value[1])
			i++
		} else {
			result.X, result.Y = crypto.S256().Add(result.X, result.Y, &value[0], &value[1])
		}
	}

	if !mpccrypto.ValidatePublicKey(result) {
		mpcsyslog.Err("mpcPointGenerator.ValidatePublicKey fail. err:%s", mpcprotocol.ErrPointZero.Error())
		return mpcprotocol.ErrPointZero
	}

	point.result = [2]big.Int{*result.X, *result.Y}
	log.Warn("-----------------mpcPointGenerator.calculateResult", "result0", point.result[0].String(), "result1", point.result[1].String())
	return nil
}
