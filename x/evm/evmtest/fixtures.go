package evmtest

import (
	"path"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/testutil"
)

type SmartContractFixture string

const (
	SmartContract_FunToken SmartContractFixture = "FunToken.sol"
)

type CompiledEvmContract struct {
	ABI      gethabi.ABI `json:"abi"`
	Bytecode []byte      `json:"bytecode"`
}

// HardhatOutput: Expected format for smart contract test fixtures.
type HardhatOutput struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode HexString       `json:"bytecode"`
}

// HexString: Hexadecimal-encoded string
type HexString string

func (h HexString) Bytes() []byte  { return gethcommon.Hex2Bytes(string(h)) }
func (h HexString) String() string { return string(h) }
func (h HexString) FromBytes(bz []byte) HexString {
	return HexString(gethcommon.Bytes2Hex(bz))
}

func NewHardhatOutputFromJson(
	t *testing.T,
	jsonBz []byte,
) HardhatOutput {
	rawJsonBz := make(map[string]json.RawMessage)
	err := json.Unmarshal(jsonBz, &rawJsonBz)
	require.NoError(t, err)
	return HardhatOutput{
		ABI:      rawJsonBz["abi"],
		Bytecode: HexString(rawJsonBz["bytecode"]),
	}
}

func (jsonObj HardhatOutput) EvmContract(t *testing.T) CompiledEvmContract {
	newAbi := new(gethabi.ABI)
	err := newAbi.UnmarshalJSON(jsonObj.ABI)
	require.NoError(t, err)

	contract := new(CompiledEvmContract)
	return CompiledEvmContract{
		ABI:      *newAbi,
		Bytecode: contract.Bytecode,
	}
}

func (sc SmartContractFixture) Load(t *testing.T) CompiledEvmContract {
	contractsDirPath := pathToContractsDir(t)
	baseName := strings.TrimSuffix(string(sc), ".sol")
	compiledPath := fmt.Sprintf("%s/%sCompiled.json", contractsDirPath, baseName)

	jsonBz, err := os.ReadFile(compiledPath)
	require.NoError(t, err)

	compiledJson := NewHardhatOutputFromJson(t, jsonBz)
	require.NoError(t, err)
	return compiledJson.EvmContract(t)
}

// pathToContractsDir: Returns the absolute path to the E2E test contract
// directory located at path, "NibiruChain/nibiru/e2e/evm/contracts".
func pathToContractsDir(t *testing.T) string {
	dirEvmTest, err := testutil.GetPackageDir()
	require.NoError(t, err)
	dirOfRepo := path.Dir(path.Dir(path.Dir(dirEvmTest)))
	dirEvmE2e := path.Join(dirOfRepo, "e2e/evm")
	require.Equal(t, "evm", path.Base(dirEvmE2e))
	return dirEvmE2e + "/contracts"
}
