package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/inflation/types"
)

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	params, _ := k.Params.Get(ctx)
	return params
}

func (k Keeper) GetPolynomialFactors(ctx sdk.Context) (res []sdk.Dec) {
	params, _ := k.Params.Get(ctx)
	return params.PolynomialFactors
}

func (k Keeper) GetInflationDistribution(ctx sdk.Context) (res types.InflationDistribution) {
	params, _ := k.Params.Get(ctx)
	return params.InflationDistribution
}

func (k Keeper) GetInflationEnabled(ctx sdk.Context) (res bool) {
	params, _ := k.Params.Get(ctx)
	return params.InflationEnabled
}

func (k Keeper) GetEpochsPerPeriod(ctx sdk.Context) (res uint64) {
	params, _ := k.Params.Get(ctx)
	return params.EpochsPerPeriod
}

func (k Keeper) GetPeriodsPerYear(ctx sdk.Context) (res uint64) {
	params, _ := k.Params.Get(ctx)
	return params.PeriodsPerYear
}
