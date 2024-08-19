// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/x/common/omap"
)

func (k *Keeper) AddPrecompiles(
	precompileMap map[gethcommon.Address]vm.PrecompiledContract,
) {
	if k.precompiles.Len() == 0 {
		newPrecompileMap := omap.SortedMap_EthAddress[vm.PrecompiledContract](
			precompileMap,
		)
		k.precompiles = newPrecompileMap
	} else {
		for addr, precompile := range precompileMap {
			k.precompiles.Set(addr, precompile)
		}
	}
}

// IsAvailablePrecompile returns true if the given precompile address is contained in the
// EVM keeper's available precompiles map.
func (k Keeper) IsAvailablePrecompile(address gethcommon.Address) bool {
	return k.precompiles.Has(address)
}
