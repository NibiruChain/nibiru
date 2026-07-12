package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/mint"
)

func (k Keeper) GetParams(ctx sdk.Context) mint.Params {
	params, _ := k.Params.Get(ctx)
	return params
}

func (k Keeper) GetPolynomialFactors(ctx sdk.Context) (res []sdkmath.LegacyDec) {
	params, _ := k.Params.Get(ctx)
	return params.PolynomialFactors
}

func (k Keeper) GetInflationDistribution(ctx sdk.Context) (res mint.InflationDistribution) {
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
