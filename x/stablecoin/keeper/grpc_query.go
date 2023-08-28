package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/stablecoin/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(
	goCtx context.Context, req *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k Keeper) ModuleAccountBalances(
	goCtx context.Context, req *types.QueryModuleAccountBalances,
) (*types.QueryModuleAccountBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	var balances sdk.Coins = k.BankKeeper.GetAllBalances(
		ctx, k.AccountKeeper.GetModuleAddress(types.ModuleName),
	)
	return &types.QueryModuleAccountBalancesResponse{
		ModuleAccountBalances: balances,
	}, nil
}

func (k Keeper) CirculatingSupplies(
	goCtx context.Context, req *types.QueryCirculatingSupplies,
) (*types.QueryCirculatingSuppliesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryCirculatingSuppliesResponse{
		Nibi: k.GetSupplyNIBI(ctx),
		Nusd: k.GetSupplyNUSD(ctx),
	}, nil
}

func (k Keeper) LiquidityRatioInfo(
	goCtx context.Context, req *types.QueryLiquidityRatioInfoRequest,
) (res *types.QueryLiquidityRatioInfoResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	liqRatio, err := k.GetLiquidityRatio(ctx)
	if err != nil {
		return res, err
	}
	lowerBand, upperBand, err := k.GetLiquidityRatioBands(ctx)
	if err != nil {
		return res, err
	}

	return &types.QueryLiquidityRatioInfoResponse{
		Info: types.LiquidityRatioInfo{
			LiquidityRatio: liqRatio,
			UpperBand:      upperBand,
			LowerBand:      lowerBand,
		},
	}, nil
}
