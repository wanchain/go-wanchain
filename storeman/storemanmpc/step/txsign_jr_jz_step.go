package step

import (
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
	"strconv"
)

type TXSignJR_JZ_Step struct {
	BaseMpcStep
	signNum int
}

func CreateTXSignJR_JZ_Step(degree int, peers *[]mpcprotocol.PeerInfo, signNum int) *TXSignJR_JZ_Step {
	log.Warn("-----------------CreateTXSignJR_JZ_Step begin")
	log.Warn("-----------------CreateTXSignJR_JZ_Step", "degree", degree)
	log.Warn("-----------------CreateTXSignJR_JZ_Step", "signNum", signNum)
	log.Warn("-----------------CreateTXSignJR_JZ_Step", "peers", len(*peers))

	mpc := &TXSignJR_JZ_Step{*CreateBaseMpcStep(peers, 4*signNum), signNum}
	for i := 0; i < signNum; i++ {
		mpc.messages[4*i + 0] = createJRSSValue(degree, len(*peers))
		mpc.messages[4*i + 1] = createJRSSValue(degree, len(*peers))
		mpc.messages[4*i + 2] = createJZSSValue(degree*2, len(*peers))
		mpc.messages[4*i + 3] = createJZSSValue(degree*2, len(*peers))
	}

	return mpc
}

func (jrjz *TXSignJR_JZ_Step) CreateMessage() []mpcprotocol.StepMessage {
	log.Warn("-----------------TXSignJR_JZ_Step.CreateMessage begin")

	message := make([]mpcprotocol.StepMessage, len(*jrjz.peers))

	for i := 0; i < len(*jrjz.peers); i++ {
		message[i].Msgcode = mpcprotocol.MPCMessage
		message[i].PeerID = &(*jrjz.peers)[i].PeerID
	}

	for i := 0; i < jrjz.signNum; i++ {
		a := jrjz.messages[4*i + 0].(*RandomPolynomialValue)
		r := jrjz.messages[4*i + 1].(*RandomPolynomialValue)
		b := jrjz.messages[4*i + 2].(*RandomPolynomialValue)
		c := jrjz.messages[4*i + 3].(*RandomPolynomialValue)

		for j := 0; j < len(*jrjz.peers); j++ {
			message[j].Data = append(message[j].Data, []big.Int{a.polyValue[j], r.polyValue[j], b.polyValue[j], c.polyValue[j]}...)
			log.Warn("-----------------TXSignJR_JZ_Step.CreateMessage", "j", j, "aj", a.polyValue[j].String(), "rj", r.polyValue[j].String(), "bj", b.polyValue[j].String(), "cj", c.polyValue[j].String())
		}
	}

	for j := 0; j < len(*jrjz.peers); j++ {
		log.Warn("-----------------TXSignJR_JZ_Step.CreateMessage", "j", j, "data num", len(message[j].Data))
	}

	return message
}


func (jrjz *TXSignJR_JZ_Step) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.Warn("-----------------TXSignJR_JZ_Step.HandleMessage begin")
	log.Warn("-----------------TXSignJR_JZ_Step.HandleMessage", "PeerID", msg.PeerID, "DataLen:", len(msg.Data))

	log.Debug("TXSignJR_JZ_Step getMessage:", "peerID", msg.PeerID)
	seed := jrjz.getPeerSeed(msg.PeerID)
	if seed == 0 {
		mpcsyslog.Err("TXSignJR_JZ_Step.HandleMessage, get seed fail. peer:%s", msg.PeerID.String())
		log.Error("TXSignJR_JZ_Step Not Find:", "peerID", msg.PeerID)
	}

	if len(msg.Data) != jrjz.signNum*4 {
		log.Error("TXSignJR_JZ_Step HandleMessage, received data len doesn't match requirement", "data len", len(msg.Data))
		return false
	}

	for i := 0; i < jrjz.signNum; i++ {
		a := jrjz.messages[4*i + 0].(*RandomPolynomialValue)
		r := jrjz.messages[4*i + 1].(*RandomPolynomialValue)
		b := jrjz.messages[4*i + 2].(*RandomPolynomialValue)
		c := jrjz.messages[4*i + 3].(*RandomPolynomialValue)
		_, exist := a.message[seed]
		if exist {
			mpcsyslog.Err("TXSignJR_JZ_Step.HandleMessage, get msg fail. peer:%s", msg.PeerID.String())
			return false
		}

		a.message[seed] = msg.Data[4*i + 0] //message.Value
		r.message[seed] = msg.Data[4*i + 1] //message.Value
		b.message[seed] = msg.Data[4*i + 2] //message.Value
		c.message[seed] = msg.Data[4*i + 3] //message.Value

		log.Warn("-----------------TXSignJR_JZ_Step.HandleMessage", "i", i, "a", msg.Data[4*i + 0].String(), "r", msg.Data[4*i + 1].String(), "b", msg.Data[4*i + 2].String(), "c", msg.Data[4*i + 3].String())
	}

	return true
}


