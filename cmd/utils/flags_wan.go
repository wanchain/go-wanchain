package utils
import (
	"gopkg.in/urfave/cli.v1"

)
var (
	TestnetFlag = cli.BoolFlag{
		Name:  "testnet",
		Usage: "Testnet network: pre-configured POS test network",
	}
	PlutoFlag = cli.BoolFlag{
		Name:  "pluto",
		Usage: "Pluto network: pos private network",
	}
)