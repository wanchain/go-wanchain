package storemanmpc

import (
	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpccrypto "github.com/wanchain/go-wanchain/storeman/storemanmpc/crypto"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	"testing"
)

func TestMpcSelectPeers(t *testing.T) {
	var mpcID uint64
	var err error
	for {
		mpcID, err = mpccrypto.UintRand(uint64(1<<64 - 1))
		if err != nil {
		} else {
			break
		}
	}

	t.Log("mpcIDValue", mpcID)
	msger := testP2pMessager{}
	mpcDistributor := CreateMpcDistributor(nil, &msger, "", "", "", "1111")
	nThread := 21
	nodeId := common.Hex2Bytes("8f8581b96c387d80c64b8924ec466b17d3994db98ea5601c4ccb4b0a5acead74f43b7ffde50cfcf96efd19005e3986186268d0c0865b4217d28eff61d693bc16")
	peers := make([]mpcprotocol.PeerInfo, nThread)
	for i := 0; i < nThread; i++ {
		copy(peers[i].PeerID[:], nodeId)
		peers[i].PeerID[0] = byte(i + 1)
		peers[i].PeerID[1] = byte(i + 1)
		peers[i].PeerID[2] = byte(i + 1)
		peers[i].Seed = uint64(i + 1)
	}

	mpcDistributor.Self = &discover.Node{ID: peers[0].PeerID}
	mpcDistributor.StoreManGroup = make([]discover.NodeID, nThread)
	for i, item := range peers {
		mpcDistributor.StoreManGroup[i] = item.PeerID
	}

	txHash := common.HexToHash("340dd630ad21bf010b4e676dbfa9ba9a02175262d1fa356232cfde6cb5b47ef2")
	mpcDistributor.selectPeers(mpcprotocol.MpcSignLeader, peers, MpcValue{mpcprotocol.MpcTxHash, []big.Int{*txHash.Big()}, nil})
	//common.HexToHash("426fcb404ab2d5d8e61a3d918108006bbb0a9be65e92235bb10eefbdb6dcd053"),
	//common.HexToHash("48078cfed56339ea54962e72c37c7f588fc4f8e5bc173827ba75cb10a63a96a5"),
	//common.HexToHash("5723d2c3a83af9b735e3b7f21531e5623d183a9095a56604ead41f3582fdfb75"),
}
