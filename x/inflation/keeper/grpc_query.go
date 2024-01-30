package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over q
type querier struct {
	Keeper
}

// NewQuerier returns an implementation of the oracle QueryServer interface
// for the provided Keeper.
func NewQuerier(keeper Keeper) types.QueryServer {
	return &querier{Keeper: keeper}
}

var _ types.QueryServer = querier{}

// Params queries params of distribution module
func (q querier) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)

	return &types.QueryParamsResponse{Params: params}, nil
}

// Period returns the current period of the inflation module.
func (k Keeper) Period(
	c context.Context,
	_ *types.QueryPeriodRequest,
) (*types.QueryPeriodResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	period := k.CurrentPeriod.Peek(ctx)
	return &types.QueryPeriodResponse{Period: period}, nil
}

// EpochMintProvision returns the EpochMintProvision of the inflation module.
func (k Keeper) EpochMintProvision(
	c context.Context,
	_ *types.QueryEpochMintProvisionRequest,
) (*types.QueryEpochMintProvisionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	epochMintProvision := k.GetEpochMintProvision(ctx)
	coin := sdk.NewDecCoinFromDec(denoms.NIBI, epochMintProvision)
	return &types.QueryEpochMintProvisionResponse{EpochMintProvision: coin}, nil
}

// SkippedEpochs returns the number of skipped Epochs of the inflation module.
func (k Keeper) SkippedEpochs(
	c context.Context,
	_ *types.QuerySkippedEpochsRequest,
) (*types.QuerySkippedEpochsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	skippedEpochs := k.NumSkippedEpochs.Peek(ctx)
	return &types.QuerySkippedEpochsResponse{SkippedEpochs: skippedEpochs}, nil
}

// InflationRate returns the inflation rate for the current period.
func (k Keeper) InflationRate(
	c context.Context,
	_ *types.QueryInflationRateRequest,
) (*types.QueryInflationRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	inflationRate := k.GetInflationRate(ctx, denoms.NIBI)
	return &types.QueryInflationRateResponse{InflationRate: inflationRate}, nil
}

// CirculatingSupply returns the total supply in circulation excluding the team
// allocation in the first year
func (k Keeper) CirculatingSupply(
	c context.Context,
	_ *types.QueryCirculatingSupplyRequest,
) (*types.QueryCirculatingSupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	circulatingSupply := k.GetCirculatingSupply(ctx, denoms.NIBI)
	circulatingToDec := sdk.NewDecFromInt(circulatingSupply)
	coin := sdk.NewDecCoinFromDec(denoms.NIBI, circulatingToDec)

	return &types.QueryCirculatingSupplyResponse{CirculatingSupply: coin}, nil
}
