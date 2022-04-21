package keeper

import (
	epochstypes "github.com/NibiruChain/nibiru/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
}

func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) (err error) {
	params := k.GetParams(ctx)
	if epochIdentifier == params.DistrEpochIdentifier {
		err := k.EvaluateCollRatio(ctx)

		if err != nil {
			return err
		}
	}
	return
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
	err := h.k.AfterEpochEnd(ctx, epochIdentifier, epochNumber)
	if err != nil {
		panic(err)
	}
}
