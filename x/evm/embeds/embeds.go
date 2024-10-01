// Package "embeds" adds access to files (smart contracts) embedded in the Go
// runtime. Go source files that import "embed" can use the //go:embed directive
// to initialize a variable of type string, \[]byte, or \[FS] with the contents
// of files read from the package directory or subdirectories at compile time.
package embeds

import (
	// The `_ "embed"` import adds access to files embedded in the running Go
	// program (smart contracts).
	_ "embed"
	"encoding/json"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	//go:embed artifacts/contracts/ERC20Minter.sol/ERC20Minter.json
	erc20MinterContractJSON []byte
	//go:embed artifacts/contracts/IFunToken.sol/IFunToken.json
	funtokenContractJSON []byte
	//go:embed artifacts/contracts/IOracle.sol/IOracle.json
	oracleContractJSON []byte
	//go:embed artifacts/contracts/TestERC20.sol/TestERC20.json
	testErc20Json []byte
)

var (
	// Contract_ERC20Minter: The default ERC20 contract deployed during the
	// creation of a `FunToken` mapping from a bank coin.
	SmartContract_ERC20Minter = CompiledEvmContract{
		Name:      "ERC20Minter.sol",
		EmbedJSON: erc20MinterContractJSON,
	}

	// SmartContract_Funtoken: Precompile contract interface for
	// "IFunToken.sol". This precompile enables transfers of ERC20 tokens
	// to non-EVM accounts. Only the ABI is used.
	SmartContract_FunToken = CompiledEvmContract{
		Name:      "FunToken.sol",
		EmbedJSON: funtokenContractJSON,
	}
	SmartContract_Oracle = CompiledEvmContract{
		Name:      "Oracle.sol",
		EmbedJSON: oracleContractJSON,
	}
	SmartContract_TestERC20 = CompiledEvmContract{
		Name:      "TestERC20.sol",
		EmbedJSON: testErc20Json,
	}
)

func init() {
	SmartContract_ERC20Minter.MustLoad()
	SmartContract_FunToken.MustLoad()
	SmartContract_Oracle.MustLoad()
	SmartContract_TestERC20.MustLoad()
}

type CompiledEvmContract struct {
	Name      string
	EmbedJSON []byte

	// filled in post-load
	ABI      *gethabi.ABI `json:"abi"`
	Bytecode []byte       `json:"bytecode"`
}

func (sc *CompiledEvmContract) MustLoad() {
	if sc.EmbedJSON == nil {
		panic("missing compiled contract embed")
	}

	rawJsonBz := make(map[string]json.RawMessage)
	err := json.Unmarshal(sc.EmbedJSON, &rawJsonBz)
	if err != nil {
		panic(err)
	}

	abi := new(gethabi.ABI)
	err = abi.UnmarshalJSON(rawJsonBz["abi"])
	if err != nil {
		panic(err)
	}

	var bytecodeStr string
	err = json.Unmarshal(rawJsonBz["bytecode"], &bytecodeStr)
	if err != nil {
		panic(err)
	}
	sc.Bytecode = gethcommon.FromHex(bytecodeStr)
	sc.ABI = abi
}
