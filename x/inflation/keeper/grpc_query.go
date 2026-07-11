package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/inflation"
)

// querier implements the module's gRPC "QueryServer" interface
type querier struct {
	Keeper
}

// NewQuerier returns an implementation of the [inflation.QueryServer] interface
// for the provided [Keeper].
func NewQuerier(keeper Keeper) inflation.QueryServer {
	return &querier{Keeper: keeper}
}

var _ inflation.QueryServer = querier{}

// Params is a gRPC query for the module parameters
func (q querier) Params(c context.Context, _ *inflation.QueryParamsRequest) (*inflation.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)

	return &inflation.QueryParamsResponse{Params: params}, nil
}

// Period returns the current period of the inflation module.
func (k Keeper) Period(
	c context.Context,
	_ *inflation.QueryPeriodRequest,
) (*inflation.QueryPeriodResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	period := k.CurrentPeriod.Peek(ctx)
	return &inflation.QueryPeriodResponse{Period: period}, nil
}

// EpochMintProvision returns the EpochMintProvision of the inflation module.
func (k Keeper) EpochMintProvision(
	c context.Context,
	_ *inflation.QueryEpochMintProvisionRequest,
) (*inflation.QueryEpochMintProvisionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	epochMintProvision := k.GetEpochMintProvision(ctx)
	coin := sdk.NewDecCoinFromDec(appconst.DENOM_UNIBI, epochMintProvision)
	return &inflation.QueryEpochMintProvisionResponse{EpochMintProvision: coin}, nil
}

// SkippedEpochs returns the number of skipped Epochs of the inflation module.
func (k Keeper) SkippedEpochs(
	c context.Context,
	_ *inflation.QuerySkippedEpochsRequest,
) (*inflation.QuerySkippedEpochsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	skippedEpochs := k.NumSkippedEpochs.Peek(ctx)
	return &inflation.QuerySkippedEpochsResponse{SkippedEpochs: skippedEpochs}, nil
}

// InflationRate returns the inflation rate for the current period.
func (k Keeper) InflationRate(
	c context.Context,
	_ *inflation.QueryInflationRateRequest,
) (*inflation.QueryInflationRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	inflationRate := k.GetInflationRate(ctx, appconst.DENOM_UNIBI)
	return &inflation.QueryInflationRateResponse{InflationRate: inflationRate}, nil
}

// CirculatingSupply returns the total supply in circulation excluding the team
// allocation in the first year
func (k Keeper) CirculatingSupply(
	c context.Context,
	_ *inflation.QueryCirculatingSupplyRequest,
) (*inflation.QueryCirculatingSupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	circulatingSupply := k.GetCirculatingSupply(ctx, appconst.DENOM_UNIBI)
	circulatingToDec := sdkmath.LegacyNewDecFromInt(circulatingSupply)
	coin := sdk.NewDecCoinFromDec(appconst.DENOM_UNIBI, circulatingToDec)

	return &inflation.QueryCirculatingSupplyResponse{CirculatingSupply: coin}, nil
}
