package precompile

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	evmkeeper "github.com/NibiruChain/nibiru/v2/x/evm/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

var _ vm.PrecompiledContract = (*precompileWasm)(nil)

// Precompile address for "Wasm.sol",
var PrecompileAddr_Wasm = gethcommon.HexToAddress("0x0000000000000000000000000000000000000802")

// Contract methods from Wasm.sol
const (
	WasmMethod_execute      PrecompileMethod = "execute"
	WasmMethod_query        PrecompileMethod = "query"
	WasmMethod_instantiate  PrecompileMethod = "instantiate"
	WasmMethod_executeMulti PrecompileMethod = "executeMulti"
	WasmMethod_queryRaw     PrecompileMethod = "queryRaw"
)

// Run runs the precompiled contract
func (p precompileWasm) Run(
	evm *vm.EVM, contract *vm.Contract, readonly bool,
) (bz []byte, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()
	startResult, err := OnRunStart(evm, contract.Input, p.ABI(), contract.Gas)
	if err != nil {
		return nil, err
	}

	// Gracefully handles "out of gas"
	defer HandleOutOfGasPanic(&err)()

	// NOTE: The NibiruBankKeeper needs to reference the current [vm.StateDB] before
	// any operation that has the potential to use Bank send methods. This will
	// guarantee that [evmkeeper.Keeper.SetAccBalance] journal changes are
	// recorded if wei (NIBI) is transferred.
	p.Bank.StateDB = startResult.StateDB
	switch PrecompileMethod(startResult.Method.Name) {
	case WasmMethod_execute:
		bz, err = p.execute(startResult, contract.CallerAddress, readonly)
	case WasmMethod_query:
		bz, err = p.query(startResult, contract)
	case WasmMethod_instantiate:
		bz, err = p.instantiate(startResult, contract.CallerAddress, readonly)
	case WasmMethod_executeMulti:
		bz, err = p.executeMulti(startResult, contract.CallerAddress, readonly)
	case WasmMethod_queryRaw:
		bz, err = p.queryRaw(startResult, contract)
	default:
		// Note that this code path should be impossible to reach since
		// "[decomposeInput]" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", startResult.Method.Name)
		return
	}
	if err != nil {
		return nil, err
	}

	// Gas consumed by a local gas meter
	// The reason it's unnecessary to check for a success value is because
	// GasConsumed is guaranteed to be less than the contract.Gas because the gas
	// meter was initialized....
	contract.UseGas(startResult.CacheCtx.GasMeter().GasConsumed())
	return bz, err
}

type precompileWasm struct {
	*evmkeeper.Keeper
	Wasm Wasm
}

func (p precompileWasm) Address() gethcommon.Address {
	return PrecompileAddr_Wasm
}

// RequiredGas calculates the cost of calling the precompile in gas units.
func (p precompileWasm) RequiredGas(input []byte) (gasCost uint64) {
	return requiredGas(input, p.ABI())
}

func (p precompileWasm) ABI() *gethabi.ABI {
	return embeds.SmartContract_Wasm.ABI
}

// Wasm: A struct embedding keepers for read and write operations in Wasm, such
// as execute, query, and instantiate.
type Wasm struct {
	*wasmkeeper.PermissionedKeeper
	wasmkeeper.Keeper
}

func PrecompileWasm(keepers keepers.PublicKeepers) vm.PrecompiledContract {
	return precompileWasm{
		Keeper: keepers.EvmKeeper,
		Wasm: Wasm{
			wasmkeeper.NewDefaultPermissionKeeper(keepers.WasmKeeper),
			keepers.WasmKeeper,
		},
	}
}

// execute invokes a Wasm contract's "ExecuteMsg", which corresponds to
// "wasm/types/MsgExecuteContract". This enables arbitrary smart contract
// execution using the Wasm VM from the EVM.
//
// Implements "execute" from evm/embeds/contracts/Wasm.sol:
//
//	```solidity
//	 function execute(
//	   string memory contractAddr,
//	   bytes memory msgArgs,
//	   BankCoin[] memory funds
//	 ) payable external returns (bytes memory response);
//	```
//
// Contract Args:
//   - contractAddr: nibi-prefixed Bech32 address of the wasm contract
//   - msgArgs: JSON encoded wasm execute invocation
//   - funds: Optional funds to supply during the execute call. It's
//     uncommon to use this field, so you'll pass an empty array most of the time.
func (p precompileWasm) execute(
	start OnRunStartResult,
	caller gethcommon.Address,
	readOnly bool,
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertNotReadonlyTx(readOnly, method); err != nil {
		return nil, err
	}

	wasmContract, msgArgsBz, funds, err := p.parseArgsWasmExecute(args)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}
	data, err := p.Wasm.Execute(ctx, wasmContract, eth.EthAddrToNibiruAddr(caller), msgArgsBz, funds)
	if err != nil {
		return
	}
	return method.Outputs.Pack(data)
}

// query runs a smart query. In Rust, this is the "WasmQuery::Smart" variant.
// In protobuf/gRPC, it's type URL is
// "/cosmwasm.wasm.v1.Query/SmartContractState".
//
// Implements "query" from evm/embeds/contracts/Wasm.sol:
//
//	```solidity
//	function query(
//	  string memory contractAddr,
//	  bytes memory req
//	) external view returns (bytes memory response);
//	```
func (p precompileWasm) query(
	start OnRunStartResult,
	contract *vm.Contract,
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}

	wasmContract, req, err := p.parseArgsWasmQuery(args)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}
	respBz, err := p.Wasm.QuerySmart(ctx, wasmContract, req)
	if err != nil {
		return
	}
	return method.Outputs.Pack(respBz)
}

