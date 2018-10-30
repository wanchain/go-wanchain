package step

import (
	"testing"
	"github.com/wanchain/go-wanchain/p2p/discover"
	mpcprotocol "github.com/wanchain/go-wanchain/storeman/storemanmpc/protocol"
	"math/big"
	"github.com/wanchain/go-wanchain/crypto"
)


type mpcPointGeneratorTestContext struct {
	preValueKey string
	peers []mpcprotocol.PeerInfo
}

func (ctx *mpcPointGeneratorTestContext) Init()  {
	if len(ctx.peers) != 0 {
		return
	}

	ctx.preValueKey = "pointKey"

	nodeId1, _ := discover.HexID("0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001")
	nodeId2, _ := discover.HexID("0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002")
	nodeId3, _ := discover.HexID("0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003")

	ctx.peers = make([]mpcprotocol.PeerInfo, 0, 3)
	ctx.peers = append(ctx.peers, mpcprotocol.PeerInfo{nodeId1, 1})
	ctx.peers = append(ctx.peers, mpcprotocol.PeerInfo{nodeId2, 2})
	ctx.peers = append(ctx.peers, mpcprotocol.PeerInfo{nodeId3, 3})

}

func TestPotGeneratorCalculateResult(t *testing.T)  {
	var ctx mpcPointGeneratorTestContext
	ctx.Init()

	point := createPointGenerator(ctx.preValueKey)

	curve := crypto.S256()
	for i := 1; i <= 10; i++ {
		xi, yi := curve.ScalarBaseMult(big.NewInt(int64(i)).Bytes())
		point.message[uint64(i)] = [2]big.Int{*xi, *yi}
	}

	resultX, resultY := curve.ScalarBaseMult(big.NewInt(int64(1)).Bytes())
	for i := 2; i <= 10; i++ {
		xi, yi := curve.ScalarBaseMult(big.NewInt(int64(i)).Bytes())
		resultXTmp, resultYTmp := curve.Add(resultX, resultY, xi, yi)
		resultX = resultXTmp
		resultY = resultYTmp
		t.Logf("resultX:%s, resultY:%s", resultX.String(), resultY.String())
	}

	t.Logf("resultX:%s, resultY:%s", resultX.String(), resultY.String())

	err := point.calculateResult()
	if err != nil {
		t.Error("point calculateResult fail")
	}

	x55, y55 := curve.ScalarBaseMult(big.NewInt(55).Bytes())
	t.Logf("x10:%s, y10:%s", x55.String(), y55.String())
	t.Logf("point.x:%s, point.y:%s", point.result[0].String(), point.result[1].String())

	if x55.Cmp(&point.result[0]) != 0 || y55.Cmp(&point.result[1]) != 0 {
		t.Error("point calculate result is wrong")
	}

}



