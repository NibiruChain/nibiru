package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/inflation/types"
)

func (k Keeper) UpdateParams(ctx sdk.Context, params types.Params) {
	k.Params.Set(ctx, params)
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	params, _ := k.Params.Get(ctx)
	return params
}

func (k Keeper) PolynomialFactors(ctx sdk.Context) (res []sdk.Dec) {
	params, _ := k.Params.Get(ctx)
	return params.PolynomialFactors
}

func (k Keeper) InflationDistribution(ctx sdk.Context) (res types.InflationDistribution) {
	params, _ := k.Params.Get(ctx)
	return params.InflationDistribution
}

func (k Keeper) InflationEnabled(ctx sdk.Context) (res bool) {
	params, _ := k.Params.Get(ctx)
	return params.InflationEnabled
}

func (k Keeper) EpochsPerPeriod(ctx sdk.Context) (res uint64) {
	params, _ := k.Params.Get(ctx)
	return params.EpochsPerPeriod
}

func (k Keeper) PeriodsPerYear(ctx sdk.Context) (res uint64) {
	params, _ := k.Params.Get(ctx)
	return params.PeriodsPerYear
}
