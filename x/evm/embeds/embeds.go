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
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	// Contract_ERC20Minter: The default ERC20 contract deployed during the
	// creation of a `FunToken` mapping from a bank coin.
	Contract_ERC20Minter CompiledEvmContract

	//go:embed ERC20MinterCompiled.json
	erc20MinterContractJSON []byte

	// Contract_Funtoken: Precompile contract interface for
	// "IFunToken.sol". This precompile enables transfers of ERC20 tokens
	// to non-EVM accounts. Only the ABI is used.
	Contract_Funtoken CompiledEvmContract
	//go:embed IFunTokenCompiled.json
	funtokenContractJSON []byte
)

func init() {
	Contract_ERC20Minter = SmartContract_ERC20Minter.MustLoad()
	Contract_Funtoken = SmartContract_FunToken.MustLoad()
}

var (
	SmartContract_TestERC20 = SmartContractFixture{
		Name:        "TestERC20.sol",
		FixtureType: FixtueType_Test,
	}

	SmartContract_ERC20Minter = SmartContractFixture{
		Name:        "ERC20Minter.sol",
		FixtureType: FixtueType_Prod,
		EmbedJSON:   erc20MinterContractJSON,
	}
	SmartContract_FunToken = SmartContractFixture{
		Name:        "FunToken.sol",
		FixtureType: FixtueType_Prod,
		EmbedJSON:   funtokenContractJSON,
	}
)

// CompiledEvmContract: EVM contract that can be deployed into the EVM state and
// used as a valid precompile.
type CompiledEvmContract struct {
	ABI      gethabi.ABI `json:"abi"`
	Bytecode []byte      `json:"bytecode"`
}

type SmartContractFixture struct {
	Name        string
	FixtureType ContractFixtureType
	EmbedJSON   []byte
}

// ContractFixtureType: Enum type for embedded smart contracts. This type
// expresses whether a contract is used in production or only for testing.
type ContractFixtureType string

const (
	FixtueType_Prod = "prod"
	FixtueType_Test = "test"
)

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

func ParseCompiledJson(
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

func (sc SmartContractFixture) MustLoad() (out CompiledEvmContract) {
	out, err := sc.Load()
	if err != nil {
		panic(err)
	}
	return out
}

func (sc SmartContractFixture) Load() (out CompiledEvmContract, err error) {
	var jsonBz []byte

	// Locate the contracts directory.
	switch sc.FixtureType {
	case FixtueType_Prod:
		if sc.EmbedJSON == nil {
			return out, fmt.Errorf("missing compiled contract embed")
		}
		jsonBz = sc.EmbedJSON
	case FixtueType_Test:
		contractsDirPath, err := pathToE2EContracts()
		if err != nil {
			return out, err
		}
		baseName := strings.TrimSuffix(sc.Name, ".sol")
		compiledPath := fmt.Sprintf("%s/%sCompiled.json", contractsDirPath, baseName)

		jsonBz, err = os.ReadFile(compiledPath)
		if err != nil {
			return out, err
		}
	default:
		panic(fmt.Errorf("unexpected case type \"%s\"", sc.FixtureType))
	}

	abi, bytecode, err := ParseCompiledJson(jsonBz)
	if err != nil {
		return
	}

	return CompiledEvmContract{
		ABI:      *abi,
		Bytecode: bytecode,
	}, err
}

// pathToE2EContracts: Returns the absolute path to the E2E test contract
// directory located at path, "NibiruChain/nibiru/e2e/evm/contracts".
func pathToE2EContracts() (thePath string, err error) {
	dirEvmTest, _ := GetPackageDir()
	dirOfRepo := path.Dir(path.Dir(path.Dir(dirEvmTest)))
	dirEvmE2e := path.Join(dirOfRepo, "e2e/evm")
	if path.Base(dirEvmE2e) != "evm" {
		return thePath, fmt.Errorf("failed to locate the e2e/evm directory")
	}
	return dirEvmE2e + "/contracts", nil
}

// GetPackageDir: Returns the absolute path of the Golang package that
// calls this function.
func GetPackageDir() (string, error) {
	// Get the import path of the current package
	_, filename, _, _ := runtime.Caller(0)
	pkgDir := path.Dir(filename)
	pkgPath := path.Join(path.Base(pkgDir), "..")

	// Get the directory path of the package
	return filepath.Abs(pkgPath)
}
