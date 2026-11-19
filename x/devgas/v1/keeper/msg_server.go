package keeper

import (
	"context"

	"github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

var _ types.MsgServer = &Keeper{}

// The x/devgas transaction messages are disabled. Each handler returns
// [types.ErrDevgasDeprecated] without reading or writing state. The message
// types stay registered so that historical transactions still decode.

func (k Keeper) RegisterFeeShare(
	goCtx context.Context,
	msg *types.MsgRegisterFeeShare,
) (*types.MsgRegisterFeeShareResponse, error) {
	return nil, types.ErrDevgasDeprecated
}

func (k Keeper) UpdateFeeShare(
	goCtx context.Context,
	msg *types.MsgUpdateFeeShare,
) (*types.MsgUpdateFeeShareResponse, error) {
	return nil, types.ErrDevgasDeprecated
}

func (k Keeper) CancelFeeShare(
	goCtx context.Context,
	msg *types.MsgCancelFeeShare,
) (*types.MsgCancelFeeShareResponse, error) {
	return nil, types.ErrDevgasDeprecated
}

func (k Keeper) UpdateParams(
	goCtx context.Context, req *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	return nil, types.ErrDevgasDeprecated
}
