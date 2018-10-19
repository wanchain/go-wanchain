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
	log.Info("CreateTxSign_CalSignStep begin")
	mpcsyslog.Info("CreateTxSign_CalSignStep begin")

	signSeedKeys := mpcprotocol.GetPreSetKeyArr(mpcprotocol.MpcTxSignSeed, signNum)
	resultKeys := mpcprotocol.GetPreSetKeyArr(resultKey, signNum)
	mpc := &TxSign_CalSignStep{*CreateTXSign_Lagrange_Step(peers, signSeedKeys, resultKeys), signNum}
	return mpc
}

func (txStep *TxSign_CalSignStep) InitStep(result mpcprotocol.MpcResultInterface) error {
	log.Info("TxSign_CalSignStep.InitStep begin")
	mpcsyslog.Info("TxSign_CalSignStep.InitStep begin")

	privateKey, err := result.GetValue(mpcprotocol.MpcPrivateShare)
	if err != nil {
		return err
	}

	for i := 0; i < txStep.signNum; i++ {
		ar, err := result.GetValue(mpcprotocol.MpcSignARResult + "_" + strconv.Itoa(i))
		if err != nil {
			return err
		}

		aPoint, err := result.GetValue(mpcprotocol.MpcSignAPoint + "_" + strconv.Itoa(i))
		if err != nil {
			return err
		}

		r, err := result.GetValue(mpcprotocol.MpcSignR + "_" + strconv.Itoa(i))
		if err != nil {
			return err
		}

		c, err := result.GetValue(mpcprotocol.MpcSignC + "_" + strconv.Itoa(i))
		if err != nil {
			return err
		}

		txHash, err := result.GetValue(mpcprotocol.MpcTxHash + "_" + strconv.Itoa(i))
		if err != nil {
			return err
		}

		arInv := ar[0]
		arInv.ModInverse(&arInv, crypto.Secp256k1_N)
		invRPoint := new(ecdsa.PublicKey)
		invRPoint.Curve = crypto.S256()
		invRPoint.X, invRPoint.Y = crypto.S256().ScalarMult(&aPoint[0], &aPoint[1], arInv.Bytes())
		if invRPoint.X == nil || invRPoint.Y == nil {
			log.Error("TxSign_CalSignStep.InitStep, invalid r point")
			mpcsyslog.Err("TxSign_CalSignStep.InitStep, invalid r point")
			return mpcprotocol.ErrPointZero
		}

		log.Info("TxSign_CalSignStep.InitStep, calsign", "x", invRPoint.X.String(), "y", invRPoint.Y.String())
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

		log.Info("TxSign_CalSignStep.InitStep", mpcprotocol.MpcTxSignResultR + "_" + strconv.Itoa(i), SignSeed.String(), mpcprotocol.MpcTxSignResultV + "_" + strconv.Itoa(i), v)
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

		result.SetValue(mpcprotocol.MpcTxSignSeed + "_" + strconv.Itoa(i), []big.Int{*SignSeed})
		log.Info("TxSign_CalSignStep.InitStep", mpcprotocol.MpcTxSignSeed + "_" + strconv.Itoa(i), SignSeed.String())
	}

	err = txStep.TXSign_Lagrange_Step.InitStep(result)
	if err != nil {
		log.Error("TxSign_CalSignStep.InitStep, initStep fail", "err", err)
		mpcsyslog.Info("TxSign_CalSignStep.InitStep, initStep fail, err:%s", err.Error())
		return err
	} else {
		log.Info("TxSign_CalSignStep.InitStep succeed")
		mpcsyslog.Info("TxSign_CalSignStep.InitStep succeed")
		return nil
	}
}

func (txStep *TxSign_CalSignStep) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.Info("TxSign_CalSignStep.FinishStep begin")
	mpcsyslog.Info("TxSign_CalSignStep.FinishStep begin")

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
		return nil
	}

	address := common.BigToAddress(&from[0])
	signedFrom, err := result.GetByteValue(mpcprotocol.MPCSignedFrom)
	if err != nil {
		return nil
	}

	log.Info("TxSign_CalSignStep.FinishStep. check signed from", "require", common.ToHex(address[:]), "actual", common.ToHex(signedFrom))
	mpcsyslog.Info("TxSign_CalSignStep.FinishStep. check signed from. require:%s, actual:%s", common.ToHex(address[:]), common.ToHex(signedFrom))
	if !bytes.Equal(address[:], signedFrom) {
		log.Error("TxSign_CalSignStep.FinishStep, unexpect signed data from address. ", "require", common.ToHex(address[:]), "actual", common.ToHex(signedFrom))
		mpcsyslog.Err("TxSign_CalSignStep.FinishStep, unexpect signed data from address. require:%s, actual:%s", common.ToHex(address[:]), common.ToHex(signedFrom))
		return mpcprotocol.ErrFailSignRetVerify
	}

	log.Info("TxSign_CalSignStep.FinishStep succeed")
	mpcsyslog.Info("TxSign_CalSignStep.FinishStep succeed")
	return nil
}
