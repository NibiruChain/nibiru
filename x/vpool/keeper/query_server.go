package keeper

import (
	"context"

	"github.com/NibiruChain/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

type queryServer struct {
	k Keeper
}

func NewQuerier(k Keeper) types.QueryServer {
	return queryServer{k: k}
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

	pool, err := q.k.Pools.Get(ctx, req.Pair)
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

	var pools []types.Vpool
	var pricesForPools []types.PoolPrices
	for _, pool := range q.k.Pools.Iterate(ctx, collections.Range[asset.Pair]{}).Values() {
		poolPrices, err := q.k.GetPoolPrices(ctx, pool)
		if err != nil {
			return nil, err
		}

		pricesForPools = append(pricesForPools, poolPrices)
		pools = append(pools, pool)
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

	priceInQuoteDenom, err := q.k.GetBaseAssetPrice(
		ctx,
		req.Pair,
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
