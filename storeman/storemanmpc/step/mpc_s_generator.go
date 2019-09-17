package step

import (
	"bytes"
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/storeman/shcnorrmpc"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
)

type mpcSGenerator struct {
	seed        big.Int
	message     map[uint64]big.Int
	result      big.Int
	preValueKey string
}

func createSGenerator(preValueKey string) *mpcSGenerator {
	return &mpcSGenerator{message: make(map[uint64]big.Int), preValueKey: preValueKey}
}

func (msg *mpcSGenerator) initialize(peers *[]mpcprotocol.PeerInfo, result mpcprotocol.MpcResultInterface) error {
	log.SyslogInfo("mpcSGenerator.initialize begin")

	// rgpk R
	rgpkValue, err := result.GetValue(mpcprotocol.RPublicKeyResult)

	if err != nil {
		log.SyslogErr("mpcSGenerator.initialize get RPublicKeyResult fail")
		return err
	}

	var rgpk ecdsa.PublicKey
	rgpk.Curve = crypto.S256()
	rgpk.X, rgpk.Y = &rgpkValue[0], &rgpkValue[1]

	// M
	MBytes, err := result.GetByteValue(mpcprotocol.MpcM)
	if err != nil {
		log.SyslogErr("mpcSGenerator.initialize get MpcM fail")
		return err
	}

	// compute m
	var buffer bytes.Buffer
	buffer.Write(MBytes[:])
	buffer.Write(crypto.FromECDSAPub(&rgpk))

	M := crypto.Keccak256(buffer.Bytes())
	m := new(big.Int).SetBytes(M)

	rskShare, err := result.GetValue(mpcprotocol.RMpcPrivateShare)
	if err != nil {
		log.SyslogErr("mpcSGenerator.initialize get RMpcPrivateShare fail")
		return err
	}

	gskShare, err := result.GetValue(mpcprotocol.MpcPrivateShare)
	if err != nil {
		log.SyslogErr("mpcSGenerator.initialize get MpcPrivateShare fail")
		return err
	}
	sigShare := shcnorrmpc.SchnorrSign(gskShare[0], rskShare[0], *m)
	msg.seed = sigShare

	log.SyslogInfo("mpcSGenerator.initialize succeed")
	return nil
}

func (msg *mpcSGenerator) calculateResult() error {
	log.SyslogInfo("mpcSGenerator.calculateResult begin")
	// x
	seeds := make([]big.Int, 0)
	sigshares := make([]big.Int, 0)
	for seed, value := range msg.message {
		// get seeds, need sort seeds, and make seeds as a key of map, and check the map's count??
		seeds = append(seeds, *big.NewInt(0).SetUint64(seed))
		// sigshares
		sigshares = append(sigshares, value)
	}

	// Lagrange
	result := shcnorrmpc.Lagrange(sigshares, seeds[:], mpcprotocol.MPCDegree)
	msg.result = result
	log.SyslogInfo("mpcSGenerator.calculateResult succeed")

	return nil
}
