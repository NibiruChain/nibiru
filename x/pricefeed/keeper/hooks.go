package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) AfterEpochEnd(ctx sdk.Context) error {
	err := k.UpdateTWAPPrices(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) BeforeEpochStart(ctx sdk.Context) {
}

// ___________________________________________________________________________________________________

// Hooks wrapper struct for pricefeed keeper.
type Hooks struct {
	k Keeper
}

// Return the wrapper struct.
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// epochs hooks.
func (h Hooks) BeforeEpochStart(ctx sdk.Context) {
	h.k.BeforeEpochStart(ctx)
}

func (h Hooks) AfterEpochEnd(ctx sdk.Context) {
	err := h.k.AfterEpochEnd(ctx)
	if err != nil {
		panic(err)
	}
}
