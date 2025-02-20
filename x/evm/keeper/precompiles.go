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
		k.precompiles = omap.SortedMap_EthAddress(
			precompileMap,
		)
	} else {
		for addr, precompile := range precompileMap {
			k.precompiles.Set(addr, precompile)
		}
	}
}

func (k *Keeper) IsPrecompile(addr gethcommon.Address) bool {
	return k.precompiles.Has(addr)
}
