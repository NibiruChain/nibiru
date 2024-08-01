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
	"fmt"
	"strings"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	//go:embed artifacts/contracts/ERC20Minter.sol/ERC20Minter.json
	erc20MinterContractJSON []byte
	//go:embed artifacts/contracts/IFunToken.sol/IFunToken.json
	funtokenContractJSON []byte
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
)

func init() {
	SmartContract_ERC20Minter.MustLoad()
	SmartContract_FunToken.MustLoad()
}

type CompiledEvmContract struct {
	Name      string
	EmbedJSON []byte

	// filled in post-load
	ABI      *gethabi.ABI `json:"abi"`
	Bytecode []byte       `json:"bytecode"`
}

// HexString: Hexadecimal-encoded string
type HexString string

func (h HexString) Bytes() []byte {
	return gethcommon.Hex2Bytes(
		strings.TrimPrefix(string(h), "0x"),
	)
}
func (h HexString) String() string { return string(h) }
func (h HexString) FromBytes(bz []byte) HexString {
	return HexString(gethcommon.Bytes2Hex(bz))
}

func parseCompiledJson(
	jsonBz []byte,
) (abi *gethabi.ABI, bytecode []byte, err error) {
	rawJsonBz := make(map[string]json.RawMessage)
	err = json.Unmarshal(jsonBz, &rawJsonBz)
	if err != nil {
		return nil, nil, err
	}

	newAbi := new(gethabi.ABI)
	err = newAbi.UnmarshalJSON(rawJsonBz["abi"])
	if err != nil {
		return nil, nil, err
	}

	rawBytecodeBz := HexString(rawJsonBz["bytecode"])

	return newAbi, rawBytecodeBz.Bytes(), err
}

func (sc *CompiledEvmContract) MustLoad() {
	err := sc.Load()
	if err != nil {
		panic(err)
	}
}

func (sc CompiledEvmContract) Load() error {
	if sc.EmbedJSON == nil {
		return fmt.Errorf("missing compiled contract embed")
	}

	abi, bytecode, err := parseCompiledJson(sc.EmbedJSON)
	if err != nil {
		return err
	}

	sc.ABI = abi
	sc.Bytecode = bytecode

	return nil
}
