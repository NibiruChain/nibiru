package keeper

import (
	"context"

	types "github.com/NibiruChain/nibiru/x/perp/types/v1"
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
	MsgFooResponse:
	error:
*/
func (k msgServer) MsgFoo(
	goCtx context.Context, msg *types.MsgFoo) (
	fooResponse *types.MsgFooResponse, err error) {
	if err != nil {
		return nil, err
	}

	return fooResponse, nil
}

// Messages

func (k Keeper) Foo(
	goCtx context.Context, msg *types.MsgFoo,
) (res *types.MsgFooResponse, err error) {
	return res, err
}
