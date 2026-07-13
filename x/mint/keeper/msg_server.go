package keeper

import (
	"context"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/mint"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the inflation MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) mint.MsgServer {
	return &msgServer{Keeper: keeper}
}

// EditInflationParams: gRPC tx msg for editing the inflation module params.
// [SUDO] Only callable by sudoers.
func (ms msgServer) EditInflationParams(
	goCtx context.Context, msg *mint.MsgEditInflationParams,
) (resp *mint.MsgEditInflationParamsResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Stateless field validation was already performed in msg.ValidateBasic()
	// before the current scope is reached.
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	err = ms.Sudo().EditInflationParams(ctx, *msg, sender)

	resp = &mint.MsgEditInflationParamsResponse{}
	return resp, err
}

// ToggleInflation: gRPC tx msg for enabling or disabling token inflation.
// [SUDO] Only callable by sudoers.
func (ms msgServer) ToggleInflation(
	goCtx context.Context, msg *mint.MsgToggleInflation,
) (resp *mint.MsgToggleInflationResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Stateless field validation was already performed in msg.ValidateBasic()
	// before the current scope is reached.
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	err = ms.Sudo().ToggleInflation(ctx, msg.Enable, sender)
	resp = &mint.MsgToggleInflationResponse{}
	return resp, err
}
