// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlock sets the sdk Context and EIP155 chain id to the Keeper.
func (k *Keeper) BeginBlock(ctx sdk.Context, _ abci.RequestBeginBlock) {
	// TODO: feat(evm): impl BeginBlock
	// Is it necessary to set a local variable, or can we use ctx everywhere?
	// k.SetEvmChainID(ctx)
}

// EndBlock also retrieves the bloom filter value from the transient store and commits it to the
// KVStore. The EVM end block logic doesn't update the validator set, thus it returns
// an empty slice.
func (k *Keeper) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	// TODO: Do we care about bloom here?
	return []abci.ValidatorUpdate{}
}
