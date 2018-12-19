package vm

import (
	"github.com/wanchain/go-wanchain/common"
	"errors"
	"math/big"
	"crypto/ecdsa"
	"bytes"
	"math"
	"github.com/wanchain/go-wanchain/crypto"
	"github.com/wanchain/pos/cloudflare"
	"github.com/wanchain/go-wanchain/params"
	"time"
	"encoding/json"
	"sort"
)

//store information to the list
func StoreInfo(statedb StateDB,listAddr common.Address,pubHash common.Hash, info []byte) error {
	if statedb == nil {
		return ErrUnknown
	}

	if pubHash == (common.Hash{})  {
		return errors.New("public key hash is not right")
	}

	statedb.SetStateByteArray(listAddr,pubHash, info)
	return nil
}

//get stored info
func GetInfo(statedb StateDB,listAddr common.Address,pubHash common.Hash) ([]byte, error) {

	if statedb == nil {
		return nil, ErrUnknown
	}

	if pubHash == (common.Hash{})  {
		return nil,errors.New("public key hash is not right")
	}

	info := statedb.GetStateByteArray(listAddr, pubHash)
	if len(info) == 0 {
		return nil, errors.New("not get data")
	}

	return info, nil
}

type Proposer struct {
	pubSec256 		    *ecdsa.PublicKey  	//staker’s ethereum public key
	pubBn256 		    *bn256.G1  			//staker’s bn256 public key
	probabilities		*big.Int
}

type ProposerSorter []Proposer

func NewProposerSorter() ProposerSorter {
	ps := make(ProposerSorter, 0)
	return ps
}

//Len()
func (s ProposerSorter) Len() int {
	return len(s)
}

//Less():成绩将有低到高排序
func (s ProposerSorter) Less(i, j int) bool {
	return s[i].probabilities.Cmp(s[j].probabilities) < 0
}

