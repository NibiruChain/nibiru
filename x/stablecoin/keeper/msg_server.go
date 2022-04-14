package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
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
