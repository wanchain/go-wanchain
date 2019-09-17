package step

import (
	"crypto/ecdsa"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
)

type MpcRSKShare_Step struct {
	BaseMpcStep
}

func CreateMpcRSKShareStep(degree int, peers *[]mpcprotocol.PeerInfo) *MpcRSKShare_Step {
	mpc := &MpcRSKShare_Step{*CreateBaseMpcStep(peers, 1)}
	mpc.messages[0] = createJRSSValue(degree, len(*peers))
	return mpc
}

func (jrss *MpcRSKShare_Step) CreateMessage() []mpcprotocol.StepMessage {
	message := make([]mpcprotocol.StepMessage, len(*jrss.peers))
	JRSSvalue := jrss.messages[0].(*RandomPolynomialValue)
	for i := 0; i < len(*jrss.peers); i++ {
		message[i].Msgcode = mpcprotocol.MPCMessage
		message[i].PeerID = &(*jrss.peers)[i].PeerID
		message[i].Data = make([]big.Int, 1)
		message[i].Data[0] = JRSSvalue.polyValue[i]
	}

	return message
}

func (jrss *MpcRSKShare_Step) FinishStep(result mpcprotocol.MpcResultInterface, mpc mpcprotocol.StoremanManager) error {
	err := jrss.BaseMpcStep.FinishStep()
	if err != nil {
		return err
	}

	// gskshare
	JRSSvalue := jrss.messages[0].(*RandomPolynomialValue)
	err = result.SetValue(mpcprotocol.RMpcPrivateShare, []big.Int{*JRSSvalue.result})
	if err != nil {
		return err
	}
	// gpkshare
	var gpkShare ecdsa.PublicKey
	gpkShare.X, gpkShare.Y = crypto.S256().ScalarBaseMult((*JRSSvalue.result).Bytes())
	err = result.SetValue(mpcprotocol.RMpcPublicShare, []big.Int{*gpkShare.X, *gpkShare.Y})
	if err != nil {
		return err
	}

	return nil
}

func (jrss *MpcRSKShare_Step) HandleMessage(msg *mpcprotocol.StepMessage) bool {
	seed := jrss.getPeerSeed(msg.PeerID)
	if seed == 0 {
		log.SyslogErr("MpcJRSS_Step, can't find peer seed. peerID:%s", msg.PeerID.String())
	}

	JRSSvalue := jrss.messages[0].(*RandomPolynomialValue)
	_, exist := JRSSvalue.message[seed]
	if exist {
		log.SyslogErr("MpcJRSS_Step, can't find msg . peerID:%s, seed:%d", msg.PeerID.String(), seed)
		return false
	}

	JRSSvalue.message[seed] = msg.Data[0] //message.Value
	return true
}
