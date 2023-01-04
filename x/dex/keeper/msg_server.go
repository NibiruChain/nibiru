package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/dex/types"
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
CreatePool handles the MsgCreatePool transaction.

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

/*
JoinPool handles the MsgJoinPool transaction.

args

	ctx: the cosmos-sdk context
	msg: a MsgJoinPool proto object

ret

	MsgJoinPoolResponse: the MsgJoinPoolResponse proto object response, containing the pool id number
	error: an error if any occurred
*/
func (k msgServer) JoinPool(ctx context.Context, msg *types.MsgJoinPool) (*types.MsgJoinPoolResponse, error) {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	pool, numSharesOut, remCoins, err := k.Keeper.JoinPool(
		sdkContext,
		sender,
		msg.PoolId,
		msg.TokensIn,
		msg.UseAllCoins,
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
ExitPool handles the MsgExitPool transaction.

args

	ctx: the cosmos-sdk context
	msg: a MsgExitPool proto object

ret

	MsgExitPoolResponse: the MsgExitPoolResponse proto object response, containing the amount of tokens returned to the user
	error: an error if any occurred
*/
func (k msgServer) ExitPool(ctx context.Context, msg *types.MsgExitPool) (*types.MsgExitPoolResponse, error) {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	tokensOut, err := k.Keeper.ExitPool(
		sdkContext,
		sender,
		msg.PoolId,
		msg.PoolShares,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgExitPoolResponse{
		TokensOut: tokensOut,
	}, nil
}

/*
SwapAssets is a Handler for the MsgJoinPool transaction.

args

	ctx: the cosmos-sdk context
	msg: a MsgJoinPool proto object

ret

	MsgJoinPoolResponse: the MsgJoinPoolResponse proto object response, containing the pool id number
	error: an error if any occurred
*/
func (k msgServer) SwapAssets(ctx context.Context, msg *types.MsgSwapAssets) (
	*types.MsgSwapAssetsResponse, error,
) {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	tokenOut, err := k.Keeper.SwapExactAmountIn(
		sdkContext,
		sender,
		msg.PoolId,
		msg.TokenIn,
		msg.TokenOutDenom,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgSwapAssetsResponse{
		TokenOut: tokenOut,
	}, nil
}
