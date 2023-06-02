package keeper

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	gogotypes "github.com/gogo/protobuf/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/x/spot/types"
)

type queryServer struct {
	Keeper
}

func NewQuerier(k Keeper) queryServer {
	return queryServer{Keeper: k}
}

var _ types.QueryServer = queryServer{}

/*
Handler for the QueryParamsRequest query.

args

	ctx: the cosmos-sdk context
	req: a QueryParamsRequest proto object

ret

	QueryParamsResponse: the QueryParamsResponse proto object response, containing the params
	error: an error if any occurred
*/
func (k queryServer) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

/*
Handler for the QueryPoolRequest query.

args

	ctx: the cosmos-sdk context
	req: a QueryPoolRequest proto object

ret

	QueryPoolResponse: the QueryPoolResponse proto object response, containing the pool
	error: an error if any occurred
*/
func (k queryServer) Pool(goCtx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	pool, err := k.FetchPool(sdk.UnwrapSDKContext(goCtx), req.PoolId)
	if err != nil {
		return nil, err
	}

	return &types.QueryPoolResponse{
		Pool: &pool,
	}, nil
}

/*
Handler for the QueryPoolNumberRequest query.

args

	ctx: the cosmos-sdk context
	req: a QueryPoolNumberRequest proto object

ret

	QueryPoolNumberResponse: the QueryPoolNumberResponse proto object response, containing the next pool id number
	error: an error if any occurred
*/
func (k queryServer) PoolNumber(goCtx context.Context, req *types.QueryPoolNumberRequest) (*types.QueryPoolNumberResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var poolNumber uint64

	bz := ctx.KVStore(k.storeKey).Get(types.KeyNextGlobalPoolNumber)
	if bz == nil {
		return nil, fmt.Errorf("pool number has not been initialized -- Should have been done in InitGenesis")
	} else {
		val := gogotypes.UInt64Value{}
		k.cdc.MustUnmarshal(bz, &val)
		poolNumber = val.GetValue()
	}

	return &types.QueryPoolNumberResponse{
		PoolId: poolNumber,
	}, nil
}

func (k queryServer) Pools(goCtx context.Context, req *types.QueryPoolsRequest) (
	*types.QueryPoolsResponse, error,
) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := ctx.KVStore(k.Keeper.storeKey)
	poolStore := prefix.NewStore(store, types.KeyPrefixPools)

	pools := []*types.Pool{}
	pageRes, err := query.Paginate(
		poolStore,
		req.Pagination,
		func(key []byte, value []byte) error {
			var pool types.Pool
			err := k.Keeper.cdc.Unmarshal(value, &pool)
			if err != nil {
				return err
			}
			pools = append(pools, &pool)
			return nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPoolsResponse{
		Pools:      pools,
		Pagination: pageRes,
	}, nil
}

// Parameters of a single pool.
func (k queryServer) PoolParams(goCtx context.Context, req *types.QueryPoolParamsRequest) (
	resp *types.QueryPoolParamsResponse, err error,
) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pool, err := k.FetchPool(ctx, req.PoolId)
	if err != nil {
		return nil, err
	}

	return &types.QueryPoolParamsResponse{
		PoolParams: &pool.PoolParams,
	}, nil
}

// Number of pools.
func (k queryServer) NumPools(ctx context.Context, _ *types.QueryNumPoolsRequest) (
	*types.QueryNumPoolsResponse, error,
) {
	nextPoolNumber, err := k.GetNextPoolNumber(sdk.UnwrapSDKContext(ctx))
	if err != nil {
		return nil, err
	}
	return &types.QueryNumPoolsResponse{
		// next pool number is the id of the next pool,
		// so we have one less than that in number of pools (id starts at 1)
		NumPools: nextPoolNumber - 1,
	}, nil
}

// Total liquidity across all pools.
func (k queryServer) TotalLiquidity(ctx context.Context, req *types.QueryTotalLiquidityRequest) (*types.QueryTotalLiquidityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	return &types.QueryTotalLiquidityResponse{
		Liquidity: k.Keeper.GetTotalLiquidity(sdkCtx),
	}, nil
}

// Total liquidity in a single pool.
func (k queryServer) TotalPoolLiquidity(ctx context.Context, req *types.QueryTotalPoolLiquidityRequest) (*types.QueryTotalPoolLiquidityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	pool, err := k.FetchPool(sdkCtx, req.PoolId)

	if err != nil {
		return &types.QueryTotalPoolLiquidityResponse{}, err
	}
	return &types.QueryTotalPoolLiquidityResponse{
		Liquidity: k.bankKeeper.GetAllBalances(sdkCtx, pool.GetAddress()),
	}, nil
}

