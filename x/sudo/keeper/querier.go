package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// Ensure the interface is properly implemented at compile time
var _ sudo.QueryServer = (*Keeper)(nil)

func (k Keeper) QuerySudoers(
	goCtx context.Context,
	req *sudo.QuerySudoersRequest,
) (resp *sudo.QuerySudoersResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	sudoers, err := k.Sudoers.Get(ctx)

	return &sudo.QuerySudoersResponse{
		Sudoers: sudoers,
	}, err
}

func (k Keeper) GetZeroGasActors(ctx sdk.Context) sudo.ZeroGasActors {
	return k.ZeroGasActors.GetOr(ctx, sudo.DefaultZeroGasActors())
}

func (k Keeper) QueryZeroGasActors(
	goCtx context.Context,
	_ *sudo.QueryZeroGasActorsRequest,
) (resp *sudo.QueryZeroGasActorsResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &sudo.QueryZeroGasActorsResponse{
		Actors: k.GetZeroGasActors(ctx),
	}, nil
}
