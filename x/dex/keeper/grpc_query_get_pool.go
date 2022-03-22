package keeper

import (
	"context"

	"github.com/MatrixDao/dex/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetPool(goCtx context.Context, req *types.QueryGetPoolRequest) (*types.QueryGetPoolResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	poolKey := types.GetKeyPrefixPools(req.PoolId)
	bz := store.Get(poolKey)

	var pool types.Pool
	err := pool.Unmarshal(bz)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetPoolResponse{
		Pool: &pool,
	}, nil
}
