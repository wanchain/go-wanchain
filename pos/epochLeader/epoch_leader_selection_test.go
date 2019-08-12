package epochLeader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/pos/posconfig"

	"github.com/wanchain/go-wanchain/common"
	"github.com/wanchain/go-wanchain/common/hexutil"
	"github.com/wanchain/go-wanchain/common/math"
	"github.com/wanchain/go-wanchain/consensus/ethash"
	"github.com/wanchain/go-wanchain/core"
	"github.com/wanchain/go-wanchain/core/state"
	"github.com/wanchain/go-wanchain/core/types"
	"github.com/wanchain/go-wanchain/core/vm"
	"github.com/wanchain/go-wanchain/ethdb"
	"github.com/wanchain/go-wanchain/params"
	"github.com/wanchain/go-wanchain/pos/util"
	//"github.com/wanchain/go-wanchain/log"
	//"crypto/rand"
	//"github.com/wanchain/pos/cloudflare"
	//"github.com/wanchain/go-wanchain/common"
	//"strconv"
)

var allocJson1 = `{
    "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e": {
		"balance": "40000000000000000000028",
		"staking":{
			"amount":"400000000000000000000",
			"s256pk":"0x04d7dffe5e06d2c7024d9bb93f675b8242e71901ee66a1bfe3fe5369324c0a75bf6f033dc4af65f5d0fe7072e98788fcfa670919b5bdc046f1ca91f28dff59db70",
			"bn256pk":"0x150b2b3230d6d6c8d1c133ec42d82f84add5e096c57665ff50ad071f6345cf45191fd8015cea72c4591ab3fd2ade12287c28a092ac0abf9ea19c13eb65fd4910"
		}
    },
	"0x8b179c2b542f47bb2fb2dc40a3cf648aaae1df16": {
		"balance": "1000000000000000000000",
		"staking":{
			"amount":"100000000000000000000",
			"s256pk":"0x04e37be2aa12f3df03953c0a172d0f964a1561f321120c8dfa061df35dac4d52d03bbeb5758c37de915209d412bf2aead300c6e530a46967dfb736a8b4f186d950",
			"bn256pk":"0x2bf18653c442fa51689c70ad3169d444c12e8279e093bf8ddb1010fbd772d0db0c322c62c475d25fde04b245d8913b8f0c38c437cc819c679a51a5263a40fadb"
		}
    },
	"0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8": {
		"balance": "1000000000000000000000",
		"staking":{
			"amount":"100000000000000000000",
			"s256pk":"0x045c6f2618a476792c14a5959e418c9038c0b347fca40403326f818c2ed5dbdba5a77ecbaa0a2822ffb9d98e2190630537387a2ca0c24afca372d41acd9c9bf7d1",
			"bn256pk":"0x102158973be3fdcc6c73263995fb2bb441b5e71a24cbca1d87516a29a94f197f08e898b88e26b67d933c5bb980b5ae6d257acd6952a98ee8a321a3b747bd197f"
		}
    },
	"0x7a22d4e2dc5c135c4e8a7554157446e206835b05": {
		"balance": "1000000000000000000000",
		"staking":{
			"amount":"100000000000000000000",
			"s256pk":"0x04a8aa21dc331a4471c0d32b4a1032812297c4c201acb48286279b701c990ea35a145eeeb5c97334e896683234d1deb28420c609c81375e4612b76ca823c80139f",
			"bn256pk":"0x25230a3e503de1a331a36157568c97d4a38be6c60846a49c44c509d97e2b0792272d486e3eef5e323a5fae10121381d5fe7c7d63ba3d82839b9b10ef1e2a982f"
		}
    },
	"0x8836c42e61310bb447cfa70aa789662bfece2832": {
		"balance": "1000000000000000000000",
		"staking":{
			"amount":"100000000000000000000",
			"s256pk":"0x04a754389f9beb3f09ff07610fb7adb6d6107bc7105b2a2bdc9b11d4b3ef55c27cd74cf829007b701c7b7b69768dccf3081d8d4dca626c1fe3c92c1017d4fb74e0",
			"bn256pk":"0x11f4d33b5eaeec1067be306e8e047f28f95f4e7a241d92bddd616abff27337f527447ad54e497a16ae431aade12e2660b7e57004ae83387bdb66ba2625245d81"
		}
    },
	"0x576713551e21b136c5b775f5fb4081441773b2d4": {
		"balance": "1000000000000000000000",
		"staking":{
			"amount":"100000000000000000000",
			"s256pk":"0x0480dc2f30861c94c3a9dd9aa6dad207c9431a771247f61c8d62e9b616435ceebcb6b60e40e9f46a6f08bbd3e05939b3c0ea3f690c116c34afde1e533c2e8bb0ff",
			"bn256pk":"0x280e59db5249403de055615028242680bde1b89c3912a5d12ed0e76ad988f75a0a8e187dbc4688eb6eb708ced261859dd2ea470be08f2f4f0cbdfe6c36d8ae4c"
		}
    },
	"0x32944ea9809c460305c269f75dbb83ff022e79da": {
		"balance": "1000000000000000000000",
		"staking":{
			"amount":"100000000000000000000",
			"s256pk":"0x049fe5b62c6300d3c913bc1d0082af669ee4ebdcc9fe967f4296bdb12e4c5d99c0a04d9712e35370e80a08a2db0e3d06f2281e0638ad92ffcb79c6473553ebe5d2",
			"bn256pk":"0x166969216beaf17f488ad267dee8cb8edbd74fccff4ddff71244656a49da0ae42a98752303b46a118d1aaa7c604fe47366bf8948feeb842b3d887b47eaa071a3"
		}
    },
	"0x435b316a70cdb8143d56b3967aacdb6392fd6125": {
		"balance": "4000000000000000000000",
		"staking":{
			"amount":"400000000000000000000",
			"s256pk":"0x04dcbab97fee67c00f6969d2e9eedf100d72bbce8fa06a40d2fb074bd8474b96c0288d1bab49c173a60d86356cddcec4e6c0f074b25e30f34e4f15cc23622409f1",
			"bn256pk":"0x022134f210c03cbc04b185b45c413163a968c101d349225b2a2eb75edc34179a0fa19a9677be92993db3ddeb8de327ecb0a5bead05bbdcfe665e291a3c0954ef"
		}
    },
	"0x23fc2eda99667fd3df3caa7ce7e798d94eec06eb": {
		"balance": "1000000000000000000000",
		"staking":{
			"amount":"100000000000000000000",
			"s256pk":"0x04a5946c1968bbe53bfd897c06d53555292bef6e71a4c8ed92b9c1de1b1b94f797c3984581307788ff0c2a564548901f83000b1aa65a1532dacca01214e1f3fa6c",
			"bn256pk":"0x1b4626213c1af35b38d226a386e3b661a98198794e52740d2be58c14315dc1a12d8e79a95c2b6a21653550b422bb3211e62b6af6b1afe09c3232bd6c6b601ea5"
		}
    },
 	"0xb02737095f945768ca983a60e0ba92b758111111": {"balance": "10000000000000000000000000000000000000"},
	"0xb752027021340f2fec33abc91daf198915bbbbbb": {"balance": "10000000000000000000000000000000000000"},
	"0xe8ffc3d0c02c0bfc39b139fa49e2c5475f000000": {"balance": "10000000000000000000000000000000000000"}
 }`

