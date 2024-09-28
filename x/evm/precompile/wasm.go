package precompile

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
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

// Wasm: A struct embedding keepers for read and write operations in Wasm like
// execute, instantiate, and queries.
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

type precompileWasm struct {
	Wasm Wasm
}

func (p precompileWasm) Address() gethcommon.Address {
	return PrecompileAddr_Wasm
}

func (p precompileWasm) RequiredGas(input []byte) (gasPrice uint64) {
	return gethparams.TxGas
	// TODO: Lower gas requirement for queries in comparison to txs
}

// TODO: solidity: document params in Wasm.sol

// Run runs the precompiled contract
func (p precompileWasm) Run(
	evm *vm.EVM, contract *vm.Contract, readonly bool,
) (bz []byte, err error) {
	// This is a `defer` pattern to add behavior that runs in the case that the error is
	// non-nil, creating a concise way to add extra information.
	defer func() {
		if err != nil {
			precompileType := reflect.TypeOf(p).Name()
			err = fmt.Errorf("precompile error: failed to run %s: %w", precompileType, err)
		}
	}()

	method, args, err := DecomposeInput(embeds.SmartContract_Wasm.ABI, contract.Input)
	if err != nil {
		return nil, err
	}

	stateDB, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		err = fmt.Errorf("failed to load the sdk.Context from the EVM StateDB")
		return
	}
	ctx := stateDB.GetContext()

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
		bz, err = p.queryRaw(ctx, contract.CallerAddress, method, args, readonly)
	default:
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}

	return // TODO: ...
}

// TODO: docs
func (p precompileWasm) execute(
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
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
		err = fmt.Errorf("Execute failed: %w", err)
		return
	}
	return method.Outputs.Pack(data)
}

// TODO: impl wasm method
// TODO: test happy path
// TODO: docs
func (p precompileWasm) query(
	ctx sdk.Context,
	method *gethabi.Method,
	args []any,
	contract *vm.Contract,
) (bz []byte, err error) {
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
		err = fmt.Errorf("Query failed: %w", err)
		return
	}
	fmt.Printf("respBz: (%s)\n", respBz)
	return method.Outputs.Pack(respBz)
}

// TODO: docs
func (p precompileWasm) instantiate(
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
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
		err = fmt.Errorf("Instantiate failed: %w", err)
		return
	}

	return method.Outputs.Pack(contractAddr.String(), data)
}

// TODO: impl wasm method
// TODO: test happy path
// TODO: docs
func (p precompileWasm) executeMulti(
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
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
			err = fmt.Errorf("Execute failed: %w", e)
			return
		}
		responses = append(responses, respBz)
	}
	return method.Outputs.Pack(responses)
}

func (p precompileWasm) queryRaw(
	ctx sdk.Context,
	caller gethcommon.Address,
	method *gethabi.Method,
	args []any,
	readOnly bool,
) (bz []byte, err error) {
	// TODO: impl wasm method
	return
}
