package keeper

import (
	"context"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/mint"
)

// querier implements the module's gRPC "QueryServer" interface
type querier struct {
	Keeper
}

// NewQuerier returns an implementation of the [mint.QueryServer] interface
// for the provided [Keeper].
func NewQuerier(keeper Keeper) mint.QueryServer {
	return &querier{Keeper: keeper}
}

var _ mint.QueryServer = querier{}

// Params is a gRPC query for the module parameters
func (q querier) Params(c context.Context, _ *mint.QueryParamsRequest) (*mint.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)

	return &mint.QueryParamsResponse{Params: params}, nil
}

// Period returns the current period of the inflation module.
func (k Keeper) Period(
	c context.Context,
	_ *mint.QueryPeriodRequest,
) (*mint.QueryPeriodResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	period := k.CurrentPeriod.Peek(ctx)
	return &mint.QueryPeriodResponse{Period: period}, nil
}

// EpochMintProvision returns the EpochMintProvision of the inflation module.
func (k Keeper) EpochMintProvision(
	c context.Context,
	_ *mint.QueryEpochMintProvisionRequest,
) (*mint.QueryEpochMintProvisionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	epochMintProvision := k.GetEpochMintProvision(ctx)
	coin := sdk.NewDecCoinFromDec(appconst.DENOM_UNIBI, epochMintProvision)
	return &mint.QueryEpochMintProvisionResponse{EpochMintProvision: coin}, nil
}

// SkippedEpochs returns the number of skipped Epochs of the inflation module.
func (k Keeper) SkippedEpochs(
	c context.Context,
	_ *mint.QuerySkippedEpochsRequest,
) (*mint.QuerySkippedEpochsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	skippedEpochs := k.NumSkippedEpochs.Peek(ctx)
	return &mint.QuerySkippedEpochsResponse{SkippedEpochs: skippedEpochs}, nil
}

// InflationRate returns the inflation rate for the current period.
func (k Keeper) InflationRate(
	c context.Context,
	_ *mint.QueryInflationRateRequest,
) (*mint.QueryInflationRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	inflationRate := k.GetInflationRate(ctx, appconst.DENOM_UNIBI)
	return &mint.QueryInflationRateResponse{InflationRate: inflationRate}, nil
}

// CirculatingSupply returns the total supply in circulation excluding the team
// allocation in the first year
func (k Keeper) CirculatingSupply(
	c context.Context,
	_ *mint.QueryCirculatingSupplyRequest,
) (*mint.QueryCirculatingSupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	circulatingSupply := k.GetCirculatingSupply(ctx, appconst.DENOM_UNIBI)
	circulatingToDec := sdkmath.LegacyNewDecFromInt(circulatingSupply)
	coin := sdk.NewDecCoinFromDec(appconst.DENOM_UNIBI, circulatingToDec)

	return &mint.QueryCirculatingSupplyResponse{CirculatingSupply: coin}, nil
}
