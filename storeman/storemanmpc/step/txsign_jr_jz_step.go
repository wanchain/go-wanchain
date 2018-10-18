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
	log.Info("CreateTXSignJR_JZ_Step begin", "degree", degree, "signNum", signNum, "peerNum", len(*peers))
	mpcsyslog.Info("CreateTXSignJR_JZ_Step, degree:%d, signNum:%d, peerNum:%d", degree, signNum, len(*peers))

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
	log.Info("TXSignJR_JZ_Step.CreateMessage begin")

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
		}
	}

	return message
}


func (jrjz *TXSignJR_JZ_Step) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.Info("TXSignJR_JZ_Step.HandleMessage", "PeerID", msg.PeerID, "DataLen:", len(msg.Data))
	seed := jrjz.getPeerSeed(msg.PeerID)
	if seed == 0 {
		log.Error("TXSignJR_JZ_Step Not Find:", "peerID", msg.PeerID)
		mpcsyslog.Err("TXSignJR_JZ_Step.HandleMessage, get seed fail. peer:%s", msg.PeerID.String())
	}

	if len(msg.Data) != jrjz.signNum*4 {
		log.Error("TXSignJR_JZ_Step HandleMessage, received data len doesn't match requirement", "dataLen", len(msg.Data))
		mpcsyslog.Err("TXSignJR_JZ_Step HandleMessage, received data len doesn't match requirement, dataLen:%d", len(msg.Data))
		return false
	}

	for i := 0; i < jrjz.signNum; i++ {
		a := jrjz.messages[4*i + 0].(*RandomPolynomialValue)
		r := jrjz.messages[4*i + 1].(*RandomPolynomialValue)
		b := jrjz.messages[4*i + 2].(*RandomPolynomialValue)
		c := jrjz.messages[4*i + 3].(*RandomPolynomialValue)
		_, exist := a.message[seed]
		if exist {
			log.Warn("TXSignJR_JZ_Step.HandleMessage, repeat resp", "peer", msg.PeerID.String())
			mpcsyslog.Warning("TXSignJR_JZ_Step.HandleMessage, repeat resp. peer:%s", msg.PeerID.String())
			return false
		}

		a.message[seed] = msg.Data[4*i + 0] //message.Value
		r.message[seed] = msg.Data[4*i + 1] //message.Value
		b.message[seed] = msg.Data[4*i + 2] //message.Value
		c.message[seed] = msg.Data[4*i + 3] //message.Value
	}

	return true
}


func (jrjz *TXSignJR_JZ_Step) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := jrjz.BaseMpcStep.FinishStep()
	if err != nil {
		log.Error("TXSignJR_JZ_Step.BaseMpcStep.FinishStep fail", "err", err)
		mpcsyslog.Err("TXSignJR_JZ_Step.BaseMpcStep.FinishStep fail, err:%s", err.Error())
		return err
	}

	for i := 0; i < jrjz.signNum; i++ {
		iStr := "_" + strconv.Itoa(i)
		a := jrjz.messages[4*i + 0].(*RandomPolynomialValue)
		r := jrjz.messages[4*i + 1].(*RandomPolynomialValue)
		b := jrjz.messages[4*i + 2].(*RandomPolynomialValue)
		c := jrjz.messages[4*i + 3].(*RandomPolynomialValue)
		err = result.SetValue(mpcprotocol.MpcSignA + iStr, []big.Int{*a.result})
		err = result.SetValue(mpcprotocol.MpcSignA0 + iStr, []big.Int{a.randCoefficient[0]})
		err = result.SetValue(mpcprotocol.MpcSignR + iStr, []big.Int{*r.result})
		err = result.SetValue(mpcprotocol.MpcSignR0 + iStr, []big.Int{r.randCoefficient[0]})
		err = result.SetValue(mpcprotocol.MpcSignB + iStr, []big.Int{*b.result})
		err = result.SetValue(mpcprotocol.MpcSignC + iStr, []big.Int{*c.result})

		ar := make([]big.Int, 1)
		ar[0].Mul(a.result, r.result)
		ar[0].Mod(&ar[0], crypto.Secp256k1_N)
		ar[0].Add(&ar[0], b.result)
		ar[0].Mod(&ar[0], crypto.Secp256k1_N)
		err = result.SetValue(mpcprotocol.MpcSignARSeed + iStr, ar)
	}

	log.Info("TXSignJR_JZ_Step.FinishStep succeed")
	mpcsyslog.Info("TXSignJR_JZ_Step.FinishStep succeed")
	return nil
}