func jsonPrealloc(data string) core.GenesisAlloc {
	var ga core.GenesisAlloc
	if err := json.Unmarshal([]byte(data), &ga); err != nil {
		panic(err)
	}
	return ga
}

// func testPlutoGenesisBlock() *core.Genesis {
// 	return core.DefaultPlutoGenesisBlock()
// }
func testGenesisBlock1() *core.Genesis {
	return &core.Genesis{
		Config:    params.PlutoChainConfig,
		Timestamp: 0x59f83144,
		ExtraData: hexutil.MustDecode("0x00000000000000000000000000000000000000000000000000000000000000002d0e7c0813a51d3bd1d08246af2a8a7a57d8922e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		//GasLimit:   0x47b760,	// 4700000
		GasLimit:   0x2cd29c0, // 47000000
		Difficulty: big.NewInt(1),
		Alloc:      jsonPrealloc(allocJson1),
	}
}

type bproc struct{}

func (bproc) ValidateBody(*types.Block) error { return nil }
func (bproc) ValidateState(block, parent *types.Block, state *state.StateDB, receipts types.Receipts, usedGas *big.Int) error {
	return nil
}
func (bproc) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, *big.Int, error) {
	return nil, nil, new(big.Int), nil
}

func newTestBlockChain(fake bool) (*core.BlockChain, *core.ChainEnv) {

	db, _ := ethdb.NewMemDatabase()
	gspec := testGenesisBlock1()
	gspec.Difficulty = big.NewInt(1)
	gspec.MustCommit(db)
	engine := ethash.NewFullFaker(db)
	if !fake {
		engine = ethash.NewTester(db)
	}
	blockchain, err := core.NewBlockChain(db, gspec.Config, engine, vm.Config{},nil)
	if err != nil {
		panic(err)
	}
	chainEnv := core.NewChainEnv(params.TestChainConfig, gspec, engine, blockchain, db)
	blockchain.SetValidator(bproc{})

	return blockchain, chainEnv
}

