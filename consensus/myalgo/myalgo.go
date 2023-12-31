package myalgo

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/Knetic/govaluate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"sync"
)

type Problem struct {
	Index    int    `json:"index"`
	Equation string `json:"equation"`
}

func getProblems() []Problem {

	raw, err := ioutil.ReadFile("./problems.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var c []Problem
	json.Unmarshal(raw, &c)
	return c
}

func (p Problem) toString() string {
	return toJson(p)
}

func toJson(p interface{}) string {
	bytes, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return string(bytes)
}

var problems []Problem

// New creates a Clique proof-of-authority consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.MyAlgoConfig, db ethdb.Database) *MyAlgo {
	// Set any missing consensus parameters to their defaults



	conf := *config
	problems = getProblems()
	return &MyAlgo{
		config:     &conf,
		db:         db,
	}
}



// Clique is the proof-of-authority consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type MyAlgo struct {
	config *params.MyAlgoConfig // Consensus engine configuration parameters
	db     ethdb.Database       // Database to store and retrieve snapshot checkpoints
	lock   sync.RWMutex   // Protects the signer fields
}


// Author retrieves the Ethereum address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
func (MyAlgo *MyAlgo) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}
// VerifyHeader checks whether a header conforms to the consensus rules of a
// given engine. Verifying the seal may be done optionally here, or explicitly
// via the VerifySeal method.
func (MyAlgo *MyAlgo) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	log.Info("will verfiyHeader")
	hasher:=sha256.New()
	runes:=[]byte(header.ParentHash.String())
	hasher.Write(runes)
	runes1:=[]byte(header.Root.String())
	hasher.Write(runes1)

	runes3:=[]byte(header.TxHash.String())
	hasher.Write(runes3)
	runes4:=[]byte(header.Number.String())
	hasher.Write(runes4)
	hash := hasher.Sum(nil)






	var ab common.Hash = common.BytesToHash(hash)
	a:=ab.String()
	b:=header.MixDigest.String()
	if (a!=b){
		return nil
	}else {
		return errors.New("Invalid solution to the problem ")
	}



	p, _ := getProblemFromHeader(header)
	result := solveProblem(p);
	correct := checkResult(result, header)
	if (correct){
		return nil
	}else {
		return errors.New("Invalid solution to the problem ")
	}
}

func checkResult(result float64, header *types.Header) bool {
	fmt.Print("result : ")
	fmt.Println(result)

	fmt.Print("to compare with  : ")
	fmt.Println(header.Nonce.Uint64())
	toCompare := header.Nonce.Uint64()
	return toCompare == uint64(result);

}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications (the order is that of
// the input slice).
func (MyAlgo *MyAlgo) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error){
	log.Info("will verfiyHeaders")
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for _, header := range headers {
			err := MyAlgo.VerifyHeader(chain, header, false)

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of a given engine.
func (MyAlgo *MyAlgo) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	log.Info("will verfiy uncles")
	return nil
}

// VerifySeal checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (MyAlgo *MyAlgo)  VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error{
	log.Info("will verfiy VerifySeal")
	return nil
}

// Prepare initializes the consensus fields of a block header according to the
// rules of a particular engine. The changes are executed inline.
func (MyAlgo *MyAlgo) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error{
	log.Info("will prepare the block")
	header.Coinbase = common.Address{}
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = MyAlgo.CalcDifficulty(chain, header.Time, parent)
	//changed to this
	//header.Difficulty = MyAlgo.CalcDifficulty(chain, real(header.Time), parent)
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficult
// that a new block should have.
func (MyAlgo *MyAlgo) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	//return calcDifficultyFrontier(time, parent)
	return calcDifficultyHomestead(time, parent)
}

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)
	big2999999    = big.NewInt(2999999)
)

// calcDifficultyHomestead is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyHomestead(time uint64, parent *types.Header) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	//changed this
	//bigParentTime := new(big.Int).Set(parent.Time)

	var bigParentTime  = new(big.Int).SetInt64(int64(parent.Time)) // holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)
	//till here
	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}



