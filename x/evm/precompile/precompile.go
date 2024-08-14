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
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/collections"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/common/set"
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
	initMutex.Lock()
	defer initMutex.Unlock()

	precompiles = make(map[gethcommon.Address]vm.PrecompiledContract)

	// Default precompiles
	for addr, pc := range vm.PrecompiledContractsBerlin {
		precompiles[addr] = pc
	}

	// Custom precompiles
	for _, precompileSetupFn := range []func(k keepers.PublicKeepers) vm.PrecompiledContract{
		PrecompileFunToken,
	} {
		pc := precompileSetupFn(k)
		addPrecompileToVM(pc)
		precompiles[pc.Address()] = pc
	}
	return precompiles
}

// initMutex: Mutual exclusion lock (mutex) to prevent race conditions with
// consecutive calls of InitPrecompiles.
var initMutex = &sync.Mutex{}

// addPrecompileToVM adds a precompiled contract to the EVM's set of recognized
// precompiles. It updates both the contract map and the list of precompile
// addresses for the latest major upgrade or hard fork of Ethereum (Berlin).
func addPrecompileToVM(p vm.PrecompiledContract) {
	addr := p.Address()

	vm.PrecompiledContractsBerlin[addr] = p
	// TODO: 2024-07-05 feat: Cancun after go-ethereum upgrade
	// https://github.com/NibiruChain/nibiru/issues/1921
	// vm.PrecompiledContractsCancun,

	// Done if the precompiled contracts are already added
	// This check is only relevant during tests to prevent races. The iteration
	// doesn't get repeated in production.
	vmSet := set.New(vm.PrecompiledAddressesBerlin...)
	if vmSet.Has(addr) {
		return
	}

	vm.PrecompiledAddressesBerlin = append(vm.PrecompiledAddressesBerlin, addr)
	// TODO: 2024-07-05 feat: Cancun after go-ethereum upgrade
	// https://github.com/NibiruChain/nibiru/issues/1921
	// vm.PrecompiledAddressesCancun,
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

func OnRunStart(
	abi *gethabi.ABI, evm *vm.EVM, input []byte,
) (ctx sdk.Context, method *gethabi.Method, args []interface{}, err error) {
	// 1 | Get context from StateDB
	stateDB, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		err = fmt.Errorf("failed to load the sdk.Context from the EVM StateDB")
		return
	}
	ctx = stateDB.GetContext()

	// 2 | Parse the ABI method
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

	return ctx, method, args, nil
}
