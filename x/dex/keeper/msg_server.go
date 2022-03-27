package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/dex/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

/*
Handler for the MsgCreatePool transaction.

args
  ctx: the cosmos-sdk context
  msg: a MsgCreatePool proto object

ret
  MsgCreatePoolResponse: the MsgCreatePoolResponse proto object response, containing the pool id number
  error: an error if any occurred
*/
func (k msgServer) CreatePool(goCtx context.Context, msg *types.MsgCreatePool) (*types.MsgCreatePoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	poolId, err := k.NewPool(ctx, sender, *msg.PoolParams, msg.PoolAssets)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreatePoolResponse{
		PoolId: poolId,
	}, nil
}
