package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
)

// Sudo extends the Keeper with sudo functions. See sudo.go. Sudo is syntactic
// sugar to separate admin calls off from the other Keeper methods.
//
// These Sudo functions should:
// 1. Not be called in other methods in the x/perp module.
// 2. Only be callable by the x/sudo root or sudo contracts.
//
// The intention behind "Keeper.Sudo()" is to make it more obvious to the
// developer that an unsafe function is being used when it's called.
func (k Keeper) Sudo() sudoExtension { return sudoExtension{k} }

type sudoExtension struct{ Keeper }

// ------------------------------------------------------------------
// Admin.EditInflationParams

func (k sudoExtension) EditInflationParams(
	ctx sdk.Context, newParams inflationtypes.MsgEditInflationParams,
	sender sdk.AccAddress,
) (err error) {
	if err = k.sudoKeeper.CheckPermissions(sender, ctx); err != nil {
		return
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to read inflation params", err)
	}

	paramsAfter, err := MergeInflationParams(newParams, params)
	if err != nil {
		return
	}
	k.UpdateParams(ctx, paramsAfter)
	return paramsAfter.Validate()
}

func (k sudoExtension) ToggleInflation(
	ctx sdk.Context, enabled bool, sender sdk.AccAddress,
) (err error) {
	if err = k.sudoKeeper.CheckPermissions(sender, ctx); err != nil {
		return
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return
	}

	params.InflationEnabled = enabled
	k.UpdateParams(ctx, params)
	return
}

// MergeInflationParams: Takes the given Inflation params and merges them into the
// existing partial params, keeping any existing values that are not set in the
// partial.
func MergeInflationParams(
	partial inflationtypes.MsgEditInflationParams,
	inflationParams inflationtypes.Params,
) (inflationtypes.Params, error) {
	if partial.PolynomialFactors != nil {
		inflationParams.PolynomialFactors = partial.PolynomialFactors
	}

	if partial.InflationDistribution != nil {
		inflationParams.InflationDistribution = *partial.InflationDistribution
	}

	if partial.EpochsPerPeriod != nil {
		inflationParams.EpochsPerPeriod = partial.EpochsPerPeriod.Uint64()
	}
	if partial.PeriodsPerYear != nil {
		inflationParams.PeriodsPerYear = partial.PeriodsPerYear.Uint64()
	}
	if partial.MaxPeriod != nil {
		inflationParams.MaxPeriod = partial.MaxPeriod.Uint64()
	}

	return inflationParams, inflationParams.Validate()
}