/*
	var status string = "p"
	var arr []byte = []byte(status)
	fmt.Printf("array: %v (%T)\n", arr, arr)
	fmt.Println(string(arr[:]))
*/
// Finalize runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
// Note: The block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
func (MyAlgo *MyAlgo) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header){
	log.Info("will Finalize the block and finalize")
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	//b := types.NewBlock(header, txs, uncles,new(trie.Trie))


}

func (MyAlgo *MyAlgo) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// Finalize block
	MyAlgo.Finalize(chain, header, state, txs, uncles)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts, new(trie.Trie)), nil
}


func getProblemFromHeader (header *types.Header) (Problem, int64){
	runes := []rune(header.Difficulty.String())
	index_in_hash := string(runes[0:3])
	index_in_decimal, _ := strconv.ParseInt(index_in_hash , 0, 64)
	index_in_decimal = index_in_decimal % 10
	return problems[index_in_decimal], index_in_decimal
}

func solveProblem(p Problem) (float64){
	expression, _ := govaluate.NewEvaluableExpression(p.Equation)
	result, _ := expression.Evaluate(nil)
	result_in_float := result.(float64)
	return result_in_float
}



// Seal generates a new block for the given input block with the local miner's
// seal place on top.
func (MyAlgo *MyAlgo) Seal(chain consensus.ChainHeaderReader, block *types.Block,results chan<- *types.Block ,stop <-chan struct{}) ( error){
	log.Info("will Seal the block brother")
	//time.Sleep(15 * time.Second)
	header := block.Header()
	/*
		runes := []rune(header.ParentHash.String())
		index_in_hash := string(runes[0:3])
		index_in_decimal, _ := strconv.ParseInt(index_in_hash , 0, 64)
		index_in_decimal = index_in_decimal % 10
	*/
	p, index_in_decimal := getProblemFromHeader(header)

	fmt.Print("hash is : ")
	fmt.Print(header.ParentHash.String())
	fmt.Print("problem number is : ")
	fmt.Println(index_in_decimal)



	fmt.Print("problem is : ")
	fmt.Println(p.Equation)
	result_in_float := solveProblem(p)
	fmt.Print("solution is : ")
	fmt.Println(result_in_float)
	//problem_number := header.Hash().String()[2]

	/*
		fmt.Print("hash is : ")
		fmt.Print(header.Hash().String())
		fmt.Print("problem number is : ")
		fmt.Println(index_in_decimal)
		for _, p := range problems {
			fmt.Print("problem is : ")
			fmt.Println(p.Equation)
			expression, _ := govaluate.NewEvaluableExpression(p.Equation);
			result, _ := expression.Evaluate(nil);
			fmt.Print("solution is : ")
			fmt.Println(result)
		}
	*/

	//MyAlgo.SealHash(header)
	//results<- block.WithSeal(header)


	//default:
	//log.Warn("Sealing result is not read by miner", "sealhash", MyAlgo.SealHash(header))






	hasher := sha256.New()


	runes:=[]byte(header.ParentHash.String())
	hasher.Write(runes)
	runes1:=[]byte(header.Root.String())
	hasher.Write(runes1)

	runes3:=[]byte(header.TxHash.String())
	hasher.Write(runes3)
	runes4:=[]byte(header.Number.String())
	hasher.Write(runes4)
	hash := hasher.Sum(nil)






	header.MixDigest = common.BytesToHash(hash)


	header.Nonce = getRequiredHeader(result_in_float)
	//log.Info(fmt.Println("header.MixDigest"))
	results <- block.WithSeal(header)




	return  nil
}



func (MyAlgo *MyAlgo) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
	})
	hasher.Sum(hash[:0])
	return hash
}





func (c *MyAlgo) Close() error {
	return nil
}


func getRequiredHeader(result float64) (types.BlockNonce){
	return getNonce(result)
}

func getNonce(result float64) (types.BlockNonce) {
	var i uint64 = uint64(result)
	var n types.BlockNonce

	binary.BigEndian.PutUint64(n[:], i)
	return n
}



func rangeIn(low, hi int) int {

	return low + rand.Intn(hi-low)
}

// APIs returns the RPC APIs this consensus engine provides.
func (myAlgo *MyAlgo) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "myalgo",
		Version:   "1.0",
		Service:   &API{chain: chain, myAlgo: myAlgo},
		Public:    false,
	}}
}