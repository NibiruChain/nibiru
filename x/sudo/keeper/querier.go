package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/sudo/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure the interface is properly implemented at compile time
var _ types.QueryServer = Querier{}

type Querier struct {
	keeper Keeper
}

func NewQuerier(k Keeper) types.QueryServer {
	return Querier{keeper: k}
}

func (q Querier) QuerySudoers(
	goCtx context.Context,
	req *types.QuerySudoersRequest,
) (resp *types.QuerySudoersResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	sudoers, err := q.keeper.Sudoers.Get(ctx)

	return &types.QuerySudoersResponse{
		Sudoers: sudoers,
	}, err
}
