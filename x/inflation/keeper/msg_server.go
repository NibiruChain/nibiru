package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/inflation/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the inflation MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// EditOracleParams: gRPC tx msg for editing the inflation module params.
// [SUDO] Only callable by sudoers.
func (ms msgServer) EditInflationParams(
	goCtx context.Context, msg *types.MsgEditInflationParams,
) (resp *types.MsgEditInflationParamsResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Stateless field validation is already performed in msg.ValidateBasic()
	// before the current scope is reached.
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	err = ms.Sudo().EditInflationParams(ctx, *msg, sender)

	resp = &types.MsgEditInflationParamsResponse{}
	return resp, err
}

func (ms msgServer) ToggleInflation(
	goCtx context.Context, msg *types.MsgToggleInflation,
) (resp *types.MsgToggleInflationResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Stateless field validation is already performed in msg.ValidateBasic()
	// before the current scope is reached.
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	err = ms.Sudo().ToggleInflation(ctx, msg.Enable, sender)
	resp = &types.MsgToggleInflationResponse{}
	return resp, err
}
