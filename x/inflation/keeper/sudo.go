package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	inflationtypes "github.com/NibiruChain/nibiru/x/inflation/types"
)

// Sudo extends the Keeper with sudo functions. See [x/sudo].
//
// These sudo functions should:
// 1. Not be called in other methods in the module.
// 2. Only be callable by the x/sudo root or sudo contracts.
//
// The intention behind "[Keeper.Sudo]" is to make it more obvious to the
// developer that an unsafe function is being used when it's called.
// [x/sudo]: https://pkg.go.dev/github.com/NibiruChain/nibiru@v1.1.0/x/sudo/keeper
func (k Keeper) Sudo() sudoExtension { return sudoExtension{k} }

type sudoExtension struct{ Keeper }

// EditInflationParams performs a partial struct update, or struct merge, on the
// module parameters, given a subset of the params, `newParams`. Only the new
// params are overwritten.
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
	k.Params.Set(ctx, paramsAfter)
	return paramsAfter.Validate()
}

// ToggleInflation disables (pauses) or enables (unpauses) inflation.
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
	if enabled && !params.HasInflationStarted {
		params.HasInflationStarted = true
	}

	k.Params.Set(ctx, params)
	return
}

// MergeInflationParams: Performs a partial struct update using [partial] and
// merges its params into the existing [inflationParams], keeping any existing
// values that are not set in the partial. For use with
// [Keeper.EditInflationParams].
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
