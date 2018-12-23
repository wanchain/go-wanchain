package vm

import (
	"errors"
	"github.com/wanchain/go-wanchain/accounts/abi"
	"strings"
	"testing"
)


func TestRBDkg(t *testing.T) {

}

type genRParam struct {
	EpochId uint32
}

type sigshareParam struct {
	EpochId uint32
	Gsigs 	[]uint32
}

type dkgParam struct {
	EpochId uint32
	Pks []uint32
	EnSs []uint32
	Cs []uint32
	Proofs []uint32
}

var (
	ErrUnknown          = errors.New("unknown error")
)

func TestPackDate(t *testing.T) {
}

func TestUnpackData(t *testing.T) {

}