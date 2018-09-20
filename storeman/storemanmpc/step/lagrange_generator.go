package step

import (
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
	"github.com/wanchain/go-wanchain/log"
)

type lagrangeGenerator struct {
	seed        big.Int
	message     map[uint64]big.Int
	result      big.Int
	preValueKey string
}

func createLagrangeGenerator(preValueKey string) *lagrangeGenerator {
	log.Warn("-----------------createLagrangeGenerator begin", "preValueKey", preValueKey)
	return &lagrangeGenerator{message: make(map[uint64]big.Int), preValueKey: preValueKey}
}

func (lag *lagrangeGenerator) initialize(peers *[]mpcprotocol.PeerInfo, result mpcprotocol.MpcResultInterface) error {
	log.Warn("-----------------lagrangeGenerator.initialize begin")
	value, err := result.GetValue(lag.preValueKey)
	if err != nil {
		mpcsyslog.Err("lagrangeGenerator.initialize get preValueKey fail. preValueKey:%s", lag.preValueKey)
		return err
	}

	lag.seed = value[0]
	log.Warn("-----------------lagrangeGenerator.initialize", "seed", lag.seed.String())
	return nil
}

func (lag *lagrangeGenerator) calculateResult() error {
	log.Warn("-----------------lagrangeGenerator.calculateResult begin")

	f := []big.Int{}
	seed := []big.Int{}
	for key, value := range lag.message {
		log.Warn("-----------------lagrangeGenerator.calculateResult", "key", key, "value", value.String())

		f = append(f, value)
		seed = append(seed, *new(big.Int).SetUint64(key))
	}

	lag.result = mpccrypto.Lagrange(f, seed)
	log.Warn("-----------------lagrangeGenerator.calculateResult", "result", lag.result.String())
	return nil
}
