package keeper

import (
	"context"
	"fmt"

	"github.com/MatrixDao/dex/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogotypes "github.com/gogo/protobuf/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetPoolNumber(goCtx context.Context, req *types.QueryGetPoolNumberRequest) (*types.QueryGetPoolNumberResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var poolNumber uint64

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.KeyNextGlobalPoolNumber)
	if bz == nil {
		k.Logger(ctx).Error("Could not get pool number. Not initialized.")
		panic(fmt.Errorf("pool number has not been initialized -- Should have been done in InitGenesis"))
	} else {
		val := gogotypes.UInt64Value{}

		err := k.cdc.Unmarshal(bz, &val)
		if err != nil {
			panic(err)
		}

		poolNumber = val.GetValue()
	}

	return &types.QueryGetPoolNumberResponse{
		PoolId: poolNumber,
	}, nil
}
