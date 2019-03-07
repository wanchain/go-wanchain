package postools

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/go-wanchain/log"
	"github.com/wanchain/go-wanchain/pos"
)

func CalEpochSlotID(time uint64) (epochId, slotId uint64) {
	if pos.EpochBaseTime == 0 {
		return
	}
	//timeUnix := uint64(time.Now().Unix())
	timeUnix := time
	epochTimespan := uint64(pos.SlotTime * pos.SlotCount)
	epochId = uint64((timeUnix - pos.EpochBaseTime) / epochTimespan)
	slotId = uint64((timeUnix - pos.EpochBaseTime) / pos.SlotTime % pos.SlotCount)
	fmt.Println("CalEpochSlotID:", epochId, slotId)
	return epochId, slotId
}

//PkEqual only can use in same curve. return whether the two points equal
func PkEqual(pk1, pk2 *ecdsa.PublicKey) bool {
	if pk1 == nil || pk2 == nil {
		return false
	}

	if hex.EncodeToString(pk1.X.Bytes()) == hex.EncodeToString(pk2.X.Bytes()) &&
		hex.EncodeToString(pk1.Y.Bytes()) == hex.EncodeToString(pk2.Y.Bytes()) {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Uint64ToBytes use a big.Int to transfer uint64 to bytes
// Must use big.Int to reverse
func Uint64ToBytes(input uint64) []byte {
	if input == 0 {
		return []byte{0}
	}
	return big.NewInt(0).SetUint64(input).Bytes()
}

// BytesToUint64 use a big.Int to transfer uint64 to bytes
// Must input a big.Int bytes
func BytesToUint64(input []byte) uint64 {
	return big.NewInt(0).SetBytes(input).Uint64()
}

// Uint64ToString can change uint64 to string through a big.Int, output is a 10 base number
func Uint64ToString(input uint64) string {
	str := big.NewInt(0).SetUint64(input).String()
	if len(str) == 0 {
		str = "00"
	}
	return str
}

// Uint64StringToByte can change uint64  string to bytes through a big.Int, Input must be a 10 base number
func Uint64StringToByte(input string) []byte {
	num, ok := big.NewInt(0).SetString(input, 10)
	if !ok {
		return []byte{0}
	}

	if len(num.Bytes()) == 0 {
		return []byte{0}
	}

	return num.Bytes()
}

// StringToUint64 can change string to uint64 through a big.Int, Input must be a 10 base number
func StringToUint64(input string) uint64 {
	num, ok := big.NewInt(0).SetString(input, 10)
	if !ok {
		log.Error("StringToUint64 only support 10 base number string", "input", input)
		return 0
	}
	return num.Uint64()
}

// BigIntArrayToByteArray can change []*big.Int to [][]byte
func BigIntArrayToByteArray(input []*big.Int) [][]byte {
	ret := make([][]byte, len(input))
	for i := 0; i < len(input); i++ {
		ret[i] = input[i].Bytes()
	}
	return ret
}

// ByteArrayToBigIntArray can change [][]byte to big.Int
func ByteArrayToBigIntArray(input [][]byte) []*big.Int {
	ret := make([]*big.Int, len(input))
	for i := 0; i < len(input); i++ {
		ret[i] = big.NewInt(0).SetBytes(input[i])
	}
	return ret
}

// PkArrayToByteArray can change []*ecdsa.PublicKey to [][]byte
func PkArrayToByteArray(input []*ecdsa.PublicKey) [][]byte {
	ret := make([][]byte, len(input))
	for i := 0; i < len(input); i++ {
		ret[i] = crypto.FromECDSAPub(input[i])
	}
	return ret
}

// ByteArrayToPkArray can change [][]byte to []*ecdsa.PublicKey
func ByteArrayToPkArray(input [][]byte) []*ecdsa.PublicKey {
	ret := make([]*ecdsa.PublicKey, len(input))
	for i := 0; i < len(input); i++ {
		ret[i] = crypto.ToECDSAPub(input[i])
	}
	return ret
}