//Swap()
func (s ProposerSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

const Accuracy float64 = 1024.0 //accuracy to magnificate
//wanhumber*locktime*(exp-(t) ),t=(locktime - passedtime/locktime)
func GenerateProblility(pstaker *stakerInfo) (*Proposer,error) {

	amount := big.NewInt(0).Div(pstaker.amount,big.NewInt(params.Wan)).Int64()
	lockTime := pstaker.lockPeriod
	leftTimePercent := (float64(lockTime - (time.Now().Unix() - pstaker.stakingTime))/float64(lockTime))
	pb := float64(amount)*float64(lockTime)*math.Exp(-leftTimePercent)*Accuracy


	gb := new(bn256.G1)
	_, err := gb.Unmarshal(pstaker.pubBn256)
	if err != nil {
		return  nil,err
	}


	p := &Proposer{
					pubSec256: crypto.ToECDSAPub(pstaker.pubSec256),
					pubBn256:  gb,
					probabilities:big.NewInt(int64(pb)),
	              }

	return p,nil

}

func createStakerProbabilityArray(statedb StateDB,listAddr common.Address) (ProposerSorter,*big.Int,error) {
	if statedb == nil {
		return nil,nil,ErrUnknown
	}

	ps := NewProposerSorter()

	statedb.ForEachStorageByteArray(listAddr, func(key common.Hash, value []byte) bool {

			staker := stakerInfo{}
			err :=json.Unmarshal(value,&staker)
			if(err != nil) {
				return false
			}

			pitem,err := GenerateProblility(&staker)
			if(err != nil) {
				return false
			}

			ps = append(ps,*pitem)

			return true
	})

	sort.Sort(ps)

	for idx, item := range ps {
		if (idx == 0) {
			continue
		}

		ps[idx].probabilities = big.NewInt(0).Add(item.probabilities,ps[idx-1].probabilities)
	}

	return ps,nil,nil
}

//getEpochLeaders get epochLeaders of epochID in StateDB
func  getEpochLeaders(epochID uint64) []*ecdsa.PublicKey {
	return nil
}

func GetRBProposerGroup(epochId uint64) []bn256.G1 {
	return nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//Parameters
var (
	Big1                                   = big.NewInt(1)
	Big0                                   = big.NewInt(0)
	ErrInvalidRandomProposerSelection      = errors.New("Invalid Random Proposer Selection")                  //Invalid Random Proposer Selection
	ErrInvalidProbabilityfloat2big         = errors.New("Invalid Transform Probability From Float To Bigint") //Invalid Transform Probability From Float To Bigint
	ErrInvalidGenerateCommitment           = errors.New("Invalid Commitment Generation")                      //Invalid Commitment Generation
	ErrInvalidArrayPieceGeneration         = errors.New("Invalid ArrayPiece Generation")                      //Invalid ArrayPiece Generation
	ErrInvalidDleqProofGeneration          = errors.New("Invalid DLEQ Proof Generation")                      //Invalid DLEQ Proof Generation
	ErrInvalidSecretMessageArrayGeneration = errors.New("Invalid Secret Message Array Generation")            //Invalid Secret Message Array Generation
	ErrInvalidSortPublicKeys               = errors.New("Invalid PublicKeys Sort Operation")                  //Invalid PublicKeys Sort Operation
	ErrInvalidSlotLeaderSequenceGeneration = errors.New("Invalid Slot Leader Sequence Generation")            //Invalid Slot Leader Sequence Generation
	ErrInvalidSlotLeaderLocation           = errors.New("Invalid Slot Leader Location")                       //Invalid Slot Leader Location
	ErrInvalidSlotLeaderProofGeneration    = errors.New("Invalid Slot Leader Proof Generation")               //Invalid Slot Leader Proof Generation
)


/*_________________________________________________Random proposer + Epoch leader selection___________________________________________________*/

//Transform Probabilities from float to bigint
func ProbabilityFloat2big(Probabilities []*float64) ([]*big.Int, error) {
	if len(Probabilities) == 0 {
		return nil, ErrInvalidProbabilityfloat2big
	}
	for _, probability := range Probabilities {
		if probability == nil {
			return nil, ErrInvalidProbabilityfloat2big
		}
	}

	n := len(Probabilities)
	var temp int64
	var probabilitiesBig = make([]*big.Int, n) //probabilities_big as new probability array

	for i := 0; i < n; i++ {
		temp = int64(*Probabilities[i] * Accuracy)
		probabilitiesBig[i] = big.NewInt(temp)
	}

	return probabilitiesBig, nil
}

//samples nr random proposers by random number r（Random Beacon) from PublicKeys based on proportion of Probabilities
func RandomProposerSelection(r []byte, nr int, PublicKeys []*ecdsa.PublicKey, Probabilities []*float64) ([]*ecdsa.PublicKey, error) {
	if r == nil || nr <= 0 || len(PublicKeys) == 0 || len(Probabilities) == 0 || len(PublicKeys) != len(Probabilities) {
		return nil, ErrInvalidRandomProposerSelection
	}
	for _, publicKey := range PublicKeys {
		if publicKey == nil || publicKey.X == nil || publicKey.Y == nil {
			return nil, ErrInvalidRandomProposerSelection
		}
	}
	for _, probability := range Probabilities {
		if probability == nil {
			return nil, ErrInvalidRandomProposerSelection
		}
	}

	probabilitiesBig, _ := ProbabilityFloat2big(Probabilities) //transform probabilities from float64 to bigint
	tp := new(big.Int).SetInt64(0)                             //total probability of probabilities_big
	randomProposerPublicKeys := make([]*ecdsa.PublicKey, 0)    //store the selected publickeys
	n := len(probabilitiesBig)

	for _, probabilityBig := range probabilitiesBig {
		tp.Add(tp, probabilityBig)
	}

	var Byte0 = []byte{byte(0)}
	var buffer bytes.Buffer
	buffer.Write(Byte0)
	buffer.Write(r)
	r0 := buffer.Bytes()       //r0 = 0||r
	cr := crypto.Keccak256(r0) //cr = hash(r0)

	for i := 0; i < nr; i++ {

		crBig := new(big.Int).SetBytes(cr)
		crBig.Mod(crBig, tp) //cr_big = cr mod tp

		//select pki whose probability bigger than cr_big left
		sumtemp := new(big.Int).SetInt64(0)
		for j := 0; j < n; j++ {
			sumtemp.Add(sumtemp, probabilitiesBig[j])
			if sumtemp.Cmp(crBig) == 1 {
				pkselected := new(ecdsa.PublicKey) //new publickey to store the selected one
				pkselected.Curve = crypto.S256()
				pkselected.X = new(big.Int).Set(PublicKeys[j].X)
				pkselected.Y = new(big.Int).Set(PublicKeys[j].Y)
				randomProposerPublicKeys = append(randomProposerPublicKeys, pkselected)
				break
			}
		}
		cr = crypto.Keccak256(cr)
	}

	return randomProposerPublicKeys, nil
}

//samples ne epoch leaders by random number r from PublicKeys based on proportion of Probabilities
func EpochLeaderSelection(r []byte, ne int, PublicKeys []*ecdsa.PublicKey, Probabilities []*float64) ([]*ecdsa.PublicKey, error) {
	if r == nil || ne <= 0 || len(PublicKeys) == 0 || len(Probabilities) == 0 || len(PublicKeys) != len(Probabilities) {
		return nil, ErrInvalidRandomProposerSelection
	}
	for _, publicKey := range PublicKeys {
		if publicKey == nil || publicKey.X == nil || publicKey.Y == nil {
			return nil, ErrInvalidRandomProposerSelection
		}
	}
	for _, probability := range Probabilities {
		if probability == nil {
			return nil, ErrInvalidRandomProposerSelection
		}
	}

	probabilitiesBig, _ := ProbabilityFloat2big(Probabilities) //transform probabilities from float64 to bigint
	tp := new(big.Int).SetInt64(0)                             //total probability of probabilities_big
	EpochLeaderPublicKeys := make([]*ecdsa.PublicKey, 0)       //store the selected publickeys
	n := len(probabilitiesBig)

	for _, probabilityBig := range probabilitiesBig {
		tp.Add(tp, probabilityBig)
	}

	var Byte1 = []byte{byte(1)}
	var buffer bytes.Buffer
	buffer.Write(Byte1)
	buffer.Write(r)
	r1 := buffer.Bytes()       //r1 = 1||r
	cr := crypto.Keccak256(r1) //cr = hash(r1)

	for i := 0; i < ne; i++ {

		crBig := new(big.Int).SetBytes(cr)
		crBig.Mod(crBig, tp) //cr_big = cr mod tp

		//select pki whose probability bigger than cr_big left
		sumtemp := new(big.Int).SetInt64(0)
		for j := 0; j < n; j++ {
			sumtemp.Add(sumtemp, probabilitiesBig[j])
			if sumtemp.Cmp(crBig) == 1 {
				pkselected := new(ecdsa.PublicKey) //new publickey to store the selected one
				pkselected.Curve = crypto.S256()
				pkselected.X = new(big.Int).Set(PublicKeys[j].X)
				pkselected.Y = new(big.Int).Set(PublicKeys[j].Y)
				EpochLeaderPublicKeys = append(EpochLeaderPublicKeys, pkselected)
				break
			}
		}
		cr = crypto.Keccak256(cr)
	}

	return EpochLeaderPublicKeys, nil
}
