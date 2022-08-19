package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

type queryServer struct {
	Keeper
}

func NewQuerier(k Keeper) queryServer {
	return queryServer{Keeper: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) ReserveAssets(
	goCtx context.Context,
	req *types.QueryReserveAssetsRequest,
) (resp *types.QueryReserveAssetsResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	tokenPair, err := common.NewAssetPair(req.Pair)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pool, err := q.getPool(ctx, tokenPair)
	if err != nil {
		return nil, err
	}

	return &types.QueryReserveAssetsResponse{
		BaseAssetReserve:  pool.BaseAssetReserve,
		QuoteAssetReserve: pool.QuoteAssetReserve,
	}, nil
}

func (q queryServer) AllPools(
	goCtx context.Context,
	req *types.QueryAllPoolsRequest,
) (resp *types.QueryAllPoolsResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pools := q.GetAllPools(ctx)
	if err != nil {
		return nil, err
	}
	var pricesForPools []types.PoolPrices
	for _, pool := range pools {
		poolPrices, err := q.GetPoolPrices(ctx, *pool)
		if err != nil {
			return nil, err
		}
		pricesForPools = append(pricesForPools, poolPrices)
	}

	return &types.QueryAllPoolsResponse{
		Pools:  pools,
		Prices: pricesForPools,
	}, nil
}

func (q queryServer) BaseAssetPrice(
	goCtx context.Context,
	req *types.QueryBaseAssetPriceRequest,
) (resp *types.QueryBaseAssetPriceResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pair, err := common.NewAssetPair(req.Pair)
	if err != nil {
		return nil, err
	}

	priceInQuoteDenom, err := q.GetBaseAssetPrice(
		ctx,
		pair,
		req.Direction,
		req.BaseAssetAmount,
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryBaseAssetPriceResponse{
		PriceInQuoteDenom: priceInQuoteDenom,
	}, nil
}
