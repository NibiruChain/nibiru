package precompile

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
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
	res, err := OnRunStart(evm, contract, p.ABI())
	if err != nil {
		return nil, err
	}
	method, args, ctx := res.Method, res.Args, res.Ctx

	switch PrecompileMethod(method.Name) {
	case WasmMethod_execute:
		bz, err = p.execute(ctx, contract.CallerAddress, method, args, readonly)
	case WasmMethod_query:
		bz, err = p.query(ctx, method, args, contract)
	case WasmMethod_instantiate:
		bz, err = p.instantiate(ctx, contract.CallerAddress, method, args, readonly)
	case WasmMethod_executeMulti:
		bz, err = p.executeMulti(ctx, contract.CallerAddress, method, args, readonly)
	case WasmMethod_queryRaw:
		bz, err = p.queryRaw(ctx, method, args, contract)
	default:
		// Note that this code path should be impossible to reach since
		// "DecomposeInput" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	if err != nil {
		return nil, err
	}

	if err := OnRunEnd(res.StateDB, res.SnapshotBeforeRun, p.Address()); err != nil {
		return nil, err
	}
	res.WriteCtx()
	return
}

type precompileWasm struct {
	Wasm Wasm
}

func (p precompileWasm) Address() gethcommon.Address {
	return PrecompileAddr_Wasm
}

func (p precompileWasm) ABI() *gethabi.ABI {
	return embeds.SmartContract_Wasm.ABI
}

// RequiredGas calculates the cost of calling the precompile in gas units.
func (p precompileWasm) RequiredGas(input []byte) (gasCost uint64) {
	return RequiredGas(input, p.ABI())
}

// Wasm: A struct embedding keepers for read and write operations in Wasm, such
// as execute, query, and instantiate.
type Wasm struct {
	*wasmkeeper.PermissionedKeeper
	wasmkeeper.Keeper
}

func PrecompileWasm(keepers keepers.PublicKeepers) vm.PrecompiledContract {
	return precompileWasm{
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
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()

	if err := assertNotReadonlyTx(readOnly, true); err != nil {
		return bz, err
	}
	wasmContract, msgArgs, funds, err := p.parseExecuteArgs(args)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}
	callerBech32 := eth.EthAddrToNibiruAddr(caller)
	data, err := p.Wasm.Execute(ctx, wasmContract, callerBech32, msgArgs, funds)
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
	ctx sdk.Context,
	method *gethabi.Method,
	args []any,
	contract *vm.Contract,
) (bz []byte, err error) {
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}
	wasmContract, req, err := p.parseQueryArgs(args)
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
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertNotReadonlyTx(readOnly, true); err != nil {
		return bz, err
	}

	callerBech32 := eth.EthAddrToNibiruAddr(caller)
	txMsg, err := p.parseInstantiateArgs(args, callerBech32.String())
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
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertNotReadonlyTx(readOnly, true); err != nil {
		return bz, err
	}

	wasmExecMsgs, err := p.parseExecuteMultiArgs(args)
	if err != nil {
		err = ErrInvalidArgs(err)
		return
	}
	callerBech32 := eth.EthAddrToNibiruAddr(caller)

	var responses [][]byte
	for _, m := range wasmExecMsgs {
		wasmContract, e := sdk.AccAddressFromBech32(m.ContractAddr)
		if e != nil {
			err = fmt.Errorf("Execute failed: %w", e)
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
			err = e
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
	ctx sdk.Context,
	method *gethabi.Method,
	args []any,
	contract *vm.Contract,
) (bz []byte, err error) {
	defer func() {
		if err != nil {
			err = ErrMethodCalled(method, err)
		}
	}()
	if err := assertContractQuery(contract); err != nil {
		return bz, err
	}

	if e := assertNumArgs(len(args), 2); e != nil {
		err = e
		return
	}

	argIdx := 0
	wasmContract, e := parseContractAddrArg(args[argIdx])
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
