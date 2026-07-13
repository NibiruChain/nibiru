package keeper

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// AfterEpochEnd epoch hook
func (k Keeper) AfterEpochEnd(ctx sdk.Context, identifier string, epochNumber uint64) {
	k.hooks.AfterEpochEnd(ctx, identifier, epochNumber)
}

// BeforeEpochStart epoch hook
func (k Keeper) BeforeEpochStart(ctx sdk.Context, identifier string, epochNumber uint64) {
	k.hooks.BeforeEpochStart(ctx, identifier, epochNumber)
}
