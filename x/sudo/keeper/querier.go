package keeper

import (
	"context"
	"github.com/NibiruChain/nibiru/x/sudo/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure the interface is properly implemented at compile time
var _ types.QueryServer = Keeper{}

func (k Keeper) QuerySudoers(
	goCtx context.Context,
	req *types.QuerySudoersRequest,
) (resp *types.QuerySudoersResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	sudoers, err := k.Sudoers.Get(ctx)

	return &types.QuerySudoersResponse{
		Sudoers: sudoers,
	}, err
}
