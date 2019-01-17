package slotleader

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/wanchain/go-wanchain/crypto"
)

func TestScarMulti(t *testing.T) {

	bigTemp := new(big.Int).SetInt64(int64(0))
	var publicKey *ecdsa.PublicKey
	privateKey, _ := crypto.GenerateKey()
	publicKey = &privateKey.PublicKey

	var beginTime int64
	var endTime int64

	publicKeyTemp := new(ecdsa.PublicKey)
	publicKeyTemp.Curve = crypto.S256()

	beginTime = time.Now().UnixNano() / 1e6

	fmt.Printf("Begin test ScarlarMult: %v\n", time.Now().UnixNano()/1e6)
	fmt.Printf("Begin test ScarlarMult: %v\n", beginTime)

	bigTwo := big.NewInt(2)
	for i := 1; i < 65535; i++ {
		bigTemp.SetUint64(uint64(i))
		//publicKeyTemp.X, publicKeyTemp.Y = crypto.S256().ScalarMult(publicKey.X, publicKey.Y, bigTemp.Bytes())
		publicKeyTemp.X, publicKeyTemp.Y = crypto.S256().ScalarMult(publicKey.X, publicKey.Y, bigTwo.Bytes())
	}
	endTime = time.Now().UnixNano() / 1e6

	//fmt.Printf("End test ScarlarMult: %v\n",time.Now().UnixNano()/1e6)
	fmt.Printf("End test ScarlarMult: %v\n", endTime)

	fmt.Printf("Avarage time = %v\n", (endTime-beginTime)*1e6/65535)

	fmt.Printf("=======================================\n")

	//fmt.Printf("Begin test ScarlarMult: %v\n",time.Now().UnixNano()/1e6)
	fmt.Printf("Begin test ScarlarMult: %v\n", beginTime)
	var j *big.Int
	j = new(big.Int).Set(privateKey.Curve.Params().N)
	j = j.Div(j, new(big.Int).SetInt64(int64(2)))
	bigOne := new(big.Int).SetInt64(int64(1))
	endJ := new(big.Int).Add(j, big.NewInt(65535))
	fmt.Printf("N is %v\n", privateKey.Curve.Params().N)
	fmt.Println("j", j.String())
	fmt.Println("endJ", endJ.String())
	fmt.Println("bigOne", bigOne.String())

	times := 0
	beginTime = time.Now().UnixNano() / 1e6
	for ; j.Cmp(endJ) < 0; j.Add(j, bigOne) {
		times++
		//bigTemp.SetUint64(uint64(i))
		publicKeyTemp.X, publicKeyTemp.Y = crypto.S256().ScalarMult(publicKey.X, publicKey.Y, j.Bytes())
	}
	endTime = time.Now().UnixNano() / 1e6

	fmt.Println("times:", times)
	//fmt.Printf("End test ScarlarMult: %v\n",time.Now().UnixNano()/1e6)
	fmt.Printf("End test ScarlarMult: %v\n", endTime)

	fmt.Printf("Avarage time = %v\n", (endTime-beginTime)*1e6/65535)

}

func TestMult(t *testing.T) {
	key, _ := crypto.GenerateKey()

	key2, _ := crypto.GenerateKey()

	t1 := time.Now()

	crypto.S256().ScalarMult(key.X, key.Y, key2.D.Bytes())

	elapsed := time.Since(t1)

	fmt.Println("App elapsed: ", elapsed)
	fmt.Println(key2.D.String())

	var j *big.Int
	j = new(big.Int).Set(key.Curve.Params().N)

	fmt.Println(j.String())

	t1 = time.Now()

	for i := 0; i < 100; i++ {
		crypto.S256().ScalarMult(key.X, key.Y, key2.D.Bytes())
	}

	elapsed = time.Since(t1)

	fmt.Println("App elapsed: ", elapsed)
}
