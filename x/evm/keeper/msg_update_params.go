// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

func (k *Keeper) UpdateParams(
	goCtx context.Context, req *evm.MsgUpdateParams,
) (resp *evm.MsgUpdateParamsResponse, err error) {
	if k.authority.String() != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, expected %s, got %s", k.authority.String(), req.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	err = k.SetParams(ctx, req.Params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to set params")
	}
	return &evm.MsgUpdateParamsResponse{}, nil
}
