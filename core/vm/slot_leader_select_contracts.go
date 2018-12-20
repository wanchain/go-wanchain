package vm

import (
	"github.com/wanchain/go-wanchain/accounts/abi"
	"github.com/wanchain/go-wanchain/core/types"
	"strings"
)

var (
	slotLeaderSCDef = `
	[{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [{"name": "OtaAddr","type":"string"},{"name": "Value","type": "uint256"}],"name": "buyCoinNote","outputs": [{"name": "OtaAddr","type":"string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","inputs": [{"name":"RingSignedData","type": "string"},{"name": "Value","type": "uint256"}],"name": "refundCoin","outputs": [{"name": "RingSignedData","type": "string"},{"name": "Value","type": "uint256"}]},{"constant": false,"type": "function","stateMutability": "nonpayable","inputs": [],"name": "getCoins","outputs": [{"name":"Value","type": "uint256"}]}]`

	slotLeaderAbi, errSlotLeaderSCInit  = abi.JSON(strings.NewReader(slotLeaderSCDef))
	stgOneIdArr, stgTwoIdArr 			[4]byte

	//StampValueSet   = make(map[string]string, 5)
	//WanCoinValueSet = make(map[string]string, 10)
)
func init() {
	if errSlotLeaderSCInit != nil {
		panic("err in slot leader sc initialize ")
	}
}
type slotLeaderSC struct {

}

func (c *slotLeaderSC) RequiredGas(input []byte) uint64 {

	// A_i=α_i*PKi i = {1,2,....n}. size = sizeof(ecdsa.PublicKey)*N
	// π_i							size = sizeof(uint64)x2 w[0]=e w[1]=z
	//return params.SlsStgTwoPerByteGas * uint64(len(input))
	return 0
}

func (c *slotLeaderSC) Run(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	if len(in) < 4 {
		return nil, errParameters
	}

	var methodId [4]byte
	copy(methodId[:], in[:4])

	if methodId == stgOneIdArr {
		return c.handleStgOne(in[4:], contract, evm)
	}else if methodId == stgTwoIdArr {
		return c.handleStgTwo(in[4:], contract, evm)
	}
	return nil, errMethodId
}

func (c *slotLeaderSC) handleStgOne(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	return nil, nil
}


func (c *slotLeaderSC) handleStgTwo(in []byte, contract *Contract, evm *EVM) ([]byte, error) {
	return nil, nil
}

func (c *slotLeaderSC) ValidTx(stateDB StateDB, signer types.Signer, tx *types.Transaction) error {
	// 1. get transaction data
	// 2. parse data to get the Pie[i] and A[i]
	// 3. verify A[i]
	// 4. verify Pie[i]
	return nil
}
