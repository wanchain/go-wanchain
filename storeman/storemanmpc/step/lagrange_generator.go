package step

import (
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
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
	value, err := result.GetValue(lag.preValueKey)
	if err != nil {
		mpcsyslog.Err("lagrangeGenerator.initialize get preValueKey fail. preValueKey:%s", lag.preValueKey)
		return err
	}

	lag.seed = value[0]
	return nil
}

func (lag *lagrangeGenerator) calculateResult() error {
	f := []big.Int{}
	seed := []big.Int{}
	for key, value := range lag.message {
		f = append(f, value)

		seed = append(seed, *new(big.Int).SetUint64(key))
	}

	lag.result = mpccrypto.Lagrange(f, seed)
	return nil
}
