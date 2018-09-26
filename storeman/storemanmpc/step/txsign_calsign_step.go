package step

import (
	"bytes"
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
	"strconv"
)

type TxSign_CalSignStep struct {
	TXSign_Lagrange_Step
	signNum int
}

func CreateTxSign_CalSignStep(peers *[]mpcprotocol.PeerInfo, resultKey string, signNum int) *TxSign_CalSignStep {
	log.Warn("-----------------CreateTxSign_CalSignStep begin")
	signSeedKeys := mpcprotocol.GetPreSetKeyArr(mpcprotocol.MpcTxSignSeed, signNum)
	resultKeys := mpcprotocol.GetPreSetKeyArr(resultKey, signNum)
	mpc := &TxSign_CalSignStep{*CreateTXSign_Lagrange_Step(peers, signSeedKeys, resultKeys), signNum}
	return mpc
}

func (txStep *TxSign_CalSignStep) InitStep(result mpcprotocol.MpcResultInterface) error {
	log.Warn("-----------------TxSign_CalSignStep.InitStep begin")

	privateKey, err := result.GetValue(mpcprotocol.MpcPrivateShare)
	log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcPrivateShare, common.ToHex(privateKey[0].Bytes()))
	if err != nil {
		mpcsyslog.Err("TxSign_CalSignStep.InitStep, GetValue fail. key:%s", mpcprotocol.MpcPrivateShare)
		return err
	}

	for i := 0; i < txStep.signNum; i++ {
		log.Warn("-----------------TxSign_CalSignStep.InitStep", "i", i)

		ar, err := result.GetValue(mpcprotocol.MpcSignARResult + "_" + strconv.Itoa(i))
		log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcSignARResult + "_" + strconv.Itoa(i), ar[0].String())
		if err != nil {
			mpcsyslog.Err("TxSign_CalSignStep.InitStep, GetValue fail. key:%s, i:%d", mpcprotocol.MpcSignARResult, i)
			log.Error("TxSign_CalSignStep InitStep, getValue fail.", "key", mpcprotocol.MpcSignARResult, "i", i)
			return err
		}

		aPoint, err := result.GetValue(mpcprotocol.MpcSignAPoint + "_" + strconv.Itoa(i))
		log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcSignAPoint + "_" + strconv.Itoa(i), aPoint[0].String())
		if err != nil {
			mpcsyslog.Err("TxSign_CalSignStep.InitStep, GetValue fail. key:%s, i:%d", mpcprotocol.MpcSignAPoint, i)
			log.Error("TxSign_CalSignStep InitStep, getValue fail.", "key", mpcprotocol.MpcSignAPoint, "i", i)
			return err
		}

		r, err := result.GetValue(mpcprotocol.MpcSignR + "_" + strconv.Itoa(i))
		log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcSignR + "_" + strconv.Itoa(i), r[0].String())
		if err != nil {
			mpcsyslog.Err("TxSign_CalSignStep.InitStep, GetValue fail. key:%s, i:%d", mpcprotocol.MpcSignR, i)
			log.Error("TxSign_CalSignStep InitStep, getValue fail.", "key", mpcprotocol.MpcSignR, "i", i)
			return err
		}

		c, err := result.GetValue(mpcprotocol.MpcSignC + "_" + strconv.Itoa(i))
		log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcSignC + "_" + strconv.Itoa(i), c[0].String())
		if err != nil {
			mpcsyslog.Err("TxSign_CalSignStep.InitStep, GetValue fail. key:%s, i:%d", mpcprotocol.MpcSignC, i)
			log.Error("TxSign_CalSignStep InitStep, getValue fail.", "key", mpcprotocol.MpcSignC, "i", i)
			return err
		}

		txHash, err := result.GetValue(mpcprotocol.MpcTxHash + "_" + strconv.Itoa(i))
		log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcTxHash + "_" + strconv.Itoa(i), common.ToHex(txHash[0].Bytes()))
		if err != nil {
			mpcsyslog.Err("TxSign_CalSignStep.InitStep, GetValue fail. key:%s, i:%d", mpcprotocol.MpcTxHash, i)
			log.Error("TxSign_CalSignStep InitStep, getValue fail.", "key", mpcprotocol.MpcTxHash, "i", i)
			return err
		}

		arInv := ar[0]
		arInv.ModInverse(&arInv, crypto.Secp256k1_N)
		invRPoint := new(ecdsa.PublicKey)
		invRPoint.Curve = crypto.S256()
		invRPoint.X, invRPoint.Y = crypto.S256().ScalarMult(&aPoint[0], &aPoint[1], arInv.Bytes())
		if invRPoint.X == nil || invRPoint.Y == nil {
			mpcsyslog.Err("TxSign_CalSignStep.InitStep, invalid r point")
			return mpcprotocol.ErrPointZero
		}

		log.Debug("calsign", "x", invRPoint.X.String(), "y", invRPoint.Y.String())
		SignSeed := new(big.Int).Set(invRPoint.X)
		SignSeed.Mod(SignSeed, crypto.Secp256k1_N)
		var v int64
		if invRPoint.X.Cmp(SignSeed) == 0 {
			v = 0
		} else {
			v = 2
		}

		invRPoint.Y.Mod(invRPoint.Y, big.NewInt(2))
		if invRPoint.Y.Cmp(big.NewInt(0)) != 0 {
			v |= 1
		}

		log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcTxSignResultR + "_" + strconv.Itoa(i), SignSeed.String(), mpcprotocol.MpcTxSignResultV + "_" + strconv.Itoa(i), v)

		log.Debug("calsign", "v", v)
		result.SetValue(mpcprotocol.MpcTxSignResultR + "_" + strconv.Itoa(i), []big.Int{*SignSeed})
		result.SetValue(mpcprotocol.MpcTxSignResultV + "_" + strconv.Itoa(i), []big.Int{*big.NewInt(v)})
		SignSeed.Mul(SignSeed, &privateKey[0])
		SignSeed.Mod(SignSeed, crypto.Secp256k1_N)
		hash := txHash[0]
		SignSeed.Add(SignSeed, &hash)
		SignSeed.Mod(SignSeed, crypto.Secp256k1_N)
		SignSeed.Mul(SignSeed, &r[0])
		SignSeed.Mod(SignSeed, crypto.Secp256k1_N)
		SignSeed.Add(SignSeed, &c[0])
		SignSeed.Mod(SignSeed, crypto.Secp256k1_N)

		log.Debug("calsign", "seed", SignSeed.String())
		result.SetValue(mpcprotocol.MpcTxSignSeed + "_" + strconv.Itoa(i), []big.Int{*SignSeed})
		log.Warn("-----------------TxSign_CalSignStep.InitStep", mpcprotocol.MpcTxSignSeed + "_" + strconv.Itoa(i), SignSeed.String())
	}

	return txStep.TXSign_Lagrange_Step.InitStep(result)
}

