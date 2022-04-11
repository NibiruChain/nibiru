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

	sender, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	poolId, err := k.NewPool(sdk.UnwrapSDKContext(goCtx), sender, *msg.PoolParams, msg.PoolAssets)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreatePoolResponse{
		PoolId: poolId,
	}, nil
}

/*
Handler for the MsgJoinPool transaction.

args
  ctx: the cosmos-sdk context
  msg: a MsgJoinPool proto object

ret
  MsgJoinPoolResponse: the MsgJoinPoolResponse proto object response, containing the pool id number
  error: an error if any occurred
*/
func (k msgServer) JoinPool(ctx context.Context, msg *types.MsgJoinPool) (*types.MsgJoinPoolResponse, error) {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	pool, numSharesOut, remCoins, err := k.JoinPoolNoSwap(
		sdk.UnwrapSDKContext(ctx),
		sender,
		msg.PoolId,
		msg.TokensIn,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgJoinPoolResponse{
		Pool:             &pool,
		NumPoolSharesOut: numSharesOut,
		RemainingCoins:   remCoins,
	}, nil
}

/*
Handler for the MsgJoinPool transaction.

args
  ctx: the cosmos-sdk context
  msg: a MsgJoinPool proto object

ret
  MsgJoinPoolResponse: the MsgJoinPoolResponse proto object response, containing the pool id number
  error: an error if any occurred
*/
func (k msgServer) ExitPool(ctx context.Context, msg *types.MsgExitPool) (*types.MsgExitPoolResponse, error) {
	// TODO(https://github.com/MatrixDao/matrix/issues/46) implement this

	return &types.MsgExitPoolResponse{}, nil
}
