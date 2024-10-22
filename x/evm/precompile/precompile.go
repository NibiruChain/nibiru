// Package precompile implements custom precompiles for the Nibiru EVM.
//
// Precompiles are special, built-in contract interfaces that exist at
// predefined addresses and run custom logic outside of what is possible
// in standard Solidity contracts. This package extends the default Ethereum
// precompiles with Nibiru-specific functionality.
//
// Key components:
//   - InitPrecompiles: Initializes and returns a map of precompiled contracts.
//   - PrecompileFunToken: Implements the FunToken precompile for ERC20-to-bank transfers.
//
// The package also provides utility functions for working with precompiles, such
// as "ABIMethodByID" and "OnRunStart" for common precompile execution setup.
package precompile

import (
	"bytes"
	"fmt"

	"github.com/NibiruChain/collections"
	store "github.com/cosmos/cosmos-sdk/store/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	gethparams "github.com/ethereum/go-ethereum/params"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

// InitPrecompiles initializes and returns a map of precompiled contracts for the EVM.
// It combines default Ethereum precompiles with custom Nibiru precompiles.
//
// Parameters:
//   - k: A keepers.PublicKeepers instance providing access to various blockchain state.
//
// Returns:
//   - A map of Ethereum addresses to PrecompiledContract implementations.
func InitPrecompiles(
	k keepers.PublicKeepers,
) (precompiles map[gethcommon.Address]vm.PrecompiledContract) {
	precompiles = make(map[gethcommon.Address]vm.PrecompiledContract)

	// Default precompiles
	for addr, pc := range vm.PrecompiledContractsBerlin {
		precompiles[addr] = pc
	}

	// Custom precompiles
	for _, precompileSetupFn := range []func(k keepers.PublicKeepers) vm.PrecompiledContract{
		PrecompileFunToken,
		PrecompileWasm,
		PrecompileOracle,
	} {
		pc := precompileSetupFn(k)
		precompiles[pc.Address()] = pc
	}

	// TODO: feat(evm): implement precompiled contracts for ibc transfer
	// Check if there is sufficient demand for this.

	// TODO: feat(evm): implement precompiled contracts for staking
	// Note that liquid staked assets can be a useful alternative to adding a
	// staking precompile.
	// Check if there is sufficient demand for this.

	return precompiles
}

// methodById: Looks up an ABI method by the 4-byte id.
// Copy of "ABI.MethodById" from go-ethereum version > 1.10
func methodById(abi *gethabi.ABI, sigdata []byte) (*gethabi.Method, error) {
	if len(sigdata) != 4 {
		return nil, fmt.Errorf("data (%d bytes) insufficient for abi method lookup", len(sigdata))
	}

	for _, method := range abi.Methods {
		if bytes.Equal(method.ID, sigdata[:4]) {
			return &method, nil
		}
	}

	return nil, fmt.Errorf("no method with id: %#x", sigdata[:4])
}

func DecomposeInput(
	abi *gethabi.ABI, input []byte,
) (method *gethabi.Method, args []interface{}, err error) {
	// ABI method IDs are exactly 4 bytes according to "gethabi.ABI.MethodByID".
	if len(input) < 4 {
		readableBz := collections.HumanizeBytes(input)
		err = fmt.Errorf("input \"%s\" too short to extract method ID (less than 4 bytes)", readableBz)
		return
	}
	method, err = methodById(abi, input[:4])
	if err != nil {
		err = fmt.Errorf("unable to parse ABI method by its 4-byte ID: %w", err)
		return
	}

	args, err = method.Inputs.Unpack(input[4:])
	if err != nil {
		err = fmt.Errorf("unable to unpack input args: %w", err)
		return
	}

	return method, args, nil
}

func RequiredGas(input []byte, abi *gethabi.ABI) uint64 {
	method, _, err := DecomposeInput(abi, input)
	if err != nil {
		// It's appropriate to return a reasonable default here
		// because the error from DecomposeInput will be handled automatically by
		// "Run". In go-ethereum/core/vm/contracts.go, you can see the execution
		// order of a precompile in the "runPrecompiledContract" function.
		return gethparams.TxGas // return reasonable default
	}
	gasCfg := store.KVGasConfig()

	// Map access could panic. We know that it won't panic because all methods
	// are in the map, which is verified by unit tests.
	methodIsTx := precompileMethodIsTxMap[PrecompileMethod(method.Name)]
	var costPerByte, costFlat uint64
	if methodIsTx {
		costPerByte, costFlat = gasCfg.WriteCostPerByte, gasCfg.WriteCostFlat
	} else {
		costPerByte, costFlat = gasCfg.ReadCostPerByte, gasCfg.ReadCostFlat
	}

	argsBzLen := uint64(len(input[4:]))
	return (costPerByte * argsBzLen) + costFlat
}

type OnRunStartResult struct {
	// Args contains the decoded (ABI unpacked) arguments passed to the contract
	// as input.
	Args []any

	// Ctx is a cached SDK context that allows isolated state
	// operations to occur that can be reverted by the EVM's [statedb.StateDB].
	Ctx sdk.Context

	// WriteCtx commits the cached context changes to the parent context.
	WriteCtx func()

	// Method is the ABI method for the precompiled contract call.
	Method *gethabi.Method

	// SnapshotBeforeRun captures the state before precompile execution to enable
	// proper state reversal if the call fails or if [statedb.JournalChange]
	// is reverted in general.
	SnapshotBeforeRun statedb.PrecompileSnapshotBeforeRun
	StateDB           *statedb.StateDB
}

// OnRunStart prepares the execution environment for a precompiled contract call.
// It handles decoding the input data according the to contract ABI, creates an
// isolated cache context for state changes, and sets up a snapshot for potential
// EVM "reverts".
//
// Args:
//   - evm: Instance of the EVM executing the contract
//   - contract: Precompiled contract being called
//   - abi: [gethabi.ABI] defining the contract's invokable methods.
//
// Example Usage:
//
//	```go
//	func (p newPrecompile) Run(
//	  evm *vm.EVM, contract *vm.Contract, readonly bool
//	) (bz []byte, err error) {
//		res, err := OnRunStart(evm, contract, p.ABI())
//		if err != nil {
//		    return nil, err
//		}
//		// ...
//		// Use res.Ctx for state changes
//		// Use res.WriteCtx to commit changes to the multistore
//		// after successful execution
//		res.WriteCtx()
//	}
//	```
func OnRunStart(
	evm *vm.EVM, contract *vm.Contract, abi *gethabi.ABI,
) (res OnRunStartResult, err error) {
	method, args, err := DecomposeInput(abi, contract.Input)
	if err != nil {
		return res, err
	}

	stateDB, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		err = fmt.Errorf("failed to load the sdk.Context from the EVM StateDB")
		return
	}
	cacheCtx, writeCacheCtx, stateDBSnapshot := stateDB.CacheCtxForPrecompile()
	if err = stateDB.CommitContext(cacheCtx); err != nil {
		return res, fmt.Errorf("error committing dirty journal entries for the precompile call to the cache ctx: %w", err)
	}

	return OnRunStartResult{
		Args:              args,
		Ctx:               cacheCtx,
		WriteCtx:          writeCacheCtx,
		Method:            method,
		SnapshotBeforeRun: stateDBSnapshot,
		StateDB:           stateDB,
	}, nil
}

// OnRunEnd finalizes a precompile execution by saving its state snapshot to the
// journal. This ensures that any state changes can be properly reverted if needed.
//
// Args:
//   - stateDB: The EVM state database
//   - snapshot: The state snapshot taken before the precompile executed
//   - precompileAddr: The address of the precompiled contract
//
// The snapshot is critical for maintaining state consistency when:
//   - The operation gets reverted ([statedb.JournalChange] Revert).
//   - The precompile modifies state in other modules (e.g., bank, wasm)
//   - Multiple precompiles are called within a single transaction
func OnRunEnd(
	stateDB *statedb.StateDB,
	snapshot statedb.PrecompileSnapshotBeforeRun,
	precompileAddr gethcommon.Address,
) error {
	return stateDB.SavePrecompileSnapshotToJournal(precompileAddr, snapshot)
}

var precompileMethodIsTxMap map[PrecompileMethod]bool = map[PrecompileMethod]bool{
	WasmMethod_execute:      true,
	WasmMethod_instantiate:  true,
	WasmMethod_executeMulti: true,
	WasmMethod_query:        false,
	WasmMethod_queryRaw:     false,

	FunTokenMethod_BankSend: true,

	OracleMethod_queryExchangeRate: false,
}
