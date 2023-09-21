package keeper

import (
	"context"

	types "github.com/NibiruChain/nibiru/x/tokenfactory/types"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the keeper with functions for gRPC queries.
type Querier struct {
	Keeper
}

func (k Keeper) Querier() Querier {
	return Querier{
		Keeper: k,
	}
}

func (q Querier) Params(
	goCtx context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params, err := q.Keeper.Store.ModuleParams.Get(ctx)
	if err != nil {
		return nil, grpcstatus.Errorf(
			grpccodes.NotFound,
			"failed to query module params",
		)
	}
	return &types.QueryParamsResponse{Params: params}, nil
}
