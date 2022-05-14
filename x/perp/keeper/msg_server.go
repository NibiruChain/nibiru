package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/x/perp/types"
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
	MsgRemoveMarginResponse:
	error:
*/
func (k msgServer) MsgRemoveMargin(goCtx context.Context, msg *types.MsgRemoveMargin,
) (*types.MsgRemoveMarginResponse, error) {
	removeMarginResponse, err := k.RemoveMargin(goCtx, msg)
	if err != nil {
		return nil, err
	}

	return removeMarginResponse, nil
}

func (k msgServer) MsgAddMargin(goCtx context.Context, msg *types.MsgAddMargin,
) (*types.MsgAddMarginResponse, error) {
	removeMarginResponse, err := k.AddMargin(goCtx, msg)
	if err != nil {
		return nil, err
	}

	return removeMarginResponse, nil
}