func (txStep *TxSign_CalSignStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.Warn("-----------------TxSign_CalSignStep.FinishStep begin")

	err := txStep.TXSign_Lagrange_Step.FinishStep(result, mpc)
	if err != nil {
		return err
	}

	err = mpc.SignTransaction(result, txStep.signNum)
	if err != nil {
		return err
	}

	from, err := result.GetValue(mpcprotocol.MpcAddress)
	if err != nil {
		mpcsyslog.Debug("TxSign_CalSignStep.FinishStep, GetValue fail. key:%s", mpcprotocol.MpcAddress)
		return nil
	}

	chainType, err := result.GetByteValue(mpcprotocol.MpcChainType)
	if err != nil {
		log.Error("-----------------TxSign_CalSignStep FinishStep. get byte value fail.", "err", err)
		return err
	}

	if string(chainType) == "BTC" {
		// *********************
	} else {
		address := common.BigToAddress(&from[0])
		signedFrom, err := result.GetByteValue(mpcprotocol.MPCSignedFrom)
		if err != nil {
			mpcsyslog.Debug("TxSign_CalSignStep.FinishStep, GetValue fail. key:%s", mpcprotocol.MPCSignedFrom)
			return nil
		}

		log.Info("calsign finish. check signed from", "require", common.ToHex(address[:]), "actual", common.ToHex(signedFrom))
		mpcsyslog.Info("calsign finish. check signed from. require:%s, actual:%s", common.ToHex(address[:]), common.ToHex(signedFrom))
		if !bytes.Equal(address[:], signedFrom) {
			mpcsyslog.Err("TxSign_CalSignStep.FinishStep, unexpect signed data from address. require:%s, actual:%s", common.ToHex(address[:]), common.ToHex(signedFrom))
			return mpcprotocol.ErrFailSignRetVerify
		}
	}

	log.Warn("-----------------TxSign_CalSignStep.FinishStep succeed")
	return nil
}
