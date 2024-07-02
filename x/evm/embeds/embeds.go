// Package "embeds" adds access to files (smart contracts) embedded in the Go
// runtime. Go source files that import "embed" can use the //go:embed directive
// to initialize a variable of type string, \[]byte, or \[FS] with the contents
// of files read from the package directory or subdirectories at compile time.
package embeds

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	// Adds access to files (smart contracts, in this case) embedded in the Go

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	//go:embed ERC20MinterCompiled.json
	erc20MinterContractJSON []byte

	EmbeddedContractERC20Minter CompiledEvmContract
)

func init() {
	out, err := SmartContract_ERC20Minter.Load()
	if err != nil {
		panic(err)
	}
	EmbeddedContractERC20Minter = out
}

var (
	SmartContract_FunToken = SmartContractFixture{
		Name:        "FunToken.sol",
		FixtureType: FixtueType_Test,
	}

	SmartContract_ERC20Minter = SmartContractFixture{
		Name:        "ERC20Minter.sol",
		FixtureType: FixtueType_Embed,
		EmbedJSON:   &erc20MinterContractJSON,
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
	EmbedJSON   *[]byte
}

type ContractFixtureType string

const (
	FixtueType_Embed = "embed"
	FixtueType_Test  = "test"
)

// HardhatOutput: Expected format for smart contract test fixtures.
type HardhatOutput struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode HexString       `json:"bytecode"`
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

func NewHardhatOutputFromJson(
	jsonBz []byte,
) (out HardhatOutput, err error) {
	rawJsonBz := make(map[string]json.RawMessage)
	err = json.Unmarshal(jsonBz, &rawJsonBz)
	if err != nil {
		return
	}
	var rawBytecodeBz HexString
	err = json.Unmarshal(rawJsonBz["bytecode"], &rawBytecodeBz)
	if err != nil {
		return
	}

	return HardhatOutput{
		ABI:      rawJsonBz["abi"],
		Bytecode: rawBytecodeBz,
	}, err
}

func (jsonObj HardhatOutput) EvmContract() (out CompiledEvmContract, err error) {
	newAbi := new(gethabi.ABI)
	err = newAbi.UnmarshalJSON(jsonObj.ABI)
	if err != nil {
		return
	}

	return CompiledEvmContract{
		ABI:      *newAbi,
		Bytecode: jsonObj.Bytecode.Bytes(),
	}, err
}

func (sc SmartContractFixture) Load() (out CompiledEvmContract, err error) {
	var jsonBz []byte

	// Locate the contracts directory.
	switch sc.FixtureType {
	case FixtueType_Embed:
		if sc.EmbedJSON == nil {
			return out, fmt.Errorf("missing compiled contract embed")
		}
		jsonBz = *sc.EmbedJSON
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

	compiledJson, err := NewHardhatOutputFromJson(jsonBz)
	if err != nil {
		return
	}

	return compiledJson.EvmContract()
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