// Total shares in a single pool.
func (k queryServer) TotalShares(ctx context.Context, req *types.QueryTotalSharesRequest) (
	*types.QueryTotalSharesResponse, error,
) {
	pool, err := k.FetchPool(sdk.UnwrapSDKContext(ctx), req.PoolId)
	if err != nil {
		return nil, err
	}

	return &types.QueryTotalSharesResponse{
		TotalShares: pool.TotalShares,
	}, nil
}

// Instantaneous price of an asset in a pool.
func (k queryServer) SpotPrice(ctx context.Context, req *types.QuerySpotPriceRequest) (
	*types.QuerySpotPriceResponse, error,
) {
	pool, err := k.FetchPool(sdk.UnwrapSDKContext(ctx), req.PoolId)
	if err != nil {
		return nil, err
	}

	price, err := pool.CalcSpotPrice(req.TokenInDenom, req.TokenOutDenom)
	if err != nil {
		return nil, err
	}

	return &types.QuerySpotPriceResponse{
		SpotPrice: price.String(),
	}, nil
}

// Estimates the amount of assets returned given an exact amount of tokens to
// swap.
func (k queryServer) EstimateSwapExactAmountIn(
	ctx context.Context, req *types.QuerySwapExactAmountInRequest,
) (*types.QuerySwapExactAmountInResponse, error) {
	pool, err := k.FetchPool(sdk.UnwrapSDKContext(ctx), req.PoolId)
	if err != nil {
		return nil, err
	}

	tokenOut, fee, err := pool.CalcOutAmtGivenIn(req.TokenIn, req.TokenOutDenom, false)
	if err != nil {
		return nil, err
	}

	return &types.QuerySwapExactAmountInResponse{
		TokenOut: tokenOut,
		Fee:      fee,
	}, nil
}

// Estimates the amount of tokens required to return the exact amount of
// assets requested.
func (k queryServer) EstimateSwapExactAmountOut(
	ctx context.Context, req *types.QuerySwapExactAmountOutRequest,
) (*types.QuerySwapExactAmountOutResponse, error) {
	pool, err := k.FetchPool(sdk.UnwrapSDKContext(ctx), req.PoolId)
	if err != nil {
		return nil, err
	}

	tokenIn, err := pool.CalcInAmtGivenOut(req.TokenOut, req.TokenInDenom)
	if err != nil {
		return nil, err
	}

	return &types.QuerySwapExactAmountOutResponse{
		TokenIn: tokenIn,
	}, nil
}

// Estimates the amount of pool shares returned given an amount of tokens to
// join.
func (k queryServer) EstimateJoinExactAmountIn(
	ctx context.Context, req *types.QueryJoinExactAmountInRequest,
) (*types.QueryJoinExactAmountInResponse, error) {
	pool, err := k.FetchPool(sdk.UnwrapSDKContext(ctx), req.PoolId)
	if err != nil {
		return nil, err
	}
	numShares, remCoins, err := pool.AddTokensToPool(req.TokensIn)
	if err != nil {
		return nil, err
	}
	return &types.QueryJoinExactAmountInResponse{
		PoolSharesOut: numShares,
		RemCoins:      remCoins,
	}, nil
}

// Estimates the amount of tokens required to obtain an exact amount of pool
// shares.
func (k queryServer) EstimateJoinExactAmountOut(context.Context, *types.QueryJoinExactAmountOutRequest) (*types.QueryJoinExactAmountOutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not Implemented")
}

// Estimates the amount of tokens returned to the user given an exact amount
// of pool shares.
func (k queryServer) EstimateExitExactAmountIn(
	ctx context.Context, req *types.QueryExitExactAmountInRequest,
) (*types.QueryExitExactAmountInResponse, error) {
	pool, err := k.FetchPool(sdk.UnwrapSDKContext(ctx), req.PoolId)
	if err != nil {
		return nil, err
	}
	tokensOut, fees, err := pool.ExitPool(req.PoolSharesIn)
	if err != nil {
		return nil, err
	}
	return &types.QueryExitExactAmountInResponse{
		TokensOut: tokensOut,
		Fees:      fees,
	}, nil
}

// Estimates the amount of pool shares required to extract an exact amount of
// tokens from the pool.
func (k queryServer) EstimateExitExactAmountOut(context.Context, *types.QueryExitExactAmountOutRequest) (*types.QueryExitExactAmountOutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not Implemented")
}
