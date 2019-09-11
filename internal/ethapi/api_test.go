// Copyright 2018 Wanchain Foundation Ltd

package ethapi

import (
	"context"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/common"
)

func TestGenerateOneTimeAddress(t *testing.T) {
	s := new(PublicTransactionPoolAPI)

	vailidWaddrs := []string{
		"0x02e37be2aa12f3df03953c0a172d0f964a1561f321120c8dfa061df35dac4d52d0030dfc2b696438f942a9c187edb10691346a0d68cdfbbc590f85ba46f3b5f9e2a9",
		"0x03a8aa21dc331a4471c0d32b4a1032812297c4c201acb48286279b701c990ea35a037061ac75a8a89b2dc4454953275edaced7d3ae16ac0ddce5fbddd2bc04bfe16d",
		"0x024230cabb18b57b216e4f2865090e5a042150704a1c020b2ba87d319b7d3b5c5703fa8e37f3707803978c5e154ce05b251d82dd4247712493df9a094d62a17bbd97",
		"0x03059dee5729f28b64edd3e4c79e18af99e155acd1c66aadd81b01e8a43c3150f50240bf3059bcf95ac65ddd71b74fedd5800c1c90a4ae376f3319dffeda3990a6a8",
		"0x03918d923c5cd59e5bbc04efd595b21a9248783a4f3d7dc149a0202646db03779d023acb92fbca55476154a90447b784cbdfde6270fd8acb8170b03178285ba44e6d",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	for _, waddr := range vailidWaddrs {
		ota, err := s.GenerateOneTimeAddress(ctx, waddr)
		if err != nil {
			t.Errorf("waddr:%s, err:%s", waddr, err.Error())
		}

		if len(ota) != common.WAddressLength*2+2 {
			t.Errorf("invalid ota length! waddr:%s, ota:%s", waddr, ota)
		}

		if ota[0] != '0' || (ota[1] != 'x' && ota[1] != 'X') {
			t.Errorf("invalid ota! waddr:%s, ota:%s", waddr, ota)
		}
	}

	invalidWaddr := []string{
		"",
		"4324324242",
		"0x324324324324",
		"dsfsfsfds",
		"0xfsfdsfdsfhhhhjj",
		"0x5435436lefjeerw9998",
		"0x03918d923c5cd59e5bbc04efd595b21a9248783a4f3d7dc149a0202646db03779d023acb92fbca55476154a90447b784cbdfde6270fd8acb8170b03178285ba44e6d654654",
	}

	for _, waddr := range invalidWaddr {
		ota, err := s.GenerateOneTimeAddress(ctx, waddr)
		if err == nil {
			t.Errorf("succeed from invalid wanaddress. waddr:%s, ota:%s", waddr, ota)
		}
	}
}