// instantiate creates a new instance of a Wasm smart contract for some code id.
//
// Implements "instantiate" from evm/embeds/contracts/Wasm.sol:
//
//	```solidity
//	/// @notice InstantiateContract creates a new smart contract instance for the given code id.
//	/// @param admin The address of the contract admin (optional, can be empty string)
//	/// @param codeID The ID of the code to instantiate
//	/// @param msgArgs JSON encoded instantiation message
//	/// @param label A human-readable label for the contract
//	/// @param funds Optional funds to send to the contract upon instantiation
//	function instantiate(
//	  string memory admin,
//	  uint64 codeID,
//	  bytes memory msgArgs,
//	  string memory label,
//	  BankCoin[] memory funds
//	) payable external returns (string memory contractAddr, bytes memory data);
//	```
func (p precompileWasm) instantiate(
	start OnRunStartResult,
	caller gethcommon.Address,
	readOnly bool,
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertNotReadonlyTx(readOnly, method); err != nil {
		return nil, err
	}

	callerBech32 := eth.EthAddrToNibiruAddr(caller)
	txMsg, err := p.parseArgsWasmInstantiate(args, callerBech32.String())
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}

	var adminAddr sdk.AccAddress
	if len(txMsg.Admin) > 0 {
		adminAddr = sdk.MustAccAddressFromBech32(txMsg.Admin) // validated in parse
	}
	contractAddr, data, err := p.Wasm.Instantiate(
		ctx, txMsg.CodeID, callerBech32, adminAddr, txMsg.Msg, txMsg.Label, txMsg.Funds,
	)
	if err != nil {
		return
	}

	return method.Outputs.Pack(contractAddr.String(), data)
}

// executeMulti allows executing multiple Wasm contract calls in a single transaction.
// It corresponds to the "executeMulti" method in the IWasm interface.
//
// Implements "executeMulti" from evm/embeds/contracts/Wasm.sol:
//
//	```solidity
//	/// @notice Identical to "execute", except for multiple contract calls.
//	/// @param executeMsgs An array of WasmExecuteMsg structs, each containing:
//	///   - contractAddr: nibi-prefixed Bech32 address of the wasm contract
//	///   - msgArgs: JSON encoded wasm execute invocation
//	///   - funds: Optional funds to supply during the execute call
//	function executeMulti(
//	  WasmExecuteMsg[] memory executeMsgs
//	) payable external returns (bytes[] memory responses);
//	```
func (p precompileWasm) executeMulti(
	start OnRunStartResult,
	caller gethcommon.Address,
	readOnly bool,
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertNotReadonlyTx(readOnly, method); err != nil {
		return nil, err
	}

	wasmExecMsgs, err := p.parseArgsWasmExecuteMulti(args)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}
	callerBech32 := eth.EthAddrToNibiruAddr(caller)

	var responses [][]byte
	for i, m := range wasmExecMsgs {
		wasmContract, e := sdk.AccAddressFromBech32(m.ContractAddr)
		if e != nil {
			err = fmt.Errorf("Execute failed at index %d: %w", i, e)
			return
		}
		msgArgsCopy := wasm.RawContractMessage(m.MsgArgs)
		if e := msgArgsCopy.ValidateBasic(); e != nil {
			err = fmt.Errorf("Execute failed at index %d: error parsing msg args: %w", i, e)
			return
		}
		var funds sdk.Coins
		for _, fund := range m.Funds {
			funds = append(funds, sdk.Coin{
				Denom:  fund.Denom,
				Amount: sdk.NewIntFromBigInt(fund.Amount),
			})
		}
		respBz, e := p.Wasm.Execute(ctx, wasmContract, callerBech32, m.MsgArgs, funds)
		if e != nil {
			err = fmt.Errorf("Execute failed at index %d: %w", i, e)
			return
		}
		responses = append(responses, respBz)
	}
	return method.Outputs.Pack(responses)
}

// queryRaw queries the raw key-value store of a Wasm contract. This implements
// the 'queryRaw' method of Wasm.sol:
//
//	```solidity
//	function queryRaw(
//	  string memory contractAddr,
//	  bytes memory key
//	) external view returns (bytes memory response);
//	```
//
// Parameters:
//   - ctx: The SDK context for the query
//   - method: The ABI method being called
//   - args: The arguments passed to the method
//   - contract: The EVM contract context
//
// Returns:
//   - bz: The encoded raw data stored at the queried key
//   - err: Any error that occurred during the query
func (p precompileWasm) queryRaw(
	start OnRunStartResult,
	contract *vm.Contract,
) (bz []byte, err error) {
	method, args, ctx := start.Method, start.Args, start.CacheCtx
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}

	if e := assertNumArgs(args, 2); e != nil {
		err = e
		return
	}

	argIdx := 0
	wasmContract, e := parseArgContractAddr(args[argIdx])
	if e != nil {
		err = e
		return
	}

	argIdx++
	key, ok := args[argIdx].([]byte)
	if !ok {
		err = ErrArgTypeValidation("bytes req", args[argIdx])
		return
	}

	respBz := p.Wasm.QueryRaw(ctx, wasmContract, []byte(key))
	return method.Outputs.Pack(respBz)
}
