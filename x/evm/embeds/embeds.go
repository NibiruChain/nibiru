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
	//go:embed artifacts/contracts/IOracle.sol/IOracle.json
	oracleContractJSON []byte
	//go:embed artifacts/contracts/IFunToken.sol/IFunToken.json
	funtokenPrecompileJSON []byte
	//go:embed artifacts/contracts/Wasm.sol/IWasm.json
	wasmPrecompileJSON []byte
	//go:embed artifacts/contracts/TestERC20.sol/TestERC20.json
	testErc20Json []byte
	//go:embed artifacts/contracts/TestERC20MaliciousName.sol/TestERC20MaliciousName.json
	testErc20MaliciousNameJson []byte
	//go:embed artifacts/contracts/TestERC20MaliciousTransfer.sol/TestERC20MaliciousTransfer.json
	testErc20MaliciousTransferJson []byte
	//go:embed artifacts/contracts/TestFunTokenPrecompileLocalGas.sol/TestFunTokenPrecompileLocalGas.json
	testFunTokenPrecompileLocalGasJson []byte
	//go:embed artifacts/contracts/TestERC20TransferThenPrecompileSend.sol/TestERC20TransferThenPrecompileSend.json
	testERC20TransferThenPrecompileSendJson []byte
	//go:embed artifacts/contracts/TestNativeSendThenPrecompileSend.sol/TestNativeSendThenPrecompileSend.json
	testNativeSendThenPrecompileSendJson []byte
	//go:embed artifacts/contracts/TestPrecompileSelfCallRevert.sol/TestPrecompileSelfCallRevert.json
	testPrecompileSelfCallRevertJson []byte
	//go:embed artifacts/contracts/TestInfiniteRecursionERC20.sol/TestInfiniteRecursionERC20.json
	testInfiniteRecursionERC20Json []byte
	//go:embed artifacts/contracts/TestERC20TransferWithFee.sol/TestERC20TransferWithFee.json
	testERC20TransferWithFee []byte
	//go:embed artifacts/contracts/TestRandom.sol/TestRandom.json
	testRandom []byte
	//go:embed artifacts/contracts/MKR.sol/DSToken.json
	testMetadataBytes32 []byte
	//go:embed artifacts/contracts/TestPrecompileSendToBankThenERC20Transfer.sol/TestPrecompileSendToBankThenERC20Transfer.json
	testPrecompileSendToBankThenERC20Transfer []byte
	//go:embed artifacts/contracts/TestDirtyStateAttack4.sol/TestDirtyStateAttack4.json
	testDirtyStateAttack4 []byte
	//go:embed artifacts/contracts/TestDirtyStateAttack5.sol/TestDirtyStateAttack5.json
	testDirtyStateAttack5 []byte
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
		Name:      "IFunToken.sol",
		EmbedJSON: funtokenPrecompileJSON,
	}

	// SmartContract_Funtoken: Precompile contract interface for
	// "Wasm.sol". This precompile enables contract invocations in the Wasm VM
	// from EVM accounts. Only the ABI is used.
	SmartContract_Wasm = CompiledEvmContract{
		Name:      "Wasm.sol",
		EmbedJSON: wasmPrecompileJSON,
	}
	SmartContract_Oracle = CompiledEvmContract{
		Name:      "Oracle.sol",
		EmbedJSON: oracleContractJSON,
	}
	SmartContract_TestERC20 = CompiledEvmContract{
		Name:      "TestERC20.sol",
		EmbedJSON: testErc20Json,
	}
	// SmartContract_TestERC20MaliciousName is a test contract
	// which simulates malicious ERC20 behavior by adding gas intensive operation
	// for function name() intended to attack funtoken creation
	SmartContract_TestERC20MaliciousName = CompiledEvmContract{
		Name:      "TestERC20MaliciousName.sol",
		EmbedJSON: testErc20MaliciousNameJson,
	}
	// SmartContract_TestERC20MaliciousTransfer is a test contract
	// which simulates malicious ERC20 behavior by adding gas intensive operation
	// for function transfer() intended to attack funtoken conversion from erc20 to bank coin
	SmartContract_TestERC20MaliciousTransfer = CompiledEvmContract{
		Name:      "TestERC20MaliciousTransfer.sol",
		EmbedJSON: testErc20MaliciousTransferJson,
	}
	// SmartContract_TestFunTokenPrecompileLocalGas is a test contract
	// which allows precompile execution with custom local gas set (calling precompile within contract)
	SmartContract_TestFunTokenPrecompileLocalGas = CompiledEvmContract{
		Name:      "TestFunTokenPrecompileLocalGas.sol",
		EmbedJSON: testFunTokenPrecompileLocalGasJson,
	}
	// SmartContract_TestNativeSendThenPrecompileSendJson is a test contract that
	// performs two sends in a single call: a native nibi send and a precompile
	// sendToBank. It tests a race condition where the state DB commit may
	// overwrite the state after the precompile execution, potentially causing a
	// loss of funds.
	SmartContract_TestNativeSendThenPrecompileSendJson = CompiledEvmContract{
		Name:      "TestNativeSendThenPrecompileSend.sol",
		EmbedJSON: testNativeSendThenPrecompileSendJson,
	}
	// SmartContract_TestERC20TransferThenPrecompileSend is a test contract that
	// performs two sends in a single call: an erc20 token transfer and a
	// precompile sendToBank. It tests a race condition where the state DB commit
	// may overwrite the state after the precompile execution, potentially
	// causing an infinite token mint.
	SmartContract_TestERC20TransferThenPrecompileSend = CompiledEvmContract{
		Name:      "TestERC20TransferThenPrecompileSend.sol",
		EmbedJSON: testERC20TransferThenPrecompileSendJson,
	}
	// SmartContract_TestPrecompileSelfCallRevert is a test contract
	// that creates another instance of itself, calls the precompile method and then force reverts.
	// It tests a race condition where the state DB commit
	// may save the wrong state before the precompile execution, not revert it entirely,
	// potentially causing an infinite mint of funds.
	SmartContract_TestPrecompileSelfCallRevert = CompiledEvmContract{
		Name:      "TestPrecompileSelfCallRevert.sol",
		EmbedJSON: testPrecompileSelfCallRevertJson,
	}
	// SmartContract_TestInfiniteRecursionERC20 is a test contract
	// which simulates malicious ERC20 behavior by adding infinite recursion in transfer() and balanceOf() functions
	SmartContract_TestInfiniteRecursionERC20 = CompiledEvmContract{
		Name:      "TestInfiniteRecursionERC20.sol",
		EmbedJSON: testInfiniteRecursionERC20Json,
	}
	// SmartContract_TestERC20TransferWithFee is a test contract
	// which simulates malicious ERC20 behavior by adding fee to the transfer() function
	SmartContract_TestERC20TransferWithFee = CompiledEvmContract{
		Name:      "TestERC20TransferWithFee.sol",
		EmbedJSON: testERC20TransferWithFee,
	}
	// SmartContract_TestRandom is a test contract which tests random function
	SmartContract_TestRandom = CompiledEvmContract{
		Name:      "TestRandom.sol",
		EmbedJSON: testRandom,
	}
	// SmartContract_TestBytes32Metadata is a test contract which tests contract that have bytes32 as metadata
	SmartContract_TestBytes32Metadata = CompiledEvmContract{
		Name:      "MKR.sol",
		EmbedJSON: testMetadataBytes32,
	}
	// SmartContract_TestPrecompileSendToBankThenERC20Transfer is a test contract that sends to bank then calls ERC20 transfer
	SmartContract_TestPrecompileSendToBankThenERC20Transfer = CompiledEvmContract{
		Name:      "TestPrecompileSendToBankThenERC20Transfer.sol",
		EmbedJSON: testPrecompileSendToBankThenERC20Transfer,
	}
	// SmartContract_TestDirtyStateAttack4 is a test contract that composes manual send and funtoken sendToBank with a reversion
	SmartContract_TestDirtyStateAttack4 = CompiledEvmContract{
		Name:      "TestDirtyStateAttack4.sol",
		EmbedJSON: testDirtyStateAttack4,
	}
	// SmartContract_TestDirtyStateAttack5 is a test contract that calls a wasm contract with 5 NIBI
	SmartContract_TestDirtyStateAttack5 = CompiledEvmContract{
		Name:      "TestDirtyStateAttack5.sol",
		EmbedJSON: testDirtyStateAttack5,
	}
)

func init() {
	SmartContract_ERC20Minter.MustLoad()
	SmartContract_FunToken.MustLoad()
	SmartContract_Wasm.MustLoad()
	SmartContract_Oracle.MustLoad()
	SmartContract_TestERC20.MustLoad()
	SmartContract_TestERC20MaliciousName.MustLoad()
	SmartContract_TestERC20MaliciousTransfer.MustLoad()
	SmartContract_TestFunTokenPrecompileLocalGas.MustLoad()
	SmartContract_TestNativeSendThenPrecompileSendJson.MustLoad()
	SmartContract_TestERC20TransferThenPrecompileSend.MustLoad()
	SmartContract_TestPrecompileSelfCallRevert.MustLoad()
	SmartContract_TestInfiniteRecursionERC20.MustLoad()
	SmartContract_TestERC20TransferWithFee.MustLoad()
	SmartContract_TestRandom.MustLoad()
	SmartContract_TestBytes32Metadata.MustLoad()
	SmartContract_TestPrecompileSendToBankThenERC20Transfer.MustLoad()
	SmartContract_TestDirtyStateAttack4.MustLoad()
	SmartContract_TestDirtyStateAttack5.MustLoad()
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
