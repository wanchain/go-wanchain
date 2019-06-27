package cfm

import (
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/pos/posconfig"
)

func TestIsInWhiteList(t *testing.T) {
	var WhiteList = [...]string{
		"0x04dc40d03866f7335e40084e39c3446fe676b021d1fcead11f2e2715e10a399b498e8875d348ee40358545e262994318e4dcadbc865bcf9aac1fc330f22ae2c786",
		"0x0406a5c2c0524968089b8e7fdaddd642732e04e0f1da4c49dcb7810aa37dd471317b77936015a86e8efbc2002485e9d146ee392a3021e0c5bf53e5c0f6b158de09",
	}
	var networkId uint64
	networkId = 6
	posconfig.Init(nil,networkId)
	InitCFM(nil)
	c := GetCFM()
	c.whiteList = make(map[common.Address]int, 0)
	for _, value := range WhiteList {
		b := hexutil.MustDecode(value)
		address := crypto.PubkeyToAddress(*(crypto.ToECDSAPub(b)))
		c.whiteList[address] = 1
	}

	var coinBase common.Address
	coinBase.SetBytes(hexutil.MustDecode("0xcf696d8eea08a311780fb89b20d4f0895198a489"))
	if !c.isInWhiteList(coinBase) {
		t.Fail()
	}

	coinBase = common.StringToAddress("0xcf696d8eea08a311780fb89b20d4f0895198a480")

	if c.isInWhiteList(coinBase) {
		t.Fail()
	}
}

func TestGetCFM(t *testing.T) {

	InitCFM(nil)
	c := GetCFM()

	if c == nil {
		t.Fail()
	}
}

func TestInitCFM(t *testing.T) {
	var networkId uint64
	networkId = 6
	posconfig.Init(nil,networkId)
	InitCFM(nil)
	c := GetCFM()

	if len(c.whiteList) == 0 {
		t.Logf("No white list coinbase exisit")
		t.Fail()
	}
	t.Logf("white list coinbase length is %d\n", len(c.whiteList))
}

func TestGetSlotsCount(t *testing.T) {
	var networkId uint64
	networkId = 6
	posconfig.Init(nil,networkId)
	InitCFM(nil)
	c := GetCFM()

	start := uint64(time.Now().Unix())
	stop := uint64(start + posconfig.SlotTime - 1)

	if c.getSlotsCount(start, stop, posconfig.SlotTime) != 1 {
		t.Fail()
	}

	stop = uint64(start + posconfig.SlotTime + 1)
	if c.getSlotsCount(start, stop, posconfig.SlotTime) != 2 {
		t.Fail()
	}

	stop = uint64(start + posconfig.SlotTime)
	if c.getSlotsCount(start, stop, posconfig.SlotTime) != 2 {
		t.Fail()
	}

	stop = uint64(start - posconfig.SlotTime)
	if c.getSlotsCount(start, stop, posconfig.SlotTime) != 0 {
		t.Fail()
	}
}

func TestGetMaxStableBlkNumber(t *testing.T) {
	var networkId uint64
	networkId = 6
	posconfig.Init(nil,networkId)

	blkStatusArr := make([]*BlkStatus, 0)
	InitCFM(nil)
	c := GetCFM()

	if c.getMaxStableBlkNumber(blkStatusArr, 0, 0, nil) != 0 {
		t.Fail()
	}

	if c.getMaxStableBlkNumber(blkStatusArr, 0, 9, ErrNullBlk) != 0 {
		t.Fail()
	}

	if c.getMaxStableBlkNumber(blkStatusArr, 9, 0, nil) != 0 {
		t.Fail()
	}

	if c.getMaxStableBlkNumber(blkStatusArr, 9, 0, ErrNullBlk) != 0 {
		t.Fail()
	}

	blkStatusArr = append(blkStatusArr, &BlkStatus{100, false})
	blkStatusArr = append(blkStatusArr, &BlkStatus{99, true})
	blkStatusArr = append(blkStatusArr, &BlkStatus{98, true})

	if c.getMaxStableBlkNumber(blkStatusArr, 97, 100, nil) != 99 {
		t.Fail()
	}

	blkStatusArr = make([]*BlkStatus, 0)
	blkStatusArr = append(blkStatusArr, &BlkStatus{100, false})
	blkStatusArr = append(blkStatusArr, &BlkStatus{99, false})
	blkStatusArr = append(blkStatusArr, &BlkStatus{98, false})

	if c.getMaxStableBlkNumber(blkStatusArr, 97, 100, nil) != 97 {
		t.Fail()
	}

	blkStatusArr = make([]*BlkStatus, 0)
	blkStatusArr = append(blkStatusArr, &BlkStatus{0, false})

	if c.getMaxStableBlkNumber(blkStatusArr, 0, 1, nil) != 0 {
		t.Fail()
	}

	blkStatusArr = make([]*BlkStatus, 0)
	blkStatusArr = append(blkStatusArr, &BlkStatus{1, false})

	if c.getMaxStableBlkNumber(blkStatusArr, 0, 1, nil) != 0 {
		t.Fail()
	}
}
