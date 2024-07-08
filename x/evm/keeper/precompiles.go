// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"bytes"
	"fmt"
	"maps"
	"sort"

	"github.com/NibiruChain/nibiru/x/common/set"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// PrecompileSet is the set of all known precompile addresses. It includes defaults
// from go-ethereum and the custom ones specific to the Nibiru EVM.
func (k Keeper) PrecompileSet() set.Set[gethcommon.Address] {
	precompiles := set.New[gethcommon.Address]()
	for addr := range k.precompiles {
		precompiles.Add(addr)
	}
	return precompiles
}

func (k *Keeper) AddPrecompiles(precompileMap map[gethcommon.Address]vm.PrecompiledContract) {
	if len(k.precompiles) == 0 {
		k.precompiles = make(map[gethcommon.Address]vm.PrecompiledContract)
	}
	for addr, precompile := range precompileMap {
		k.precompiles[addr] = precompile
	}
	// The following TODOs can go in an epic together.

	// TODO: feat(evm): implement precompiled contracts for fungible tokens
	// https://github.com/NibiruChain/nibiru/issues/1898

	// TODO: feat(evm): implement precompiled contracts for ibc transfer
	// Check if there is sufficient demand for this.

	// TODO: feat(evm): implement precompiled contracts for staking
	// Note that liquid staked assets can be a useful alternative to adding a
	// staking precompile.
	// Check if there is sufficient demand for this.

	// TODO: feat(evm): implement precompiled contracts for wasm calls
	// Check if there is sufficient demand for this.
}

// AddEVMExtensions adds the given precompiles to the list of active precompiles in the EVM parameters
// and to the available precompiles map in the Keeper. This function returns an error if
// the precompiles are invalid or duplicated.
func (k *Keeper) AddEVMExtensions(
	ctx sdk.Context, precompiles ...vm.PrecompiledContract,
) error {
	params := k.GetParams(ctx)

	// precompileAddrs := make([]string, len(precompiles))
	precompileAddrs := set.New[string]()
	precompilesMap := maps.Clone(k.precompiles)

	for _, precompile := range precompiles {
		// add to active precompiles
		address := precompile.Address()
		precompileAddrs.Add(address.String())

		// add to available precompiles, but check for duplicates
		if _, ok := precompilesMap[address]; ok {
			return fmt.Errorf("precompile already registered: %s", address)
		}
		precompilesMap[address] = precompile
	}

	params.ActivePrecompiles = append(params.ActivePrecompiles, precompileAddrs.ToSlice()...)

	// NOTE: the active precompiles are sorted and validated before setting them
	// in the params
	k.SetParams(ctx, params)
	// update the pointer to the map with the newly added EVM Extensions
	k.precompiles = precompilesMap
	return nil
}

// IsAvailablePrecompile returns true if the given precompile address is contained in the
// EVM keeper's available precompiles map.
func (k Keeper) IsAvailablePrecompile(address gethcommon.Address) bool {
	_, ok := k.precompiles[address]
	return ok
}

// PrecompileAddrsSorted returns the list of available precompile addresses.
//
// NOTE: uses index based approach instead of append because it's supposed to be faster.
// Check https://stackoverflow.com/questions/21362950/getting-a-slice-of-keys-from-a-map.
func (k Keeper) PrecompileAddrsSorted() []gethcommon.Address {
	addresses := make([]gethcommon.Address, len(k.precompiles))
	i := 0

	//#nosec G705 -- two operations in for loop here are fine
	for address := range k.precompiles {
		addresses[i] = address
		i++
	}

	sort.Slice(addresses, func(i, j int) bool {
		return bytes.Compare(addresses[i].Bytes(), addresses[j].Bytes()) == -1
	})

	return addresses
}
