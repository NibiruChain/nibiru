package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/stablecoin/types"
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
Args:
	goCtx

Returns
	MsgMintStableResponse:
	error:
*/
func (k msgServer) MsgMintStable(
	goCtx context.Context, msg *types.MsgMintStable) (
	*types.MsgMintStableResponse, error) {
	mintStableResponse, err := k.MintStable(goCtx, msg)
	if err != nil {
		return nil, err
	}

	return mintStableResponse, nil
}

func (k msgServer) MsgBurnStable(
	goCtx context.Context, msg *types.MsgBurnStable) (
	*types.MsgBurnStableResponse, error) {
	burnStableResponse, err := k.BurnStable(goCtx, msg)
	if err != nil {
		return nil, err
	}
	return burnStableResponse, nil
}

func (k msgServer) MsgRecollateralize(
	goCtx context.Context, msg *types.MsgRecollateralize,
) (*types.MsgRecollateralizeResponse, error) {
	response, err := k.Recollateralize(goCtx, msg)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (k msgServer) MsgBuyback(
	goCtx context.Context, msg *types.MsgBuyback,
) (*types.MsgBuybackResponse, error) {
	response, err := k.Buyback(goCtx, msg)
	if err != nil {
		return nil, err
	}
	return response, nil
}