func TestGetEpochLeaders(t *testing.T) {
	var networkId uint64
	networkId = 6
	posconfig.Init(nil,networkId)
	epochID, slotID := util.GetEpochSlotID()
	fmt.Println("epochID:", epochID, " slotID:", slotID)

	blkChain, _ := newTestBlockChain(true)

	epocher1 := NewEpocherWithLBN(blkChain, "rb1", "epdb1")
	epocher2 := NewEpocherWithLBN(blkChain, "rb2", "epdb2")

	epocher1.SelectLeadersLoop(0)

	time.Sleep(30 * time.Second)
	epocher2.SelectLeadersLoop(0)

	epl1 := epocher1.GetEpochLeaders(0)
	epl2 := epocher2.GetEpochLeaders(0)

	if len(epl1) != len(epl2) {
		t.Fail()
	}

	for idx, val := range epl1 {
		if !bytes.Equal(val, epl2[idx]) {
			t.Fail()
		}
	}

	rbl1 := epocher1.GetRBProposerGroup(0)
	rbl2 := epocher2.GetRBProposerGroup(0)

	if len(epl1) != len(epl2) {
		t.Fail()
	}

	for idx, val := range rbl1 {
		if !bytes.Equal(val.PubBn256, rbl2[idx].PubBn256) {
			t.Fail()
		}
	}

}

//func TestGetGetEpochLeadersCapability(t *testing.T) {
//
//	blkChain, _ := newTestBlockChain(true)
//
//	stateDb, err := blkChain.StateAt(blkChain.GetBlockByNumber(0).Root())
//	if err != nil {
//		t.Fail()
//	}
//
//	epocher1 := NewEpocherWithLBN(blkChain, "countrb1", "countepdb1")
//
//	loopCount := 10000
//	nr := 30
//	ne := 30
//	for i:=0;i<loopCount;i++ {
//		_, ga, err := bn256.RandomG1(rand.Reader)
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		epocher1.SelectLeaders(ga.Marshal(),ne,nr,stateDb,uint64(i))
//	}
//
//	epochLeadersMap := make(map[string]int)
//	for i:=0;i<loopCount;i++ {
//		epochLeadersArray := epocher1.GetEpochLeaders(uint64(i))
//		for _,value := range epochLeadersArray {
//			key := common.ToHex(value)
//			if count,ok:= epochLeadersMap[key];ok {
//				epochLeadersMap[key] = count + 1
//			} else {
//				epochLeadersMap[key] = 1
//			}
//		}
//	}
//
//	fmt.Println("the select epoch leader count=" + strconv.Itoa(len(epochLeadersMap)))
//	if len(epochLeadersMap) != 9 {
//		t.Fail()
//	}
//
//	for key,value := range epochLeadersMap {
//		fmt.Println("key=",key,"count=",value,"selected percent=",(float32(value)/float32(loopCount*ne))*100,"%")
//	}
//
//}
//
//
//
//
//func TestGetGetEpochLeaderAddress(t *testing.T) {
//
//	checkArray := [3][2]string{
//		{
//		 "0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e",
//		 "0x150b2b3230d6d6c8d1c133ec42d82f84add5e096c57665ff50ad071f6345cf45191fd8015cea72c4591ab3fd2ade12287c28a092ac0abf9ea19c13eb65fd4910",
//		},
//
//		{"0x8b179c2b542f47bb2fb2dc40a3cf648aaae1df16",
//		"0x2bf18653c442fa51689c70ad3169d444c12e8279e093bf8ddb1010fbd772d0db0c322c62c475d25fde04b245d8913b8f0c38c437cc819c679a51a5263a40fadb",
//		},
//
//		{"0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8",
//		"0x102158973be3fdcc6c73263995fb2bb441b5e71a24cbca1d87516a29a94f197f08e898b88e26b67d933c5bb980b5ae6d257acd6952a98ee8a321a3b747bd197f",
//		},
//	}
//
//
//	blkChain,_ := newTestBlockChain(true)
//
//	stateDb, err := blkChain.StateAt(blkChain.GetBlockByNumber(0).Root())
//	if err != nil {
//		t.Fail()
//	}
//
//	epocher1 := NewEpocherWithLBN(blkChain, "countrb1", "countepdb1")
//
//	nr := 30
//	ne := 30
//
//	_, ga, err := bn256.RandomG1(rand.Reader)
//	epocher1.SelectLeaders(ga.Marshal(),ne,nr,stateDb,0)
//
//	rbl1 := epocher1.GetRBProposerGroup(0)
//	epMap :=  make(map[string]int)
//	for idx,val := range rbl1 {
//		v := common.ToHex(val.Marshal())
//		epMap[v] = idx
//	}
//
//	for _,val := range checkArray {
//		g1 := val[1]
//
//		if idx,ok := epMap[g1];ok {
//			addrBytes := common.FromHex(val[0])
//			addr := common.BytesToAddress(addrBytes)
//			ret := epocher1.GetProposerBn256PK(0, uint64(idx), addr)
//			fmt.Println(common.ToHex(addrBytes),common.ToHex(ret))
//			if !bytes.Equal(ret,common.FromHex(g1)) {
//				t.Fail()
//			}
//		}
//	}
//}