func (jrjz *TXSignJR_JZ_Step) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	log.Warn("-----------------TXSignJR_JZ_Step.FinishStep begin")

	err := jrjz.BaseMpcStep.FinishStep()
	if err != nil {
		log.Error("-----------------TXSignJR_JZ_Step.BaseMpcStep.FinishStep fail", "err", err)
		return err
	}

	for i := 0; i < jrjz.signNum; i++ {
		iStr := "_" + strconv.Itoa(i)
		a := jrjz.messages[4*i + 0].(*RandomPolynomialValue)
		r := jrjz.messages[4*i + 1].(*RandomPolynomialValue)
		b := jrjz.messages[4*i + 2].(*RandomPolynomialValue)
		c := jrjz.messages[4*i + 3].(*RandomPolynomialValue)
		err = result.SetValue(mpcprotocol.MpcSignA + iStr, []big.Int{*a.result})
		log.Warn("-----------------TXSignJR_JZ_Step.FinishStep SetValue", "i", i, "key", mpcprotocol.MpcSignA + iStr, "value", a.result.String())
		if err != nil {
			return err
		}

		err = result.SetValue(mpcprotocol.MpcSignA0 + iStr, []big.Int{a.randCoefficient[0]})
		log.Warn("-----------------TXSignJR_JZ_Step.FinishStep SetValue", "i", i, "key", mpcprotocol.MpcSignA0 + iStr, "value", a.randCoefficient[0].String())
		if err != nil {
			return err
		}

		err = result.SetValue(mpcprotocol.MpcSignR + iStr, []big.Int{*r.result})
		log.Warn("-----------------TXSignJR_JZ_Step.FinishStep SetValue", "i", i, "key", mpcprotocol.MpcSignR + iStr, "value", r.result.String())
		if err != nil {
			return err
		}

		err = result.SetValue(mpcprotocol.MpcSignR0 + iStr, []big.Int{r.randCoefficient[0]})
		log.Warn("-----------------TXSignJR_JZ_Step.FinishStep SetValue", "i", i, "key", mpcprotocol.MpcSignR0 + iStr, "value", r.randCoefficient[0].String())
		if err != nil {
			return err
		}

		err = result.SetValue(mpcprotocol.MpcSignB + iStr, []big.Int{*b.result})
		log.Warn("-----------------TXSignJR_JZ_Step.FinishStep SetValue", "i", i, "key", mpcprotocol.MpcSignB + iStr, "value", b.result.String())
		if err != nil {
			return err
		}

		err = result.SetValue(mpcprotocol.MpcSignC + iStr, []big.Int{*c.result})
		log.Warn("-----------------TXSignJR_JZ_Step.FinishStep SetValue", "i", i, "key", mpcprotocol.MpcSignC + iStr, "value", c.result.String())
		if err != nil {
			return err
		}

		ar := make([]big.Int, 1)
		ar[0].Mul(a.result, r.result)
		ar[0].Mod(&ar[0], crypto.Secp256k1_N)
		ar[0].Add(&ar[0], b.result)
		ar[0].Mod(&ar[0], crypto.Secp256k1_N)
		err = result.SetValue(mpcprotocol.MpcSignARSeed + iStr, ar)
		log.Warn("-----------------TXSignJR_JZ_Step.FinishStep SetValue", "i", i, "key", mpcprotocol.MpcSignARSeed + iStr, "value", ar[0].String())
		if err != nil {
			return nil
			log.Error("-----------------TXSignJR_JZ_Step.FinishStep SetValue succeed", "i", i, "err", err)
		}
	}

	log.Warn("-----------------TXSignJR_JZ_Step.FinishStep succeed")
	return nil
}
