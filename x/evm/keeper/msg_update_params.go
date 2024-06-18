// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/evm/types"
)

func (k *Keeper) UpdateParams(
	goCtx context.Context, req *types.MsgUpdateParams,
) (resp *types.MsgUpdateParamsResponse, err error) {
	if k.authority.String() != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, expected %s, got %s", k.authority.String(), req.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, req.Params)
	return &types.MsgUpdateParamsResponse{}, nil
}
