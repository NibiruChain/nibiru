package keeper

import (
	"context"
	"fmt"

	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogotypes "github.com/gogo/protobuf/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	pool := k.FetchPool(sdk.UnwrapSDKContext(goCtx), req.PoolId)

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
		panic(fmt.Errorf("pool number has not been initialized -- Should have been done in InitGenesis"))
	} else {
		val := gogotypes.UInt64Value{}
		k.cdc.MustUnmarshal(bz, &val)
		poolNumber = val.GetValue()
	}

	return &types.QueryPoolNumberResponse{
		PoolId: poolNumber,
	}, nil
}