//
//func TestGenerateLeader(t *testing.T) {
//	blkChain, _ := newTestBlockChain(true)
//	e := NewEpocherWithLBN(blkChain, "countrb1", "leaderdb", "countepdb1")
//	targetBlkNum := uint64(0)
//
//	epochId := uint64(0)
//
//	stateDb, err := e.blkChain.StateAt(e.blkChain.GetBlockByNumber(targetBlkNum).Root())
//	if err != nil {
//		t.Fail()
//	}
//	err = e.GenerateLeader(stateDb, epochId)
//	if err != nil {
//		t.Fail()
//	}
//	t.Log("GenerateLeader done")
//
//	i:=0
//	epLeaders := e.GetEpochLeadersInfo(epochId)
//	t.Log(epLeaders)
//	for i=0; i<len(epLeaders); i++ {
//		t.Log(common.ToHex(epLeaders[i].PubSec256))
//	}
//
//	rbGroup := e.GetRBProposerGroup(epochId)
//	t.Log(rbGroup)
//	for i=0; i<len(rbGroup); i++ {
//		t.Log(common.ToHex(rbGroup[i].PubSec256))
//	}
//
//}
//
//func TestGetEpochProbability(t *testing.T) {
//	blkChain, _ := newTestBlockChain(true)
//	epocherInst := NewEpocherWithLBN(blkChain, "countrb1", "leaderdb", "countepdb1")
//
//	epochID := uint64(0)
//	addr := common.HexToAddress("0x2d0e7c0813a51d3bd1d08246af2a8a7a57d8922e")
//
//	infors, feeRate, total, err := epocherInst.GetEpochProbability(epochID, addr)
//	if err != nil {
//		log.Fatal("failed to GetEpochProbability: ", err)
//	}
//	log.Println(infors)
//	log.Println(feeRate)
//	log.Println(total)
//}
func TestCalProbability(t *testing.T) {
	blkChain, _ := newTestBlockChain(true)
	epocherInst := NewEpocherWithLBN(blkChain, "countrb1", "countepdb1")
	addr := common.Address{}
	addr.SetString("0xd1d1079cdb7249eee955ce34d90f215571c0781d")
	amount := math.MustParseBig256("1000000000000000000000000")
	t.Log("amount: ", amount)
	item := vm.StakerInfo{
		Address:      addr,
		Amount:       amount,
		Clients:      make([]vm.ClientInfo, 0),
		FeeRate:      100,
		From:         addr,
		LockEpochs:   6,
		PubBn256:     hexutil.MustDecode("0x2c9a9b8dfb23dfd62c3abb3de1be1ff9c3495edc7ec40d6a816643f1d561f0f81fd29a3218ccd8fcae7996f390b91b71b77195e31a09608aa67f9c33243abfff"),
		PubSec256:    hexutil.MustDecode("0x04e78171373e7d4671fe7a0ab7c3983f46874fab1db2cce81ce512059e5b7e94373abf9875a1dd339aca8d36bdaad6d7542f3243f488155fc12e3a26c6e2f753cd"),
		StakingEpoch: uint64(1),
	}
	c := vm.ClientInfo{}
	addrc := common.Address{}
	addrc.SetString("0x6e6f37b8463b541fd6d07082f30f0296c5ac2118")
	c.Address = addrc
	c.Amount = math.MustParseBig256("1000000000000000000000000")
	item.Clients = append(item.Clients, c)
	for epochid := uint64(1); epochid < 11; epochid++ {
		pb := epocherInst.CalProbability(item.Amount, item.LockEpochs)
		t.Log("pb: ", epochid, pb)
		for i := 0; i < len(item.Clients); i++ {
			lockEpoch := item.LockEpochs
			cp := epocherInst.CalProbability(item.Clients[i].Amount, lockEpoch)
			t.Log("cp: ", epochid, cp)
			pb.Add(pb, cp)
		}

		t.Log("total:", epochid, pb)
		t.Log("===========")
	}
}
