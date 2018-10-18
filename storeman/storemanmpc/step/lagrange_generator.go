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
	return &lagrangeGenerator{message: make(map[uint64]big.Int), preValueKey: preValueKey}
}

func (lag *lagrangeGenerator) initialize(peers *[]mpcprotocol.PeerInfo, result mpcprotocol.MpcResultInterface) error {
	log.Info("lagrangeGenerator.initialize begin")
	mpcsyslog.Info("lagrangeGenerator.initialize begin")

	value, err := result.GetValue(lag.preValueKey)
	if err != nil {
		return err
	}

	lag.seed = value[0]

	log.Info("lagrangeGenerator.initialize succeed")
	mpcsyslog.Info("lagrangeGenerator.initialize succeed")
	return nil
}

func (lag *lagrangeGenerator) calculateResult() error {
	log.Info("lagrangeGenerator.calculateResult begin")
	mpcsyslog.Info("lagrangeGenerator.calculateResult begin")

	f := []big.Int{}
	seed := []big.Int{}
	for key, value := range lag.message {
		f = append(f, value)
		seed = append(seed, *new(big.Int).SetUint64(key))
	}

	lag.result = mpccrypto.Lagrange(f, seed)

	log.Info("lagrangeGenerator.calculateResult succeed")
	mpcsyslog.Info("lagrangeGenerator.calculateResult succeed")
	return nil
}
