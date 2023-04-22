package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// GetParams returns the total set of inflation parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return
}

// SetParams sets the inflation params in a single key
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

// VotePeriod returns the number of blocks during which voting takes place.
func (k Keeper) ExponentialCalculation(ctx sdk.Context) (res types.ExponentialCalculation) {
	k.paramSpace.Get(ctx, types.KeyExponentialCalculation, &res)
	return
}

// VoteThreshold returns the minimum percentage of votes that must be received for a ballot to pass.
func (k Keeper) InflationDistribution(ctx sdk.Context) (res types.InflationDistribution) {
	k.paramSpace.Get(ctx, types.KeyInflationDistribution, &res)
	return
}

// VoteThreshold returns the minimum percentage of votes that must be received for a ballot to pass.
func (k Keeper) InflationEnabled(ctx sdk.Context) (res bool) {
	k.paramSpace.Get(ctx, types.KeyInflationEnabled, &res)
	return
}

func (k Keeper) EpochsPerPeriod(ctx sdk.Context) (res uint64) {
	k.paramSpace.Get(ctx, types.KeyEpochsPerPeriod, &res)
	return
}
