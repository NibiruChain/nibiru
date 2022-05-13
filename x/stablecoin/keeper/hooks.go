package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
)

func (k Keeper) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
}

func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	params := k.GetParams(ctx)
	if epochIdentifier == params.DistrEpochIdentifier {
		err := k.EvaluateCollRatio(ctx)

		params = k.GetParams(ctx)
		params.IsCollateralRatioValid = err == nil

		k.SetParams(ctx, params)
	}
}

// ___________________________________________________________________________________________________

// Hooks wrapper struct for incentives keeper.
type Hooks struct {
	k Keeper
}

var _ epochstypes.EpochHooks = Hooks{}

// Return the wrapper struct.
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// epochs hooks.
func (h Hooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	h.k.BeforeEpochStart(ctx, epochIdentifier, epochNumber)
}

func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	h.k.AfterEpochEnd(ctx, epochIdentifier, epochNumber)
}
