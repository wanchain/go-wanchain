package step

import (
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	mpcsyslog "github.com/wanchain/go-wanchain/storeman/syslog"
	"math/big"
)

type TXSignJR_JZ_Step struct {
	BaseMpcStep
}

func CreateTXSignJR_JZ_Step(degree int, peers *[]mpcprotocol.PeerInfo) *TXSignJR_JZ_Step {
	mpc := &TXSignJR_JZ_Step{*CreateBaseMpcStep(peers, 4)}
	mpc.messages[0] = createJRSSValue(degree, len(*peers))
	mpc.messages[1] = createJRSSValue(degree, len(*peers))
	mpc.messages[2] = createJZSSValue(degree*2, len(*peers))
	mpc.messages[3] = createJZSSValue(degree*2, len(*peers))
	return mpc
}

func (jrjz *TXSignJR_JZ_Step) CreateMessage() []mpcprotocol.StepMessage {
	message := make([]mpcprotocol.StepMessage, len(*jrjz.peers))
	a := jrjz.messages[0].(*RandomPolynomialValue)
	r := jrjz.messages[1].(*RandomPolynomialValue)
	b := jrjz.messages[2].(*RandomPolynomialValue)
	c := jrjz.messages[3].(*RandomPolynomialValue)
	for i := 0; i < len(*jrjz.peers); i++ {
		message[i].Msgcode = mpcprotocol.MPCMessage
		message[i].PeerID = &(*jrjz.peers)[i].PeerID
		message[i].Data = []big.Int{a.polyValue[i], r.polyValue[i], b.polyValue[i], c.polyValue[i]}
	}

	return message
}

func (jrjz *TXSignJR_JZ_Step) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := jrjz.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	a := jrjz.messages[0].(*RandomPolynomialValue)
	r := jrjz.messages[1].(*RandomPolynomialValue)
	b := jrjz.messages[2].(*RandomPolynomialValue)
	c := jrjz.messages[3].(*RandomPolynomialValue)
	err = result.SetValue(mpcprotocol.MpcSignA, []big.Int{*a.result})
	if err != nil {
		return err
	}

	err = result.SetValue(mpcprotocol.MpcSignA0, []big.Int{a.randCoefficient[0]})
	if err != nil {
		return err
	}

	err = result.SetValue(mpcprotocol.MpcSignR, []big.Int{*r.result})
	if err != nil {
		return err
	}

	err = result.SetValue(mpcprotocol.MpcSignR0, []big.Int{r.randCoefficient[0]})
	if err != nil {
		return err
	}

	err = result.SetValue(mpcprotocol.MpcSignB, []big.Int{*b.result})
	if err != nil {
		return err
	}

	err = result.SetValue(mpcprotocol.MpcSignC, []big.Int{*c.result})
	if err != nil {
		return err
	}

	ar := make([]big.Int, 1)
	ar[0].Mul(a.result, r.result)
	ar[0].Mod(&ar[0], crypto.Secp256k1_N)
	ar[0].Add(&ar[0], b.result)
	ar[0].Mod(&ar[0], crypto.Secp256k1_N)
	return result.SetValue(mpcprotocol.MpcSignARSeed, ar)
}

func (jrjz *TXSignJR_JZ_Step) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	log.Debug("TXSignJR_JZ_Step getMessage:", "peerID", msg.PeerID)
	seed := jrjz.getPeerSeed(msg.PeerID)
	if seed == 0 {
		mpcsyslog.Err("TXSignJR_JZ_Step.HandleMessage, get seed fail. peer:%s", msg.PeerID.String())
		log.Error("TXSignJR_JZ_Step Not Find:", "peerID", msg.PeerID)
	}

	a := jrjz.messages[0].(*RandomPolynomialValue)
	r := jrjz.messages[1].(*RandomPolynomialValue)
	b := jrjz.messages[2].(*RandomPolynomialValue)
	c := jrjz.messages[3].(*RandomPolynomialValue)
	_, exist := a.message[seed]
	if exist {
		mpcsyslog.Err("TXSignJR_JZ_Step.HandleMessage, get msg fail. peer:%s", msg.PeerID.String())
		return false
	}

	a.message[seed] = msg.Data[0] //message.Value
	r.message[seed] = msg.Data[1] //message.Value
	b.message[seed] = msg.Data[2] //message.Value
	c.message[seed] = msg.Data[3] //message.Value

	return true
}
